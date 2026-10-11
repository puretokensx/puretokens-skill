import assert from "node:assert/strict";
import { execFile as execFileCallback } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test, { after, before } from "node:test";
import { promisify } from "node:util";

import { repositoryRoot } from "../scripts/skill-registry.mjs";
import { compileSchema } from "./support/schema-validator.mjs";

const schemaNames = [
  "executor-request.schema.json", "executor-receipt.schema.json", "task-record.schema.json",
  "model-query.schema.json", "model-query-receipt.schema.json", "balance-snapshot.schema.json",
  "balance-receipt.schema.json", "init-receipt.schema.json", "doctor-receipt.schema.json",
  "support-summary.schema.json", "evaluation-request.schema.json", "evaluation-receipt.schema.json",
  "audio-request.schema.json", "audio-receipt.schema.json", "audio-artifact-verify-request.schema.json",
  "model-groups-request.schema.json", "model-groups-receipt.schema.json", "timings.schema.json"
];
const documents = await Promise.all(schemaNames.map(async (name) =>
  JSON.parse(await readFile(path.join(repositoryRoot, "schemas", name), "utf8"))));
const validators = new Map(schemaNames.map((name, index) => [name, compileSchema(documents[index], documents)]));
const execFile = promisify(execFileCallback);
let temporary;
let examples;

before(async () => {
  temporary = await mkdtemp(path.join(os.tmpdir(), "puretokens-schema-fixtures-"));
  const output = path.join(temporary, "examples.json");
  try {
    await execFile("go", ["test", "-run", "^TestContractExamples$", "-count=1", "."], {
      cwd: path.join(repositoryRoot, "runtime", "executor"),
      env: { ...process.env, PURETOKENS_SCHEMA_EXAMPLES_OUT: output },
      timeout: 120_000,
      maxBuffer: 2 * 1024 * 1024
    });
  } catch (error) {
    throw new Error(`Native contract examples failed:\n${error.stdout ?? ""}\n${error.stderr ?? ""}`, { cause: error });
  }
  examples = JSON.parse(await readFile(output, "utf8"));
});
after(async () => {
  if (temporary) await rm(temporary, { recursive: true, force: true });
});

function example(name) {
  const found = examples.find((entry) => entry.name === name);
  assert.ok(found, `native fixture missing: ${name}`);
  return structuredClone(found.document);
}

function rejects(schema, document, keyword) {
  const errors = validators.get(schema)(document);
  assert.ok(errors.length > 0, `${schema} accepted an invalid document`);
  if (keyword) assert.ok(errors.some((error) => error.includes(keyword)), errors.join("\n"));
}

test("actual executor JSON and executable documentation conform to their schemas", async () => {
  const coveredSchemas = new Set();
  const names = new Set();
  for (const entry of examples) {
    assert.ok(!names.has(entry.name), `duplicate example name: ${entry.name}`);
    names.add(entry.name);
    const validator = validators.get(entry.schema);
    assert.ok(validator, `unexpected schema: ${entry.schema}`);
    assert.deepEqual(validator(entry.document), [], entry.name);
    coveredSchemas.add(entry.schema);
  }
  assert.deepEqual([...coveredSchemas].sort(), [...schemaNames].sort());
  const expectedNames = [];
  for (const kind of ["image", "video"]) {
    const text = await readFile(path.join(repositoryRoot, "skills", `puretokens-${kind}`, "references/executor-usage.md"), "utf8");
    const blocks = [...text.matchAll(/```json\s*\n([\s\S]*?)\n```/g)];
    assert.ok(blocks.length >= 4, `${kind}: generation, attachment, status and content examples must remain`);
    for (const [index, block] of blocks.entries()) {
      const name = `docs-${kind}-${index}`;
      expectedNames.push(name);
      assert.deepEqual(example(name), JSON.parse(block[1]), `${name}: executed request differs from documentation`);
    }
  }
  assert.deepEqual(examples.filter(entry => /^docs-(image|video)-\d+$/.test(entry.name)).map(entry => entry.name).sort(), expectedNames.sort());
});

