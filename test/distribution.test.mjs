import assert from "node:assert/strict";
import { test } from "node:test";
import { execFile as execFileCallback } from "node:child_process";
import { promisify } from "node:util";
import { cp, mkdir, mkdtemp, readFile, readdir, rm, writeFile, chmod, stat, symlink } from "node:fs/promises";
import { createHash } from "node:crypto";
import os from "node:os";
import path from "node:path";
import { repositoryRoot } from "../scripts/skill-registry.mjs";
const execFile = promisify(execFileCallback);
const commit = "a".repeat(40);
const version = "0.17.0";
const scripts = ["puretokens-skill-install.sh", "puretokens-skill-install.ps1", "puretokens-skill-fetch.sh", "puretokens-skill-fetch.ps1"];

test("PowerShell download diagnostics classify exceptions and suppress private messages", async t => {
  try { await execFile("pwsh", ["-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"]); }
  catch (error) { if (error.code === "ENOENT") { t.skip("PowerShell is unavailable; Windows execution remains unverified"); return; } throw error; }
  const { stdout } = await execFile("pwsh", ["-NoProfile", "-File", path.join(repositoryRoot, "test/support/windows-download-diagnostics.ps1")]);
  assert.match(stdout, /14 failure cases and one explicit successful continuation passed/);
});

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-distribution-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const source = path.join(root, "puretokens-skill");
  await mkdir(path.join(source, "runtime/executor/bin"), { recursive: true });
  await mkdir(path.join(source, "scripts"));
  await mkdir(path.join(root, "home"));
  await mkdir(path.join(root, "tmp"));
  await writeFile(path.join(source, "README.md"), "fixture\n");
  await writeFile(path.join(source, "package.json"), JSON.stringify({ version }, null, 2));
  await cp(path.join(repositoryRoot, "skills"), path.join(source, "skills"), { recursive: true });
  for (const script of scripts) await cp(path.join(repositoryRoot, "runtime", script), path.join(source, "runtime", script));
  // Match the shell installer, including a translated shell on Apple Silicon.
  const architecture = (await execFile("uname", ["-m"])).stdout.trim();
  const platform = `${process.platform === "darwin" ? "darwin" : "linux"}-${["arm64", "aarch64"].includes(architecture) ? "arm64" : "amd64"}`;
  // Exercise inventory validation with the actual candidate executable.
  const binary = await readFile(path.join(repositoryRoot, "runtime/executor/bin", `puretokens-api-${platform}`));
  const binaryPath = `bin/puretokens-api-${platform}`;
  await writeFile(path.join(source, "runtime/executor", binaryPath), binary, { mode: 0o755 });
  await writeFile(path.join(source, "runtime/executor/manifest.json"), JSON.stringify({
    schemaVersion: 1, name: "puretokens-api-executor", version,
    artifacts: { [platform]: { path: binaryPath, sha256: createHash("sha256").update(binary).digest("hex") } }
  }, null, 2));
  const env = { ...process.env, HOME: path.join(root, "home"), CODEX_HOME: path.join(root, "home/.codex"), TMPDIR: path.join(root, "tmp"), GITHUB_SHA: commit };
  return { root, source, platform, env, target: path.join(root, "target") };
}

test("repackaging replaces the archive and removes deleted members", async t => {
  const f = await fixture(t);
  const builder = path.join(f.source, "scripts/package-platform-releases.mjs");
  await cp(path.join(repositoryRoot, "scripts/package-platform-releases.mjs"), builder);
  const obsolete = path.join(f.source, "skills/puretokens-image/obsolete.json");
  await writeFile(obsolete, "{}");
  await execFile(process.execPath, [builder], { env: f.env });
  await rm(obsolete);
  await execFile(process.execPath, [builder], { env: f.env });
  const releases = path.join(f.source, "dist/releases");
  const { stdout } = await execFile("unzip", ["-Z1", path.join(releases, `puretokens-skill-${version}-${f.platform}.zip`)]);
  assert.doesNotMatch(stdout, /obsolete.json/);
  assert.equal(stdout.split("\n").filter(p => /runtime\/executor\/bin\/.+/.test(p)).length, 1);
  assert.equal(JSON.parse(await readFile(path.join(releases, "release-manifest.json"), "utf8")).sourceCommit, null);
  assert.equal((await readdir(releases)).some(p => p.endsWith(".tmp")), false);
});

