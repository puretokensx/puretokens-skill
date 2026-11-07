// Maintainer-only gate. User installers never need Git or this script.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { lstat, readFile, readdir, realpath } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Match the packager's source boundary, excluding executor tests only.
const sourceScopes = ["LICENSE", "README.md", "package.json", "skills", "runtime", "scripts/build-executor.mjs", "scripts/executor-build-proof.mjs", "scripts/package-platform-releases.mjs"];
const isTest = file => file.startsWith("runtime/executor/") && file.endsWith("_test.go");
const equal = (a, b) => a.size === b.size && [...a].every(([file, hash]) => b.get(file) === hash);

function compareVersions(a, b) {
  if (![a, b].every(value => /^\d+\.\d+\.\d+$/.test(value))) throw new Error("Release history contains an invalid version.");
  const left = a.split(".").map(BigInt), right = b.split(".").map(BigInt);
  for (let i = 0; i < 3; i++) {
    if (left[i] !== right[i]) return left[i] > right[i] ? 1 : -1;
  }
  return 0;
}

async function workingSnapshot(root) {
  const entries = new Map();
  async function visit(file) {
    if (isTest(file)) return;
    let info;
    try { info = await lstat(path.join(root, file)); }
    catch (error) { if (error.code === "ENOENT") return; throw error; }
    if (info.isDirectory()) {
      for (const name of await readdir(path.join(root, file))) await visit(`${file}/${name}`);
    } else if (info.isFile()) {
      const bytes = await readFile(path.join(root, file));
      entries.set(file, createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex"));
    } else {
      throw new Error("Distribution sources must contain regular files, not links or special files.");
    }
  }
  for (const scope of sourceScopes) await visit(scope);
  return entries;
}

export async function validateReleaseVersion(root) {
  const git = args => execFileSync("git", args, {cwd: root, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"]}).trim();
  try {
    if (await realpath(git(["rev-parse", "--show-toplevel"])) !== await realpath(root) ||
        git(["rev-parse", "--is-shallow-repository"]) !== "false") {
      throw new Error("incomplete history");
    }
  } catch {
    throw new Error("Release version validation requires this repository's complete Git history.");
  }
  const versionAt = commit => JSON.parse(git(["show", `${commit}:package.json`])).version;
  const snapshotAt = commit => {
    const entries = new Map();
    for (const line of git(["ls-tree", "-rz", "--full-tree", commit, "--", ...sourceScopes]).split("\0").filter(Boolean)) {
      const entry = line.match(/^(\d+) blob ([0-9a-f]{40})\t(.+)$/);
      if (!entry || !["100644", "100755"].includes(entry[1])) throw new Error("Unsupported distribution file type in release history.");
      if (!isTest(entry[3])) entries.set(entry[3], entry[2]);
    }
    return entries;
  };
  const commits = git(["rev-list", "--first-parent", "HEAD"]).split("\n");
  const head = commits[0];
  const headVersion = versionAt(head);
  const version = JSON.parse(await readFile(path.join(root, "package.json"), "utf8")).version;
  const comparison = compareVersions(version, headVersion);
  const working = await workingSnapshot(root);
  if (comparison < 0) throw new Error("The distribution version must not decrease.");
  if (comparison > 0) {
    return {version, status: "candidate", message: "Version-bumped working candidate; publication still requires clean committed source proof."};
  }
  const sameVersion = [];
  for (const commit of commits) {
    const previous = versionAt(commit);
    if (previous !== version) {
      if (compareVersions(version, previous) <= 0) throw new Error("The distribution version did not increase from its preceding version.");
      break;
    }
    sameVersion.push(commit);
  }
  const first = snapshotAt(sameVersion.at(-1));
  // Check every commit, including a same-version change subsequently reverted.
  for (const commit of sameVersion) {
    if (!equal(first, snapshotAt(commit))) throw new Error("A distributed version changed within its history; increment the version before distributing new content.");
  }
  if (!equal(first, working)) throw new Error("Distribution content changed without a version increase, including untracked or deleted files.");
  return {version, status: "immutable", message: "Current distribution bytes match every commit in this version's first-parent history."};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const result = await validateReleaseVersion(root);
  console.log(`Release version ${result.version}: ${result.status}. ${result.message}`);
}