test("media receipt schemas reject lost task context and unsafe metadata", () => {
  const receipt = example("record-0-submit-receipt");
  assert.equal(receipt.retry_not_before, "2026-09-06T00:00:05Z");
  assert.equal(receipt.retry_after_seconds, undefined, "local delay is not an API Retry-After");
  const initialRecord = example("record-0-submit-artifact");
  assert.equal(initialRecord.format, "puretokens-task-v1");
  assert.equal(initialRecord.retry_not_before, receipt.retry_not_before);
  assert.equal(example("record-1-resume-receipt").retry_not_before, undefined);
  for (const field of ["kind", "operation", "task_id", "status"]) {
    const changed = structuredClone(receipt);
    delete changed[field];
    rejects("executor-receipt.schema.json", changed, "required");
  }
  for (const addition of [
    { raw_response_body: "private upstream data" }, { prompt: "private prompt" },
    { parameters: { image_urls: ["https://reference.example.test/private.png"] } },
    { requested_count: 7 }, { http_status: 700 }, { submission_outcome: "assumed_success" }
  ]) rejects("executor-receipt.schema.json", { ...receipt, ...addition });
  const failure = example("submission-400");
  assert.equal(failure.submission_outcome, "rejected");
  assert.equal(example("submission-503").submission_outcome, "unknown");
  delete failure.next_action;
  rejects("executor-receipt.schema.json", failure, "required next_action");
  const retry = example("status-retry");
  rejects("executor-receipt.schema.json", { ...retry, retry_not_before: "2026-02-30T12:00:00Z" }, "format");
  rejects("executor-receipt.schema.json", { ...retry, retry_not_before: "tomorrow" }, "format");
});

test("music schemas preserve asynchronous identity, paid intent and private lyrics", () => {
  const input = example("docs-music-0");
  for (const addition of [
    {model:"other"}, {operation:"edit"}, {requested_count:2}, {index:1},
    {prompt:"字".repeat(1001)}, {output_dir:"relative"},
    {parameters:{response_format:"mp3"}},
    {parameters:{instrumental:true,response_format:"mp3",lyrics:"not instrumental"}},
    {parameters:{instrumental:false,response_format:"wav",lyrics:"字".repeat(4001)}},
    {parameters:{instrumental:false,response_format:"mp3",group:"other"}},
  ]) rejects("executor-request.schema.json",{...input,...addition});
  const record=example("music-1-content-record");
  assert.equal(record.parameters.instrumental,false);
  assert.equal(record.parameters.response_format,"wav");
  assert.equal(record.parameters.lyrics,undefined);
  assert.equal(record.prompt,undefined);
  rejects("task-record.schema.json",{...record,parameters:{...record.parameters,lyrics:"private"}});
  rejects("task-record.schema.json",{...record,kind:"audio"});
  assert.equal(example("music-0-resume").next_step,"deliver");
  assert.equal(example("music-0-delivered").next_step,"done");
  assert.deepEqual(example("music-model-receipt").result.data.map(item=>item.id),["stepaudio-3-music-preview"]);
});

test("evaluation schemas reject malformed questions and invented result fields", () => {
  const input = example("evaluation-request");
  rejects("evaluation-request.schema.json", {...input, async: true});
  rejects("evaluation-request.schema.json", {...input, model: "jev"});
  rejects("evaluation-request.schema.json", {...input, questions: {}});
  rejects("evaluation-request.schema.json", {...input, questions: Object.fromEntries(Array.from({length: 65}, (_, i) => [`q${i}`, {type:"noul",instructions:"Is this relevant?"}]))});
  rejects("evaluation-request.schema.json", {...input, questions: {q: {type:"score",instructions:"Rate",criteria:["single"]}}});
  const receipt = example("evaluation-success");
  for (const addition of [{state:"private"}, {task_id:"invented"}, {next_step:"wait"}]) {
    rejects("evaluation-receipt.schema.json", {...receipt,...addition});
  }
  const invalid = structuredClone(receipt);
  invalid.result.answers.urgency.noul = 2;
  rejects("evaluation-receipt.schema.json", invalid);
  const unknown = example("evaluation-invalid-response");
  rejects("evaluation-receipt.schema.json", {...unknown,result:receipt.result});
  rejects("evaluation-receipt.schema.json", {...unknown,next_step:"done"});
  const support = example("support-evaluation-success");
  for (const key of ["state","questions","answers","result"]) rejects("support-summary.schema.json", {...support,[key]:{}});
});

