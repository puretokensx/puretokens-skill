import assert from "node:assert/strict";
import { test } from "node:test";
import { execFile as callback } from "node:child_process";
import { promisify } from "node:util";
import { mkdir, mkdtemp, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

const execFile = promisify(callback);
const installer = path.join(repositoryRoot, "runtime/puretokens-skill-install.sh");

test("TRAE installation selects the existing edition without creating a second skill directory", async t => {
  for (const edition of [".trae", ".trae-cn"]) {
    const home = await mkdtemp(path.join(os.tmpdir(), "pt-trae-home-"));
    t.after(() => rm(home, { recursive: true, force: true }));
    await mkdir(path.join(home, edition));
    const { stdout } = await execFile("sh", [installer, "locate", "--host", "trae"], {
      env: { ...process.env, HOME: home }
    });
    assert.equal(stdout.trim(), path.join(home, edition, "skills"));
  }
});

test("TRAE with both edition roots needs an explicit target instead of guessing", async t => {
  const home = await mkdtemp(path.join(os.tmpdir(), "pt-trae-home-"));
  t.after(() => rm(home, { recursive: true, force: true }));
  await mkdir(path.join(home, ".trae"));
  await mkdir(path.join(home, ".trae-cn"));
  await assert.rejects(execFile("sh", [installer, "locate", "--host", "trae"], {
    env: { ...process.env, HOME: home }
  }), /explicit.*target|--target/);
});

test("PowerShell TRAE installer selects the domestic root and rejects ambiguous editions", async t => {
  try {
    await execFile("pwsh", ["-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"]);
  } catch (error) {
    if (error.code === "ENOENT") { t.skip("PowerShell unavailable on this host"); return; }
    throw error;
  }
  const home = await mkdtemp(path.join(os.tmpdir(), "pt-trae-ps-home-"));
  t.after(() => rm(home, { recursive: true, force: true }));
  await mkdir(path.join(home, ".trae-cn"));
  const script = path.join(repositoryRoot, "runtime/puretokens-skill-install.ps1");
  const args = ["-NoProfile", "-File", script, "locate", "-Host", "trae"];
  const env = { ...process.env, USERPROFILE: home };
  const { stdout } = await execFile("pwsh", args, { env });
  assert.equal(stdout.trim(), path.join(home, ".trae-cn", "skills"));
  await mkdir(path.join(home, ".trae"));
  await assert.rejects(execFile("pwsh", args, { env }), /explicit.*target|-Target/);
});
