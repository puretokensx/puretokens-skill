import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { repositoryRoot } from "../scripts/skill-registry.mjs";
import { renderHostReferences } from "../scripts/host-reference-guidance.mjs";
import { renderSelectedHostDelivery } from "../scripts/host-delivery-guidance.mjs";

const read = file => readFile(path.join(repositoryRoot, file), "utf8");

test("host lookup is bounded and each client reads only its own execution reference", async () => {
  const support = JSON.parse(await read("references/host-support.json"));
  const registry = JSON.parse(await read("skills/index.json"));
  for (const skill of registry.skills) {
    const root = `${path.dirname(skill.entry)}/references`;
    const index = await read(`${root}/desktop-hosts.md`);
    assert.ok(Buffer.byteLength(index) < 6000, `${skill.name}: host index loads the full client manual`);
    assert.match(index, /hosts\/[^ ]+|hosts\/</);
    for (const host of support.supported) {
      assert.ok(index.includes(`hosts/${host.id}.md`), `${skill.name}: missing ${host.id} lookup`);
      const selected = await read(`${root}/hosts/${host.id}.md`);
      assert.ok(selected.includes(`--host ${host.id}`));
      assert.ok(Buffer.byteLength(selected) < 6500, `${host.id}: selected reference is too broad`);
      assert.doesNotMatch(selected, /## New Switch clients|## 按宿主选择/);
    }
    const codex = await read(`${root}/hosts/codex.md`);
    assert.doesNotMatch(codex, /PURETOKENS_SWITCH_QWEN_API_KEY|workbuddy_record_missing|OMP_PROFILE/);
  }
});

test("media references expose only protocols mapped to the bound host", async () => {
  const source = JSON.parse(await read("references/host-delivery.json"));
  for (const kind of ["image", "video", "audio", "evaluate"]) {
    const root = `skills/puretokens-${kind}/references`;
    const index = await read(`${root}/host-delivery.md`);
    assert.ok(Buffer.byteLength(index) < 6000, "delivery guide eagerly includes every protocol");
    for (const host of source.hosts) {
      const selected = await read(`${root}/delivery/${host.id}.md`);
      for (const protocol of source.protocols) {
        assert.equal(selected.includes(`## ${protocol.title}`), protocol.id !== source.defaultProtocol && host.protocols.includes(protocol.id), `${host.id}: ${protocol.id}`);
      }
      assert.ok(selected.includes("../host-delivery.md"));
    }
    const shared = source.protocols.find(protocol => protocol.id === source.defaultProtocol);
    assert.ok(index.includes(`## ${shared.title}`), "shared protocol must be stored once per Skill");
  }
});

test("unknown and duplicate host sections cannot create reference paths or fallback bindings", async () => {
  const support = JSON.parse(await read("references/host-support.json"));
  const source = await read("references/desktop-hosts.md");
  const binding = (await read("references/skill-fragments/host-binding.md")).trim();
  assert.throws(() => renderHostReferences(source + "\n## ../other\nunsafe\n", support, binding), /Unknown or duplicate/);
  assert.throws(() => renderHostReferences(source + "\n## codex\nduplicate\n", support, binding), /Unknown or duplicate/);
  assert.throws(() => renderSelectedHostDelivery(JSON.parse("{\"hosts\":[]}"), "missing"), /Unknown delivery host/);
});