test("audio schemas distinguish local verification, synchronous text and binary handoff", () => {
  const speech = example("docs-audio-0");
  for (const extra of [
    { voice: "invented" }, { speed: 2.1 }, { input: "字".repeat(1001) },
    { file: "/private/extra.wav" }, { async: true }, { response_format: "url" },
    { output_dir: "relative" }, { model: "stepaudio-3-tts" }
  ]) rejects("audio-request.schema.json", { ...speech, ...extra });
  const transcript = example("audio-transcribe");
  for (const extra of [{task_id:"invented"}, {next_step:"wait"}, {delivery_status:"downloaded"}, {artifact:example("audio-speech").artifact}]) {
    rejects("audio-receipt.schema.json", { ...transcript, ...extra });
  }
  const generated = example("audio-speech");
  for (const extra of [{next_step:"done"}, {delivery_status:"delivered"}, {result:{text:"invented"}}, {submission_outcome:"not_submitted"}]) {
    rejects("audio-receipt.schema.json", { ...generated, ...extra });
  }
  const unknown = example("audio-unknown");
  for (const extra of [{artifact:generated.artifact}, {next_step:"deliver"}, {submission_outcome:"accepted"}]) {
    rejects("audio-receipt.schema.json", { ...unknown, ...extra });
  }
  const verify = example("audio-verify-false");
  rejects("audio-receipt.schema.json", {...verify,submission_outcome:"accepted"});
  rejects("audio-receipt.schema.json", {...verify,http_status:200});
  rejects("audio-artifact-verify-request.schema.json", {artifact:{...generated.artifact,sha256:"wrong"}});
  const support = example("support-audio-transcribe");
  for (const field of ["input", "instruction", "text", "artifact", "file"]) {
    rejects("support-summary.schema.json", {...support,[field]:"private"});
  }
  rejects("model-query.schema.json", {kind:"audio",operation:"clone"});
  rejects("model-query.schema.json", {kind:"audio",parameters:{speed:1}});
});

test("support summaries reject private data and invented network evidence", () => {
  const submission = example("support-docs-image-0-receipt");
  assert.equal(submission.api_request_attempted, true);
  assert.equal(submission.request_phase, "submission");
  assert.equal(submission.http_status, 200);
  assert.equal(submission.request_id, "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99");
  assert.equal(submission.submission_outcome, "accepted");
  assert.equal(example("support-doctor-success").request_phase, "catalog");
  assert.equal(example("support-doctor-no-credential").api_request_attempted, false);
  assert.equal(example("support-balance").request_phase, "balance_metadata");
  const summary = example("support-continuation-credential-failure");
  assert.equal(summary.api_request_attempted, false);
  assert.equal(summary.command, "status");
  assert.equal(summary.local_error_code, "host_credential_adapter_unavailable");
  for (const addition of [
    { prompt: "private" }, { raw_response: "private" }, { parameters: { image: "private" } },
    { base_url: "https://private.example.test" }, { downloaded_paths: ["/private/output"] },
    { request_id: "private-header" }, { host: "another-host" }, { api_error_code: "private-provider-code" },
    { local_error_code: "private-error" }, { request_id: "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99" },
    { http_status: 200 }, { request_phase: "submission" }
  ]) rejects("support-summary.schema.json", { ...summary, ...addition });
  const response = {
    ...summary, api_request_attempted: true, http_status: 503, request_phase: "submission",
    request_id: "55a3d6e3-42dd-4e67-8b08-e5c6058b2f99"
  };
  assert.deepEqual(validators.get("support-summary.schema.json")(response), []);
});

