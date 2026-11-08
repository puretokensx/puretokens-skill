import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const read = file => readFile(new URL(`../${file}`, import.meta.url), "utf8");

for (const [kind, limit] of [["image", 10500], ["video", 10500], ["audio", 8000]]) {
  test(`${kind} entrypoint stays compact with conditional detail and one next action`, async () => {
    const entry = await read(`skills/puretokens-${kind}/SKILL.md`);
    assert.ok(Buffer.byteLength(entry) <= limit, `${kind}: ${Buffer.byteLength(entry)} > ${limit} UTF-8 bytes`);
    assert.match(entry, /已读且仍在有效上下文中的说明不重读|已读且仍有效的资料不重读/);
    assert.match(entry, /普通请求不读取按需说明/);
    assert.match(entry, /结果优先.*实际附件|结果优先.*转写文本/);
    assert.match(entry, /失败只给一个.*下一步/);
    assert.match(entry, /不要求用户编辑 JSON|不把 JSON.*交给用户/);
    assert.match(entry, /references\/receipt-guide\.md/);
  });
}

test("receipt presentation selects one evidence-based next action without extra requests", async () => {
  const guide = await read("references/skill-fragments/receipt-guide.md");
  assert.match(guide, /结果优先/);
  assert.match(guide, /失败只给一个.*下一步/);
  assert.match(guide, /原执行器.*next_action|next_action.*原执行器/);
  assert.match(guide, /不为补充说明再发请求/);
  assert.match(guide, /不.*用户.*JSON|不把 JSON 编辑交给用户/);
  assert.match(guide, /unknown.*不能|unknown[\s\S]*不自动重发/);
});

test("video recovery preserves explicitly requested reviewed tail frame indexes", async () => {
  const usage = await read("skills/puretokens-video/references/executor-usage.md");
  assert.doesNotMatch(usage, /video has no next index|video tasks only allow one output|retrieve only the undelivered index 0/);
  assert.match(usage, /reviewed Super Vega Pro.*return_last_frame=true/);
  assert.match(usage, /requested_count.*1.*video.*index 1/);
  assert.match(usage, /only.*authorized.*indexes.*same task/s);
  assert.match(usage, /Never infer.*tail.frame|Do not infer.*tail.frame/);
});
