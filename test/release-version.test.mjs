import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import { validateReleaseVersion } from "../scripts/validate-release-version.mjs";

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-version-test-"));
  t.after(() => rm(root, {recursive: true, force: true}));
  const git = args => execFileSync("git", args, {cwd: root, stdio: "pipe"});
  git(["init", "-q"]);
  const write = async (file, value) => {
    await mkdir(path.dirname(path.join(root, file)), {recursive: true});
    await writeFile(path.join(root, file), value);
  };
  const version = value => write("package.json", JSON.stringify({version: value}));
  const commit = () => {
    git(["add", "."]);
    git(["-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture"]);
  };
  await version("0.18.0");
  await write("skills/puretokens-image/SKILL.md", "first");
  commit();
  await version("0.18.1");
  commit();
  return {root, git, write, version, commit};
}

test("unchanged committed versions and non-distribution tests stay valid", async t => {
  const f = await fixture(t);
  assert.equal((await validateReleaseVersion(f.root)).status, "immutable");
  await f.write("runtime/executor/probe_test.go", "fixture");
  await f.write("test/probe.mjs", "fixture");
  f.commit();
  assert.equal((await validateReleaseVersion(f.root)).status, "immutable");
});

test("dirty same-version distribution changes require a version increase", async t => {
  const f = await fixture(t);
  await f.write("skills/puretokens-image/SKILL.md", "changed");
  await assert.rejects(validateReleaseVersion(f.root), /without a version increase/);
  await f.version("0.18.2");
  assert.equal((await validateReleaseVersion(f.root)).status, "candidate");
  f.commit();
  assert.equal((await validateReleaseVersion(f.root)).status, "immutable");
});

test("untracked and deleted distribution files are not invisible to the gate", async t => {
  const f = await fixture(t);
  await f.write("skills/puretokens-image/new_test.go", "this is distributed, not an executor test");
  await assert.rejects(validateReleaseVersion(f.root), /without a version increase/);
  await rm(path.join(f.root, "skills/puretokens-image/new_test.go"));
  await rm(path.join(f.root, "skills/puretokens-image/SKILL.md"));
  await assert.rejects(validateReleaseVersion(f.root), /without a version increase/);
});

test("license changes require a distribution version increase", async t => {
  const f = await fixture(t);
  await f.write("LICENSE", "MIT License fixture\n");
  await assert.rejects(validateReleaseVersion(f.root), /without a version increase/);
});

test("a same-version change remains invalid even after reverting its content", async t => {
  const f = await fixture(t);
  await f.write("skills/puretokens-image/SKILL.md", "changed");
  f.commit();
  await assert.rejects(validateReleaseVersion(f.root), /within its history/);
  await f.write("skills/puretokens-image/SKILL.md", "first");
  f.commit();
  await assert.rejects(validateReleaseVersion(f.root), /within its history/);
  await f.version("0.18.2");
  f.commit();
  assert.equal((await validateReleaseVersion(f.root)).status, "immutable");
});

test("version regression is rejected for dirty and committed candidates", async t => {
  const f = await fixture(t);
  await f.version("0.17.0");
  await assert.rejects(validateReleaseVersion(f.root), /must not decrease/);
  f.commit();
  await assert.rejects(validateReleaseVersion(f.root), /did not increase/);
});

test("missing or shallow Git history cannot prove version immutability", async t => {
  const f = await fixture(t);
  const shallow = path.join(f.root, "shallow");
  f.git(["clone", "-q", "--depth=1", `file://${f.root}`, shallow]);
  await assert.rejects(validateReleaseVersion(shallow), /complete Git history/);
  const empty = path.join(f.root, "empty");
  await mkdir(empty);
  await assert.rejects(validateReleaseVersion(empty), /complete Git history/);
});