test("task records reject credentials, raw inputs and invalid progress", () => {
  const record = example("record-5-delivered-artifact");
  assert.deepEqual(record.delivered, [0, 1]);
  assert.equal(record.original_operation, "generate");
  for (const addition of [
    { prompt: "private prompt" }, { api_key: "synthetic-private-value" }, { reference_url: "https://reference.example.test" },
    { parameters: { generate_audio: "true" } }, { parameters: { n: 0 } },
    { downloaded: { 6: "/output/out.png" } }, { delivered: [0, 0] }, { delivered: [6] },
    { retry_not_before: "later" }, { format: "puretokens-task-v2" }
  ]) rejects("task-record.schema.json", { ...record, ...addition });
  for (const field of ["format", "kind"]) {
    const changed = structuredClone(record);
    delete changed[field];
    rejects("task-record.schema.json", changed, "required");
  }
});

test("recorded task failures and explicit redelivery preserve safe evidence", () => {
  const record = example("recorded-failure-artifact");
  assert.deepEqual(record.last_failure, { category: "model_permission", phase: "status", task_status: "failed", api_error_code: "model_not_allowed" });
  const restored = example("recorded-failure-resume");
  assert.equal(restored.failure_category, "model_permission");
  assert.equal(restored.failure_source, "task_record");
  assert.equal(restored.support.api_request_attempted, false);
  assert.equal(restored.support.api_error_code, undefined, "historical task evidence is not this invocation's API failure");
  for (const extra of [{ message: "private" }, { api_error_code: "private_provider_code" }, { category: "private_provider" }, { phase: "content" }, { task_status: "pending" }]) {
    rejects("task-record.schema.json", { ...record, last_failure: { ...record.last_failure, ...extra } });
  }
  const pending = example("explicit-redelivery-artifact");
  assert.deepEqual(pending.delivered, [0]);
  assert.equal(pending.pending_redelivery_index, 0);
  for (const name of ["explicit-redelivery-receipt", "explicit-redelivery-resume"]) {
    const result = example(name);
    assert.equal(result.redelivery_index, 0);
    assert.equal(result.next_step, "deliver");
    assert.deepEqual(result.delivered_indexes, [0]);
    assert.equal(result.downloaded_paths.length, 1);
  }
  const done = example("explicit-redelivery-ack");
  assert.equal(done.next_step, "done");
  assert.equal(done.redelivery_index, undefined);
  assert.equal(example("adaptive-receipt").parameters.aspect_ratio, "adaptive");
  rejects("task-record.schema.json", { ...pending, pending_redelivery_index: 6 });
});

test("invocation timings are measured phases without private or unobserved labels", () => {
  const samples = examples.filter(entry => entry.schema === "timings.schema.json");
  assert.ok(samples.length > 0, "actual command timing examples missing");
  for (const { document } of samples) {
    const sum = Object.entries(document).filter(([phase]) => phase !== "total").reduce((sum, [, ms]) => sum + ms, 0);
    assert.ok(sum <= document.total, "overlapping phases exceed observed command time");
  }
  const sample = samples[0].document;
  for (const addition of [{ total: -1 }, { request: 1.5 }, { server_generation: 12 }, { host_handoff: 2 }, { path: "/private/result.png" }]) {
    rejects("timings.schema.json", { ...sample, ...addition });
  }
  for (const entry of examples.filter(entry => entry.schema === "support-summary.schema.json" || entry.schema === "task-record.schema.json")) {
    assert.equal(entry.document.timings_ms, undefined, `${entry.name}: per-command timing was persisted or shared`);
  }
});

