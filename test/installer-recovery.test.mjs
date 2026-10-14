import assert from "node:assert/strict";
import { test } from "node:test";
import { execFile as execFileCallback } from "node:child_process";
import { promisify } from "node:util";
import { mkdtemp, mkdir, readFile, readdir, rm, writeFile, chmod, rename, cp, stat, symlink, lstat } from "node:fs/promises";
import { createHash } from "node:crypto";
import path from "node:path";
import os from "node:os";
import { repositoryRoot } from "../scripts/skill-registry.mjs";
const execFile = promisify(execFileCallback);
const names = ["puretokens-balance", "puretokens-connection", "puretokens-models", "puretokens-image", "puretokens-video", "puretokens-audio", "puretokens-evaluate", "puretokens-update"];
const installer = path.join(repositoryRoot, "runtime/puretokens-skill-install.sh");
async function fixture(t) {
 const root = await mkdtemp(path.join(os.tmpdir(), "pt-installer-test-"));
 t.after(() => rm(root, { recursive: true, force: true }));
 const target = path.join(root, "target");
 const home = path.join(root, "home"); await mkdir(home);
 return { root, target, env: { ...process.env, HOME: home, CODEX_HOME: path.join(home, ".codex") } };
}
async function install(f, env = f.env, more = []) {
 return execFile("sh", [installer, "sync", "--target", f.target, ...more], { cwd: repositoryRoot, env });
}
async function waitForFile(file) {
 const deadline = Date.now() + 8000;
 while (Date.now() < deadline) {
  try { await readFile(file); return; } catch (error) { if (error.code !== "ENOENT") throw error; }
  await new Promise(resolve => setTimeout(resolve, 10));
 }
 assert.fail(`Timed out waiting for fixture marker ${path.basename(file)}`);
}
test("OpenCode directory selection honors explicit and XDG roots without probing other directories", async t => {
 const f = await fixture(t);
 const xdg = path.join(f.root, "xdg config");
 const custom = path.join(f.root, "opencode custom");
 for (const [overrides, expected] of [
  [{}, path.join(f.env.HOME, ".config/opencode/skills")],
  [{ XDG_CONFIG_HOME: xdg }, path.join(xdg, "opencode/skills")],
  [{ XDG_CONFIG_HOME: xdg, OPENCODE_CONFIG_DIR: custom }, path.join(custom, "skills")]
 ]) {
  const env = { ...f.env, XDG_CONFIG_HOME: "", OPENCODE_CONFIG_DIR: "", ...overrides };
  const { stdout } = await execFile("sh", [installer, "locate", "--host", "opencode"], { env });
  assert.equal(stdout.trim(), expected);
 }
 for (const key of ["XDG_CONFIG_HOME", "OPENCODE_CONFIG_DIR"]) {
  await assert.rejects(execFile("sh", [installer, "locate", "--host", "opencode"], {
   env: { ...f.env, XDG_CONFIG_HOME: "", OPENCODE_CONFIG_DIR: "", [key]: "relative" }
  }));
 }
});
test("desktop hosts locate and install into isolated local skill roots", async t => {
 const f = await fixture(t);
 for (const [host, key] of [["claude-desktop", "CLAUDE_CONFIG_DIR"], ["dsh-desktop", "DSH_HOME"], ["deepseek-harness", "DSH_HOME"]]) {
  const directory = path.join(f.root, `${host} with spaces`);
  const env = { ...f.env, [key]: directory };
  const { stdout: located } = await execFile("sh", [installer, "locate", "--host", host], { env });
  assert.equal(located.trim(), path.join(directory, "skills"));
  await execFile("sh", [installer, "sync", "--host", host], { env });
  for (const name of names) {
   const manifest = JSON.parse(await readFile(path.join(directory, "skills", name, "skill.json"), "utf8"));
   assert.ok(manifest.supportedClients.includes(host));
  }
  assert.equal((await readdir(path.join(directory, "skills"))).filter(x => x === ".puretokens-executor").length, 1);
  await assert.rejects(execFile("sh", [installer, "locate", "--host", host], { env: { ...env, [key]: "relative" } }));
 }
 const harnessDefault = await execFile("sh", [installer, "locate", "--host", "deepseek-harness"], { env: { ...f.env, DSH_HOME: "" } });
 assert.equal(harnessDefault.stdout.trim(), path.join(f.env.HOME, ".dsh", "skills"));
 await assert.rejects(execFile("sh", [installer, "locate", "--host", "deepseek-harness"], { env: { ...f.env, DSH_HOME: path.join(f.root, "safe") + "/../unsafe" } }));
 // Default paths are platform-owned; no synthetic connection is opened.
 const env = { ...f.env, CLAUDE_CONFIG_DIR: "", DSH_HOME: "" };
 const { stdout } = await execFile("sh", [installer, "locate", "--host", "claude-desktop"], { env });
 assert.equal(stdout.trim(), path.join(env.HOME, ".claude", "skills"));
 if (process.platform === "darwin") {
  const { stdout } = await execFile("sh", [installer, "locate", "--host", "dsh-desktop"], { env });
  assert.equal(stdout.trim(), path.join(env.HOME, "Library/Application Support/dsh-desktop/harness/skills"));
 } else {
  await assert.rejects(execFile("sh", [installer, "locate", "--host", "dsh-desktop"], { env }), /local macOS or Windows/);
 }
});
test("ZCode honors the data base and installs all managed components", async t => {
 const f = await fixture(t);
 const base = path.join(f.root, "zcode data with spaces");
 const env = { ...f.env, ZCODE_DATA_BASE_DIR: base };
 const target = path.join(base, ".zcode", "skills");
 const { stdout } = await execFile("sh", [installer, "locate", "--host", "zcode"], { env });
 assert.equal(stdout.trim(), target);
 await execFile("sh", [installer, "sync", "--host", "zcode"], { env });
 for (const name of names) assert.ok(JSON.parse(await readFile(path.join(target, name, "skill.json"), "utf8")).supportedClients.includes("zcode"));
 const executor = path.join(target, ".puretokens-executor", "puretokens-api");
 await execFile(executor, ["--version"]);
 await execFile("sh", [installer, "sync", "--host", "zcode"], { env });
 await assert.rejects(execFile("sh", [installer, "locate", "--host", "zcode"], { env: { ...env, ZCODE_DATA_BASE_DIR: "relative" } }));
 const result = await execFile("sh", [installer, "locate", "--host", "zcode"], { env: { ...env, ZCODE_DATA_BASE_DIR: "" } });
 assert.equal(result.stdout.trim(), path.join(env.HOME, ".zcode", "skills"));
});

