import assert from "node:assert/strict";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { loadHostDelivery, renderHostDeliveryGuide } from "../scripts/host-delivery-guidance.mjs";
import { repositoryRoot } from "../scripts/skill-registry.mjs";

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "pt-host-delivery-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  await mkdir(path.join(root, "references"));
  for (const file of ["host-delivery.json", "host-support.json"]) {
    await cp(path.join(repositoryRoot, "references", file), path.join(root, "references", file));
  }
  return root;
}

test("every registered host retains a conditional current-session media delivery protocol", async () => {
  const source = await loadHostDelivery(repositoryRoot);
  const support = JSON.parse(await readFile(path.join(repositoryRoot, "references/host-support.json"), "utf8"));
  assert.deepEqual(source.hosts.map(row => row.id).sort(), support.supported.map(row => row.id).sort());
  const fallback = source.protocols.find(row => row.id === source.defaultProtocol);
  assert.deepEqual([...fallback.media].sort(), ["audio", "image", "video"]);
  assert.deepEqual([...fallback.modes].sort(), ["cli-local", "desktop-local", "ide-local", "remote"]);
  assert.equal(fallback.evidenceKind, "current_session_contract");
  for (const host of source.hosts) assert.ok(host.protocols.includes(source.defaultProtocol));
});

test("delivery metadata cannot create a generation preflight, weaken file delivery or claim real-host acceptance", async t => {
  const root = await fixture(t);
  const file = path.join(root, "references/host-delivery.json");
  const original = JSON.parse(await readFile(file, "utf8"));
  for (const [key, value] of [
    ["ordinaryGenerationPreflight", true], ["unknownMeansUnsupported", true],
    ["nativeFileRequired", false], ["scope", "real_host_verified"],
    ["defaultProtocol", "codex-desktop-local-media"]
  ]) {
    await writeFile(file, JSON.stringify({ ...original, [key]: value }));
    await assert.rejects(loadHostDelivery(root), /preserve native files/, key);
  }
});

test("unknown, duplicate and incomplete host delivery mappings fail instead of silently inheriting another host", async t => {
  const root = await fixture(t);
  const file = path.join(root, "references/host-delivery.json");
  const original = JSON.parse(await readFile(file, "utf8"));
  const changes = [
    value => value.hosts.push(structuredClone(value.hosts[0])),
    value => value.hosts.pop(),
    value => { value.hosts[0].id = "invented-client"; },
    value => { value.hosts[0].protocols = ["missing-tool"]; },
    value => value.protocols.push(structuredClone(value.protocols[0])),
    value => { value.protocols[0].modes = ["desktop-local"]; }
  ];
  for (const change of changes) {
    const altered = structuredClone(original);
    change(altered);
    await writeFile(file, JSON.stringify(altered));
    await assert.rejects(loadHostDelivery(root), /Invalid|must match|every mode/);
  }
});

test("reviewed desktop and IDE examples do not extend into CLI, remote or undocumented audio delivery", async () => {
  const source = await loadHostDelivery(repositoryRoot);
  const protocol = id => source.protocols.find(row => row.id === id);
  const codex = protocol("codex-desktop-local-media");
  assert.deepEqual(codex.modes, ["desktop-local"]);
  assert.equal(codex.evidenceKind, "current_session_contract");
  assert.deepEqual(codex.sources, []);
  const vscode = protocol("vscode-artifacts");
  assert.deepEqual(vscode.modes, ["ide-local"]);
  assert.deepEqual(vscode.media, ["image", "video"]);
  const claude = protocol("claude-desktop-preview-entry");
  assert.deepEqual(claude.modes, ["desktop-local"]);
  assert.deepEqual(claude.media, ["image", "video"]);
  for (const [host, id] of [["codex", codex.id], ["vscode", vscode.id], ["claude-desktop", claude.id]]) {
    assert.deepEqual(source.hosts.filter(row => row.protocols.includes(id)).map(row => row.id), [host]);
  }
});

test("documentation provenance cannot be substituted for a current-session contract or carry credentialed URLs", async t => {
  const root = await fixture(t);
  const file = path.join(root, "references/host-delivery.json");
  const original = JSON.parse(await readFile(file, "utf8"));
  const changes = [
    value => { value.protocols[0].evidenceKind = "official_documentation"; },
    value => { value.protocols[2].sources = []; },
    value => { value.protocols[2].sources = ["http://code.visualstudio.com/docs/agents/run/artifacts"]; },
    value => { value.protocols[2].sources = ["https://user:private@code.visualstudio.com/docs/agents/run/artifacts"]; },
    value => { value.protocols[2].evidenceKind = "real_host_passed"; }
  ];
  for (const change of changes) {
    const altered = structuredClone(original);
    change(altered);
    await writeFile(file, JSON.stringify(altered));
    await assert.rejects(loadHostDelivery(root), /Invalid|sources|evidence/);
  }
});

test("all installed media and evaluation copies contain the same locally usable delivery guide", async () => {
  const rendered = renderHostDeliveryGuide(await loadHostDelivery(repositoryRoot));
  const files = ["references/skill-fragments/host-delivery.md", ...["image", "video", "audio", "evaluate"].map(kind => `skills/puretokens-${kind}/references/host-delivery.md`)];
  for (const file of files) assert.equal(await readFile(path.join(repositoryRoot, file), "utf8"), rendered, file);
});