async function mockDownloads(f, mode = "platform") {
  const tools = path.join(f.root, "tools");
  await mkdir(tools);
  await execFile("zip", ["-qr", path.join(f.root, "platform.zip"), "puretokens-skill"], { cwd: f.root });
  const checksum = createHash("sha256").update(await readFile(path.join(f.root, "platform.zip"))).digest("hex");
  const selectorChecksum = createHash("sha256").update(await readFile(path.join(f.source, "runtime/puretokens-skill-install.sh"))).digest("hex");
  const executorChecksum = createHash("sha256").update(await readFile(path.join(f.source, "runtime/executor/bin", `puretokens-api-${f.platform}`))).digest("hex");
  await writeFile(path.join(f.root, "release.json"), JSON.stringify({ schemaVersion: 2, version, sourceCommit: commit, installers: {
    shell: {filename: "puretokens-skill-install.sh", sha256: selectorChecksum},
  }, files: {
    [f.platform]: { filename: `puretokens-skill-${version}-${f.platform}.zip`, sha256: mode === "bad-checksum" ? "0".repeat(64) : checksum, executorSha256: executorChecksum }
  } }, null, 2));
  const curl = path.join(tools, "curl");
  await writeFile(curl, `#!/bin/sh
while [ "$#" -gt 0 ]; do
  case "$1" in --output) destination=$2; shift 2 ;; https:*) request_url=$1; shift ;; *) shift ;; esac
done
printf '%s\\n' "$request_url" >> "$PT_FETCH_FIXTURE/requests"
if [ -n "\${PT_FAIL_MATCH:-}" ]; then
  case "$request_url" in *"$PT_FAIL_MATCH"*)
    printf '%s' "$PT_FAIL_HTTP"
    printf '%s' 'private-exception-must-not-leak' >&2
    exit "$PT_FAIL_EXIT" ;;
  esac
fi
case "$request_url" in
  https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json)
    if [ "$PT_FETCH_MODE" = missing ]; then printf 404; exit 0; fi
    file=release.json ;;
  https://github.com/puretokensx/puretokens-skill/releases/download/v${version}/puretokens-skill-install.sh) file=puretokens-skill/runtime/puretokens-skill-install.sh ;;
  https://github.com/puretokensx/puretokens-skill/releases/download/v${version}/puretokens-skill-${version}-${f.platform}.zip)
    if [ -n "\${PT_INSTALL_DURING_DOWNLOAD:-}" ]; then
      sh "$PT_INSTALL_DURING_DOWNLOAD/runtime/puretokens-skill-install.sh" sync --target "$PT_FIXTURE_TARGET" --source "$PT_INSTALL_DURING_DOWNLOAD" > "$PT_FETCH_FIXTURE/concurrent-update"
    fi
    file=platform.zip ;;
  *) exit 99 ;;
esac
cp "$PT_FETCH_FIXTURE/$file" "$destination"
printf 200
`);
  await chmod(curl, 0o700);
  return { ...f.env, PATH: `${tools}${path.delimiter}${f.env.PATH}`, PT_FETCH_FIXTURE: f.root, PT_FETCH_MODE: mode };
}

test("check-update reads stable metadata and its verified selector only", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  const { stdout } = await execFile("sh", [wrapper, "check-update", "--target", f.target], { env });
  assert.match(stdout, /installed=not_installed available=0.17.0/);
  await assert.rejects(readFile(path.join(f.target, ".puretokens-executor/runtime.json")));
  const requests = (await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n");
  assert.equal(requests.length, 2);
  assert.ok(requests[0].includes("/releases/latest/download/"));
  assert.ok(requests[1].includes(`/releases/download/v${version}/`));
  assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
});

