import assert from "node:assert/strict";
import {test} from "node:test";
import {execFileSync} from "node:child_process";
import {createHash} from "node:crypto";
import {cp, mkdir, mkdtemp, readFile, rm, writeFile} from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import {validatePlatformReleases} from "../scripts/validate-platform-releases.mjs";
import {repositoryRoot} from "../scripts/skill-registry.mjs";

const platforms = ["darwin-amd64","darwin-arm64","linux-amd64","linux-arm64","windows-amd64","windows-arm64"];
test("release workflows package the checked-out commit, not the dispatch ref", async () => {
  for (const workflow of ["release.yml", "publish.yml"]) {
    const source = await readFile(path.join(repositoryRoot, ".github/workflows", workflow), "utf8");
    assert.match(source, /run: GITHUB_SHA="\$\(git rev-parse HEAD\)" npm run release:package/);
  }
});

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(),"pt-platform-gate-"));
  t.after(()=>rm(root,{recursive:true,force:true}));
  await mkdir(path.join(root,"runtime/executor/bin"),{recursive:true});
  await mkdir(path.join(root,"scripts"));
  await writeFile(path.join(root,"README.md"),"Release fixture\n");
  await writeFile(path.join(root,"LICENSE"),"MIT License fixture\n");
  await writeFile(path.join(root,"package.json"),JSON.stringify({version:"0.18.1"}));
  await cp(path.join(repositoryRoot,"skills"),path.join(root,"skills"),{recursive:true});
  for (const extension of ["sh","ps1"]) for (const command of ["install","fetch"]) {
    const filename = `puretokens-skill-${command}.${extension}`;
    await cp(path.join(repositoryRoot,"runtime",filename),path.join(root,"runtime",filename));
  }
  const artifacts = {};
  for (const platform of platforms) {
    const bytes = Buffer.from(`fixture-not-an-executable:${platform}`);
    const relative = `bin/puretokens-api-${platform}${platform.startsWith("windows-")?".exe":""}`;
    await writeFile(path.join(root,"runtime/executor",relative),bytes);
    artifacts[platform] = {path:relative,sha256:createHash("sha256").update(bytes).digest("hex")};
  }
  await writeFile(path.join(root,"runtime/executor/manifest.json"),JSON.stringify({schemaVersion:1,name:"puretokens-api-executor",version:"0.18.1",artifacts}));
  await cp(path.join(repositoryRoot,"scripts/package-platform-releases.mjs"),path.join(root,"scripts/package-platform-releases.mjs"));
  execFileSync(process.execPath,[path.join(root,"scripts/package-platform-releases.mjs")]);
  return {root,directory:path.join(root,"dist/releases")};
}
test("six real archive inventories contain only their system scripts and one binary",async t=>{
  const f = await fixture(t);
  const release = await validatePlatformReleases(f.root);
  assert.equal(release.sourceCommit,null);
  assert.equal(Object.keys(release.files).length,6);
  await assert.rejects(validatePlatformReleases(f.root,{publishable:true}),/Draft candidates/);
});
test("every platform archive includes the exact source license",async t=>{
  const f = await fixture(t);
  const release = await validatePlatformReleases(f.root);
  const license = await readFile(path.join(f.root,"LICENSE"));
  for (const entry of Object.values(release.files)) {
    const bundled = execFileSync("unzip",["-p",path.join(f.directory,entry.filename),"puretokens-skill/LICENSE"]);
    assert.deepEqual(bundled,license);
  }
});
test("stable archive gate rejects missing platforms and altered selectors",async t=>{
  const f = await fixture(t);
  const file = path.join(f.directory,"release-manifest.json");
  const release = JSON.parse(await readFile(file,"utf8"));
  delete release.files["windows-arm64"];
  await writeFile(file,JSON.stringify(release));
  await assert.rejects(validatePlatformReleases(f.root),/all six/);
  execFileSync(process.execPath,[path.join(f.root,"scripts/package-platform-releases.mjs")]);
  await writeFile(path.join(f.directory,"puretokens-skill-install.sh"),"altered script");
  await assert.rejects(validatePlatformReleases(f.root),/differs from source/);
});
test("archive gate rejects unneeded OS scripts even with a recomputed outer checksum",async t=>{
  const f = await fixture(t);
  const file = path.join(f.directory,"release-manifest.json");
  const release = JSON.parse(await readFile(file,"utf8"));
  const entry = release.files["darwin-arm64"];
  const extra = path.join(f.root,"extra");
  await mkdir(path.join(extra,"puretokens-skill/runtime"),{recursive:true});
  await writeFile(path.join(extra,"puretokens-skill/runtime/puretokens-skill-install.ps1"),"unneeded");
  execFileSync("zip",["-q",path.join(f.directory,entry.filename),"puretokens-skill/runtime/puretokens-skill-install.ps1"],{cwd:extra});
  const bytes = await readFile(path.join(f.directory,entry.filename));
  entry.bytes = bytes.length; entry.sha256 = createHash("sha256").update(bytes).digest("hex");
  await writeFile(file,JSON.stringify(release));
  await assert.rejects(validatePlatformReleases(f.root),/missing or extra files/);
});