test("installation summaries stay compact and reflect host activation and API support", async t => {
 const f = await fixture(t);
 for(const host of ["codex","zcode","trae"]){
  const target=path.join(f.root,host);
  const {stdout}=await execFile("sh",[installer,"sync","--target",target,"--host",host],{cwd:repositoryRoot,env:f.env});
  assert.ok(Buffer.byteLength(stdout)<2200, `${host} dumped a long guide`);
  assert.doesNotMatch(stdout,/## 操作与恢复|# Pure Tokens Skill 使用须知/);
  if(host==="zcode")assert.match(stdout,/Refresh and enable the installed Skills in ZCode/);
  if(host==="trae"){
   assert.match(stdout,/API tasks: unsupported in this host/);
   assert.doesNotMatch(stdout,/Example \(not executed\):|Available instructions:/);
  }else assert.equal(stdout.split("Example (not executed):").length-1,2);
 }
});
test("a failed update restores every existing Skill and releases its lock", async t => {
 const f = await fixture(t); await install(f);
 const originals = new Map();
 for (const name of names) {
  const file = path.join(f.target, name, "SKILL.md");
  originals.set(name, await readFile(file, "utf8"));
 }
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 const wrapper = path.join(tools, "mv");
 await writeFile(wrapper, '#!/bin/sh\ncase "$2" in */backup/puretokens-connection) exit 73 ;; esac\nexec /bin/mv "$@"\n'); await chmod(wrapper, 0o700);
 await assert.rejects(install(f, { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}` }));
 for (const name of names) assert.equal(await readFile(path.join(f.target, name, "SKILL.md"), "utf8"), originals.get(name));
 assert.equal((await readdir(f.target)).some(x => x.includes("stage") || x.includes("lock")), false);
 await install(f);
});
test("failed-update rollback preserves unknown stage content before restoring any target", async t => {
 const f = await fixture(t); await install(f);
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 await writeFile(path.join(tools, "mv"), [
  "#!/bin/sh",
  'case "$2" in */backup/puretokens-connection)',
  ' stage=${2%/backup/puretokens-connection}',
  ' printf "keep my work" > "$stage/personal.txt"',
  ' exit 73 ;;',
  'esac',
  'exec /bin/mv "$@"'
 ].join("\n"));
 await chmod(path.join(tools, "mv"), 0o700);
 await assert.rejects(install(f, { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}` }));
 const stages = (await readdir(f.target)).filter(name => name.startsWith(".puretokens-skill-stage."));
 assert.equal(stages.length, 1, "unrecognized stage content must survive EXIT recovery");
 const stage = path.join(f.target, stages[0]);
 assert.equal(await readFile(path.join(stage, "personal.txt"), "utf8"), "keep my work");
 assert.ok(await readFile(path.join(stage, "backup/puretokens-balance/SKILL.md")), "global preflight must precede rollback mutations");
 assert.equal((await readdir(f.target)).includes(".puretokens-install-lock"), false);
});