for (const [options, detected, relativeTarget] of [
  [[], "codex", ".agents/skills"],
  [[], "opencode", ".config/opencode/skills"],
  [["--host", "opencode"], "node", ".config/opencode/skills"],
  [["--target"], "node", null]
]) {
  test(`fetch resolves the correct target through its verified selector (${options.join(" ") || detected})`, async t => {
    const f = await fixture(t);
    const env = await mockDownloads(f);
    const calls = path.join(f.root, "process-queries");
    env.PT_PROCESS_QUERIES = calls;
    const ps = path.join(f.root, "tools/ps");
    await writeFile(ps, `#!/bin/sh
printf '%s\\n' "$2" >> "$PT_PROCESS_QUERIES"
printf '1 /opt/${detected}\\n'
`);
    await chmod(ps, 0o700);
    const target = relativeTarget ? path.join(f.env.HOME, relativeTarget) : f.target;
    await mkdir(path.join(target, ".puretokens-executor"), { recursive: true });
    const runtime = path.join(target, ".puretokens-executor/runtime.json");
    const before = JSON.stringify({ name: "puretokens-api-executor", version: "0.16.0" });
    await writeFile(runtime, before);
    const args = options[0] === "--target" ? ["--target", target] : options;
    const { stdout } = await execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "check-update", ...args], { env });
    assert.match(stdout, /installed=0.16.0 available=0.17.0/);
    assert.equal(await readFile(runtime, "utf8"), before);
    if (options.length) await assert.rejects(readFile(calls), { code: "ENOENT" });
    else assert.equal((await readFile(calls, "utf8")).trim().split("\n").length, 1);
    assert.equal((await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n").length, 2);
    assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
  });
}

test("download failures identify the stage and safe category without retries or target changes", async t => {
  const f = await fixture(t);
  const base = await mockDownloads(f);
  await mkdir(f.target);
  await writeFile(path.join(f.target, "sentinel"), "keep");
  const cases = [
    ["release-manifest.json", "read_release_manifest", "403", "0", "http_error", 1],
    ["install.sh", "download_selector", "429", "0", "http_error", 2],
    ["install.sh", "download_selector", "000", "28", "timeout", 2],
    ["release-manifest.json", "read_release_manifest", "000", "6", "dns_failure", 1],
    [".zip", "download_platform_archive", "000", "35", "tls_failure", 3],
    ["release-manifest.json", "read_release_manifest", "000", "7", "connection_failure", 1],
    ["release-manifest.json", "read_release_manifest", "404", "0", "http_error", 1],
    ["release-manifest.json", "read_release_manifest", "200", "28", "timeout", 1],
    [".zip", "download_platform_archive", "404", "0", "http_error", 3],
  ];
  for (const [match, stage, http, exit, category, count] of cases) {
    await writeFile(path.join(f.root, "requests"), "");
    await assert.rejects(execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "install", "--target", f.target], {
      env: { ...base, PT_FAIL_MATCH: match, PT_FAIL_HTTP: http, PT_FAIL_EXIT: exit }
    }), error => {
      assert.ok(error.stderr.includes(`stage=${stage} error_code=${category} http_status=${Number(http)}`));
      assert.match(error.stderr, /installation_status=not_completed installed_files_changed=false/);
      assert.doesNotMatch(error.stderr, /private-exception-must-not-leak/);
      return true;
    });
    assert.equal((await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n").length, count);
    assert.deepEqual(await readdir(f.target), ["sentinel"]);
    assert.equal(await readFile(path.join(f.target, "sentinel"), "utf8"), "keep");
  }
});