test("audio construction diagnostics identify only the invalid field and reviewed limit", () => {
  const voice = example("audio-diagnostic-unknown-voice");
  const limit = example("audio-diagnostic-unicode-limit");
  assert.deepEqual(voice.validation, {field: "voice", reason: "unsupported"});
  assert.deepEqual(limit.validation, {field: "input", reason: "max_characters", limit: 1000});
  for (const receipt of [voice, limit]) {
    assert.equal(receipt.submission_outcome, "not_submitted");
    assert.equal(receipt.support.api_request_attempted, false);
    assert.equal(receipt.support.validation, undefined);
    assert.equal(receipt.support.input, undefined);
    assert.equal(receipt.support.voice, undefined);
    assert.equal(receipt.timings_ms.request, undefined);
    for (const extra of [{field: "private-name"}, {reason: "private-reason"}, {value: "private-input"}, {limit: 0}]) {
      rejects("audio-receipt.schema.json", {...receipt, validation: {...receipt.validation, ...extra}});
    }
  }
});

test("raw quotas and legacy billing responses cannot masquerade as a balance projection", () => {
  const balance = example("balance-projection");
  assert.equal(balance.used, 19.672244);
  assert.equal(balance.remaining, 10);
  assert.equal(balance.unit, "USD");
  assert.equal(balance.scope, "account_wallet");
  for (const raw of [{}, { hard_limit_usd: 100000000 }, { total_usage: 275 }, { total_available: 5000000, total_used: 9836122, total_granted: 14836122, unlimited_quota: true }]) {
    rejects("balance-snapshot.schema.json", raw, "required");
  }
  for (const field of Object.keys(balance)) {
    const changed = structuredClone(balance);
    delete changed[field];
    rejects("balance-snapshot.schema.json", changed, `required ${field}`);
  }
  for (const addition of [{ unit: "CNY" }, { scope: "unlimited" }, { currency: "USD" }, { used: "2.75" }, { includes_subscription_quota: true }, { remaining: "infinity" }]) {
    rejects("balance-snapshot.schema.json", { ...balance, ...addition });
  }
  const envelope = example("balance");
  rejects("balance-receipt.schema.json", { ...envelope, result: { hard_limit_usd: 12.5 } });
  const usageFailure = example("balance-usage-failure");
  const unitFailure = example("balance-unit-failure");
  assert.equal(usageFailure.local_error_code, "balance_usage_auth_rejected");
  assert.equal(unitFailure.local_error_code, "balance_unit_metadata_unavailable");
  assert.equal(usageFailure.api_error_code, "auth_required");
  assert.equal(unitFailure.api_error_code, "auth_required");
  assert.match(usageFailure.next_action, /keep the existing connection/);
});

test("console-origin balance exception does not change media transport or allow legacy billing", async () => {
  const read = async (relative) => JSON.parse(await readFile(path.join(repositoryRoot, relative), "utf8"));
  const schema = await read("schemas/media-execution-contract.schema.json");
  const validate = compileSchema(schema, [schema]);
  const balance = await read("skills/puretokens-balance/references/execution-contract.json");
  assert.deepEqual(validate(balance), []);
  for (const patch of [
    { url: "https://api.puretokensx.com/v1/dashboard/billing/subscription" },
    { url: "https://another-provider.invalid/api/usage/token/" },
    { requiresBrowserSession: true }, { unitMetadataRequiresConfiguredApiKey: true },
    { unitMetadataUrl: "https://another-provider.invalid/status" },
    { maxRequests: 3 }, { bearerNormalization: "retry_after_401_with_another_key" }
  ]) {
    const changed = structuredClone(balance);
    Object.assign(changed.operations.read, patch);
    assert.ok(validate(changed).length > 0, JSON.stringify(patch));
  }
  for (const kind of ["image", "video", "models", "connection"]) {
    const contract = await read(`skills/puretokens-${kind}/references/execution-contract.json`);
    assert.deepEqual(validate(contract), [], kind);
    contract.transport.fixedApiOrigin = "https://console.puretokensx.com";
    assert.ok(validate(contract).length > 0, `${kind} accepted the balance-only origin`);
  }
  const directSchema = await read("schemas/direct-api-execution-contract.schema.json");
  const validateDirect = compileSchema(directSchema, [directSchema]);
  const direct = await read("references/direct-api-execution-contract.json");
  assert.deepEqual(validateDirect(direct), []);
  direct.balance.unit = "CNY";
  assert.ok(validateDirect(direct).length > 0);
});

