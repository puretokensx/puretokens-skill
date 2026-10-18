import assert from "node:assert/strict";
import { test } from "node:test";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { buildRoot, executorInputs, verifyExecutorBuild } from "../scripts/executor-build-proof.mjs";

test("source changes and re-labelled old binaries fail the build proof gate", async t => {
 const root=await mkdtemp(path.join(os.tmpdir(),"pt-proof-test-"));t.after(()=>rm(root,{recursive:true,force:true}));
 await mkdir(path.join(root,"scripts"));await mkdir(path.join(root,"runtime"));
 for(const file of ["package.json","scripts/build-executor.mjs","scripts/executor-build-proof.mjs"])await cp(path.join(buildRoot,file),path.join(root,file));
 await cp(path.join(buildRoot,"runtime/executor"),path.join(root,"runtime/executor"),{recursive:true,filter:src=>!["bin","testdata"].includes(path.basename(src))});
 const policyFile=path.join(root,"runtime/executor/reviewed-media-policy.json");
 const policyBytes=await readFile(policyFile);
 const previousInputs=await executorInputs(root);
 assert.ok(previousInputs.inputs["runtime/executor/reviewed-media-policy.json"]);
 await writeFile(policyFile,Buffer.concat([policyBytes,Buffer.from("\n")]));
 assert.notEqual((await executorInputs(root)).sourceSha256,previousInputs.sourceSha256);
 await assert.rejects(verifyExecutorBuild(root),/inputs changed/);
 await writeFile(policyFile,policyBytes);
 const source=path.join(root,"runtime/executor/main.go");await writeFile(source,(await readFile(source,"utf8"))+"\n// changed after last build\n");
 await assert.rejects(verifyExecutorBuild(root),/inputs changed/);
 const proofFile=path.join(root,"runtime/executor/build-proof.json");const proof=JSON.parse(await readFile(proofFile));
 const inputs=await executorInputs(root);proof.sourceSha256=inputs.sourceSha256;proof.inputs=inputs.inputs;await writeFile(proofFile,JSON.stringify(proof));
 const platform=inputs.config.platforms[0];const file=proof.artifacts[platform].path;
 await mkdir(path.join(root,"runtime/executor/bin"));await cp(path.join(buildRoot,"runtime/executor",file),path.join(root,"runtime/executor",file));
 await assert.rejects(verifyExecutorBuild(root),/embedded build identity differs/);
});