test("download wrapper installs only the verified stable platform package", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const standalone = path.join(f.root, "puretokens-skill-fetch.sh");
  await cp(path.join(f.source, "runtime/puretokens-skill-fetch.sh"), standalone);
  const { stdout } = await execFile("sh", [standalone, "install", "--target", f.target], { env });
  assert.match(stdout, /synchronized with the native API executor/);
  const installed = JSON.parse(await readFile(path.join(f.target, ".puretokens-executor/runtime.json"), "utf8"));
  assert.equal(installed.version, version);
  const requests = await readFile(path.join(f.root, "requests"), "utf8");
  assert.doesNotMatch(requests, /codeload|commits\/main|raw\.githubusercontent/);
  assert.equal(requests.trim().split("\n").length, 3);
  assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
});

test("an unavailable child installer reports safe state without exposing shell errors or retrying", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  await mkdir(f.target);
  await writeFile(path.join(f.target, "sentinel"), "keep");
  const launcher = path.join(f.root, "tools/sh");
  await writeFile(launcher, `#!/bin/sh
if [ "$2" = "$PT_REJECT_INSTALLER_COMMAND" ]; then
  printf 'private-execution-error %s\\n' "$1" >&2
  exit 126
fi
exec /bin/sh "$@"
`);
  await chmod(launcher, 0o700);
  for (const [command, phase, state, changed, count] of [
    ["locate", "select_target", "not_completed", "false", 2],
    ["sync", "sync", "unconfirmed", "unknown", 3]
  ]) {
    await writeFile(path.join(f.root, "requests"), "");
    await assert.rejects(execFile("/bin/sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "install", "--target", f.target], {
      env: { ...env, PT_REJECT_INSTALLER_COMMAND: command }
    }), error => {
      assert.match(error.stderr, new RegExp(`stage=${phase} error_code=installer_unavailable installation_status=${state} installed_files_changed=${changed}`));
      assert.match(error.stderr, /next_step=review_host_execution_permissions/);
      assert.doesNotMatch(error.stderr, /private-execution-error|puretokens-download\./);
      return true;
    });
    assert.equal((await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n").length, count);
    assert.deepEqual(await readdir(f.target), ["sentinel"]);
  }
});

test("published checksum mismatch stops before target mutation", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f, "bad-checksum");
  await assert.rejects(execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "update", "--target", f.target], { env }), /checksum mismatch/);
  await assert.rejects(readFile(path.join(f.target, ".puretokens-executor/runtime.json")));
  assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
});

