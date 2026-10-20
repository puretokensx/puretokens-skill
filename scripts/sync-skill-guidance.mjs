import { createHash } from "node:crypto";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { repositoryRoot } from "./skill-registry.mjs";
import { loadOperationGuidance, renderOperationGuidance, replaceGuidanceBlock } from "./operation-guidance.mjs";

import { loadHostDelivery, renderHostDeliveryGuide } from "./host-delivery-guidance.mjs";
import { validModelID } from "./model-id.mjs";
import { buildAudioRequestSchema } from "./audio-request-schema.mjs";

const bindingPattern = /^宿主绑定与本地失败停止：[^\n]+$/gm;

export async function syncSkillGuidance(root, { write = false } = {}) {
  const read = file => readFile(path.join(root, file), "utf8");
  const registry = JSON.parse(await read("skills/index.json"));
  const support = JSON.parse(await read("references/host-support.json"));
  const binding = (await read("references/skill-fragments/host-binding.md")).trim();
  if (!/^宿主绑定与本地失败停止：[^\n]+$/.test(binding)) throw new Error("Invalid host-binding fragment");
  const source = await read("references/desktop-hosts.md");
  if (source.split("<!-- host-binding -->").length !== 2) throw new Error("Desktop guidance needs one host-binding marker");
  const desktop = source.replace("<!-- host-binding -->", binding);
  const hostList = support.supported.map(host => host.id).join("、");
  const deliveryGuide = renderHostDeliveryGuide(await loadHostDelivery(root));
  const changed = [];
  const operations = await loadOperationGuidance(root);
  const directContract = JSON.parse(await read("references/direct-api-execution-contract.json"));
  const modelAlternatives = directContract.asyncTaskHandling.modelAlternatives;
  if (!modelAlternatives || modelAlternatives.requiresExplicitNewTaskConsent !== true || modelAlternatives.automaticModelSwitch !== false) {
    throw new Error("Model alternatives must require explicit new-task consent without automatic switching");
  }
  async function output(file, next) {
    let current = "";
    try { current = await read(file); } catch (error) { if (error.code !== "ENOENT") throw error; }
    if (current === next) return;
    changed.push(file);
    if (write) {
      await mkdir(path.dirname(path.join(root, file)), { recursive: true });
      await writeFile(path.join(root, file), next);
    }
  }
  await output("references/skill-fragments/host-delivery.md", deliveryGuide);
  const audio = JSON.parse(await read("runtime/executor/audio-profiles.json"));
  const audioRoot = "skills/puretokens-audio/references";
  const ids = Object.keys(audio.models);
  await output("schemas/audio-request.schema.json", JSON.stringify(buildAudioRequestSchema(audio), null, 2) + "\n");
  const existing = await readdir(path.join(root, audioRoot, "profiles")).catch(error => {
    if (error.code !== "ENOENT") throw error;
    return [];
  });
  if (existing.some(file => !ids.some(id => file === `${id}.json`))) {
    throw new Error("Audio profiles contain an unreviewed file; review and remove it explicitly");
  }
  await output(`${audioRoot}/model-index.json`, JSON.stringify({
    schemaVersion: 1, reviewedAt: audio.reviewedAt,
    scope: "reviewed_contract_not_live_availability",
    defaults: { speech: "stepaudio-2.5-tts", transcribe: "stepaudio-2.5-asr", generate: "stepaudio-3-gen-preview", music: "stepaudio-3-music-preview" },
    models: ids.map(id => ({ id, operation: audio.models[id].operation, profile: `profiles/${id}.json` }))
  }, null, 2) + "\n");
  for (const [id, profile] of Object.entries(audio.models)) {
    if (!validModelID(id) || profile.model !== id) throw new Error("Invalid audio model identifier");
    await output(`${audioRoot}/profiles/${id}.json`, JSON.stringify({
      schemaVersion: 1, reviewedAt: audio.reviewedAt, ...profile
    }, null, 2) + "\n");
  }
  for (const skill of registry.skills) {
    const text = await read(skill.entry);
    if ([...text.matchAll(bindingPattern)].length !== 1) throw new Error(`${skill.name}: expected one host-binding paragraph`);
    let next = text.replace(bindingPattern, () => binding);
    const selected = operations.filter(row => row.skill === skill.name);
    if (selected.length) {
      next = replaceGuidanceBlock(next, "operations", renderOperationGuidance(selected));
      if (selected.some(row => row.delivery !== "file")) {
        await output(`skills/${skill.name}/references/receipt-guide.md`, await read("references/skill-fragments/receipt-guide.md"));
      }
    }
    if (["puretokens-image", "puretokens-video", "puretokens-audio"].includes(skill.name)) {
      await output(`skills/${skill.name}/references/workflows.md`, await read("references/skill-fragments/media-workflows.md"));
    }
    if (["puretokens-image", "puretokens-video"].includes(skill.name)) {
      await output(`skills/${skill.name}/references/model-alternatives.md`, await read("references/skill-fragments/model-alternatives.md"));
      const contractPath = `skills/${skill.name}/references/execution-contract.json`;
      const contract = JSON.parse(await read(contractPath));
      contract.submissionFailure.modelAlternatives = modelAlternatives;
      await output(contractPath, JSON.stringify(contract, null, 2) + "\n");
    }
    if (skill.name === "puretokens-models") {
      const contractPath = `skills/${skill.name}/references/execution-contract.json`;
      const contract = JSON.parse(await read(contractPath));
      contract.result.groupGuidance = directContract.modelGroupGuidance;
      await output(contractPath, JSON.stringify(contract, null, 2) + "\n");
    }
    if (["puretokens-image", "puretokens-video", "puretokens-models"].includes(skill.name)) {
      await output(`skills/${skill.name}/references/model-groups.md`, await read("references/skill-fragments/model-groups.md"));
    }
    if (["puretokens-image", "puretokens-video", "puretokens-audio", "puretokens-evaluate"].includes(skill.name)) {
      await output(`skills/${skill.name}/references/host-delivery.md`, deliveryGuide);
      if (!next.includes("references/host-delivery.md")) throw new Error(`${skill.name}: missing host delivery guide`);
    }
    if (skill.name === "puretokens-update") {
      if (!/^宿主 ID 为 [^；\n]+；/m.test(next)) throw new Error("Update: missing host list");
      next = next.replace(/^宿主 ID 为 [^；\n]+；/m, `宿主 ID 为 ${hostList}；`);
    }
    await output(skill.entry, next);
    await output(`skills/${skill.name}/references/desktop-hosts.md`, desktop);
    const manifest = JSON.parse(await read(skill.manifest));
    manifest.supportedClients = support.supported.map(host => host.id);
    for (const host of support.supported) {
      const key = host.id.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
      const delivery = manifest.distribution[key] ??= { manualInstallationSupported: true };
      if (host.globalSkillDirectory) delivery.globalSkillDirectory = host.globalSkillDirectory;
      else delete delivery.globalSkillDirectory;
      if (host.workspaceSkillDirectory) delivery.workspaceSkillDirectory = host.workspaceSkillDirectory;
      else delete delivery.workspaceSkillDirectory;
    }
    if (manifest.sourceSha256 !== undefined) manifest.sourceSha256 = createHash("sha256").update(next).digest("hex");
    await output(skill.manifest, JSON.stringify(manifest, null, 2) + "\n");
  }
  const usagePath = "skills/puretokens-update/references/usage-guide.md";
  await output(usagePath, replaceGuidanceBlock(await read(usagePath), "operations", renderOperationGuidance(operations)));
  return changed;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const write = process.argv.includes("--write"), check = process.argv.includes("--check");
  if (write === check) throw new Error("Use exactly one of --write or --check");
  const changed = await syncSkillGuidance(repositoryRoot, { write });
  if (check && changed.length) {
    console.error(`Shared Skill guidance is out of sync; run npm run docs:sync-guidance:\n${changed.join("\n")}`);
    process.exitCode = 1;
  }
}