test("explicit repair resumes an interrupted missing-file replacement", async t => {
 const f = await fixture(t); await install(f);
 const name = "puretokens-image";
 const dir = path.join(f.target, name);
 const stage = path.join(f.target, ".puretokens-skill-stage.repair");
 await mkdir(path.join(stage, "backup"), { recursive: true });
 await cp(dir, path.join(stage, name), { recursive: true });
 await writeFile(path.join(stage, "transaction-v1"), "");
 await writeFile(path.join(stage, "repair-v1"), "");
 await writeFile(path.join(stage, "plan"), `replace ${name}\n`);
 await rm(path.join(dir, "SKILL.md"));
 await execFile("sh", [installer, "repair", "--target", f.target], { cwd: repositoryRoot, env: f.env });
 assert.ok(await readFile(path.join(dir, "SKILL.md")));
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
});
test("repair preserves and rejects non-directory or symlink staging entries before changing installation", async t => {
 const f = await fixture(t); await install(f);
 const original = await readFile(path.join(f.target, "puretokens-image/SKILL.md"));
 for (const kind of ["file", "dangling-link", "directory-link"]) {
  const entry = path.join(f.target, `.puretokens-skill-stage.${kind}`);
  if (kind === "file") await writeFile(entry, "preserve unknown transaction");
  else await symlink(kind === "dangling-link" ? path.join(f.root, "absent") : f.target, entry);
  await assert.rejects(execFile("sh", [installer, "repair", "--target", f.target], {cwd:repositoryRoot,env:f.env}), /unrecognized staging entry/);
  assert.deepEqual(await readFile(path.join(f.target, "puretokens-image/SKILL.md")), original);
  assert.ok(await lstat(entry), "unknown entry must be retained");
  assert.equal((await readdir(f.target)).includes(".puretokens-install-lock"), false);
  await rm(entry);
 }
});
test("failure before a transaction plan preserves unverified stage contents and existing installation", async t => {
 const f = await fixture(t); await install(f);
 const original = await readFile(path.join(f.target, "puretokens-balance/SKILL.md"));
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 await writeFile(path.join(tools, "cp"), [
  "#!/bin/sh",
  'case "$3" in */.puretokens-skill-stage.*/puretokens-balance)',
  ' stage=${3%/puretokens-balance}',
  ' printf "preserve unknown work" > "$stage/personal.txt"',
  ' exit 73 ;;',
  'esac',
  'exec /bin/cp "$@"'
 ].join("\n"));
 await chmod(path.join(tools, "cp"), 0o700);
 await assert.rejects(install(f, { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}` }));
 const stages = (await readdir(f.target)).filter(name => name.startsWith(".puretokens-skill-stage."));
 assert.equal(stages.length, 1);
 const stage = path.join(f.target, stages[0]);
 assert.equal(await readFile(path.join(stage, "personal.txt"), "utf8"), "preserve unknown work");
 await assert.rejects(readFile(path.join(stage, "plan")), { code: "ENOENT" });
 assert.deepEqual(await readFile(path.join(f.target, "puretokens-balance/SKILL.md")), original);
 assert.equal((await readdir(f.target)).includes(".puretokens-install-lock"), false);
});
test("denied committed-stage cleanup reports pending, releases the lock and runs init once", async t => {
 const f = await fixture(t);
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 const attempts = path.join(f.root, "cleanup-attempts");
 await writeFile(path.join(tools, "rm"), [
  "#!/bin/sh",
  'for arg in "$@"; do',
  ' case "$arg" in */.puretokens-skill-stage.*)',
  '  printf "denied\\n" >> "$PT_CLEANUP_ATTEMPTS"',
  '  exit 73 ;;',
  ' esac',
  'done',
  'exec /bin/rm "$@"'
 ].join("\n"));
 await chmod(path.join(tools, "rm"), 0o700);
 const { stdout } = await install(f, { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}`, PT_CLEANUP_ATTEMPTS: attempts });
 assert.match(stdout, /synchronized with the native API executor/);
 assert.match(stdout, /cleanup_status: pending/);
 assert.equal(stdout.split("Pure Tokens Skill init:").length - 1, 1);
 assert.equal(await readFile(attempts, "utf8"), "denied\n", "do not retry a denied deletion in EXIT");
 const entries = await readdir(f.target);
 assert.equal(entries.includes(".puretokens-install-lock"), false);
 assert.equal(entries.filter(name => name.startsWith(".puretokens-skill-stage.")).length, 1);
 for (const name of names) assert.ok(await readFile(path.join(f.target, name, "SKILL.md")));
 await install(f);
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
});
test("next sync restores a recoverable interrupted transaction", async t => {
 const f = await fixture(t); await install(f);
 const stage = path.join(f.target, ".puretokens-skill-stage.fixture");
 await mkdir(path.join(stage, "backup"), { recursive: true });
 await writeFile(path.join(stage, "transaction-v1"), "");
 await writeFile(path.join(stage, "plan"), "replace puretokens-image\n");
 await rename(path.join(f.target, "puretokens-image"), path.join(stage, "backup/puretokens-image"));
 await cp(path.join(repositoryRoot, "skills/puretokens-image"), path.join(f.target, "puretokens-image"), { recursive: true });
  const {stdout: inventory}=await execFile(path.join(f.target,".puretokens-executor/puretokens-api"),["install-inventory","--directory",path.join(f.target,"puretokens-image"),"--name","puretokens-image"]);
  await writeFile(path.join(f.target,"puretokens-image/.puretokens-managed.json"),inventory);
 await install(f);
 assert.equal((await readdir(f.target)).includes(".puretokens-skill-stage.fixture"), false);
 assert.match(await readFile(path.join(f.target, "puretokens-image/SKILL.md"), "utf8"), /submit/);
});
test("successful sync with unverified init allows init-only continuation without rewriting installation", async t => {
 const f = await fixture(t);
 // Missing synthetic Codex configuration stops credential resolution before any API call.
 const env = { ...f.env, PTP_OPERATIONS_RECEIPTS: "0" };
 const { stdout: installed } = await install(f, env, ["--host", "codex"]);
 assert.match(installed, /synchronized with the native API executor/);
 assert.match(installed.split("\n").find(line => line.startsWith("Pure Tokens Skill init:")), /\[active_connection_record_missing\]/);
 async function snapshot(directory) {
  const entries = {};
  async function visit(relative = "") {
   for (const item of await readdir(path.join(directory, relative), { withFileTypes: true })) {
    const name = path.join(relative, item.name);
    if (item.isDirectory()) await visit(name);
    else {
     const file = path.join(directory, name);
     entries[name] = { sha256: createHash("sha256").update(await readFile(file)).digest("hex"), mtime: (await stat(file)).mtimeMs };
    }
   }
  }
  await visit(); return entries;
 }
 const before = await snapshot(f.target);
 const { stdout: result } = await execFile(path.join(f.target, ".puretokens-executor/puretokens-api"), ["init", "--host", "codex"], { env }).catch(error => {
  // An unverified connection may have a nonzero exit, but still has a typed receipt.
  if (error.stdout) return { stdout: error.stdout }; throw error;
 });
 const receipt = JSON.parse(result);
 assert.equal(receipt.configuration_status, "active_connection_record_missing");
 assert.equal(receipt.credential_verified, false);
 assert.equal(receipt.api_request_executed, false);
 assert.deepEqual(await snapshot(f.target), before);
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
});
test("live update lock prevents all installation writes", async t => {
 const f = await fixture(t); await mkdir(path.join(f.target, ".puretokens-install-lock"), { recursive: true });
 await writeFile(path.join(f.target, ".puretokens-install-lock/pid"), `${process.pid}\n`);
 await assert.rejects(install(f), /another installation is in progress/);
 assert.deepEqual(await readdir(f.target), [".puretokens-install-lock"]);
});
test("a filesystem without hard links supports installation and preserves incomplete locks", async t => {
 const f = await fixture(t);
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 await writeFile(path.join(tools, "ln"), "#!/bin/sh\nexit 1\n");
 await chmod(path.join(tools, "ln"), 0o700);
 const env = { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}` };
 await install(f, env);
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
 const lock = path.join(f.target, ".puretokens-install-lock");
 await mkdir(lock);
 await writeFile(path.join(lock, "pid"), "");
 await assert.rejects(install(f, env), /update lock needs inspection/);
 assert.equal(await readFile(path.join(lock, "pid"), "utf8"), "");
 assert.deepEqual(await readdir(lock), ["pid"]);
});
for (const hardLinks of [true, false]) test(`concurrent stale-lock recovery cannot replace the winning live updater (${hardLinks ? "hard links" : "exclusive creation"})`, async t => {
 const f = await fixture(t); await install(f);
 const lock = path.join(f.target, ".puretokens-install-lock");
 await mkdir(lock);
 await writeFile(path.join(lock, "pid"), "2147483647\n");
 const tools = path.join(f.root, "tools"); await mkdir(tools);
 if (!hardLinks) {
  await writeFile(path.join(tools, "ln"), "#!/bin/sh\nexit 1\n");
  await chmod(path.join(tools, "ln"), 0o700);
 }
 const coordination = path.join(f.root, "coordination"); await mkdir(coordination);
 const waitForRelease = [
  'attempt=0',
  'while [ ! -f "$release" ]; do',
  ' attempt=$((attempt + 1)); [ "$attempt" -lt 1500 ] || exit 91',
  ' sleep 0.01',
  'done'
 ].join("\n");
 await writeFile(path.join(tools, "cat"), [
  "#!/bin/sh",
  'case "$1" in */.puretokens-install-lock/pid) is_lock_pid=true ;; *) is_lock_pid=false ;; esac',
  'if [ "$PT_LOCK_PEER" = second ] && $is_lock_pid && [ ! -f "$PT_LOCK_COORDINATION/stale-read" ]; then',
  ' /bin/cat "$@" || exit',
  ' : > "$PT_LOCK_COORDINATION/stale-read"',
  ' release="$PT_LOCK_COORDINATION/release-second"',
  waitForRelease,
  ' exit 0',
  'fi',
  'exec /bin/cat "$@"'
 ].join("\n"));
 await writeFile(path.join(tools, "cp"), [
  "#!/bin/sh",
  'if [ "$PT_LOCK_PEER" = first ] && [ "$1" = -R ] && [ ! -f "$PT_LOCK_COORDINATION/first-copy" ]; then',
  ' : > "$PT_LOCK_COORDINATION/first-copy"',
  ' release="$PT_LOCK_COORDINATION/release-first"',
  waitForRelease,
  'fi',
  'exec /bin/cp "$@"'
 ].join("\n"));
 await chmod(path.join(tools, "cat"), 0o700);
 await chmod(path.join(tools, "cp"), 0o700);
 const env = { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}`, PT_LOCK_COORDINATION: coordination };
 const settle = promise => promise.then(value => ({ ok: true, value }), error => ({ ok: false, error }));
 const second = settle(install(f, { ...env, PT_LOCK_PEER: "second" }));
 let first;
 try {
  await waitForFile(path.join(coordination, "stale-read"));
  first = settle(install(f, { ...env, PT_LOCK_PEER: "first" }));
  await waitForFile(path.join(coordination, "first-copy"));
  await writeFile(path.join(coordination, "release-second"), "");
  const result = await second;
  assert.equal(result.ok, false, "the losing recovery must not enter the live updater's transaction");
  assert.match(result.error.stderr, /another installation is in progress/);
  assert.equal((await readdir(f.target)).filter(name => name.startsWith(".puretokens-skill-stage.")).length, 1);
 } finally {
  await writeFile(path.join(coordination, "release-second"), "");
  await writeFile(path.join(coordination, "release-first"), "");
  await second;
  if (first) {
   const result = await first;
   assert.equal(result.ok, true, result.error?.stderr);
  }
 }
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
});
test("a killed stale-lock takeover remains recoverable by the next sync", async t => {
 const f = await fixture(t); await install(f);
 const lock = path.join(f.target, ".puretokens-install-lock");
 await mkdir(lock);
 await writeFile(path.join(lock, "pid"), "2147483647\n");
 await writeFile(path.join(lock, "next.legacy-2147483647"), "2147483646 owner.interrupted\n");
 const stage = path.join(f.target, ".puretokens-skill-stage.interrupted");
 await mkdir(path.join(stage, "backup"), { recursive: true });
 await writeFile(path.join(stage, "transaction-v1"), "");
 await writeFile(path.join(stage, "plan"), "replace puretokens-image\n");
 await rename(path.join(f.target, "puretokens-image"), path.join(stage, "backup/puretokens-image"));
 await install(f);
 assert.equal((await readdir(f.target)).some(name => name.includes("stage") || name.includes("lock")), false);
 assert.match(await readFile(path.join(f.target, "puretokens-image/SKILL.md"), "utf8"), /submit/);
});
test("Gemini selects an existing shared Skill and detects its lower-priority duplicate", async t => {
 const f = await fixture(t);
 const shared = path.join(f.env.HOME, ".agents/skills");
 const legacy = path.join(f.env.HOME, ".gemini/skills");
 await mkdir(shared, { recursive: true });
 await mkdir(legacy, { recursive: true });
 await cp(path.join(repositoryRoot, "skills/puretokens-image"), path.join(shared, "puretokens-image"), { recursive: true });
 await cp(path.join(repositoryRoot, "skills/puretokens-image"), path.join(legacy, "puretokens-image"), { recursive: true });
 const { stdout: selected } = await execFile("sh", [installer, "locate", "--host", "gemini-cli"], { env: f.env });
 assert.equal(selected.trim(), shared);
 f.target = shared;
 const { stdout } = await install(f, f.env, ["--host", "gemini-cli"]);
 assert.match(stdout, /Managed duplicate detected/);
 assert.match(await readFile(path.join(legacy, "puretokens-image/SKILL.md"), "utf8"), /Pure Tokens/);
 assert.match(await readFile(path.join(shared, ".puretokens-executor/puretokens-skill-fetch.sh"), "utf8"), /check-update/);
 f.target = legacy;
 await assert.rejects(install(f, f.env, ["--host", "gemini-cli"]), /higher-priority/);
});