for (const mode of ["platform", "bad-checksum"]) {
  test(`fresh official bootstrap upgrades an old-repository installation safely (${mode})`, async t => {
    const f = await fixture(t);
    const oldSource = path.join(f.root, "old-source");
    await cp(f.source, oldSource, { recursive: true });
    await writeFile(path.join(oldSource, "package.json"), JSON.stringify({ version: "0.16.0" }, null, 2));
    const oldFetch = path.join(oldSource, "runtime/puretokens-skill-fetch.sh");
    await writeFile(oldFetch, (await readFile(oldFetch, "utf8")).replaceAll(
      "github.com/puretokensx/puretokens-skill", "github.com/PureTokens/puretokens-skill"));
    await execFile("sh", [path.join(oldSource, "runtime/puretokens-skill-install.sh"), "sync",
      "--target", f.target, "--source", oldSource], { env: f.env });
    await writeFile(path.join(f.target, "unmanaged.txt"), "keep personal files");
    const snapshot = async () => {
      const result = {};
      const walk = async directory => {
        for (const entry of await readdir(directory, { withFileTypes: true })) {
          const full = path.join(directory, entry.name);
          if (entry.isDirectory()) await walk(full);
          else result[path.relative(f.target, full)] = {
            sha: createHash("sha256").update(await readFile(full)).digest("hex"),
            mtime: (await stat(full)).mtimeMs
          };
        }
      };
      await walk(f.target);
      return result;
    };
    const before = await snapshot();
    const env = await mockDownloads(f, mode);
    await assert.rejects(execFile("sh", [path.join(f.target, ".puretokens-executor/puretokens-skill-fetch.sh"),
      "check-update", "--target", f.target], { env }), /read_release_manifest/);
    assert.match(await readFile(path.join(f.root, "requests"), "utf8"), /github\.com\/PureTokens\//);
    assert.deepEqual(await snapshot(), before);
    await writeFile(path.join(f.root, "requests"), "");
    const bootstrap = path.join(f.root, "current-official-fetch.sh");
    await cp(path.join(f.source, "runtime/puretokens-skill-fetch.sh"), bootstrap);
    if (mode === "bad-checksum") {
      await assert.rejects(execFile("sh", [bootstrap, "update", "--target", f.target], { env }), /checksum mismatch/);
      assert.deepEqual(await snapshot(), before, "failed verification rewrote the old installation");
    } else {
      await execFile("sh", [bootstrap, "update", "--target", f.target], { env });
      assert.equal(JSON.parse(await readFile(path.join(f.target, ".puretokens-executor/runtime.json"), "utf8")).version, version);
      assert.doesNotMatch(await readFile(path.join(f.target, ".puretokens-executor/puretokens-skill-fetch.sh"), "utf8"), /github\.com\/PureTokens\//);
    }
    assert.equal(await readFile(path.join(f.target, "unmanaged.txt"), "utf8"), "keep personal files");
    const requests = (await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n");
    assert.equal(requests.length, 3);
    assert.ok(requests.every(url => url.startsWith("https://github.com/puretokensx/puretokens-skill/releases/")));
    assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
  });
}

test("unpublished or invalid stable metadata never falls back to main or source", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f, "platform");
  const releaseFile = path.join(f.root, "release.json");
  const release = JSON.parse(await readFile(releaseFile, "utf8"));
  for (const change of [{sourceCommit: null}, {schemaVersion: 1}, {version: "0.18.1-rc.1"}, {version: "../unsafe"}]) {
    await writeFile(releaseFile, JSON.stringify({...release, ...change}, null, 2));
    await writeFile(path.join(f.root, "requests"), "");
    await assert.rejects(execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "install", "--target", f.target], { env }));
    assert.equal((await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n").length, 1);
    await assert.rejects(stat(f.target));
  }
  await writeFile(releaseFile, JSON.stringify(release, null, 2));
  await assert.rejects(execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "install", "--target", f.target], {env:{...env, PT_FETCH_MODE:"missing"}}), /read_release_manifest/);
  assert.doesNotMatch(await readFile(path.join(f.root, "requests"), "utf8"), /codeload|commits\/main/);
});

test("legacy sibling installer is replaced by the pinned directory selector", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const bootstrap = path.join(f.root, "bootstrap");
  await mkdir(bootstrap);
  await cp(path.join(f.source, "runtime/puretokens-skill-fetch.sh"), path.join(bootstrap, "puretokens-skill-fetch.sh"));
  await writeFile(path.join(bootstrap, "puretokens-skill-install.sh"), "#!/bin/sh\nexit 79\n");
  const { stdout } = await execFile("sh", [path.join(bootstrap, "puretokens-skill-fetch.sh"), "check-update", "--target", f.target], { env });
  assert.match(stdout, /available=0.17.0/);
  assert.match(await readFile(path.join(f.root, "requests"), "utf8"), /releases\/download\/v0\.17\.0\/puretokens-skill-install\.sh/);
  assert.deepEqual(await readdir(path.join(f.root, "tmp")), []);
});

