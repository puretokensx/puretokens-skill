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

function imageListTrace() {
  return [event("select", { skill: "puretokens-image" }),
    ...criteria["image-distinct-assets"].requiredReferences.map(readPath => ({ ...event("read"), readPath })),
    ...["红色茶壶", "蓝色陶瓷杯", "绿色玻璃瓶"].flatMap((subject, i) => {
      const id = `image-${i}`, record = `/fixture/records/${i}.json`, file = `/fixture/artifacts/${i}.png`;
      return [
        { ...command("submit", { record, request: { model: "gpt-image-2.5-flare", prompt: subject, parameters: { n: 1 } } }, { postDelta: 1 }), outcome: { ok: true, task_id: id, submission_outcome: "accepted", next_step: "wait" } },
        event("say", { text: `已提交 ${id}` }),
        { ...command("wait", { record }), outcome: { ok: true, task_id: id, next_step: "content" } },
        { ...command("content", { record }), outcome: { ok: true, task_id: id, downloaded_paths: [file], next_step: "deliver" } },
        event("attach", { path: file }, { ok: true, path: file, host_handoff_confirmed: true }),
        { ...command("delivered", { record }), outcome: { ok: true, task_id: id, delivery_status: "delivered", next_step: "done" } }
      ];
    }), event("finish", { text: "三份独立图片已逐项交付。" })];
}

test("distinct image scoring accepts three separate prompts, task records and actual simulated handoffs", () => {
  const result = assess("image-distinct-assets", imageListTrace());
  assert.equal(result.passed, true, JSON.stringify(result));
  assert.equal(result.metrics.attachmentHandoffs, 3);
});

test("distinct image scoring rejects collage prompts, n-based substitution and reused task records", () => {
  for (const defect of ["combined-prompt", "n", "record", "task-id", "duplicate-attachment", "missing-receipt", "early-submit"]) {
    const rows = imageListTrace(), submissions = rows.filter(row => row.action.command === "submit");
    if (defect === "combined-prompt") submissions[0].action.request.prompt += " 蓝色陶瓷杯 绿色玻璃瓶";
    if (defect === "n") submissions[0].action.request.parameters.n = 3;
    if (defect === "record") submissions[1].action.record = submissions[0].action.record;
    if (defect === "task-id") submissions[1].outcome.task_id = submissions[0].outcome.task_id;
    if (defect === "duplicate-attachment") rows.filter(row => row.action.type === "attach")[1].action.path = "/fixture/artifacts/0.png";
    if (defect === "missing-receipt") rows.splice(rows.findIndex(row => row.action.type === "say"), 1);
    if (defect === "early-submit") { const i = rows.indexOf(submissions[1]); rows.splice(i, 1); rows.splice(rows.indexOf(submissions[0]) + 1, 0, submissions[1]); }
    const result = assess("image-distinct-assets", rows);
    assert.equal(result.passed, false, defect);
    assert.ok(result.violations.some(v => v.startsWith("independent_asset:")), `${defect}: ${JSON.stringify(result)}`);
  }
});

test("existing icons never authorize a paid generation or require a generation Skill", () => {
  assert.equal(assess("image-existing-icons", [event("finish", { text: "需要三份独立的现有图标；本会话缺少资源检索工具，无法获取并附加。" })]).passed, true);
  const bad = assess("image-existing-icons", [event("select", { skill: "puretokens-image" }), command("submit", {}, { postDelta: 1 }), event("finish", { text: "已生成官方图标，全部完成。" })]);
  assert.ok(bad.violations.includes("unexpected_paid_post_count"));
});

test("one composed image cannot drop a subject or become three separate paid tasks", () => {
  const rows = imageListTrace().slice(0, 10);
  rows.find(row => row.action.command === "submit").action.request.prompt = "红色茶壶 蓝色陶瓷杯 绿色玻璃瓶";
  rows.push(event("finish", { text: "已交付一张海报。" }));
  assert.equal(assess("image-combined-design", rows).passed, true);
  rows.find(row => row.action.command === "submit").action.request.prompt = "红色茶壶";
  assert.ok(assess("image-combined-design", rows).violations.includes("combined_design:missing_subject"));
});

async function deliverListItem(run, record, prompt) {
  const submitted = await run({ type: "command", command: "submit", host: "codex", record, request: { kind: "image", operation: "generate", model: "gpt-image-2.5-flare", prompt, parameters: { n: 1 } } });
  assert.equal(submitted.next_step, "wait");
  await run({ type: "say", text: `已提交任务 ${submitted.task_id}。` });
  await run({ type: "command", command: "resume", host: "codex", record });
  const content = await run({ type: "command", command: "content", host: "codex", record });
  const handoff = await run({ type: "attach", path: content.downloaded_paths[0] });
  assert.equal(handoff.host_handoff_confirmed, true);
  await run({ type: "command", command: "delivered", host: "codex", record });
  return submitted.task_id;
}

