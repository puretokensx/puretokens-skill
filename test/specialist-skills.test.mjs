import assert from "node:assert/strict";
import { execFile as execFileCallback, execFileSync } from "node:child_process";
import { mkdir, mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { promisify } from "node:util";

import { collectSkillRecords, repositoryRoot, validateRepository } from "../scripts/skill-registry.mjs";

const execFile = promisify(execFileCallback);
const skillNames = [
  "puretokens-balance",
  "puretokens-connection",
  "puretokens-models",
  "puretokens-image",
  "puretokens-video",
  "puretokens-audio", "puretokens-evaluate", "puretokens-embedding", "puretokens-rerank", "puretokens-update"
];
const apiSkillNames = skillNames.filter((name) => name !== "puretokens-update");
const apiOrigin = "https://api.puretokensx.com";
const executorRequestStart = "invoke_the_checksum_verified_managed_native_executor_with_the_current_host_id_and_fixed_request";
const executorRequestActualFailureOnly = "report_only_an_actual_executor_network_credential_adapter_or_attachment_byte_failure_and_never_open_a_browser_desktop_ui_or_computer_use";
const packageManifest = JSON.parse(await readFile(path.join(repositoryRoot, "package.json"), "utf8"));

function assertSynchronized(stdout) {
  assert.ok(stdout.includes(`Pure Tokens Skills ${packageManifest.version} synchronized with the native API executor at`));
}

function currentExecutorPlatform() {
  const platform = process.platform === "darwin" ? "darwin" : process.platform === "linux" ? "linux" : "windows";
  const shellArch = process.platform === "win32" ? process.arch : execFileSync("uname", ["-m"], { encoding: "utf8" }).trim();
  const architecture = ["x64", "x86_64"].includes(shellArch) ? "amd64" : ["arm64", "aarch64"].includes(shellArch) ? "arm64" : shellArch;
  return `${platform}-${architecture}`;
}

test("repository validates its managed native executor contract", async () => {
  assert.deepEqual(await validateRepository(), []);
});

test("all specialist Skills are versioned together and use the expected source files", async () => {
  const packageManifest = JSON.parse(await readFile(path.join(repositoryRoot, "package.json"), "utf8"));
  const { registry, records } = await collectSkillRecords();
  assert.match(packageManifest.version, /^\d+\.\d+\.\d+$/);
  assert.deepEqual(registry.skills.map((entry) => entry.name), skillNames);
  assert.equal(records.length, skillNames.length);
  for (const record of records) {
    assert.equal(record.manifest.version, packageManifest.version);
    assert.equal(record.frontmatter.name, record.manifest.name);
  }
});

test("API Skills use the checksum-verified native executor without a user runtime", async () => {
  for (const name of apiSkillNames) {
    const base = path.join(repositoryRoot, "skills", name);
    const [skill, manifest, contract] = await Promise.all([
      readFile(path.join(base, "SKILL.md"), "utf8"),
      readFile(path.join(base, "skill.json"), "utf8").then(JSON.parse),
      readFile(path.join(base, "references", "execution-contract.json"), "utf8").then(JSON.parse)
    ]);
    assert.match(skill, /https:\/\/api\.puretokensx\.com/);
    assert.equal(manifest.rules.usesManagedNativeExecutor, true);
    assert.equal(manifest.rules.usesCurrentHostConfiguredCredential, true);
    assert.equal(manifest.rules.resolvesCurrentHostConnectionCredentialInMemoryForFixedRequest, true);
    assert.equal(manifest.rules.doesNotUseNodeRuntime, true);
    assert.equal(manifest.rules.doesNotUseComputerUseOrUiAutomation, true);
    assert.equal(manifest.rules.doesNotInvokeOtherMediaSkillsAsApiFallback, true);
    assert.equal(contract.transport.fixedApiOrigin, name === "puretokens-balance" ? "https://console.puretokensx.com" : apiOrigin);
    if (name === "puretokens-balance") {
      assert.equal(manifest.rules.balanceRequiresExistingAccountSession, false);
      assert.equal(contract.operations.read.requiresBrowserSession, false);
      assert.equal(contract.operations.read.unitMetadataRequiresConfiguredApiKey, false);
    }
    assert.equal(contract.transport.usesFullApiUrls, true);
    assert.equal(contract.transport.usesManagedNativeExecutor, true);
    assert.equal(contract.transport.usesCurrentHostConfiguredCredential, true);
    assert.equal(contract.transport.credentialSourceNeverBecomesRequestTarget, true);
    assert.equal(contract.transport.neverExposesCredentialsOrHostConfiguration, true);
    assert.equal(contract.transport.doesNotUseNodeRuntime, true);
    assert.equal(contract.transport.doesNotUseMcpOrFallbackTransport, true);
    assert.equal(contract.transport.doesNotUseComputerUseOrUiAutomation, true);
    assert.equal(contract.transport.doesNotInvokeOtherMediaSkillsAsApiFallback, true);
    assert.equal(contract.transport.executionStart, executorRequestStart);
    assert.equal(contract.transport.whenManagedNativeExecutorUnavailable, executorRequestActualFailureOnly);
    assert.ok(skill.includes("references/execution-contract.json"));
    assert.match(skill, /\.puretokens-executor\/puretokens-api/);
    assert.ok(skill.includes("puretokens-api.exe"));
    assert.doesNotMatch(skill, /puretokens-direct-api\.mjs|\.puretokens-runtime|--json-base64|--multipart-base64|Authorization: Bearer/);
  }
  const manifest = JSON.parse(await readFile(path.join(repositoryRoot, "runtime", "executor", "manifest.json"), "utf8"));
  assert.equal(manifest.name, "puretokens-api-executor");
  assert.match(manifest.artifacts[currentExecutorPlatform()].sha256, /^[a-f0-9]{64}$/);
});

test("media routing prioritizes Pure Tokens specialists before generic media Skills", async () => {
  const directContract = JSON.parse(await readFile(path.join(repositoryRoot, "references", "direct-api-execution-contract.json"), "utf8"));
  assert.deepEqual(directContract.mediaRouting, {
    imageAndVideoRequestsPreferPureTokensSpecialists: true,
    existingResourceLookupDoesNotAuthorizeGeneration: true,
    imageSkill: "puretokens-image",
    videoSkill: "puretokens-video",
    priority: "when_current_host_connection_uses_puretokens_select_the_matching_puretokens_specialist_before_generic_imagegen_imagen_or_video_skills",
    afterSpecialistSelection: "never_fall_back_to_a_generic_media_skill",
    connectionContext: "managed_native_executor_resolves_one_matching_current_connection_credential_privately_for_the_fixed_request",
    metadataLimitation: "skill_metadata_expresses_routing_priority_but_cannot_override_a_host_that_ignores_installed_skill_selection"
  });
  for (const [name, genericSkill] of [["puretokens-image", "imagegen"], ["puretokens-video", "通用视频"]]) {
    const base = path.join(repositoryRoot, "skills", name);
    const [skill, manifest, agentMetadata, scenarios] = await Promise.all([
      readFile(path.join(base, "SKILL.md"), "utf8"),
      readFile(path.join(base, "skill.json"), "utf8").then(JSON.parse),
      readFile(path.join(base, "agents", "openai.yaml"), "utf8"),
      readFile(path.join(base, "references", "behavior-scenarios.json"), "utf8").then(JSON.parse)
    ]);
    assert.match(skill, /生成或编辑.*续接已有.*任务或取回结果优先使用本 Skill/);
    assert.match(skill, new RegExp(genericSkill));
    assert.match(manifest.description, /Primary/);
    assert.match(agentMetadata, /Primary/);
    assert.equal(scenarios.scenarios[0].id, `${name.replace("puretokens-", "")}-specialist-routing-priority`);
  }
});

test("image and video start with the installed executor and never fall back", async () => {
  const image = await readFile(path.join(repositoryRoot, "skills", "puretokens-image", "SKILL.md"), "utf8");
  const video = await readFile(path.join(repositoryRoot, "skills", "puretokens-video", "SKILL.md"), "utf8");
  for (const [kind, skill] of [["image", image], ["video", video]]) {
    assert.match(skill, /唯一 API 传输/);
    assert.match(skill, /必须调用安装的原生执行器/);
    assert.match(skill, /不得自行发 HTTP/);
    assert.doesNotMatch(skill, /~\/.codex\/auth\.json|Authorization: Bearer/);
    const scenarios = JSON.parse(await readFile(path.join(repositoryRoot, "skills", `puretokens-${kind}`, "references", "behavior-scenarios.json"), "utf8"));
    assert.equal(scenarios.scenarios.some((scenario) => scenario.id === `${kind}-executor-request-start`), true);
    assert.equal(scenarios.scenarios.some((scenario) => scenario.id === `${kind}-executor-validation-failure`), true);
  }
  assert.match(image, /https:\/\/api\.puretokensx\.com\/v1\/images\/edits/);
  assert.match(image, /`image` 字段/);
});

test("media contracts retain fixed endpoints, asynchronous same-task handling, and native delivery", async () => {
  const image = JSON.parse(await readFile(path.join(repositoryRoot, "skills", "puretokens-image", "references", "execution-contract.json"), "utf8"));
  const video = JSON.parse(await readFile(path.join(repositoryRoot, "skills", "puretokens-video", "references", "execution-contract.json"), "utf8"));
  assert.equal(image.operations.submit.url, `${apiOrigin}/v1/images/generations`);
  assert.equal(image.operations.status.urlTemplate, `${apiOrigin}/v1/images/{task_id}`);
  assert.equal(image.operations.content.urlTemplate, `${apiOrigin}/v1/images/{task_id}/content?index={index}`);
  assert.equal(image.parameterValidation.defaultModel, "gpt-image-2.5-flare");
  assert.equal(video.parameterValidation.defaultModel, "MiniMax-H3");
  assert.equal(image.contentRetrieval.indexBase, 0);
  assert.equal(image.contentRetrieval.completeDeliveryRequiresEveryRequestedIndex, true);
  assert.equal(image.submissionFailure.hostExecutionPolicyBlockedBeforeRequestStarts, "report_validation_no_api_request_or_task_id_and_require_host_session_with_external_network_permission");
  assert.equal(video.operations.submit.url, `${apiOrigin}/v1/videos`);
  assert.equal(video.operations.status.urlTemplate, `${apiOrigin}/v1/videos/{task_id}`);
  assert.equal(video.operations.content.urlTemplate, `${apiOrigin}/v1/videos/{task_id}/content`);
  assert.equal(video.inputMediaValidation.nativeAttachmentBodyMode, "executor_multipart_only");
  assert.equal(video.submissionFailure.hostExecutionPolicyBlockedBeforeRequestStarts, "report_validation_no_api_request_or_task_id_and_require_host_session_with_external_network_permission");
  for (const contract of [image, video]) {
    assert.equal(contract.result.sameTaskOnly, true);
    assert.equal(contract.result.neverAutoResubmit, true);
    assert.equal(contract.transport.requiresNativeMediaByteDelivery, true);
  }
});

test("media Skills load a compact index and only the selected model profile", async () => {
  for (const [name, capability, defaultModel] of [
    ["puretokens-image", "image", "gpt-image-2.5-flare"],
    ["puretokens-video", "video", "MiniMax-H3"]
  ]) {
    const base = path.join(repositoryRoot, "skills", name);
    const [skill, manifest, index] = await Promise.all([
      readFile(path.join(base, "SKILL.md"), "utf8"),
      readFile(path.join(base, "skill.json"), "utf8").then(JSON.parse),
      readFile(path.join(base, "references", "model-index.json"), "utf8").then(JSON.parse)
    ]);
    assert.equal(manifest.modelIndex, "references/model-index.json");
    assert.equal(manifest.rules.usesLazyModelProfileLoading, true);
    assert.equal(manifest.rules.loadsBehaviorScenariosOnDemandOnly, true);
    assert.equal(index.capability, capability);
    assert.equal(index.defaultModel, defaultModel);
    assert.equal(manifest.rules.defaultModel, defaultModel);
    assert.ok(skill.includes(`默认 \`${defaultModel}\` 或用户精确 ID`));
    assert.equal(index.models.some((model) => model.id === defaultModel), true);
    for (const model of index.models) {
      const profile = JSON.parse(await readFile(path.join(base, "references", model.profile), "utf8"));
      assert.equal(profile.id, model.id);
      assert.equal(profile.capability, capability);
      assert.equal(typeof profile.parameterSchema, "object");
    }
    assert.ok(skill.includes("references/model-index.json"));
    assert.ok(skill.includes("references/profiles/<model>.json"));
    await assert.rejects(readFile(path.join(base, "references", "model-selection.json"), "utf8"));
  }
});

test("media entrypoints route exceptional detail without weakening support privacy", async () => {
  for (const kind of ["image", "video"]) {
    const base = path.join(repositoryRoot, "skills", `puretokens-${kind}`);
    const [skill, usage, failure, receipt, scenarios] = await Promise.all([
      readFile(path.join(base, "SKILL.md"), "utf8"),
      readFile(path.join(base, "references/executor-usage.md"), "utf8"),
      readFile(path.join(base, "references/failure-guide.md"), "utf8"),
      readFile(path.join(base, "references/task-receipt.json"), "utf8").then(JSON.parse),
      readFile(path.join(base, "references/behavior-scenarios.json"), "utf8").then(JSON.parse)
    ]);
    assert.ok(skill.includes("references/executor-usage.md#continuation-record"));
    assert.match(usage, /## Continuation record/);
    assert.match(usage, /Across computers or operating systems/);
    assert.match(skill, /不先查余额、init、doctor、preflight 或实时目录/);
    assert.match(skill, /累计两个窗口/);
    assert.match(skill, /取消、失败回执、网络超时、未知状态或对账均停止自动续接/);
    assert.match(skill, /实际交接后才执行/);
    assert.match(failure, /api_request_attempted=true/);
    assert.match(failure, /不自动导出、上传、读取凭据或追加诊断/);
    assert.match(receipt.failure.conditionalFields.support, /never the whole receipt/);
    const scenario = scenarios.scenarios.find(s => s.id === `${kind}-private-support-summary`);
    assert.match(scenario?.then ?? "", /never resubmit to obtain one/);
  }
});

for (const kind of ["image", "video"]) {
  test(`${kind} simple requests avoid planning-only rounds without bypassing safety`, async () => {
    const base = path.join(repositoryRoot, "skills", `puretokens-${kind}`);
    const [skill, manifest] = await Promise.all([
      readFile(path.join(base, "SKILL.md"), "utf8"),
      readFile(path.join(base, "skill.json"), "utf8").then(JSON.parse)
    ]);
    assert.equal(manifest.rules.simpleRequestsSkipPlanningOnlyRounds, true);
    assert.equal(manifest.rules.reusesAccessibleCurrentAttachmentPath, true);
    assert.equal(manifest.rules.followsNextStepWithoutPlanningInterludes, true);
    const shortPath = skill.match(/## 简单请求短路径\n([\s\S]*?)(?=\n## )/)?.[1];
    assert.ok(shortPath, "both media entrypoints must carry the short-path contract");
    assert.match(shortPath, /模型、素材和用途明确的单次生成或编辑/);
    assert.match(shortPath, /除非用户要求计划或任务属于复杂多阶段工作/);
    assert.match(shortPath, /不创建或更新任务清单/);
    assert.match(shortPath, /必要且互不依赖的读取放在同一轮/);
    assert.match(shortPath, /已读且仍在有效上下文中的说明不重读/);
    assert.match(shortPath, /仅缺命令用法时才读 executor-usage/);
    assert.match(shortPath, /直接使用宿主提供且执行器可访问的附件绝对路径/);
    assert.match(shortPath, /不为整理目录复制、重命名或改写原图/);
    assert.match(shortPath, /宿主必须先物化附件时才保存一份字节不变的文件/);
    assert.match(shortPath, /收到回执后直接按 `next_step` 执行/);
    assert.match(shortPath, /不在 wait、content、deliver 之间插入任务清单或额外规划/);
    assert.match(shortPath, /不能把 submit、wait、content 合成一个命令/);
    assert.match(skill, /先告知返回的 task_id 和状态/);
    assert.match(skill, /累计两个窗口/);
    assert.match(skill, /实际交接后才执行/);
    assert.match(skill, /media_operation/);
    assert.match(skill, /没有声明的传输方式就停止/);
  });
}

test("the public install prompt remains extractable in both README files", async () => {
  const expected = "Install or update the official Pure Tokens Skills from https://github.com/puretokensx/puretokens-skill.";
  let prompts;
  for (const file of ["README.md", "README.zh-CN.md"]) {
    const readme = await readFile(path.join(repositoryRoot, file), "utf8");
    const match = readme.match(/### (?:Copy this to a terminal-capable local agent|复制给可在本机执行命令的 Agent)\n\n```text\n([^\n]+)\n```/);
    assert.equal(match?.[1], expected);
    const complete = [
      readme.match(/### (?:Chinese installation prompt|中文安装提示词)\n\n```text\n([\s\S]*?)\n```/)?.[1],
      readme.match(/### (?:English installation prompt|英文安装提示词)\n\n```text\n([\s\S]*?)\n```/)?.[1]
    ];
    assert.ok(complete.every(Boolean), "both complete prompts must remain copyable");
    for (const prompt of complete) {
      assert.ok(prompt.includes("https://github.com/puretokensx/puretokens-skill/releases/latest/download/"));
      assert.ok(prompt.includes("puretokens-skill-fetch.sh") && prompt.includes("puretokens-skill-fetch.ps1"));
      assert.ok(prompt.includes("install"));
      assert.doesNotMatch(prompt, /--host codex|-Host codex/);
    }
    assert.ok(complete[0].length <= 300 && complete[1].length <= 600, "copyable prompts should contain only the task and bootstrap entry");
    if (prompts) assert.deepEqual(complete, prompts, "permission/continuation instructions must not drift between READMEs");
    prompts = complete;
  }
});

test("host matrix lists nineteen installer hosts and the bounded credential adapters", async () => {
  const support = JSON.parse(await readFile(path.join(repositoryRoot, "references", "host-support.json"), "utf8"));
  assert.deepEqual(support.supported.map((host) => host.id), ["claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop"]);
  for (const host of support.supported) {
    assert.equal(host.delivery, "native-installer");
    assert.equal(host.directMediaExecution, "managed-native-executor");
  }
  assert.deepEqual(support.supported.filter((host) => host.credentialAdapter === "fixture-tested").map((host) => host.id), ["claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop"]);
});

test("source installer synchronizes Skills and exactly one native executor", async (t) => {
  const target = await mkdtemp(path.join(os.tmpdir(), "puretokens-skill-executor-"));
  t.after(() => rm(target, { recursive: true, force: true }));
  const installer = path.join(repositoryRoot, "runtime", "puretokens-skill-install.sh");
  const { stdout } = await execFile("sh", [installer, "sync", "--target", target, "--source", repositoryRoot], { cwd: repositoryRoot });
  assertSynchronized(stdout);
  assert.match(stdout, /Pure Tokens Skill init: host ID was not supplied, so the connection check was deferred/);
  assert.match(stdout, /Example \(not executed\):/);
  assert.doesNotMatch(stdout, /Pure Tokens Skill 使用须知/);
  assert.deepEqual((await readdir(target)).filter((entry) => skillNames.includes(entry)).sort(), [...skillNames].sort());
  await assert.rejects(readFile(path.join(target, ".puretokens-runtime", "runtime.json"), "utf8"));
  const executor = path.join(target, ".puretokens-executor", process.platform === "win32" ? "puretokens-api.exe" : "puretokens-api");
  const { stdout: version } = await execFile(executor, ["--version"]);
  assert.equal(version.trim(), packageManifest.version);
});

test("source installer exposes explicit init with compact guidance and on-demand help", async (t) => {
  const target = await mkdtemp(path.join(os.tmpdir(), "puretokens-skill-init-"));
  t.after(() => rm(target, { recursive: true, force: true }));
  const installer = path.join(repositoryRoot, "runtime", "puretokens-skill-install.sh");
  await execFile("sh", [installer, "sync", "--target", target, "--source", repositoryRoot], { cwd: repositoryRoot });
  const { stdout } = await execFile("sh", [installer, "init", "--target", target], { cwd: repositoryRoot });
  assert.match(stdout, /Pure Tokens Skill init: host ID was not supplied, so the connection check was deferred/);
  assert.equal(stdout.split("Example (not executed):").length - 1, 2);
  assert.match(stdout, /read the local usage guide without a network request/);
  assert.doesNotMatch(stdout, /## 操作与恢复|# Pure Tokens Skill 使用须知/);
  assert.ok(Buffer.byteLength(stdout) < 2200);
});

test("catalog freshness gate remains present for releases", async () => {
  const policy = JSON.parse(await readFile(path.join(repositoryRoot, "references", "catalog-freshness.json"), "utf8"));
  assert.equal(policy.maxAgeDays, 7);
  assert.equal(policy.releaseCommand, "npm run release:validate");
});