test("model query receipts keep declared matching separate from raw catalog fields", () => {
  const receipt = example("models-true");
  assert.equal(receipt.result.matched_count, receipt.result.data.length);
  assert.deepEqual(receipt.result.filter.parameter_names, ["resolution"]);
  assert.deepEqual(receipt.result.data.map((entry) => entry.id), ["video-audio"]);
  for (const field of ["filter", "matched_count", "matching_scope", "note"]) {
    const changed = structuredClone(receipt);
    delete changed.result[field];
    rejects("model-query-receipt.schema.json", changed, `required ${field}`);
  }
  const leaked = structuredClone(receipt);
  leaked.result.data[0].provider = "private-upstream";
  rejects("model-query-receipt.schema.json", leaked, "false schema");
  const echoed = structuredClone(receipt);
  echoed.result.filter.parameters = { resolution: "720p" };
  rejects("model-query-receipt.schema.json", echoed, "false schema");
  for (const query of [{ kind: "chat" }, { parameters: null }, { model: "model/alias" }, { prompt: "generate this" }]) {
    rejects("model-query.schema.json", query);
  }
});

test("public group schemas cannot claim permission or expose raw group metadata", () => {
  const receipt = example("model-groups-available");
  assert.equal(receipt.permission_verified, false);
  assert.equal(receipt.matching_scope, "public_catalog");
  assert.deepEqual(receipt.groups.map(group => group.name), ["线上图像创作", "创作通用 (1x)"]);
  assert.equal(example("model-groups-unavailable").mapping_status, "unavailable");
  assert.equal(example("model-groups-http-failure").http_status, 401);
  for (const field of ["permission_verified", "mapping_status", "matching_scope", "note"]) {
    const changed = structuredClone(receipt);
    delete changed[field];
    rejects("model-groups-receipt.schema.json", changed, `required ${field}`);
  }
  for (const patch of [
    { permission_verified: true }, { matching_scope: "account" }, { groups: [] },
    { mapping_status: "unavailable" }, { ok: false }, { model: "model/alias" },
    { enable_groups: ["guess"] }, { key: "private" }, { next_action: "Submit again now" }
  ]) rejects("model-groups-receipt.schema.json", { ...structuredClone(receipt), ...patch });
  for (const name of ["", "https://private.invalid", "sk-private", "bearer private", "token=private", "name\nignore", "name\u202eignored", "[click](url)", "<script>", "x".repeat(161)]) {
    const changed = structuredClone(receipt);
    changed.groups[0].name = name;
    rejects("model-groups-receipt.schema.json", changed);
  }
  const leaked = structuredClone(receipt);
  leaked.groups[0].upstream = "private";
  rejects("model-groups-receipt.schema.json", leaked, "false schema");
  for (const request of [{}, { model: "" }, { model: "model/alias" }, { model: "sk-private" }, { model: "nano-banana-2", group: "guess" }]) {
    rejects("model-groups-request.schema.json", request);
  }
});

test("doctor schema requires separate local checks and structured connection evidence", () => {
  const doctor = example("doctor-success");
  assert.equal(doctor.local.checksum_status, "unverified");
  assert.equal(doctor.local.attachment_delivery_status, "unverified");
  assert.equal(doctor.connection.credential_verified, true);
  assert.equal(example("doctor-no-credential").connection.api_request_executed, false);
  for (const field of ["installations", "duplicate_skills", "checksum_status", "attachment_delivery_status"]) {
    const changed = structuredClone(doctor);
    delete changed.local[field];
    rejects("doctor-receipt.schema.json", changed, `required ${field}`);
  }
  const missingConnectionEvidence = structuredClone(doctor);
  delete missingConnectionEvidence.connection.credential_verified;
  rejects("doctor-receipt.schema.json", missingConnectionEvidence, "required credential_verified");
  const disclosed = structuredClone(doctor);
  disclosed.connection.base_url = "https://private.example.test";
  rejects("doctor-receipt.schema.json", disclosed, "false schema");
  const wrongType = structuredClone(doctor);
  wrongType.local.installations[0].loaded = "true";
  rejects("doctor-receipt.schema.json", wrongType, "type");
});

