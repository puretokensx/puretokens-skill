import assert from "node:assert/strict";
import { execFile as execFileCallback } from "node:child_process";
import { chmod, mkdir, mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import { promisify } from "node:util";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

const execFile = promisify(execFileCallback);
const fetch = path.join(repositoryRoot, "runtime/puretokens-skill-fetch.sh");

async function fixture(t, name, parentName = "codex") {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-host-detection-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const tools = path.join(root, "tools");
  await mkdir(tools);
  await writeFile(path.join(tools, "ps"), `#!/bin/sh
[ "$#" = 6 ] && [ "$1" = -p ] && [ "$3" = -o ] && [ "$4" = ppid= ] && [ "$5" = -o ] && [ "$6" = comm= ] || exit 98
printf '%s\\n' "$2" >> "$PT_PROCESS_CALLS"
case "$2" in
  71) printf '72 %s\\n' "$PT_PROCESS_NAME" ;;
  72) printf '1 %s\\n' "$PT_PARENT_NAME" ;;
  *) printf '71 /bin/sh\\n' ;;
esac
`);
  await writeFile(path.join(tools, "curl"), `#!/bin/sh
printf 'unexpected-network\\n' >> "$PT_PROCESS_CALLS"
exit 99
`);
  for (const tool of ["ps", "curl"]) await chmod(path.join(tools, tool), 0o700);
  return {
    root,
    env: {
      ...process.env,
      PATH: `${tools}${path.delimiter}${process.env.PATH}`,
      HOME: root,
      PT_PROCESS_NAME: name,
      PT_PARENT_NAME: parentName,
      PT_PROCESS_CALLS: path.join(root, "calls"),
      // Directory settings and inherited markers never select the current client.
      CODEX_HOME: path.join(root, ".codex"),
      CODEX_THREAD_ID: "inherited-from-an-outer-session",
      VSCODE_PID: "123",
      CLAUDECODE: "1"
    }
  };
}

for (const [executable, host] of [
  ["/Applications/ChatGPT.app/Contents/Resources/codex-cli/CodexCLI.app/Contents/MacOS/codex", "codex"],
  ["/opt/opencode/bin/opencode", "opencode"]
]) {
  test(`host detection recognizes a direct ${host} launcher without network or configuration reads`, async t => {
    const f = await fixture(t, executable);
    const { stdout, stderr } = await execFile("sh", [fetch, "detect-host"], { env: f.env });
    assert.equal(stdout.trim(), host);
    assert.equal(stderr, "");
    assert.equal((await readFile(f.env.PT_PROCESS_CALLS, "utf8")).trim().split("\n").length, 2);
    assert.deepEqual((await readdir(f.root)).sort(), ["calls", "tools"]);
  });
}

for (const executable of ["/usr/bin/node", "/usr/bin/python3", "/Applications/Visual Studio Code.app/Contents/MacOS/Electron", "/opt/claude"]) {
  test(`host detection stops at an unconfirmed launcher instead of borrowing the outer Codex session: ${path.basename(executable)}`, async t => {
    const f = await fixture(t, executable);
    await assert.rejects(execFile("sh", [fetch, "install"], { env: f.env }), error => {
      assert.match(error.stderr, /stage=detect_host error_code=current_host_unconfirmed/);
      assert.match(error.stderr, /next_step=confirm_current_client/);
      assert.doesNotMatch(error.stderr, /inherited-from|Applications|PT_PROCESS|unexpected-network/);
      return true;
    });
    const calls = (await readFile(f.env.PT_PROCESS_CALLS, "utf8")).trim().split("\n");
    assert.equal(calls.length, 2, "do not traverse past an unknown client or request release data");
  });
}

test("host detection is bounded when only shell ancestors are visible", async t => {
  const f = await fixture(t, "/bin/bash", "/bin/sh");
  await assert.rejects(execFile("sh", [fetch, "detect-host"], { env: f.env }), /current_host_unconfirmed/);
  assert.ok((await readFile(f.env.PT_PROCESS_CALLS, "utf8")).trim().split("\n").length <= 8);
});

test("PowerShell host detection preserves the same launcher boundary", async t => {
  try { await execFile("pwsh", ["-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"]); }
  catch (error) { if (error.code === "ENOENT") { t.skip("PowerShell is unavailable; Windows detection remains unverified"); return; } throw error; }
  const { stdout } = await execFile("pwsh", ["-NoProfile", "-File", path.join(repositoryRoot, "test/support/windows-host-detection.ps1")]);
  assert.match(stdout, /bounded queries passed/);
});
