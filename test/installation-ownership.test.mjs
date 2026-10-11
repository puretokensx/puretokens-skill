import assert from "node:assert/strict";
import { test } from "node:test";
import { execFile as callback } from "node:child_process";
import { promisify } from "node:util";
import { mkdir, mkdtemp, readFile, readdir, rm, writeFile, cp } from "node:fs/promises";
import { createHash } from "node:crypto";
import path from "node:path";
import os from "node:os";
import { repositoryRoot } from "../scripts/skill-registry.mjs";
import { getManagedSkillProvenances } from "../scripts/media-skill-provenance.mjs";
const execFile = promisify(callback);
const installer = path.join(repositoryRoot, "runtime/puretokens-skill-install.sh");
async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-ownership-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const target = path.join(root, "skills");
  const install = source => execFile("sh", [installer, "sync", "--target", target, "--source", source ?? repositoryRoot]);
  return { root, target, install };
}
test("upgrade preserves unmanaged same-name Skill and added or modified files", async t => {
  const f = await fixture(t);
  const dir = path.join(f.target, "puretokens-image");
  await mkdir(dir, { recursive: true });
  await writeFile(path.join(dir, "SKILL.md"), "# Independent Skill");
  await writeFile(path.join(dir, "skill.json"), JSON.stringify({ name: "puretokens-image" }, null, 2));
  await assert.rejects(f.install(), /ownership/);
  assert.equal(await readFile(path.join(dir, "SKILL.md"), "utf8"), "# Independent Skill");
  await rm(dir, { recursive: true });
  await f.install();
  for (const file of ["personal.txt", "SKILL.md"]) {
    const destination = path.join(dir, file);
    const original = await readFile(destination).catch(() => null);
    await writeFile(destination, "keep my local work");
    await assert.rejects(f.install(), /ownership/);
    assert.equal(await readFile(destination, "utf8"), "keep my local work");
    if (original) await writeFile(destination, original); else await rm(destination);
  }
  await f.install();
  assert.equal((await readdir(f.target)).some(name => name.includes("stage")), false);
});
test("interrupted update recovery preserves user changes and retains its backup", async t => {
  const f = await fixture(t);
  await f.install();
  const dir = path.join(f.target, "puretokens-image");
  const stage = path.join(f.target, ".puretokens-skill-stage.user-change");
  await mkdir(path.join(stage, "backup"), { recursive: true });
  await cp(dir, path.join(stage, "backup/puretokens-image"), { recursive: true });
  await writeFile(path.join(stage, "transaction-v1"), "");
  await writeFile(path.join(stage, "plan"), "replace puretokens-image\n");
  await writeFile(path.join(dir, "personal.txt"), "keep");
  await assert.rejects(f.install(), /recovery failed/);
  assert.equal(await readFile(path.join(dir, "personal.txt"), "utf8"), "keep");
  assert.ok(await readFile(path.join(stage, "backup/puretokens-image/SKILL.md")));
});
test("all managed provenance entries derive from the registry", async () => {
  const entries = await getManagedSkillProvenances();
  assert.equal(entries.length, 8);
  for (const entry of entries) {
    assert.match(entry.name, /^puretokens-/);
    assert.match(entry.sourceSha256, /^[a-f0-9]{64}$/);
  }
});

test("installer activation fragments match their single source and unsupported hosts have no examples", async () => {
  await execFile(process.execPath,[path.join(repositoryRoot,"scripts/sync-install-activation.mjs"),"--check"]);
  const registry=JSON.parse(await readFile(path.join(repositoryRoot,"references/install-activation.json"),"utf8"));
  assert.equal(registry.hosts.trae.apiSupport,"unsupported");
  assert.deepEqual(registry.hosts.trae.examples,[]);
  assert.match(registry.hosts.zcode.activation,/Refresh and enable/);
});

test("six-Skill upgrade adds evaluation and preserves a conflicting user-owned Skill", async t => {
  const f = await fixture(t);
  await f.install();
  const evaluation = path.join(f.target, "puretokens-evaluate");
  await rm(evaluation, { recursive: true });
  await rm(path.join(f.target, "puretokens-audio"), { recursive: true });
  const executor = path.join(f.target, ".puretokens-executor", "puretokens-api");
  // Model a verified previous release: six Skill inventories plus its executor.
  const entries = (await readdir(f.target)).filter(name => name.startsWith("puretokens-") || name === ".puretokens-executor");
  for (const name of entries) {
    const directory = path.join(f.target, name);
    const metadataFile = path.join(directory, name === ".puretokens-executor" ? "runtime.json" : "skill.json");
    const metadata = JSON.parse(await readFile(metadataFile, "utf8"));
    metadata.version = "0.18.6";
    await writeFile(metadataFile, JSON.stringify(metadata, null, 2) + "\n");
    const { stdout } = await execFile(executor, ["install-inventory", "--directory", directory, "--name", name]);
    await writeFile(path.join(directory, ".puretokens-managed.json"), stdout);
  }
  assert.equal(entries.length, 7);
  await mkdir(evaluation);
  await writeFile(path.join(evaluation, "SKILL.md"), "# My independent evaluation Skill");
  await assert.rejects(f.install(), /ownership/);
  assert.equal(await readFile(path.join(evaluation, "SKILL.md"), "utf8"), "# My independent evaluation Skill");
  assert.equal(JSON.parse(await readFile(path.join(f.target, "puretokens-image/skill.json"), "utf8")).version, "0.18.6");
  await rm(evaluation, { recursive: true });
  await f.install();
  const current = JSON.parse(await readFile(path.join(repositoryRoot, "package.json"), "utf8")).version;
  assert.equal(JSON.parse(await readFile(path.join(evaluation, "skill.json"), "utf8")).version, current);
  await execFile("sh", [installer, "verify-installed", "--target", f.target]);
  await rm(path.join(evaluation, "SKILL.md"));
  await assert.rejects(execFile("sh", [installer, "verify-installed", "--target", f.target]), /missing the managed Skill: puretokens-evaluate/);
  await assert.rejects(readFile(path.join(evaluation, "SKILL.md")), { code: "ENOENT" });
});


test("native installation ignores retired directories and preserves their bytes", async t => {
 const f=await fixture(t);
 for(const name of ["puretokens_image","puretokens_workbuddy_router",".puretokens-runtime"]){
  await mkdir(path.join(f.target,name),{recursive:true});
  await writeFile(path.join(f.target,name,"personal.txt"),"untouched");
 }
 await f.install();
 for(const name of ["puretokens_image","puretokens_workbuddy_router",".puretokens-runtime"]){
  assert.equal(await readFile(path.join(f.target,name,"personal.txt"),"utf8"),"untouched");
 }
 await f.install();
});

test("seven-Skill installation adds audio without replacing user-owned audio content", async t => {
 const f=await fixture(t);
 await f.install();
 const audio=path.join(f.target,"puretokens-audio");
 await rm(audio,{recursive:true});
 await mkdir(audio);
 await writeFile(path.join(audio,"SKILL.md"),"# User audio instructions");
 await assert.rejects(f.install(),/ownership/);
 assert.equal(await readFile(path.join(audio,"SKILL.md"),"utf8"),"# User audio instructions");
 await rm(audio,{recursive:true});
 await f.install();
 await execFile("sh",[installer,"verify-installed","--target",f.target]);
 assert.equal(JSON.parse(await readFile(path.join(audio,"skill.json"),"utf8")).name,"puretokens-audio");
 await rm(path.join(audio,"SKILL.md"));
 await assert.rejects(execFile("sh",[installer,"verify-installed","--target",f.target]),/missing the managed Skill: puretokens-audio/);
});
