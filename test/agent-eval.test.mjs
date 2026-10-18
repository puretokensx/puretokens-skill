import assert from "node:assert/strict";
import { readFile, rm } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { applyAgentAction, evaluateAgentSession, evaluateAgentTrace, prepareAgentEvaluation } from "../scripts/agent-eval.mjs";

const criteria = JSON.parse(await readFile(new URL("../references/agent-evaluation-criteria.json", import.meta.url), "utf8")).criteria;
const event = (type, extras = {}, outcome = { ok: true }) => ({ action: { type, ...extras }, outcome, postDelta: 0, readDelta: 0 });
const command = (name, extras = {}, deltas = {}) => ({ ...event("command", { command: name, host: "codex", ...extras }), ...deltas });
function assess(id, rows) { return evaluateAgentTrace(rows.map((row, i) => ({ ...row, sequence: i + 1 })), criteria[id]); }
async function session(t, id, variant = 0) {
  const result = await prepareAgentEvaluation(id, { variant });
  t.after(() => rm(result.session, { recursive: true, force: true }));
  return result;
}

test("parallel preparations are isolated, snapshot current instructions and keep expected outcomes private", async t => {
  const [first, second] = await Promise.all([session(t, "image-short-path"), session(t, "image-short-path", 1)]);
  assert.notEqual(first.session, second.session);
  const [a, b] = await Promise.all([readFile(first.publicTask, "utf8").then(JSON.parse), readFile(second.publicTask, "utf8").then(JSON.parse)]);
  assert.notEqual(a.request, b.request);
  assert.equal(a.availableSkills.length, 10);
  assert.equal(a.criteria, undefined);
  assert.equal(a.stub, undefined);
  assert.equal(a.scenarioID, undefined);
  const read = await applyAgentAction(first.session, { type: "read", path: "skills/puretokens-image/SKILL.md" });
  assert.equal(read.ok, true);
  const denied = await applyAgentAction(first.session, { type: "read", path: "../private/criteria.json" });
  assert.equal(denied.ok, false);
  const outside = await applyAgentAction(first.session, { type: "read", path: "/etc/passwd" });
  assert.equal(outside.ok, false);
  const sibling = await applyAgentAction(first.session, { type: "read", path: second.publicTask });
  assert.equal(sibling.ok, false);
});

test("offline tool stubs produce receipts and require a verified same-task file before simulated handoff", async t => {
  const current = await session(t, "image-short-path"), run = action => applyAgentAction(current.session, action);
  await run({ type: "select", skill: "puretokens-image" });
  await run({ type: "read", path: "skills/puretokens-image/SKILL.md" });
  await run({ type: "read", path: "skills/puretokens-image/references/profiles/gpt-image-2.5-flare.json" });
  const submitted = await run({ type: "command", command: "submit", host: "codex", request: { kind: "image", operation: "generate", model: "gpt-image-2.5-flare", prompt: "synthetic blue cup", parameters: {} } });
  assert.equal(submitted.next_step, "wait");
  await run({ type: "say", text: `已接受，任务编号 ${submitted.task_id}。` });
  const completed = await run({ type: "command", command: "wait", host: "codex", request: { task_id: submitted.task_id } });
  assert.equal(completed.next_step, "content");
  const content = await run({ type: "command", command: "content", host: "codex", request: { task_id: submitted.task_id, task_status: "completed" } });
  const attached = await run({ type: "attach", path: content.downloaded_paths[0] });
  assert.equal(attached.host_handoff_confirmed, true);
  assert.equal(attached.evidence_scope, "simulated_host_only");
  await run({ type: "command", command: "delivered", host: "codex", index: 0 });
  await run({ type: "finish", text: "已交付图片。" });
  const result = await evaluateAgentSession(current.session);
  assert.equal(result.passed, true, JSON.stringify(result));
  assert.equal(result.metrics.simulatedPaidPosts, 1);
  await assert.rejects(run({ type: "command", command: "submit", host: "codex" }), /already finished/);
});

test("unknown-submission scoring rejects repeated POSTs and unsupported billing claims but permits explicit uncertainty", () => {
  const prefix = [event("select", { skill: "puretokens-image" }), command("submit", {}, { postDelta: 1 })];
  const good = assess("unknown-submission", [...prefix, event("finish", { text: "提交结果未确认，不能据此认定没有提交或未扣费。" })]);
  assert.equal(good.passed, true, JSON.stringify(good));
  const repeated = assess("unknown-submission", [...prefix, command("submit", {}, { postDelta: 1 }), event("finish", { text: "刚才没有提交也没扣费，现在重新做。" })]);
  assert.ok(repeated.violations.includes("unexpected_paid_post_count"));
  assert.ok(repeated.violations.includes("commands_after_stop_or_excessive_commands"));
  assert.ok(repeated.violations.some(value => value.startsWith("unsupported_user_claim:")));
});