test("new hosts install and update in their declared isolated roots", async t => {
 const f = await fixture(t);
 for (const [host, variable] of [["kimi-code", "KIMI_CODE_HOME"], ["qoder", "QODER_CONFIG_DIR"], ["pi", "PI_CODING_AGENT_DIR"], ["hermes", "HERMES_HOME"], ["evox", "EVOX_AGENT_DIR"]]) {
  const root = path.join(f.root, `${host} \u7528\u6237 with spaces`);
  const env = {...f.env, [variable]: root};
  const target = path.join(root, "skills");
  const located = await execFile("sh", [installer, "locate", "--host", host], {env});
  assert.equal(located.stdout.trim(), target);
  for (let i=0;i<2;i++) await execFile("sh", [installer, "sync", "--host", host], {env});
  assert.ok(await readFile(path.join(target, "puretokens-image", "SKILL.md")));
  assert.ok(await readFile(path.join(target, ".puretokens-executor", "puretokens-api")));
  await assert.rejects(execFile("sh", [installer, "sync", "--host", host], {env:{...env,[variable]:"relative"}}));
 }
 const piDefault = await execFile("sh", [installer, "locate", "--host", "pi"], {env:{...f.env,PI_CODING_AGENT_DIR:""}});
 assert.equal(piDefault.stdout.trim(), path.join(f.env.HOME, ".pi", "agent", "skills"));
 const invalidPi = `${f.root}/invalid/../unexpected`;
 for (const command of ["locate", "sync"]) {
  await assert.rejects(execFile("sh", [installer, command, "--host", "pi"], {env:{...f.env,PI_CODING_AGENT_DIR:invalidPi}}));
 }
 await assert.rejects(readdir(path.join(f.root, "invalid")), {code:"ENOENT"});
 await assert.rejects(readdir(path.join(f.root, "unexpected")), {code:"ENOENT"});
 const env={...f.env,QODER_CONFIG_DIR:"",QODER_CLI_HOME:path.join(f.root,"qoder parent"),QODER_CONFIG_DIR_NAME:"custom"};
 const result=await execFile("sh",[installer,"locate","--host","qoder"],{env});
 assert.equal(result.stdout.trim(),path.join(env.QODER_CLI_HOME,"custom","skills"));
 await assert.rejects(execFile("sh",[installer,"locate","--host","qoder"],{env:{...env,QODER_CONFIG_DIR_NAME:"../bad"}}));
});

