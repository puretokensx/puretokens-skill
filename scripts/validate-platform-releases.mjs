// Maintainer-only archive gate; never installed as a user runtime dependency.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { lstat, mkdir, mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const platforms = ["darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64", "windows-amd64", "windows-arm64"];
const sha256 = bytes => createHash("sha256").update(bytes).digest("hex");
const sourceScopes = ["LICENSE", "README.md", "package.json", "skills", "runtime", "scripts/build-executor.mjs", "scripts/executor-build-proof.mjs", "scripts/package-platform-releases.mjs"];
async function filesUnder(root, relative) {
  const info = await lstat(path.join(root, relative));
  if (info.isFile()) return [relative];
  if (!info.isDirectory()) throw new Error("Release sources contain a link or unsupported file.");
  const result = [];
  for (const name of await readdir(path.join(root, relative))) result.push(...await filesUnder(root, path.posix.join(relative, name)));
  return result.sort();
}

export async function validatePlatformReleases(root, {directory = path.join(root, "dist/releases"), publishable = false} = {}) {
  const release = JSON.parse(await readFile(path.join(directory, "release-manifest.json"), "utf8"));
  const executor = JSON.parse(await readFile(path.join(root, "runtime/executor/manifest.json"), "utf8"));
  const pkg = JSON.parse(await readFile(path.join(root, "package.json"), "utf8"));
  if (release.schemaVersion !== 2 || release.version !== pkg.version || executor.version !== pkg.version ||
      !/^\d+\.\d+\.\d+$/.test(release.version) || JSON.stringify(Object.keys(release.files ?? {}).sort()) !== JSON.stringify(platforms)) {
    throw new Error("A stable release requires the current version and all six platform archives.");
  }
  if (release.sourceCommit !== null && !/^[0-9a-f]{40}$/.test(release.sourceCommit ?? "")) throw new Error("Invalid release source commit.");
  if (publishable) {
    if (!release.sourceCommit) throw new Error("Draft candidates cannot be published without committed source proof.");
    const head = execFileSync("git", ["rev-parse", "HEAD"], {cwd:root, encoding:"utf8"}).trim();
    if (head !== release.sourceCommit) throw new Error("Release source commit does not match HEAD.");
    const records = execFileSync("git", ["ls-tree", "-rz", "--full-tree", head, "--", ...sourceScopes], {cwd:root, encoding:"utf8"}).split("\0").filter(Boolean);
    const committed = new Map(records.map(record => {
      const match = record.match(/^(100644|100755) blob ([0-9a-f]{40})\t(.+)$/);
      if (!match) throw new Error("Unsupported committed release source.");
      return [match[3], match[2]];
    }));
    const actual = new Map();
    for (const scope of sourceScopes) for (const file of await filesUnder(root, scope)) {
      const bytes = await readFile(path.join(root,file));
      actual.set(file, createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex"));
    }
    if (actual.size !== committed.size || [...actual].some(([file,hash])=>committed.get(file)!==hash)) throw new Error("Uncommitted release source changes cannot be published.");
  }
  for (const [system, extension] of [["shell","sh"],["powershell","ps1"]]) {
    const entry = release.installers?.[system];
    const filename = `puretokens-skill-install.${extension}`;
    if (entry?.filename !== filename || !/^[0-9a-f]{64}$/.test(entry.sha256 ?? "")) throw new Error("Stable installer metadata is invalid.");
    for (const script of [filename, `puretokens-skill-fetch.${extension}`]) {
      const bytes = await readFile(path.join(directory,script));
      if (!bytes.equals(await readFile(path.join(root,"runtime",script)))) throw new Error("Release bootstrap/selector differs from source.");
      if (script === filename && sha256(bytes) !== entry.sha256) throw new Error("Release selector checksum mismatch.");
    }
  }
  const skills = await filesUnder(root, "skills");
  if (skills.some(file=>/(^|\/)(?:__pycache__|\.DS_Store|\.pytest_cache)|\.(?:pyc|log)$/.test(file))) throw new Error("Release Skills contain cache or noise files.");
  const temporary = await mkdtemp(path.join(os.tmpdir(), "pt-release-validation-"));
  try {
    for (const platform of platforms) {
      const entry = release.files[platform];
      const filename = `puretokens-skill-${release.version}-${platform}.zip`;
      const extension = platform.startsWith("windows-") ? "ps1" : "sh";
      const binary = `runtime/executor/bin/puretokens-api-${platform}${extension==="ps1"?".exe":""}`;
      if (entry.filename !== filename || entry.executorSha256 !== executor.artifacts[platform]?.sha256) throw new Error("Release platform or executor identity mismatch.");
      const archive = path.join(directory,filename);
      const bytes = await readFile(archive);
      if (entry.bytes !== bytes.length || entry.sha256 !== sha256(bytes)) throw new Error("Release archive checksum or length mismatch.");
      const entries = execFileSync("unzip", ["-Z1", archive], {encoding:"utf8"}).trim().split("\n");
      if (entries.some(file=>!file.startsWith("puretokens-skill/") || /\\|(^|\/)\.\.?($|\/)/.test(file)) ||
          /^l\S+\s/m.test(execFileSync("unzip",["-Z","-l",archive],{encoding:"utf8"}))) throw new Error("Release archive has unsafe paths or links.");
      const extraction = path.join(temporary,platform);
      await mkdir(extraction);
      execFileSync("unzip",["-q",archive,"-d",extraction]);
      const bundle = path.join(extraction,"puretokens-skill");
      const copied = [...skills,"LICENSE","README.md","package.json",`runtime/puretokens-skill-install.${extension}`,`runtime/puretokens-skill-fetch.${extension}`,binary];
      const expected = [...copied,"runtime/executor/manifest.json"].sort();
      if (JSON.stringify(await filesUnder(bundle,"")) !== JSON.stringify(expected)) throw new Error("Release archive contains missing or extra files.");
      for (const file of copied) {
        if (!(await readFile(path.join(bundle,file))).equals(await readFile(path.join(root,file)))) throw new Error(`Release member differs from source: ${platform}/${file}`);
      }
      const bundledExecutor = JSON.parse(await readFile(path.join(bundle,"runtime/executor/manifest.json"),"utf8"));
      if (JSON.stringify(bundledExecutor) !== JSON.stringify({...executor,artifacts:{[platform]:executor.artifacts[platform]}})) throw new Error("Release executor manifest is not platform-only.");
    }
  } finally { await rm(temporary,{recursive:true,force:true}); }
  return release;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const args = process.argv.slice(2);
  let directory;
  let publishable = false;
  while (args.length) {
    const arg = args.shift();
    if (arg === "--publishable") publishable = true;
    else if (arg === "--directory" && args[0]) directory = path.resolve(args.shift());
    else throw new Error("Use --publishable and/or --directory <release-assets-directory>.");
  }
  const release = await validatePlatformReleases(root,{directory,publishable});
  console.log(`Verified six platform-only archives and bootstrap scripts for ${release.version}; ${release.sourceCommit ? "committed source" : "draft candidate, not publishable"}.`);
}