test("sync rejects an older download after a newer version wins the installation lock", async t => {
  const f = await fixture(t);
  const newer = path.join(f.root, "newer-source");
  await cp(f.source, newer, { recursive: true });
  await writeFile(path.join(newer, "package.json"), JSON.stringify({ version: "0.18.0" }, null, 2));
  await writeFile(path.join(newer, "skills/puretokens-image/newer-marker"), "preserve newer installation");
  const env = { ...await mockDownloads(f), PT_INSTALL_DURING_DOWNLOAD: newer, PT_FIXTURE_TARGET: f.target };
  await assert.rejects(execFile("sh", [path.join(f.source, "runtime/puretokens-skill-fetch.sh"), "update", "--target", f.target], { env }), /downgrade was stopped under the update lock/);
  assert.equal(JSON.parse(await readFile(path.join(f.target, ".puretokens-executor/runtime.json"), "utf8")).version, "0.18.0");
  assert.equal(await readFile(path.join(f.target, "puretokens-image/newer-marker"), "utf8"), "preserve newer installation");
  assert.equal((await readdir(f.target)).some(file => file.includes("stage") || file.includes("lock")), false);
});

test("release provenance rejects committed source without a matching executable build proof", async t => {
  const f = await fixture(t);
  const builder = path.join(f.source, "scripts/package-platform-releases.mjs");
  await cp(path.join(repositoryRoot, "scripts/package-platform-releases.mjs"), builder);
  await writeFile(path.join(f.source, "scripts/build-executor.mjs"), "// fixture build input\n");
  await writeFile(path.join(f.source, "runtime/executor/main.go"), "package main\n");
  const git = args => execFile("git", args, { cwd: f.source, env: f.env });
  await git(["init"]);
  await git(["add", "."]);
  await git(["-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "fixture"]);
  const { stdout: committed } = await git(["rev-parse", "HEAD"]);
  const env = { ...f.env, GITHUB_SHA: committed.trim() };
  const manifestFile = path.join(f.source, "dist/releases/release-manifest.json");
  await execFile(process.execPath, [builder], { env });
  assert.equal(JSON.parse(await readFile(manifestFile, "utf8")).sourceCommit, null);
  for (const file of ["runtime/executor/main.go", "scripts/build-executor.mjs", "skills/puretokens-image/SKILL.md"]) {
    const before = await readFile(path.join(f.source, file), "utf8");
    await writeFile(path.join(f.source, file), `${before}\nmodified fixture\n`);
    await execFile(process.execPath, [builder], { env });
    assert.equal(JSON.parse(await readFile(manifestFile, "utf8")).sourceCommit, null, file);
    await writeFile(path.join(f.source, file), before);
  }
  await writeFile(path.join(f.source, "runtime/executor/untracked.go"), "package main\n");
  await execFile(process.execPath, [builder], { env });
  assert.equal(JSON.parse(await readFile(manifestFile, "utf8")).sourceCommit, null);
});

test("pinned selector handles new hosts even beside a marked old installer", async t => {
 const f = await fixture(t);
 const env = { ...await mockDownloads(f), ZCODE_DATA_BASE_DIR: path.join(f.root, "ZCode base") };
 const entry = path.join(f.root,"old-entry"); await mkdir(entry);
 await cp(path.join(f.source,"runtime/puretokens-skill-fetch.sh"),path.join(entry,"puretokens-skill-fetch.sh"));
 await writeFile(path.join(entry,"puretokens-skill-install.sh"), '#!/bin/sh\n# puretokens-locate-v1\necho unsupported-host >&2\nexit 1\n');
 await execFile("sh",[path.join(entry,"puretokens-skill-fetch.sh"),"install","--host","zcode"],{env});
 assert.ok(await readFile(path.join(env.ZCODE_DATA_BASE_DIR,".zcode/skills/puretokens-image/SKILL.md")));
 const requests=await readFile(path.join(f.root,"requests"),"utf8");
 assert.ok(requests.includes(`/releases/download/v${version}/puretokens-skill-install.sh`));
});

test("same-version install and update verify inventories without archive, writes or init", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  await execFile("sh", [wrapper, "install", "--target", f.target], {env});
  async function snapshot(directory) {
    const result = {};
    for (const entry of await readdir(directory, {withFileTypes:true})) {
      const full = path.join(directory, entry.name);
      if (entry.isDirectory()) result[entry.name] = await snapshot(full);
      else result[entry.name] = {sha: createHash("sha256").update(await readFile(full)).digest("hex"), mtime:(await stat(full)).mtimeMs};
    }
    return result;
  }
  const before = await snapshot(f.target);
  for (const command of ["install", "update"]) {
    await writeFile(path.join(f.root, "requests"), "");
    const {stdout} = await execFile("sh", [wrapper, command, "--target", f.target, "--host", "codex"], {env});
    assert.match(stdout, /already current and verified/);
    assert.doesNotMatch(stdout, /Pure Tokens Skill init:/);
    assert.deepEqual(await snapshot(f.target), before);
    const requests = (await readFile(path.join(f.root, "requests"), "utf8")).trim().split("\n");
    assert.equal(requests.length, 2);
    assert.ok(requests.every(url=>!url.endsWith(".zip")));
  }
});

test("Gemini same-version verification accepts shared roots, HOME aliases and trailing slashes", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  const shared = path.join(f.env.HOME, ".agents/skills");
  await execFile("sh", [wrapper, "install", "--target", shared], {env});
  const alias = path.join(f.root, "home-alias");
  await symlink(f.env.HOME, alias, "dir");
  for (const [home, target] of [[f.env.HOME, null], [alias, null], [alias, `${shared}/`]]) {
    await writeFile(path.join(f.root, "requests"), "");
    const args = [wrapper, "update", "--host", "gemini-cli"];
    if (target) args.push("--target", target);
    const {stdout} = await execFile("sh", args, {env:{...env, HOME:home}});
    assert.match(stdout, /already current and verified/);
    assert.doesNotMatch(stdout, /Pure Tokens Skill init:/);
    assert.equal((await readFile(path.join(f.root,"requests"),"utf8")).trim().split("\n").length, 2);
  }
});

