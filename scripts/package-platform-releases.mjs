// Maintainer-only packaging. A platform release includes ten Skills and one
// executable; no source checkout, other architectures or migration archives.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cp, lstat, mkdir, mkdtemp, readFile, readdir, rename, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const manifestBytes = await readFile(path.join(root, "runtime/executor/manifest.json"));
const manifest = JSON.parse(manifestBytes);
const destination = path.join(root, "dist", "releases");
await mkdir(destination, { recursive: true });
const files = {};
const publicFiles = ["README.md", "package.json", "runtime/puretokens-skill-install.sh", "runtime/puretokens-skill-install.ps1", "runtime/puretokens-skill-fetch.sh", "runtime/puretokens-skill-fetch.ps1"];
// Include build inputs, not just copied outputs: committed binaries cannot
// attest to uncommitted Go code, go.mod/go.sum, or a modified build recipe.
const sourceScopes = ["README.md", "package.json", "skills", "runtime", "scripts/build-executor.mjs", "scripts/executor-build-proof.mjs", "scripts/package-platform-releases.mjs"];
async function snapshot(directory, scopes) {
  const result = new Map();
  async function visit(relative) {
    const full = path.join(directory, relative);
    let info;
    try { info = await lstat(full); } catch (error) { if (error.code === "ENOENT") return; throw error; }
    if (info.isDirectory()) {
      for (const name of await readdir(full)) await visit(`${relative}/${name}`);
    } else if (info.isFile()) {
      const bytes = await readFile(full);
      result.set(relative, createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex"));
    } else {
      result.set(relative, "unsupported-file-type");
    }
  }
  for (const scope of scopes) await visit(scope);
  return result;
}
function sameSnapshot(left, right) {
  return left.size === right.size && [...left].every(([file, hash]) => right.get(file) === hash);
}
function matchingCommit(actual) {
  try {
    const git = args => execFileSync("git", args, { cwd: root, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
    if (path.resolve(git(["rev-parse", "--show-toplevel"])) !== root) return null;
    const commit = git(["rev-parse", "HEAD"]);
    if (!/^[0-9a-f]{40}$/.test(commit) || (process.env.GITHUB_SHA && process.env.GITHUB_SHA !== commit)) return null;
    const committed = new Map(git(["ls-tree", "-rz", "--full-tree", commit, "--", ...sourceScopes]).split("\0").filter(Boolean).map(record => {
      const match = record.match(/^([0-9]+) blob ([0-9a-f]{40})\t(.+)$/);
      if (!match || match[1] === "120000") throw new Error("unsupported committed source entry");
      return [match[3], match[2]];
    }));
    return sameSnapshot(actual, committed) ? commit : null;
  } catch { return null; }
}
const sourceSnapshot = await snapshot(root, sourceScopes);
let buildVerified = false;
try {
  const { verifyExecutorBuild } = await import("./executor-build-proof.mjs");
  await verifyExecutorBuild(root);
  buildVerified = true;
} catch { /* Unverified bytes can only produce draft candidates. */ }
let sourceCommit = buildVerified ? matchingCommit(sourceSnapshot) : null;
if (sourceSnapshot.get("runtime/executor/manifest.json") !== createHash("sha1").update(`blob ${manifestBytes.length}\0`).update(manifestBytes).digest("hex")) sourceCommit = null;
for (const [platform, artifact] of Object.entries(manifest.artifacts)) {
  const temp = await mkdtemp(path.join(os.tmpdir(), "pt-release-"));
  try {
    const bundle = path.join(temp, "puretokens-skill");
    await mkdir(path.join(bundle, "runtime/executor/bin"), { recursive: true });
    await cp(path.join(root, "skills"), path.join(bundle, "skills"), { recursive: true });
    const extension = platform.startsWith("windows-") ? ".ps1" : ".sh";
    const platformFiles = publicFiles.filter(file => !file.startsWith("runtime/") || file.endsWith(extension));
    for (const file of platformFiles) await cp(path.join(root, file), path.join(bundle, file));
    const binary = await readFile(path.join(root, "runtime/executor", artifact.path));
    if (createHash("sha256").update(binary).digest("hex") !== artifact.sha256) throw new Error(`Binary checksum mismatch: ${platform}`);
    await writeFile(path.join(bundle, "runtime/executor", artifact.path), binary, { mode: 0o755 });
    const expectedBundle = new Map([...sourceSnapshot].filter(([file]) => file.startsWith("skills/") || platformFiles.includes(file) || file === `runtime/executor/${artifact.path}`));
    if (!sameSnapshot(await snapshot(bundle, ["README.md", "package.json", "skills", "runtime"]), expectedBundle)) sourceCommit = null;
    await writeFile(path.join(bundle, "runtime/executor/manifest.json"), JSON.stringify({ ...manifest, artifacts: { [platform]: artifact } }, null, 2));
    const name = `puretokens-skill-${manifest.version}-${platform}.zip`;
    // zip updates existing archives, retaining removed source members. Always
    // build a fresh archive, then replace the previous candidate as one rename.
    const temporaryArchive = path.join(destination, `.${name}.${process.pid}.tmp`);
    try {
      await rm(temporaryArchive, { force: true });
      execFileSync("zip", ["-qr", temporaryArchive, "puretokens-skill"], { cwd: temp });
      await rename(temporaryArchive, path.join(destination, name));
    } finally { await rm(temporaryArchive, { force: true }); }
    const bytes = await readFile(path.join(destination, name));
    files[platform] = { filename: name, sha256: createHash("sha256").update(bytes).digest("hex"), bytes: bytes.length, executorSha256: artifact.sha256 };
  } finally { await rm(temp, { recursive: true, force: true }); }
}
const installers = {};
for (const [system, extension] of [["shell", "sh"], ["powershell", "ps1"]]) {
  const filename = `puretokens-skill-install.${extension}`;
  const bytes = await readFile(path.join(root, "runtime", filename));
  await writeFile(path.join(destination, filename), bytes);
  await cp(path.join(root, "runtime", `puretokens-skill-fetch.${extension}`), path.join(destination, `puretokens-skill-fetch.${extension}`));
  installers[system] = { filename, sha256: createHash("sha256").update(bytes).digest("hex") };
}
if (!sameSnapshot(sourceSnapshot, await snapshot(root, sourceScopes)) || matchingCommit(sourceSnapshot) !== sourceCommit) sourceCommit = null;
await writeFile(path.join(destination, "release-manifest.json"), `${JSON.stringify({ schemaVersion: 2, version: manifest.version, sourceCommit, installers, files }, null, 2)}\n`);
console.log(sourceCommit
  ? `Prepared ${Object.keys(files).length} platform-only releases matching source commit ${sourceCommit}.`
  : `Prepared ${Object.keys(files).length} draft platform candidates. sourceCommit is unavailable because source/build inputs are not verified against a clean commit; fetch will not select these as published assets.`);