test("request schemas reject missing submission or continuation identity", () => {
  for (const request of [
    { kind: "image" },
    { kind: "image", operation: "generate", model: "gpt-image-2" },
    { kind: "image", operation: "generate", model: "gpt-image-2", prompt: "  " },
    { kind: "video", task_id: "../wrong-task" },
    { kind: "image", task_id: "fixture", poll: { max_status_reads: 100 } },
    { kind: "image", task_id: "fixture", lifecycle: { status_url: "/other" } }
  ]) rejects("executor-request.schema.json", request);
  const submission = example("docs-image-0");
  for (const addition of [
    { task_id: "existing" }, { task_status: "completed" }, { original_operation: "edit" },
    { retry_not_before: "2026-09-05T00:00:00Z" }, { reconciliation_required: false }, { index: 1 }
  ]) rejects("executor-request.schema.json", { ...submission, ...addition });
  assert.deepEqual(validators.get("executor-request.schema.json")({ kind: "image", task_id: "fixture", requested_count: 0 }), []);
  assert.deepEqual(validators.get("executor-request.schema.json")({ kind: "image", task_id: "fixture", poll: { max_status_reads: 40, deadline_seconds: 120 } }), []);
  rejects("executor-request.schema.json", { kind: "image", task_id: "fixture", poll: { max_status_reads: 41 } });
  assert.deepEqual(validators.get("executor-request.schema.json")({ kind: "video", task_id: "fixture", poll: { max_status_reads: 7 } }), []);
  rejects("executor-request.schema.json", { kind: "video", task_id: "fixture", poll: { max_status_reads: 8 } });
});

test("schema evaluator fails closed and exercises branches, references and JSON equality", () => {
  const id = "https://fixture.example.test/schema.json";
  const compile = (schema) => {
    const document = { $id: id, ...schema };
    return compileSchema(document, [document]);
  };
  assert.throws(() => compile({ anyOf: [{ type: "string" }, { unsupportedKeyword: true }] }), /Unsupported schema keyword/);
  assert.throws(() => compile({ format: "invented-format" }), /Unsupported schema format/);
  assert.throws(() => compile({ $ref: "missing.json" }), /Unregistered schema reference/);
  assert.throws(() => compile({ $ref: "#/$defs/missing" }), /Unresolved schema reference/);
  const nonempty = compile({ type: "object", minProperties: 1 });
  assert.deepEqual(nonempty({ value: 0 }), []);
  assert.ok(nonempty({}).some((error) => error.includes("minProperties")));
  const branches = compile({
    $defs: { count: { type: "integer", minimum: 1, maximum: 6 } },
    type: "object",
    properties: { count: { $ref: "#/$defs/count" }, values: { type: "array", uniqueItems: true, items: { type: "string" } } },
    additionalProperties: false,
    oneOf: [{ required: ["count"] }, { required: ["values"] }]
  });
  assert.deepEqual(branches({ count: 2 }), []);
  for (const value of [{}, { count: 0 }, { count: "2" }, { count: 2, values: [] }, { values: ["a", "a"] }, { values: [1] }, { count: 2, extra: true }]) {
    assert.ok(branches(value).length > 0);
  }
  const equality = compile({ enum: [{ a: 1, b: 2 }] });
  assert.deepEqual(equality({ b: 2, a: 1 }), []);
  const conditional = compile({ if: { type: "integer" }, then: { minimum: 2 }, else: { const: "unknown" }, not: { const: 3 } });
  assert.deepEqual(conditional(2), []);
  assert.deepEqual(conditional("unknown"), []);
  for (const value of [1, 3, "other"]) assert.ok(conditional(value).length > 0);
});