test("same-version shortcut rejects a newer sync between identity and inventory verification", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  await execFile("sh", [wrapper, "install", "--target", f.target], {env});
  const newer = path.join(f.root, "newer-source");
  await cp(f.source, newer, {recursive:true});
  await writeFile(path.join(newer,"package.json"), JSON.stringify({version:"99.0.0"},null,2));
  await writeFile(path.join(newer,"skills/puretokens-image/newer-marker"),"preserve newer installation");
  const shim = path.join(f.root, "tools/sh");
  await writeFile(shim, `#!/bin/sh
if [ "\${2:-}" = verify-installed ]; then
  /bin/sh "$PT_NEWER_SOURCE/runtime/puretokens-skill-install.sh" sync --target "$PT_FIXTURE_TARGET" --source "$PT_NEWER_SOURCE" > "$PT_FETCH_FIXTURE/concurrent-update"
fi
exec /bin/sh "$@"
`, {mode:0o755});
  await writeFile(path.join(f.root,"requests"),"");
  await assert.rejects(execFile("sh", [wrapper,"update","--target",f.target], {
    env:{...env, PT_NEWER_SOURCE:newer, PT_FIXTURE_TARGET:f.target}
  }), error => {
    assert.match(error.stderr, /installed release changed during verification/);
    assert.doesNotMatch(error.stdout, /already current and verified/);
    return true;
  });
  assert.equal(JSON.parse(await readFile(path.join(f.target,".puretokens-executor/runtime.json"),"utf8")).version,"99.0.0");
  assert.equal(await readFile(path.join(f.target,"puretokens-image/newer-marker"),"utf8"),"preserve newer installation");
  assert.doesNotMatch(await readFile(path.join(f.root,"requests"),"utf8"),/\.zip/);
});

