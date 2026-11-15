// Maintainer-only build. Neither this script nor its binary is an installer input.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { buildRoot, buildEnvironment, checkToolchain, executorInputs } from "./executor-build-proof.mjs";

const platform = process.argv[2];
const inputs = await executorInputs(buildRoot);
if (process.argv.length !== 3 || !inputs.config.platforms.includes(platform)) {
  throw new Error("Choose one supported OS-architecture for the maintainer diagnostic build.");
}
const directory = path.join(buildRoot, "runtime/executor");
checkToolchain(inputs.config, directory);
const packagePath = execFileSync("go", ["list", "-f", "{{.ImportPath}}", "."],
  { cwd: directory, env: buildEnvironment(platform), encoding: "utf8" }).trim();
const { version } = JSON.parse(await readFile(path.join(buildRoot, "package.json")));
const hash = bytes => createHash("sha256").update(bytes).digest("hex");
async function sourceIdentity() {
  const identity = { source: (await executorInputs(buildRoot)).sourceSha256 };
  const tests = (await readdir(directory)).filter(name => name.endsWith("_test.go")).sort();
  for (const name of tests) identity[name] = hash(await readFile(path.join(directory, name)));
  identity.builder = hash(await readFile(new URL(import.meta.url)));
  return hash(JSON.stringify(identity));
}
const source = await sourceIdentity();
const output = path.join(buildRoot, "dist/diagnostics", `puretokens-content-diagnostic-${platform}${platform.startsWith("windows-") ? ".exe" : ""}`);
await mkdir(path.dirname(output), { recursive: true });
try {
  execFileSync("go", ["test", "-c", "-tags=content_diagnostic", ...inputs.config.flags,
    "-ldflags", `${inputs.config.linkerFlags} -X ${packagePath}.executorVersion=${version} -X ${packagePath}.diagnosticSourceSHA256=${source}`,
    "-o", output, "."], { cwd: directory, env: buildEnvironment(platform), stdio: "pipe" });
  if (await sourceIdentity() !== source) throw new Error("Diagnostic sources changed during build.");
  const bytes = await readFile(output);
  if (!bytes.includes(Buffer.from(source))) throw new Error("Diagnostic source identity was not embedded.");
  const nativePlatform = `${process.platform === "win32" ? "windows" : process.platform}-${process.arch === "x64" ? "amd64" : process.arch}`;
  if (platform === nativePlatform) {
    const info = JSON.parse(execFileSync(output, ["--build-info"], { encoding: "utf8" }));
    if (info.executor_version !== version || info.diagnostic_source_sha256 !== source) throw new Error("Diagnostic build identity differs.");
  }
  const proof = { format: "puretokens-content-diagnostic-build-v1", platform, executorVersion: version, diagnosticSourceSHA256: source,
    artifact: path.basename(output), sha256: hash(bytes) };
  await writeFile(`${output}.json`, JSON.stringify(proof, null, 2) + "\n");
  console.log(JSON.stringify({ ...proof, path: output }));
} catch (error) {
  await rm(output, { force: true });
  await rm(`${output}.json`, { force: true });
  throw error;
}
