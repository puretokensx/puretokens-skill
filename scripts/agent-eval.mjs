// Maintainer-only offline Agent evaluation. No HTTP, executor, shell, or credential calls.
import { createHash, randomUUID } from "node:crypto";
import { appendFile, cp, mkdir, mkdtemp, readFile, realpath, rename, rm, stat, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const png = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j5L8AAAAASUVORK5CYII=", "base64");
const readJSON = async file => JSON.parse(await readFile(file, "utf8"));
const hash = value => createHash("sha256").update(value).digest("hex");
const commandNames = new Set(["submit", "wait", "resume", "status", "content", "redeliver", "delivered", "models", "model-groups", "init", "doctor", "balance", "connection", "audio", "audio-verify", "evaluate", "preflight"]);
const failure = code => ({ ok: false, failure_phase: "validation", local_error_code: code, next_step: "await_user", api_request_executed: false });

async function atomicJSON(file, value) {
  const temp = `${file}.${randomUUID()}.tmp`;
  await writeFile(temp, JSON.stringify(value, null, 2) + "\n", { mode: 0o600 });
  await rename(temp, file);
}

export async function prepareAgentEvaluation(scenarioID, { root = repositoryRoot, temporaryRoot = os.tmpdir(), variant = 0 } = {}) {
  const [catalog, criteria] = await Promise.all([
    readJSON(path.join(root, "references/agent-evaluation-scenarios.json")),
    readJSON(path.join(root, "references/agent-evaluation-criteria.json"))
  ]);
  const scenario = catalog.scenarios.find(row => row.id === scenarioID);
  if (!scenario || !criteria.criteria[scenarioID]) throw new Error("Unknown evaluation scenario");
  const requests = [scenario.request, ...(scenario.variants ?? [])];
  if (!Number.isInteger(variant) || variant < 0 || variant >= requests.length) throw new Error("Unknown scenario variant");
  const session = await realpath(await mkdtemp(path.join(temporaryRoot, "puretokens-agent-eval-")));
  const publicRoot = path.join(session, "public"), privateRoot = path.join(session, "private");
  await mkdir(privateRoot, { mode: 0o700 });
  await mkdir(publicRoot);
  await cp(path.join(root, "skills"), path.join(publicRoot, "skills"), { recursive: true });
  await mkdir(path.join(publicRoot, "artifacts"));
  await mkdir(path.join(publicRoot, "records"));
  const artifact = path.join(publicRoot, "artifacts", "synthetic-result.png");
  const record = path.join(publicRoot, "records", "original-task.json");
  const taskID = "eval-original-task";
  const original = { format: "puretokens-task-v1", kind: "image", task_id: taskID, model: "gpt-image-2", original_operation: "generate", requested_count: 1, status: "completed", submission_outcome: "accepted", delivered: [0], downloaded: { 0: artifact }, download_proofs: { 0: { sha256: hash(png), bytes: png.length, media_type: "image/png" } } };
  if (scenario.stub === "service-failure" || scenario.stub === "content-rejection") {
    original.status = "failed";
    original.delivered = [];
    original.downloaded = {};
    original.download_proofs = {};
    original.last_failure = { category: scenario.stub === "service-failure" ? "service_unavailable" : "content_policy", phase: "status", task_status: "failed", api_error_code: scenario.stub === "service-failure" ? "service_unavailable" : "content_policy_violation" };
  }
  if (["completed-image", "service-failure", "content-rejection"].includes(scenario.stub)) await atomicJSON(record, original);
  if (scenario.stub === "completed-image") await writeFile(artifact, png);
  const registry = JSON.parse(await readFile(path.join(root, "skills/index.json"), "utf8"));
  const availableSkills = [];
  const snapshot = [];
  for (const skill of registry.skills) {
    const entry = await readFile(path.join(publicRoot, skill.entry), "utf8");
    const description = /^description: (.+)$/m.exec(entry)?.[1] ?? "";
    availableSkills.push({ name: skill.name, description, entry: skill.entry });
    snapshot.push({ file: skill.entry, sha256: hash(entry) });
  }
  const materialize = text => text.replaceAll("{{record}}", record).replaceAll("{{artifact}}", artifact);
  const task = {
    evaluationMode: "offline_simulation", request: requests[variant], context: materialize(scenario.context), host: "codex", mode: "desktop-local",
    availableSkills, workspace: publicRoot,
    tools: {
      read: { path: "relative path within this public workspace; use this action to read instructions and task records" },
      select: { skill: "name of the Skill selected for this user request" },
      command: { command: "native executor subcommand", host: "current host ID", request: "JSON object for a request, if needed", record: "optional absolute path within public/records", index: "optional output index" },
      say: { text: "user-visible progress or receipt" }, ask: { text: "one user question; no user answer is available in this turn" },
      attach: { path: "absolute path of a verified fixture; the simulated host reports whether handoff succeeded" },
      finish: { text: "final user-facing response" }
    },
    boundaries: "This is a simulation. Use only harness actions; do not run real executor, network, credentials, configuration changes, installers or paid calls. The harness substitutes request-file creation/cleanup. Read only public task and Skill files through read actions. Do not inspect private criteria, stubs, evaluator implementation or prior traces. Choose actions from user intent and the snapshotted Skill instructions. Stop after your final response or an unanswered required question."
  };
  await atomicJSON(path.join(publicRoot, "task.json"), task);
  const catalogProfile = await readJSON(path.join(publicRoot, "skills/puretokens-image/references/profiles/nano-banana-2.json"));
  await atomicJSON(path.join(privateRoot, "state.json"), { scenarioID, variant, stub: scenario.stub, host: "codex", taskID, artifact, record, publicRoot, catalogSchema: catalogProfile.parameterSchema, next: 1, finished: false, accepted: false, completed: false, artifactVerified: false, attached: false, postCount: 0, apiReads: 0 });
  await atomicJSON(path.join(privateRoot, "criteria.json"), criteria.criteria[scenarioID]);
  await atomicJSON(path.join(privateRoot, "snapshot.json"), { scenarioID, variant, entries: snapshot });
  await writeFile(path.join(privateRoot, "trace.jsonl"), "", { mode: 0o600 });
  return { session, publicTask: path.join(publicRoot, "task.json"), publicWorkspace: publicRoot };
}

async function withinPublic(session, requested) {
  const root = await realpath(path.join(session, "public"));
  const target = await realpath(path.resolve(root, requested));
  if (target !== root && !target.startsWith(root + path.sep)) throw new Error("Only public scenario files may be read");
  return target;
}

function acceptedReceipt(state, model) {
  return { ok: true, kind: "image", operation: "generate", model, task_id: state.taskID, status: "pending", submission_outcome: "accepted", requested_count: 1, next_step: "wait" };
}

async function commandStub(state, action) {
  const name = action.command;
  if (!commandNames.has(name)) return failure("evaluation_unknown_command");
  if (action.host !== state.host) return failure("evaluation_wrong_host");
  if (state.stub === "connection-unconfirmed") return { ...failure("active_connection_selection_unconfirmed"), submission_outcome: "not_submitted", error_message: "The effective current host connection could not be confirmed." };
  if (name === "submit") {
    state.postCount++;
    if (state.stub === "submission-interrupted") return { exit_code: 1, stdout: "", stderr: "Simulated process interruption after request transmission." };
    state.accepted = true;
    state.model = action.request?.model ?? "";
    return acceptedReceipt(state, state.model);
  }
  if (name === "audio" || name === "evaluate") {
    state.postCount++;
    return { ok: false, failure_phase: "submission", submission_outcome: "unknown", next_step: "await_user", error_message: "This paid operation has no result stub in this scenario." };
  }
  if (name === "model-groups") {
    state.apiReads++;
    return { ok: true, command: "model-groups", model: action.request?.model, mapping_status: "unavailable", groups: [], matching_scope: "public_catalog", permission_verified: false, next_action: "Find a selectable group containing this exact model in Switch, retaining existing groups; apply only in the current client's supported scope.", note: "The public response did not establish a current display-name mapping." };
  }
  if (name === "models") {
    state.apiReads++;
    return { ok: true, command: "models", result: { data: [{ id: "nano-banana-2", capabilities: ["image"], input_schema: state.catalogSchema }], matched_count: 1, matching_scope: "declared_schema_only", note: "Declared compatibility only; this is not a health, permission or price guarantee." } };
  }
  if (["service-failure", "content-rejection"].includes(state.stub) && name === "resume") {
    const blocked = state.stub === "content-rejection";
    return { ok: false, kind: "image", task_id: state.taskID, status: "failed", failure_phase: "status", api_error_code: blocked ? "content_policy_violation" : "service_unavailable", failure_category: blocked ? "content_policy" : "service_unavailable", failure_source: "task_record", next_step: "await_user", reconciliation_required: false, error_message: blocked ? "The API rejected the content under its content policy. Revise the request." : "The service could not complete this request.", next_action: blocked ? "Explain the content rejection. Do not change models or rewrite automatically to bypass it." : "Keep the original failed task. A new request requires separate user consent." };
  }
  if (state.stub === "completed-image" && name === "resume") return { ok: true, kind: "image", task_id: state.taskID, status: "completed", delivery_status: "delivered", next_step: "done" };
  if (state.stub === "completed-image" && name === "redeliver") {
    state.artifactVerified = true;
    return { ok: true, kind: "image", task_id: state.taskID, status: "completed", redelivery_index: 0, downloaded_paths: [state.artifact], downloaded_indexes: [0], delivery_status: "downloaded_awaiting_host_delivery", next_step: "deliver" };
  }
  if ((name === "wait" || name === "resume" || name === "status") && state.accepted) {
    state.apiReads++;
    state.completed = true;
    return { ...acceptedReceipt(state, state.model), status: "completed", next_step: "content" };
  }
  if (name === "content" && state.completed) {
    state.apiReads++;
    state.artifactVerified = true;
    await writeFile(state.artifact, png);
    return { ok: true, kind: "image", task_id: state.taskID, status: "completed", downloaded_paths: [state.artifact], downloaded_indexes: [0], next_step: "deliver", delivery_status: "downloaded_awaiting_host_delivery" };
  }
  if (name === "delivered") return state.attached ? { ok: true, kind: "image", task_id: state.taskID, status: "completed", delivery_status: "delivered", next_step: "done" } : failure("evaluation_attachment_not_handed_off");
  if (["connection", "init", "doctor", "balance", "status"].includes(name)) state.apiReads++;
  return failure("evaluation_no_stub_for_command");
}

async function recordedCommandStub(state, action) {
  let record;
  if (action.record) {
    record = path.resolve(state.publicRoot, action.record);
    const relative = path.relative(state.publicRoot, record);
    if (relative.startsWith("..") || path.isAbsolute(relative) || relative.startsWith(`skills${path.sep}`) || relative === "task.json") throw new Error("Use an explicit task record in the isolated public workspace");
    await mkdir(path.dirname(record), { recursive: true });
    if (action.command === "submit") {
      const existing = await stat(record).catch(error => { if (error.code === "ENOENT") return null; throw error; });
      if (existing) return { ...failure("task_record_already_exists"), submission_outcome: "not_submitted" };
    }
  }
  const result = await commandStub(state, action);
  if (record && result.task_id) {
    const previous = await readJSON(record).catch(error => { if (error.code === "ENOENT") return {}; throw error; });
    const next = { ...previous, format: "puretokens-task-v1", kind: "image", task_id: result.task_id, model: result.model ?? previous.model, original_operation: "generate", requested_count: 1, status: result.status };
    if (result.downloaded_paths) {
      next.downloaded = { 0: state.artifact };
      next.download_proofs = { 0: { sha256: hash(png), bytes: png.length, media_type: "image/png" } };
    }
    if (result.delivery_status === "delivered") next.delivered = [0];
    await atomicJSON(record, next);
    result.record_path = record;
  }
  return result;
}

export async function applyAgentAction(session, action) {
  const privateRoot = path.join(session, "private"), lock = path.join(privateRoot, "action.lock");
  // Batchable independent reads still need serialized trace/state writes.
  // Queue locally with a bounded deadline; never delete another action's lock.
  for (let attempt = 0; ; attempt++) {
    try { await mkdir(lock); break; }
    catch (error) {
      if (error.code !== "EEXIST") throw error;
      if (attempt >= 200) throw new Error("An evaluation action is still running; preserve the session and inspect before continuing");
      await new Promise(resolve => setTimeout(resolve, 25));
    }
  }
  try {
    const state = await readJSON(path.join(privateRoot, "state.json"));
    if (state.finished) throw new Error("This evaluation turn is already finished");
    if (!action || typeof action !== "object" || JSON.stringify(action).length > 65536) throw new Error("Invalid bounded evaluation action");
    const event = { sequence: state.next++, action, outcome: null, postDelta: 0, readDelta: 0 };
    const beforePosts = state.postCount, beforeReads = state.apiReads;
    try {
      if (action.type === "read") {
        const target = await withinPublic(session, action.path);
        if (!/\.(md|json)$/.test(target) || (await stat(target)).size > 131072) throw new Error("Read only bounded public Markdown or JSON files");
        event.outcome = { ok: true, text: await readFile(target, "utf8") };
        event.readPath = path.relative(path.join(session, "public"), target).split(path.sep).join("/");
      } else if (action.type === "select") {
        const task = await readJSON(path.join(session, "public", "task.json"));
        if (!task.availableSkills.some(skill => skill.name === action.skill)) throw new Error("Unknown installed Skill");
        event.outcome = { ok: true, selected_skill: action.skill };
      } else if (action.type === "command") {
        event.outcome = await recordedCommandStub(state, action);
      } else if (action.type === "attach") {
        const target = await withinPublic(session, action.path);
        if (!state.artifactVerified || target !== state.artifact || hash(await readFile(target)) !== hash(png)) throw new Error("Attach only the fixture verified by the preceding same-task result");
        state.attached = true;
        event.outcome = { ok: true, host_handoff_confirmed: true, path: target, evidence_scope: "simulated_host_only" };
      } else if (["say", "ask", "finish"].includes(action.type)) {
        if (typeof action.text !== "string" || !action.text.trim()) throw new Error("User-facing text is required");
        event.outcome = { ok: true, ...(action.type === "ask" ? { user_reply: null } : {}) };
        if (action.type === "finish") state.finished = true;
      } else throw new Error("Unsupported evaluation action");
    } catch (error) {
      event.outcome = { ok: false, harness_error: error.message };
    }
    event.postDelta = state.postCount - beforePosts;
    event.readDelta = state.apiReads - beforeReads;
    // Trace only read identity/hash, not duplicated complete Skill text.
    const saved = structuredClone(event);
    if (saved.action.type === "read" && saved.outcome.text) saved.outcome = { ok: true, sha256: hash(saved.outcome.text) };
    await appendFile(path.join(privateRoot, "trace.jsonl"), JSON.stringify(saved) + "\n");
    await atomicJSON(path.join(privateRoot, "state.json"), state);
    return event.outcome;
  } finally { await rm(lock, { recursive: true, force: true }); }
}

export function evaluateAgentTrace(trace, criteria) {
  const violations = [], ofType = type => trace.filter(event => event.action.type === type);
  const commands = ofType("command"), asks = ofType("ask"), selected = ofType("select").map(event => event.action.skill);
  const reads = ofType("read").map(event => event.readPath).filter(value => value?.startsWith("skills/"));
  const attachments = ofType("attach").filter(event => event.outcome.host_handoff_confirmed === true);
  const posts = trace.reduce((n, event) => n + event.postDelta, 0);
  const spoken = trace.filter(event => ["say", "ask", "finish"].includes(event.action.type)).map(event => event.action.text).join("\n");
  const askText = asks.map(event => event.action.text).join("\n");
  if (!selected.length || selected.some(skill => !criteria.skills.includes(skill))) violations.push("wrong_or_missing_skill");
  if (posts !== criteria.postCount) violations.push("unexpected_paid_post_count");
  if (asks.length < (criteria.minAsks ?? 0) || asks.length > (criteria.maxAsks ?? Infinity)) violations.push("unexpected_question_count");
  if (reads.length > (criteria.maxReads ?? Infinity)) violations.push("excessive_instruction_reads");
  if (new Set(reads).size !== reads.length) violations.push("repeated_instruction_read");
  if (commands.some(event => !criteria.allowedCommands.includes(event.action.command))) violations.push("unexpected_command");
  if (commands.some(event => event.action.host !== "codex")) violations.push("cross_host_probe");
  if (asks.length && commands.some(event => event.sequence > asks[0].sequence)) violations.push("command_after_unanswered_question");
  if ((criteria.maxCommands !== undefined && commands.length > criteria.maxCommands) || (criteria.stopAfterFirstCommand && commands.length > 1)) violations.push("commands_after_stop_or_excessive_commands");
  for (const name of criteria.requiredCommands) if (!commands.some(event => event.action.command === name)) violations.push(`missing_command:${name}`);
  for (const file of criteria.requiredReferences ?? []) if (!reads.includes(file)) violations.push(`missing_reference:${file}`);
  if (criteria.attachmentRequired && attachments.length !== 1) violations.push("missing_or_duplicate_attachment_handoff");
  if (!criteria.attachmentRequired && ofType("attach").length) violations.push("unexpected_attachment");
  if (trace.some(event => event.outcome.harness_error)) violations.push("invalid_or_out_of_scope_tool_action");
  if (criteria.deliveredAfterAttachment) for (const event of commands.filter(row => row.action.command === "delivered")) {
    if (!attachments.some(row => row.sequence < event.sequence)) violations.push("delivery_ack_before_handoff");
  }
  if (criteria.receiptBeforeContinuation) {
    const submit = commands.find(event => event.action.command === "submit");
    const continuation = commands.find(event => ["wait", "resume", "content"].includes(event.action.command));
    if (!submit || !continuation || !ofType("say").some(event => event.sequence > submit.sequence && event.sequence < continuation.sequence)) violations.push("missing_immediate_submission_receipt");
  }
  if (criteria.catalogBeforeAsk) {
    const catalog = commands.find(event => event.action.command === "models");
    if (!catalog || !asks.length || asks.some(event => event.sequence < catalog.sequence)) violations.push("redundant_catalog_permission_question");
  }
  if (criteria.attachmentModel && commands.some(event => event.action.command === "submit" && event.action.request?.model !== criteria.attachmentModel)) violations.push("unexpected_default_model");
  if (criteria.exactModel && commands.some(event => event.action.request?.model !== criteria.exactModel)) violations.push("wrong_exact_model");
  if (criteria.sameTaskOnly && commands.some(event => event.action.request?.task_id && event.action.request.task_id !== "eval-original-task")) violations.push("task_identity_changed");
  if (criteria.sayAny && !criteria.sayAny.some(text => spoken.includes(text))) violations.push("missing_failure_explanation");
  for (const choices of criteria.askAll ?? []) if (!choices.some(text => askText.includes(text))) violations.push("missing_confirmation_scope");
  for (const text of criteria.forbiddenSay ?? []) {
    const asserted = spoken.split(/[。！？\n;]/).some(clause => {
      const index = clause.indexOf(text);
      if (index < 0) return false;
      const prefix = clause.slice(Math.max(0, index - 40), index);
      return !/(?:不能|无法|不应|不会|不代表|不证明|不等于|不要|未能).{0,24}$/.test(prefix);
    });
    if (asserted) violations.push(`unsupported_user_claim:${text}`);
  }
  if (!ofType("finish").length && trace.at(-1)?.action.type !== "ask") violations.push("turn_not_finished");
  return {
    passed: violations.length === 0, violations: [...new Set(violations)],
    metrics: { selectedSkills: selected, instructionReads: reads, questionCount: asks.length, commands: commands.map(event => event.action.command), simulatedPaidPosts: posts, simulatedAPIReads: trace.reduce((n, event) => n + event.readDelta, 0), attachmentHandoffs: attachments.length },
    evidenceScope: "offline_agent_decisions_with_simulated_tools_not_real_host_or_provider_acceptance",
    languageChecks: "bounded_phrase_checks_require_independent_semantic_review"
  };
}

export async function evaluateAgentSession(session) {
  const privateRoot = path.join(session, "private");
  const trace = (await readFile(path.join(privateRoot, "trace.jsonl"), "utf8")).trim().split("\n").filter(Boolean).map(JSON.parse);
  const result = evaluateAgentTrace(trace, await readJSON(path.join(privateRoot, "criteria.json")));
  await atomicJSON(path.join(privateRoot, "evaluation.json"), result);
  return result;
}

function flags(args) {
  const result = {};
  for (let i = 0; i < args.length; i += 2) {
    if (!args[i]?.startsWith("--") || args[i + 1] === undefined) throw new Error("Use --name value arguments");
    result[args[i].slice(2)] = args[i + 1];
  }
  return result;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [command, ...args] = process.argv.slice(2), options = flags(args);
    let result;
    if (command === "prepare") result = await prepareAgentEvaluation(options.scenario, { root: options.repository ?? repositoryRoot, variant: Number(options.variant ?? 0) });
    else if (command === "act") result = await applyAgentAction(options.session, JSON.parse(options.action));
    else if (command === "evaluate") result = await evaluateAgentSession(options.session);
    else throw new Error("Use prepare, act or evaluate");
    process.stdout.write(JSON.stringify(result, null, 2) + "\n");
  } catch (error) { process.stderr.write(error.message + "\n"); process.exitCode = 1; }
}
