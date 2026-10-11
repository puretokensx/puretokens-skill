import assert from "node:assert/strict";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { loadOperationGuidance } from "../scripts/operation-guidance.mjs";
import { syncSkillGuidance } from "../scripts/sync-skill-guidance.mjs";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-operation-guidance-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  for (const dir of ["skills", "references"]) {
    await cp(path.join(repositoryRoot, dir), path.join(root, dir), { recursive: true });
  }
  await mkdir(path.join(root, "runtime/executor"), { recursive: true });
  await cp(path.join(repositoryRoot, "runtime/executor/audio-profiles.json"),
    path.join(root, "runtime/executor/audio-profiles.json"));
  return root;
}

test("operation summaries reject incorrect lifecycle, recovery, command and delivery", async t => {
  const root = await fixture(t);
  const file = path.join(root, "references/operation-guidance.json");
  const source = JSON.parse(await readFile(file, "utf8"));
  for (const [id, key, value] of [
    ["music", "lifecycle", "sync"], ["music", "command", "audio"],
    ["music", "recovery", "local-artifact"], ["speech", "recovery", "task-record"],
    ["transcribe", "delivery", "attachment"], ["evaluate", "command", "submit"],
    ["sound", "operations", ["music"]]
  ]) {
    const changed = structuredClone(source);
    changed.operations.find(row => row.id === id)[key] = value;
    await writeFile(file, JSON.stringify(changed));
    await assert.rejects(loadOperationGuidance(root), /mismatch|differs/, `${id}.${key}`);
  }
  await writeFile(file, JSON.stringify(source));
  const contractFile = path.join(root, "skills/puretokens-audio/references/execution-contract.json");
  const contract = JSON.parse(await readFile(contractFile, "utf8"));
  contract.result.asynchronousMusic.asynchronous = false;
  await writeFile(contractFile, JSON.stringify(contract));
  await assert.rejects(loadOperationGuidance(root), /music: async contract differs/);
});

test("common operation and recovery guidance drift is detected and repaired without overwriting local prose", async t => {
  const root = await fixture(t);
  const targets = [
    "skills/puretokens-audio/SKILL.md",
    "skills/puretokens-update/references/usage-guide.md",
    "skills/puretokens-image/references/receipt-guide.md",
    "skills/puretokens-video/references/workflows.md",
    "skills/puretokens-image/references/model-alternatives.md",
    "skills/puretokens-video/references/model-alternatives.md",
    "skills/puretokens-image/references/model-groups.md",
    "skills/puretokens-video/references/model-groups.md",
    "skills/puretokens-models/references/model-groups.md"
  ];
  for (const file of targets) {
    let text = await readFile(path.join(root, file), "utf8");
    if (file.endsWith("SKILL.md") || file.endsWith("usage-guide.md")) {
      text = text.replace("原任务 `resume`；仅实际附件交付后 `delivered`", "音乐没有任务恢复");
      text += "\n保留本文件独立说明。\n";
    } else text += "\n错误：下载后自动重新生成。\n";
    await writeFile(path.join(root, file), text);
  }
  const drift = await syncSkillGuidance(root);
  for (const file of targets) assert.ok(drift.includes(file), file);
  await syncSkillGuidance(root, { write: true });
  assert.deepEqual(await syncSkillGuidance(root), []);
  for (const file of targets.slice(0, 2)) {
    const text = await readFile(path.join(root, file), "utf8");
    assert.ok(text.endsWith("保留本文件独立说明。\n"));
    assert.doesNotMatch(text, /音乐没有任务恢复/);
  }
  const source = await readFile(path.join(root, "references/skill-fragments/receipt-guide.md"), "utf8");
  for (const skill of ["image", "video", "audio", "evaluate"]) {
    assert.equal(await readFile(path.join(root, `skills/puretokens-${skill}/references/receipt-guide.md`), "utf8"), source);
  }
  const alternatives = await readFile(path.join(root, "references/skill-fragments/model-alternatives.md"), "utf8");
  for (const skill of ["image", "video"]) {
    assert.equal(await readFile(path.join(root, `skills/puretokens-${skill}/references/model-alternatives.md`), "utf8"), alternatives);
  }
});

test("media alternative contracts detect drift and preserve unrelated task semantics", async t => {
  const root = await fixture(t);
  const source = JSON.parse(await readFile(path.join(root, "references/direct-api-execution-contract.json"), "utf8"));
  for (const skill of ["image", "video"]) {
    const file = `skills/puretokens-${skill}/references/execution-contract.json`;
    const original = JSON.parse(await readFile(path.join(root, file), "utf8"));
    const changed = structuredClone(original);
    changed.submissionFailure.modelAlternatives.requiresExplicitNewTaskConsent = false;
    changed.submissionFailure.modelAlternatives.automaticModelSwitch = true;
    await writeFile(path.join(root, file), JSON.stringify(changed));
    assert.ok((await syncSkillGuidance(root)).includes(file));
    await syncSkillGuidance(root, { write: true });
    const repaired = JSON.parse(await readFile(path.join(root, file), "utf8"));
    assert.deepEqual(repaired.submissionFailure.modelAlternatives, source.asyncTaskHandling.modelAlternatives);
    assert.deepEqual(repaired, original);
  }
  source.asyncTaskHandling.modelAlternatives.requiresExplicitNewTaskConsent = false;
  await writeFile(path.join(root, "references/direct-api-execution-contract.json"), JSON.stringify(source));
  await assert.rejects(syncSkillGuidance(root), /explicit new-task consent/);
});

test("missing generated markers fail closed instead of silently skipping a Skill", async t => {
  const root = await fixture(t);
  const file = path.join(root, "skills/puretokens-audio/SKILL.md");
  await writeFile(file, (await readFile(file, "utf8")).replace("<!-- generated:operations -->", ""));
  await assert.rejects(syncSkillGuidance(root), /Expected one ordered operations guidance block/);
});