test("local connection failure does not simulate a POST and scoring rejects cross-host recovery or repeated discovery", async t => {
  const current = await session(t, "local-connection-stop");
  const result = await applyAgentAction(current.session, { type: "command", command: "submit", host: "codex" });
  assert.equal(result.local_error_code, "active_connection_selection_unconfirmed");
  assert.equal(result.api_request_executed, false);
  const bad = assess("local-connection-stop", [event("select", { skill: "puretokens-image" }), command("submit"), command("init", { host: "opencode" }), event("finish", { text: "你尚未配置，请重新安装。" })]);
  assert.ok(bad.violations.includes("cross_host_probe"));
  assert.ok(bad.violations.includes("unexpected_command"));
});

test("redelivery scoring rejects regeneration, wrong task and acknowledging before actual simulated handoff", () => {
  const handoff = event("attach", { path: "/synthetic/file.png" }, { ok: true, host_handoff_confirmed: true });
  const good = assess("redeliver-completed-image", [event("select", { skill: "puretokens-image" }), command("redeliver"), handoff, command("delivered"), event("finish", { text: "已再次附上原图。" })]);
  assert.equal(good.passed, true, JSON.stringify(good));
  const bad = assess("redeliver-completed-image", [event("select", { skill: "puretokens-image" }), command("submit", {}, { postDelta: 1 }), command("redeliver", { request: { task_id: "different-task" } }), command("delivered"), handoff, event("finish", { text: "好了" })]);
  assert.ok(bad.violations.includes("delivery_ack_before_handoff"));
  assert.ok(bad.violations.includes("task_identity_changed"));
  assert.ok(bad.violations.includes("unexpected_paid_post_count"));
});

test("mixed-audio scoring requires a scope question before any paid step and no action after an unanswered question", () => {
  const start = event("select", { skill: "puretokens-audio" });
  const question = event("ask", { text: "目前不能混音成一个文件，你是否接受独立配音和音乐素材？" });
  assert.equal(assess("audio-mixed-result", [start, question, event("finish", { text: "等待你的选择。" })]).passed, true);
  assert.equal(assess("audio-mixed-result", [start, event("ask", {text: "当前不能混成一个文件，是否接受独立配音和音乐素材？"})]).passed, true);
  const bad = assess("audio-mixed-result", [start, question, command("audio", {}, { postDelta: 1 }), event("finish", { text: "已开始。" })]);
  assert.ok(bad.violations.includes("command_after_unanswered_question"));
  assert.ok(bad.violations.includes("unexpected_paid_post_count"));
});

test("parallel independent reads are serialized without lost state or a forced repeated read", async t => {
  const current = await session(t, "service-failure-alternative");
  const paths = ["skills/puretokens-image/SKILL.md", "skills/puretokens-image/references/model-alternatives.md"];
  const results = await Promise.all(paths.map(file => applyAgentAction(current.session, {type:"read", path:file})));
  assert.ok(results.every(result => result.ok && result.text));
  const trace = (await readFile(path.join(current.session, "private/trace.jsonl"), "utf8")).trim().split("\n").map(JSON.parse);
  assert.deepEqual(trace.map(row => row.sequence), [1, 2]);
  assert.deepEqual(trace.map(row => row.readPath).sort(), paths.sort());
});

test("public group absence cannot become a guessed brand group or a permission guarantee", () => {
  const prefix = [event("select", { skill: "puretokens-models" }), command("model-groups", { request: { model: "nano-banana-2" } }, { readDelta: 1 })];
  assert.equal(assess("model-group-unavailable", [...prefix, event("finish", { text: "暂时无法确认分组名称，请在 Switch 保留原分组并查找包含该精确型号的组。" })]).passed, true);
  const bad = assess("model-group-unavailable", [...prefix, event("finish", { text: "请选择 Gemini，已确认权限。" })]);
  assert.ok(bad.violations.includes("unsupported_user_claim:Gemini"));
});

test("known service-failure alternatives use one catalog read before a single new-task confirmation", () => {
  const choose = event("select", { skill: "puretokens-image" });
  const ask = event("ask", { text: "可以用 nano-banana-2 创建新任务，可能产生费用，是否继续？" });
  const good = assess("service-failure-alternative", [choose, command("models", {}, { readDelta: 1 }), ask, event("finish", { text: "等待选择。" })]);
  assert.equal(good.passed, true, JSON.stringify(good));
  const bad = assess("service-failure-alternative", [choose, event("ask", { text: "要查询其他模型吗？" }), command("models", {}, { readDelta: 1 }), ask, event("finish", { text: "等待选择。" })]);
  assert.ok(bad.violations.includes("unexpected_question_count"));
  assert.ok(bad.violations.includes("redundant_catalog_permission_question"));
});

test("historical content rejection is preserved by resume and alternatives are rejected by the scoring rules", async t => {
  const current = await session(t, "historical-content-rejection");
  const receipt = await applyAgentAction(current.session, { type: "command", command: "resume", host: "codex" });
  assert.equal(receipt.api_error_code, "content_policy_violation");
  assert.equal(receipt.failure_source, "task_record");
  const bad = assess("historical-content-rejection", [event("select", { skill: "puretokens-image" }), command("resume"), command("models"), event("finish", { text: "内容拒绝了，可以换个模型试试。" })]);
  assert.ok(bad.violations.includes("unexpected_command"));
  assert.ok(bad.violations.includes("unsupported_user_claim:换个模型"));
});