test("new host roots fail before writes and workspace-only Octop requires a target", async t => {
 const f = await fixture(t);
 const env = {...f.env, HERMES_HOME:"", EVOX_AGENT_DIR:"", EVOX_CODING_AGENT_DIR:""};
 for (const [host, variable] of [["hermes", "HERMES_HOME"], ["evox", "EVOX_AGENT_DIR"]]) {
  for (const invalid of ["relative", "//server/share", `${f.root}/../other`, `${f.root}\n`, `${f.root}\t`]) {
   await assert.rejects(execFile("sh", [installer, "locate", "--host", host, "--target", f.target], {env:{...env,[variable]:invalid}}));
  }
 }
 await assert.rejects(execFile("sh", [installer,"locate","--host","evox"], {env:{...env,EVOX_AGENT_DIR:f.root,EVOX_CODING_AGENT_DIR:f.target}}));
 for (const host of ["vscode","octop"]) {
  const target = host === "vscode" ? path.join(f.env.HOME, ".copilot/skills") : path.join(f.root,"workspace/.octop/skills");
  const args = ["--host",host];
  if (host === "octop") {
   await assert.rejects(execFile("sh", [installer,"locate",...args], {env}), /requires --target/);
   args.push("--target",target);
  }
  const {stdout} = await execFile("sh", [installer,"locate",...args], {env});
  assert.equal(stdout.trim(),target);
  await execFile("sh", [installer,"sync",...args], {env});
  for (const name of names) assert.ok(JSON.parse(await readFile(path.join(target,name,"skill.json"),"utf8")).supportedClients.includes(host));
 }
 for (const invalid of [f.target, `${f.root}/../skills`, "//server/share/skills"]) {
  await assert.rejects(execFile("sh",[installer,"locate","--host","octop","--target",invalid],{env}));
 }
});