test("same-version changed or missing files and recovery stages stop without repair", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  await execFile("sh", [wrapper, "install", "--target", f.target], {env});
  const file = path.join(f.target, "puretokens-image/SKILL.md");
  const original = await readFile(file);
  for (const change of ["modified", "missing", "stage", "executor"]) {
    if (change === "modified") await writeFile(file, "user changes");
    if (change === "missing") await rm(file);
    const stage = path.join(f.target, ".puretokens-skill-stage.unresolved");
    if (change === "stage") await mkdir(stage);
    const binary = path.join(f.target, ".puretokens-executor/puretokens-api");
    const originalBinary = change === "executor" ? await readFile(binary) : null;
    if (originalBinary) await writeFile(binary, "invalid executable");
    await writeFile(path.join(f.root, "requests"), "");
    await assert.rejects(execFile("sh", [wrapper, "update", "--target", f.target], {env}));
    assert.doesNotMatch(await readFile(path.join(f.root, "requests"), "utf8"), /\.zip/);
    if (change === "modified") assert.equal(await readFile(file,"utf8"), "user changes");
    if (change === "missing") await assert.rejects(stat(file));
    if (change === "stage") await rm(stage, {recursive:true});
    if (originalBinary) await writeFile(binary, originalBinary);
    await writeFile(file, original);
  }
});

test("stable selector checksum is verified before execution", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const release = JSON.parse(await readFile(path.join(f.root,"release.json"),"utf8"));
  release.installers.shell.sha256 = "0".repeat(64);
  await writeFile(path.join(f.root,"release.json"),JSON.stringify(release,null,2));
  await assert.rejects(execFile("sh",[path.join(f.source,"runtime/puretokens-skill-fetch.sh"),"install","--target",f.target],{env}),/selector checksum mismatch/);
  await assert.rejects(stat(f.target));
});

test("explicit stable repair recovers verified committed stages and missing managed files only", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source, "runtime/puretokens-skill-fetch.sh");
  await execFile("sh", [wrapper, "install", "--target", f.target], {env});
  const stage = path.join(f.target, ".puretokens-skill-stage.committed");
  await mkdir(path.join(stage, "backup"), {recursive:true});
  await cp(path.join(f.target,"puretokens-image"),path.join(stage,"backup/puretokens-image"),{recursive:true});
  await writeFile(path.join(stage,"transaction-v1"),"");
  await writeFile(path.join(stage,"committed"),"");
  await writeFile(path.join(stage,"plan"),"replace puretokens-image\n");
  await assert.rejects(execFile("sh",[wrapper,"update","--target",f.target],{env}), /transaction requires review/);
  await execFile("sh",[wrapper,"repair","--target",f.target],{env});
  await assert.rejects(stat(stage), {code:"ENOENT"});
  const file = path.join(f.target,"puretokens-image/SKILL.md");
  const original = await readFile(file,"utf8");
  await rm(file);
  await execFile("sh",[wrapper,"repair","--target",f.target],{env});
  assert.equal(await readFile(file,"utf8"),original);
  await writeFile(file,"preserve my edits");
  await assert.rejects(execFile("sh",[wrapper,"repair","--target",f.target],{env}), /local additions, changes or unknown ownership/);
  assert.equal(await readFile(file,"utf8"),"preserve my edits");
});

test("explicit repair preserves unknown or modified recovery stage data", async t => {
  const f = await fixture(t);
  const env = await mockDownloads(f);
  const wrapper = path.join(f.source,"runtime/puretokens-skill-fetch.sh");
  await execFile("sh",[wrapper,"install","--target",f.target],{env});
  const stage = path.join(f.target,".puretokens-skill-stage.personal");
  await mkdir(path.join(stage,"backup"),{recursive:true});
  await writeFile(path.join(stage,"transaction-v1"),"");
  await writeFile(path.join(stage,"committed"),"");
  await writeFile(path.join(stage,"plan"),"replace puretokens-image\n");
  const personal=path.join(stage,"personal.txt");
  await writeFile(personal,"keep transaction addition");
  const before=await readFile(path.join(f.target,"puretokens-image/SKILL.md"));
  await assert.rejects(execFile("sh",[wrapper,"repair","--target",f.target],{env}),/failed verification/);
  assert.equal(await readFile(personal,"utf8"),"keep transaction addition");
  assert.deepEqual(await readFile(path.join(f.target,"puretokens-image/SKILL.md")),before);
});
