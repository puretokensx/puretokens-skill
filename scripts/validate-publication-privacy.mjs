import { createHash } from "node:crypto";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Regression fingerprints only: do not reintroduce the removed supplier names
// into public documents through the checker itself. This is not a secret scan.
const privateIdentityDigests = new Set([
  "b7e39c3ee29d76eef0396ea74c49825e644c80332839cc04ff35f1e76f4ed9c1",
  "0e6b69fbd1836bbd274587164bdbde0221901974ec4ade407e4bf4c357273eed"
]);
export function hasPrivateIdentity(text) {
  return (text.normalize("NFKC").toLowerCase().match(/[a-z0-9]+/g) || []).some(token =>
    privateIdentityDigests.has(createHash("sha256").update(token).digest("hex")));
}

export async function validatePublicAcceptance(root) {
  const errors = [];
  async function visit(relative) {
    for (const entry of await readdir(path.join(root, relative), { withFileTypes: true })) {
      const file = path.posix.join(relative, entry.name);
      if (entry.isDirectory()) await visit(file);
      else if (entry.isFile() && /\.(?:md|json)$/.test(entry.name) &&
          hasPrivateIdentity(await readFile(path.join(root, file), "utf8"))) {
        errors.push(file + ": removed private identity reappeared in public acceptance material");
      }
    }
  }
  await visit("references/acceptance");
  return errors;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const root = fileURLToPath(new URL("../", import.meta.url));
  const errors = await validatePublicAcceptance(root);
  if (errors.length) {
    for (const error of errors) process.stderr.write("error: " + error + "\n");
    process.exitCode = 1;
  } else console.log("Public acceptance privacy regression check passed; protocol compatibility exclusions remain separately documented.");
}