test("offline list execution preserves three identities, records and separate local artifacts", async t => {
  const current = await session(t, "image-distinct-assets"), run = action => applyAgentAction(current.session, action);
  await run({ type: "select", skill: "puretokens-image" });
  for (const file of criteria["image-distinct-assets"].requiredReferences) await run({ type: "read", path: file });
  const ids = [], artifacts = [];
  for (const [i, subject] of criteria["image-distinct-assets"].independentAssets.entries()) {
    const record = path.join(current.publicWorkspace, "records", `${i}.json`);
    ids.push(await deliverListItem(run, record, subject));
    const saved = JSON.parse(await readFile(record, "utf8"));
    assert.deepEqual(saved.delivered, [0]);
    assert.equal(saved.prompt, undefined);
    assert.equal(saved.task_id, ids[i]);
    artifacts.push(saved.downloaded[0]);
  }
  assert.equal(new Set(ids).size, 3);
  assert.equal(new Set(artifacts).size, 3);
  await run({ type: "finish", text: "三份独立图片已逐项交付。" });
  const result = await evaluateAgentSession(current.session);
  assert.equal(result.passed, true, JSON.stringify(result));
  assert.equal(result.metrics.simulatedPaidPosts, 3);
});

test("interruption of item two preserves item one's handoff and stops before item three", async t => {
  const current = await session(t, "image-list-interrupted"), run = action => applyAgentAction(current.session, action);
  await run({ type: "select", skill: "puretokens-image" });
  const first = path.join(current.publicWorkspace, "records", "first.json");
  await deliverListItem(run, first, "红色茶壶");
  const before = await readFile(first, "utf8");
  const result = await run({ type: "command", command: "submit", host: "codex", record: path.join(current.publicWorkspace, "records", "second.json"), request: { model: "gpt-image-2.5-flare", prompt: "蓝色陶瓷杯", parameters: { n: 1 } } });
  assert.equal(result.submission_outcome, "unknown");
  await run({ type: "finish", text: "第一份已交付，第二份提交结果未知，第三份尚未尝试；不自动重提。" });
  assert.equal(await readFile(first, "utf8"), before);
  const assessed = await evaluateAgentSession(current.session);
  assert.equal(assessed.passed, true, JSON.stringify(assessed));
  const trace = (await readFile(path.join(current.session, "private", "trace.jsonl"), "utf8")).trim().split("\n").map(JSON.parse);
  const bad = [...trace.slice(0, -1), { ...command("submit", { request: { model: "gpt-image-2.5-flare", prompt: "绿色玻璃瓶" } }, { postDelta: 1 }), sequence: trace.at(-1).sequence }, { ...trace.at(-1), sequence: trace.at(-1).sequence + 1 }];
  assert.ok(evaluateAgentTrace(bad, criteria["image-list-interrupted"]).violations.includes("independent_asset:command_after_stop"));
});

test("independent image lists stop at exhausted pending windows or failed delivery", () => {
  for (const reason of ["pending", "delivery"]) {
    const rows = imageListTrace(), submissions = rows.filter(row => row.action.command === "submit");
    const second = submissions[1], record = second.action.record, task_id = second.outcome.task_id;
    const stop = reason === "pending"
      ? { ...command("resume", { record }), outcome: { ok: true, task_id, wait_outcome: "window_ended", wait_windows_completed: 2, next_step: "wait" } }
      : { ...command("content", { record }), outcome: { ok: false, task_id, next_step: "await_user", failure_phase: "content" } };
    const stopped = [...rows.slice(0, rows.indexOf(second) + 2), stop, event("finish", { text: "第一份已交付，第二份等待耗尽或交付失败，第三份尚未尝试。" })];
    const rule = { ...criteria["image-distinct-assets"], postCount: 2, attachmentCount: 1 };
    const score = trace => evaluateAgentTrace(trace.map((row, i) => ({ ...row, sequence: i + 1 })), rule);
    assert.equal(score(stopped).passed, true, reason);
    const bad = [...stopped.slice(0, -1), submissions[2], stopped.at(-1)];
    assert.ok(score(bad).violations.includes("independent_asset:command_after_stop"), reason);
  }
});

test("same-brief variations use a profile-supported count and three sequential output handoffs", async () => {
  const model = "nano-banana-2", record = "/fixture/records/variants.json", task_id = "same-brief-task";
  const profile = JSON.parse(await readFile(new URL(`../skills/puretokens-image/references/profiles/${model}.json`, import.meta.url), "utf8"));
  const n = profile.parameterSchema.properties.n;
  assert.ok(n.enum ? n.enum.includes(3) : 3 >= (n.minimum ?? n.min ?? 1) && 3 <= (n.maximum ?? n.max ?? Infinity));
  const rows = [event("select", { skill: "puretokens-image" }),
    { ...command("submit", { record, request: { model, prompt: "蓝色陶瓷杯", parameters: { n: 3 } } }, { postDelta: 1 }), outcome: { ok: true, task_id, submission_outcome: "accepted" } },
    event("say", { text: `已提交 ${task_id}` }), command("wait", { record }),
    ...[0, 1, 2].flatMap(index => {
      const file = `/fixture/artifacts/variant-${index}.png`;
      return [{ ...command("content", { record, index }), outcome: { ok: true, task_id, downloaded_paths: [file] } },
        event("attach", { path: file }, { ok: true, host_handoff_confirmed: true }), command("delivered", { record, index })];
    }), event("finish", { text: "同一主题的三份候选已交付。" })];
  const rule = { ...criteria["image-short-path"], attachmentModel: model, attachmentCount: 3, requiredReferences: [] };
  const result = evaluateAgentTrace(rows.map((row, i) => ({ ...row, sequence: i + 1 })), rule);
  assert.equal(result.passed, true, JSON.stringify(result));
});
