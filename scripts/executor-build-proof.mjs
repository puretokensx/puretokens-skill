// Repository development only. No compiler or build proof is needed by users.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { lstat, readFile, readdir, mkdtemp, rm } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { fileURLToPath } from "node:url";
export const buildRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const hash = bytes => createHash("sha256").update(bytes).digest("hex");
export async function executorInputs(root) {
  const config = JSON.parse(await readFile(path.join(root, "runtime/executor/build-config.json")));
  if (config.schemaVersion !== 1 || !/^go1\.[0-9]+\.[0-9]+$/.test(config.toolchain) ||
      !Array.isArray(config.platforms) || config.platforms.length !== 6 || new Set(config.platforms).size !== 6 ||
      config.platforms.some(p => !/^(darwin|linux|windows)-(amd64|arm64)$/.test(p)) ||
      JSON.stringify(config.flags) !== JSON.stringify(["-trimpath", "-buildvcs=false"]) || config.linkerFlags !== "-s -w -buildid=") throw new Error("Unsupported executor build configuration");
  const files = ["package.json", "scripts/build-executor.mjs", "scripts/executor-build-proof.mjs"];
  async function visit(relative) {
    for (const name of await readdir(path.join(root,relative))) {
      if (["bin", "testdata"].includes(name)) continue;
      const file = `${relative}/${name}`;
      const info = await lstat(path.join(root,file));
      if (info.isDirectory()) await visit(file);
      else if ((name.endsWith(".go") && !name.endsWith("_test.go")) || ["go.mod","go.sum","build-config.json","audio-profiles.json","reviewed-media-policy.json","reviewed-video-profiles.json"].includes(name)) {
        if (!info.isFile()) throw new Error("Unsupported build input");
        files.push(file);
      }
    }
  }
  await visit("runtime/executor");
  files.sort(); const digests = {};
  for (const file of files) digests[file] = hash(await readFile(path.join(root,file)));
  return { config, sourceSha256: hash(JSON.stringify(digests)), inputs: digests };
}
export function buildEnvironment(platform) {
  const [GOOS, GOARCH] = platform.split("-");
  return { ...process.env, GOOS, GOARCH, CGO_ENABLED:"0", GOAMD64:"v1", GOARM64:"v8.0", GOEXPERIMENT:"", GOFLAGS:"", GOWORK:"off", GOENV:"off", GOTOOLCHAIN:"local" };
}
export function checkToolchain(config, directory) {
  const actual = execFileSync("go",["env","GOVERSION"],{cwd:directory,env:buildEnvironment("linux-amd64"),encoding:"utf8"}).trim();
  if(actual!==config.toolchain) throw new Error(`Executor build requires ${config.toolchain}; actual compiler is ${actual}`);
}
export function compileExecutor(root, inputs, version, platform, output) {
  const linkerFlags = `${inputs.config.linkerFlags} -X main.executorVersion=${version} -X main.executorSourceSHA256=${inputs.sourceSha256}`;
  execFileSync("go",["build",...inputs.config.flags,"-ldflags",linkerFlags,"-o",output,"."],{cwd:path.join(root,"runtime/executor"),env:buildEnvironment(platform),stdio:"pipe"});
}
export async function verifyExecutorBuild(root=buildRoot,{rebuild=false}={}) {
  const inputs = await executorInputs(root);
  const directory=path.join(root,"runtime/executor");
  const proof=JSON.parse(await readFile(path.join(directory,"build-proof.json")));
  const manifest=JSON.parse(await readFile(path.join(directory,"manifest.json")));
  const pkg=JSON.parse(await readFile(path.join(root,"package.json")));
  if(proof.schemaVersion!==1 || proof.sourceSha256!==inputs.sourceSha256 || proof.toolchain!==inputs.config.toolchain || proof.version!==pkg.version || manifest.version!==pkg.version || JSON.stringify(proof.inputs)!==JSON.stringify(inputs.inputs)) throw new Error("Executor source/build inputs changed; rebuild all platform executors");
  if (Object.keys(manifest.artifacts??{}).sort().join()!==[...inputs.config.platforms].sort().join() || JSON.stringify(proof.artifacts)!==JSON.stringify(manifest.artifacts)) throw new Error("Executor artifact set differs from build proof");
  let temporary;
  if(rebuild){checkToolchain(inputs.config,directory);temporary=await mkdtemp(path.join(os.tmpdir(),"pt-build-verify-"));}
  try {
    for(const platform of inputs.config.platforms) {
      const artifact=manifest.artifacts[platform];
      const expectedPath=`bin/puretokens-api-${platform}${platform.startsWith("windows-")?".exe":""}`;
      if(artifact.path!==expectedPath || !/^[0-9a-f]{64}$/.test(artifact.sha256)) throw new Error("Invalid executor artifact path or digest");
      const file=path.join(directory,artifact.path);
      const binaryBytes = await readFile(file);
      if(hash(binaryBytes)!==artifact.sha256) throw new Error(`Executor bytes differ: ${platform}`);
      // Inspect the actual Go binary's embedded compiler/linker metadata.
      const metadata=execFileSync("go",["version","-m",file],{encoding:"utf8",env:buildEnvironment(platform)});
      if(!metadata.split("\n")[0].endsWith(`: ${inputs.config.toolchain}`) || !binaryBytes.includes(Buffer.from(inputs.sourceSha256)) || !metadata.includes(`GOOS=${platform.split("-")[0]}`) || !metadata.includes(`GOARCH=${platform.split("-")[1]}`)) throw new Error(`Executor embedded build identity differs: ${platform}`);
      if(rebuild){
        const output=path.join(temporary,platform);
        compileExecutor(root,inputs,pkg.version,platform,output);
        if(hash(await readFile(output))!==artifact.sha256) throw new Error(`Rebuilt executor does not match committed bytes: ${platform}`);
      }
    }
    if((await executorInputs(root)).sourceSha256!==inputs.sourceSha256) throw new Error("Executor inputs changed during verification");
  } finally {if(temporary)await rm(temporary,{recursive:true,force:true});}
  return proof;
}
if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  await verifyExecutorBuild(buildRoot,{rebuild:process.argv.includes("--rebuild")});
  console.log("Executor source, embedded build identities and six artifact hashes verified"+(process.argv.includes("--rebuild")?" against reproducible builds.":"."));
}
