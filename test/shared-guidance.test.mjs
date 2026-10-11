import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { syncSkillGuidance } from "../scripts/sync-skill-guidance.mjs";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

test("shared guidance is generated, detects drift and synchronizes self-contained Skills", async t => {
  assert.deepEqual(await syncSkillGuidance(repositoryRoot), []);
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-guidance-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  await cp(path.join(repositoryRoot, "skills"), path.join(root, "skills"), { recursive: true });
  await mkdir(path.join(root, "runtime/executor"), { recursive: true });
  await cp(path.join(repositoryRoot, "runtime/executor/audio-profiles.json"), path.join(root, "runtime/executor/audio-profiles.json"));
  await mkdir(path.join(root, "references/skill-fragments"), { recursive: true });
  for (const file of ["host-delivery.json", "host-support.json", "desktop-hosts.md", "direct-api-execution-contract.json", "operation-guidance.json", "skill-fragments/host-binding.md", "skill-fragments/receipt-guide.md", "skill-fragments/media-workflows.md", "skill-fragments/model-alternatives.md", "skill-fragments/model-groups.md"]) {
    await cp(path.join(repositoryRoot, "references", file), path.join(root, "references", file));
  }
  const fragment = path.join(root, "references/skill-fragments/host-binding.md");
  const next = (await readFile(fragment, "utf8")).replace("首次调用前", "开始调用前");
  await writeFile(fragment, next);
  const drift = await syncSkillGuidance(root);
  assert.equal(drift.filter(file => file.endsWith("/SKILL.md")).length, 8);
  assert.equal(drift.filter(file => file.endsWith("/desktop-hosts.md")).length, 8);
  assert.deepEqual(await syncSkillGuidance(root, { write: true }), drift);
  assert.deepEqual(await syncSkillGuidance(root), []);
  const registry = JSON.parse(await readFile(path.join(root, "skills/index.json"), "utf8"));
  for (const skill of registry.skills) {
    const text = await readFile(path.join(root, skill.entry), "utf8");
    assert.ok(text.includes(next.trim()));
    const manifest = JSON.parse(await readFile(path.join(root, skill.manifest), "utf8"));
    if (manifest.sourceSha256) assert.equal(manifest.sourceSha256, createHash("sha256").update(text).digest("hex"));
    assert.doesNotMatch(text, /\.\.\/\.\.\/references\/skill-fragments/);
  }
});

test("connection and initialization have one consistent intent mapping", async () => {
  const contract = JSON.parse(await readFile(path.join(repositoryRoot, "skills/puretokens-update/references/execution-contract.json"), "utf8"));
  assert.deepEqual(contract.operations.intentRouting, {
    publicServiceIdentity: "connection_public_identity_only",
    generalConnectionCheck: "init",
    explicitAuthenticationCheck: "init",
    installationDiagnostics: "doctor",
    usageQuestion: "local_guide_only",
    doNotChainChecksForOneIntent: true
  });
  const update = await readFile(path.join(repositoryRoot, "skills/puretokens-update/SKILL.md"), "utf8");
  assert.doesNotMatch(update, /同步后执行同宿主 init/);
  assert.match(update, /不另跑一次 init/);
});