test("installation never invokes Codex plugins and still runs host init", async t => {
 const f=await fixture(t);
 const mockBin=path.join(f.root,"mock-bin");await mkdir(mockBin);
 const marker=path.join(f.root,"codex-called");
 await writeFile(path.join(mockBin,"codex"),'#!/bin/sh\nprintf called >> "$PLUGIN_MARKER"\nprintf \'{"installed":[]}\\n\'\n');
 await chmod(path.join(mockBin,"codex"),0o755);
 const env={...f.env,PATH:mockBin+path.delimiter+f.env.PATH,PLUGIN_MARKER:marker,QODER_CONFIG_DIR:path.join(f.root,"missing-qoder")};
 const shared=path.join(f.home??f.env.HOME,".agents","skills");
 const {stdout}=await execFile("sh",[installer,"sync","--host","qoder","--target",shared],{env});
 await assert.rejects(readFile(marker),{code:"ENOENT"});
 assert.match(stdout,/Pure Tokens Skill init:/);
 assert.doesNotMatch(stdout,/connection check was deferred/);
 assert.match(stdout,/Installation host: qoder/);
 assert.match(stdout,/Installation host: qoder/);
 assert.match(stdout,/Follow this client's Skill loading controls, then start a new conversation/);
 await execFile("sh",[installer,"sync","--host","codex","--target",shared],{env});
 await assert.rejects(readFile(marker),{code:"ENOENT"});
 await execFile(path.join(mockBin,"codex"),[],{env});
 assert.equal(await readFile(marker,"utf8"),"called");
});

test("MiniMax Desktop installer uses only a declared root and refuses ambiguous preferences", { skip: process.platform !== "darwin" }, async t => {
 const f = await fixture(t);
 const env = { ...f.env, MINIMAX_DATA_DIR: "", MAVIS_DATA_DIR: "", MAVIS_PROFILE: "", MINIMAX_PROFILE: "", AGENTARCHON_PROFILE: "", AGENTARCHON_DATA_DIR: "", __MAVIS_RUNTIME_PROFILE: "", __MAVIS_RUNTIME_DATA_DIR: "" };
 const locate = overrides => execFile("sh", [installer, "locate", "--host", "minimax-code"], { env: { ...env, ...overrides } });
 assert.equal((await locate({})).stdout.trim(), path.join(env.HOME, ".minimax/skills"));
 const custom = path.join(f.root, "declared runtime");
 assert.equal((await locate({ MINIMAX_DATA_DIR: custom, MAVIS_DATA_DIR: path.join(f.root, "other") })).stdout.trim(), path.join(custom, "skills"));
 assert.equal((await locate({ MAVIS_DATA_DIR: custom })).stdout.trim(), path.join(custom, "skills"));
 await assert.rejects(locate({ MINIMAX_DATA_DIR: "relative" }));
 await assert.rejects(locate({ MINIMAX_DATA_DIR: custom + "/../other" }));
 await assert.rejects(locate({ MAVIS_PROFILE: "cli-profile" }));
 await mkdir(path.join(env.HOME, ".mavis"));
 await assert.rejects(locate({}));
 const prefs = path.join(env.HOME, "Library/Application Support/MiniMax Code");
 await mkdir(prefs, { recursive: true });
 await writeFile(path.join(prefs, "minimax-agent-config.json"), '{"config":{}}');
 await assert.rejects(locate({}));
 const result = await execFile("sh", [installer, "sync", "--host", "minimax-code"], { env: { ...env, MINIMAX_DATA_DIR: custom } });
 assert.match(result.stdout, /synchronized/);
 const manifest = JSON.parse(await readFile(path.join(custom, "skills/puretokens-image/skill.json"), "utf8"));
 assert.ok(manifest.supportedClients.includes("minimax-code"));
});
