import { createHash } from "node:crypto";
import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { validModelID } from "./model-id.mjs";

export const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
export const skillsRoot = path.join(repositoryRoot, "skills");
export const mediaDefaultModels = Object.freeze({ image: "gpt-image-2.5-flare", video: "MiniMax-H3" });

const forbiddenPattern = /(BEGIN [A-Z ]*PRIVATE|127\.0\.0\.1:|\/Users\/)/i;
const directApiOrigin = "https://api.puretokensx.com";
const balanceApiOrigin = "https://console.puretokensx.com";
const balanceUsageUrl = `${balanceApiOrigin}/api/product/console/api-keys/usage`;
const balanceUnitUrl = `${balanceApiOrigin}/api/product/console/status`;
const balanceScopeRule = "unlimited_quota_true_account_wallet_else_key_allowance";
const balanceCalculation = "each_returned_quota_divided_by_public_quota_per_unit_in_USD_never_subtract_used_from_available";
const balanceBearerNormalization = "add_sk_prefix_if_missing_in_memory_for_usage_header_only";
const balanceLocalFailureCodes = ["balance_usage_unavailable", "balance_usage_auth_rejected", "balance_unit_metadata_unavailable"];
const directAcceptanceScenarioIds = ["api-identity-read", "catalog-read", "media-submit", "same-task-status", "native-media-delivery", "executor-request-start"];
const directExecutorCapabilities = ["fixed_full_url_request", "json_task_response", "same_task_status_read", "native_media_byte_delivery", "bounded_local_resource_use"];
const mediaRoutingPriority = "when_current_host_connection_uses_puretokens_select_the_matching_puretokens_specialist_before_generic_imagegen_imagen_or_video_skills";
const mediaRoutingMetadataLimitation = "skill_metadata_expresses_routing_priority_but_cannot_override_a_host_that_ignores_installed_skill_selection";
const executorRequestStart = "invoke_the_checksum_verified_managed_native_executor_with_the_current_host_id_and_fixed_request";
const executorRequestActualFailureOnly = "report_only_an_actual_executor_network_credential_adapter_or_attachment_byte_failure_and_never_open_a_browser_desktop_ui_or_computer_use";
const receiptPhaseFields = { submission: ["task_id", "status"], continuation: ["task_id", "status"], reconciliation: ["task_id", "reconciliation_required"], completion: ["delivered_count"], failure: ["error_message", "next_action"] };
const failureReceiptRequiredFields = ["failure_phase", "error_message", "next_action"];
const failureReceiptPhases = ["validation", "submission", "status", "content"];
const failureReceiptForbiddenDisclosure = ["raw_response_body", "upstream_or_provider_identifier", "internal_hostname_or_url", "stack_trace", "request_headers", "request_body", "credentials_or_session_data", "user_reference_url_or_attachment_bytes"];

export async function readSkillRegistry() {
  const registryPath = path.join(skillsRoot, "index.json");
  const registry = JSON.parse(await readFile(registryPath, "utf8"));
  if (registry?.schemaVersion !== 1 || !Array.isArray(registry.skills)) {
    throw new Error("skills/index.json must contain schemaVersion=1 and a skills array");
  }
  return registry;
}

export async function collectSkillRecords() {
  const registry = await readSkillRegistry();
  const directoryEntries = await readdir(skillsRoot, { withFileTypes: true });
  const directories = directoryEntries.filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort();
  const records = [];

  for (const directory of directories) {
    const skillDir = path.join(skillsRoot, directory);
    const [skillText, manifestText] = await Promise.all([
      readFile(path.join(skillDir, "SKILL.md"), "utf8"),
      readFile(path.join(skillDir, "skill.json"), "utf8")
    ]);
    const manifest = JSON.parse(manifestText);
    const frontmatter = readFrontmatter(skillText);
    records.push({ directory, skillDir, skillText, manifest, frontmatter });
  }

  return { registry, records };
}

export async function validateRepository() {
  const errors = [];
  const { registry, records } = await collectSkillRecords();
  const packageManifest = JSON.parse(await readFile(path.join(repositoryRoot, "package.json"), "utf8"));
  if (!/^\d+\.\d+\.\d+$/.test(packageManifest?.version || "")) {
    errors.push("package.json must contain a semver version");
  }
  await validateInstallerSources(errors);
  await validateSchemaDocuments(errors);
  const hostSupport = await readHostSupport(errors);
  const directApiExecutionContract = await readDirectApiExecutionContract(errors);
  await validateCatalogFreshnessPolicy(errors);
  validateDirectApiExecutionContract(errors, directApiExecutionContract, hostSupport);
  const seen = new Set();
  const registryByName = new Map(registry.skills.map((skill) => [skill?.name, skill]));

  for (const record of records) {
    const { directory, skillText, manifest, frontmatter } = record;
    const name = manifest?.name;
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(directory)) errors.push(`${directory}: directory must use kebab-case`);
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(name || "")) errors.push(`${directory}: manifest name must use kebab-case`);
    if (name !== directory) errors.push(`${directory}: directory and manifest name must match`);
    if (frontmatter.name !== name) errors.push(`${directory}: SKILL.md frontmatter name must match manifest`);
    if (!frontmatter.description) errors.push(`${directory}: SKILL.md needs a non-empty frontmatter description`);
    if (!manifest?.version || !/^\d+\.\d+\.\d+$/.test(manifest.version)) errors.push(`${directory}: manifest version must be semver`);
    if (manifest?.version !== packageManifest?.version) errors.push(`${directory}: manifest version must match package.json`);
    if (manifest?.sourceSha256 !== undefined) {
      const sourceSha256 = String(manifest.sourceSha256);
      const actualSourceSha256 = createHash("sha256").update(skillText).digest("hex");
      if (!/^[a-f0-9]{64}$/.test(sourceSha256)) {
        errors.push(`${directory}: sourceSha256 must be a lowercase SHA-256 digest`);
      } else if (sourceSha256 !== actualSourceSha256) {
        errors.push(`${directory}: sourceSha256 does not match SKILL.md`);
      }
    }
    if (seen.has(name)) errors.push(`${directory}: duplicate skill name ${name}`);
    seen.add(name);
    const registryEntry = registryByName.get(name);
    if (!registryEntry) errors.push(`${directory}: missing skills/index.json entry`);
    if (registryEntry && registryEntry.version !== manifest.version) {
      errors.push(`${directory}: manifest and skills/index.json versions differ`);
    }
    if (manifest?.behaviorTests) await verifyFile(errors, manifest.behaviorTests);
    if (manifest?.naturalLanguageAliases) await verifyFile(errors, manifest.naturalLanguageAliases);
    await validateSpecialistReferences(errors, record);
    const declaredClients = Array.isArray(manifest?.supportedClients) ? [...manifest.supportedClients].sort() : [];
    const supportedHostIds = hostSupport.supported.map((host) => host.id).sort();
    if (supportedHostIds.length && JSON.stringify(declaredClients) !== JSON.stringify(supportedHostIds)) {
      errors.push(`${directory}: supportedClients must match references/host-support.json`);
    }
    if (name === "puretokens-image" || name === "puretokens-video" || name === "puretokens-audio" || name === "puretokens-evaluate") {
      if (!skillText.includes("references/host-delivery.md")) {
        errors.push(`${directory}: media entry must link its installed host delivery guide`);
      }
      await verifyFile(errors, `skills/${directory}/references/host-delivery.md`);
    }
    validateHostDistributions(errors, directory, manifest, hostSupport);
    if (forbiddenPattern.test(skillText) || forbiddenPattern.test(JSON.stringify(manifest))) {
      errors.push(`${directory}: skill content contains a forbidden credential or local endpoint marker`);
    }
  }

  const expected = ["puretokens-balance", "puretokens-connection", "puretokens-models", "puretokens-image", "puretokens-video", "puretokens-audio", "puretokens-evaluate", "puretokens-embedding", "puretokens-rerank", "puretokens-update"];
  if (registry.skills.length !== expected.length || expected.some((name, index) => registry.skills[index]?.name !== name)) {
    errors.push("skills/index.json must list the ten specialist Skills in order");
  }

  for (const entry of registry.skills) {
    if (!entry || !seen.has(entry.name)) errors.push(`skills/index.json: ${entry?.name || "unnamed entry"} has no skill directory`);
    if (!entry?.entry || !entry?.manifest) errors.push(`skills/index.json: ${entry?.name || "unnamed entry"} needs entry and manifest`);
    if (entry?.entry) await verifyFile(errors, entry.entry);
    if (entry?.manifest) await verifyFile(errors, entry.manifest);
  }

  return errors;
}

async function readHostSupport(errors) {
  const file = path.join(repositoryRoot, "references", "host-support.json");
  try {
    const support = JSON.parse(await readFile(file, "utf8"));
    if (support?.$schema !== "https://puretokensx.com/schemas/host-support.schema.json") {
      errors.push("references/host-support.json must declare the host-support schema");
    }
    if (support?.schemaVersion !== 1 || !Array.isArray(support.supported)) {
      errors.push("references/host-support.json must be schemaVersion=1 with a supported array");
      return { supported: [] };
    }
    if (support.directApiOrigin !== directApiOrigin) {
      errors.push("references/host-support.json must declare the fixed direct API origin");
    }
    const records = support.supported;
    const ids = records.map((host) => host?.id).filter((id) => typeof id === "string" && /^[a-z][a-z0-9-]*$/.test(id));
    if (ids.length !== records.length || new Set(ids).size !== ids.length) {
      errors.push("references/host-support.json must contain unique, kebab-case host IDs");
    }
    for (const host of support.supported) {
      if (!host || typeof host.guidance !== "string" || !host.guidance) {
        errors.push(`references/host-support.json: ${host?.id || "unnamed host"} needs guidance`);
      }
      const validDirectory = host?.id === "octop"
        ? host.globalSkillDirectory === undefined && host.workspaceSkillDirectory === ".octop/skills"
        : typeof host?.globalSkillDirectory === "string" && /^~\/(?:(?:\.[a-z-]+\/)*(?:[a-z-]+\/)?|Library\/Application Support\/dsh-desktop\/harness\/)skills$/.test(host.globalSkillDirectory);
      if (host?.delivery !== "native-installer" || !validDirectory ||
        host.directMediaExecution !== "managed-native-executor" || !["fixture-tested", "pending"].includes(host.credentialAdapter)) {
        errors.push(`references/host-support.json: ${host?.id || "unnamed host"} needs a declared native-installer Skill directory`);
      }
    }
    return { supported: support.supported };
  } catch {
    errors.push("references/host-support.json is missing or invalid JSON");
    return { supported: [] };
  }
}

async function readDirectApiExecutionContract(errors) {
  const file = path.join(repositoryRoot, "references", "direct-api-execution-contract.json");
  try {
    return JSON.parse(await readFile(file, "utf8"));
  } catch {
    errors.push("references/direct-api-execution-contract.json is missing or invalid JSON");
    return undefined;
  }
}

async function validateCatalogFreshnessPolicy(errors) {
  const file = path.join(repositoryRoot, "references", "catalog-freshness.json");
  try {
    const policy = JSON.parse(await readFile(file, "utf8"));
    if (policy?.$schema !== "https://puretokensx.com/schemas/catalog-freshness.schema.json" || policy?.schemaVersion !== 1 ||
      !Number.isInteger(policy?.maxAgeDays) || policy.maxAgeDays < 1 || policy.maxAgeDays > 31 ||
      policy.releaseCommand !== "npm run release:validate" || typeof policy.message !== "string" || !policy.message) {
      errors.push("references/catalog-freshness.json must declare a valid release freshness policy");
    }
  } catch {
    errors.push("references/catalog-freshness.json is missing or invalid JSON");
  }
}

function validateDirectApiExecutionContract(errors, contract, hostSupport) {
  const executorCredentialAdapterHostIds = hostSupport.supported.filter(host => host.credentialAdapter === "fixture-tested").map(host => host.id);
  const label = "references/direct-api-execution-contract.json";
  if (!contract || typeof contract !== "object") return;
  if (contract.$schema !== "https://puretokensx.com/schemas/direct-api-execution-contract.schema.json" || contract.schemaVersion !== 1) {
    errors.push(`${label} must declare schemaVersion=1 direct API execution contract`);
  }
  const transport = contract.transport;
  if (contract.apiOrigin !== directApiOrigin || !contract.authentication ||
    contract.authentication.usesCurrentHostConfiguredCredential !== true ||
    !sameArray(contract.authentication.executorCredentialAdapterHosts, executorCredentialAdapterHostIds) ||
    contract.authentication.credentialUse !== "in_memory_only_for_fixed_puretokens_api_requests" ||
    contract.authentication.credentialSourceMustMatchFixedPureTokensEndpoint !== true ||
    contract.authentication.credentialSourceNeverBecomesRequestTarget !== true ||
    contract.authentication.neverDisplaysCopiesStoresOrRequestsCredentials !== true ||
    contract.authentication.resolvesCurrentHostConnectionCredentialInMemoryForFixedRequest !== true ||
    contract.authentication.neverScansForCredentials !== true ||
    !transport || transport.usesFullApiUrls !== true || transport.usesManagedNativeExecutor !== true || transport.executorIsOneShot !== true ||
    transport.doesNotRequireNode !== true || transport.doesNotUseMcp !== true ||
    transport.doesNotUseLocalProxyOrSidecar !== true || transport.doesNotUseFallbackEndpoint !== true ||
    transport.doesNotUseComputerUseOrUiAutomation !== true || transport.doesNotInvokeOtherMediaSkillsAsApiFallback !== true ||
    transport.executionStart !== executorRequestStart || transport.whenManagedNativeExecutorUnavailable !== executorRequestActualFailureOnly) {
    errors.push(`${label} must use the managed one-shot native executor for the fixed full API origin without Node, MCP, proxy, sidecar, UI automation, Computer Use, or another media-Skill fallback`);
  }
  const mediaRouting = contract.mediaRouting;
  if (!mediaRouting || mediaRouting.imageAndVideoRequestsPreferPureTokensSpecialists !== true ||
    mediaRouting.imageSkill !== "puretokens-image" || mediaRouting.videoSkill !== "puretokens-video" ||
    mediaRouting.priority !== mediaRoutingPriority ||
    mediaRouting.afterSpecialistSelection !== "never_fall_back_to_a_generic_media_skill" ||
    mediaRouting.connectionContext !== "managed_native_executor_resolves_one_matching_current_connection_credential_privately_for_the_fixed_request" ||
    mediaRouting.metadataLimitation !== mediaRoutingMetadataLimitation) {
    errors.push(`${label} must prioritize the matching Pure Tokens media specialist before generic media Skills while resolving only one matching current connection credential privately for the fixed request and never using a fallback`);
  }
  if (!Array.isArray(contract.acceptanceScenarios) || !sameArray(contract.acceptanceScenarios.map((scenario) => scenario?.id), directAcceptanceScenarioIds)) {
    errors.push(`${label} must define the ordered direct-API acceptance scenarios`);
  }
  const installableHostIds = hostSupport.supported.map((host) => host.id);
  if (!sameArray(contract.installableHosts, installableHostIds) || !sameArray(contract.executorCredentialAdapterHosts, executorCredentialAdapterHostIds) ||
    !sameArray(contract.requiredExecutorCapabilities, directExecutorCapabilities)) {
    errors.push(`${label} must define the declared installable hosts, verified executor credential adapters, and required executor capabilities`);
  }
  const balance = contract.balance;
  if (!balance || balance.method !== "GET" || balance.url !== balanceUsageUrl ||
    balance.unitMetadataUrl !== balanceUnitUrl || balance.unitMetadataRequiresConfiguredApiKey !== false ||
    balance.requiresBrowserSession !== false || balance.maxRequests !== 2 || balance.deadlineSeconds !== 30 ||
    balance.scopeRule !== balanceScopeRule || balance.unit !== "USD" || balance.includesSubscriptionQuota !== false ||
    balance.calculation !== balanceCalculation || balance.bearerNormalization !== balanceBearerNormalization ||
    balance.requiresConfiguredApiKey !== true || balance.responseSchema !== "schemas/balance-snapshot.schema.json" ||
    typeof balance.fallback !== "string" || !balance.fallback) {
    errors.push(`${label} must define the fixed direct balance endpoint and fallback`);
  }
  const evaluation = contract.evaluation;
  if (!evaluation || evaluation.method !== "POST" ||
    evaluation.url !== `${directApiOrigin}/typesafe/v1/systemone` ||
    evaluation.command !== "evaluate" || evaluation.deadlineSeconds !== 90 ||
    evaluation.maxRequests !== 1 || evaluation.synchronous !== true ||
    evaluation.neverAutomaticallyRetries !== true || evaluation.onePostPerInvocation !== true ||
    evaluation.neverPoll !== true ||
    evaluation.requestSchema !== "schemas/evaluation-request.schema.json" ||
    evaluation.responseSchema !== "schemas/evaluation-receipt.schema.json" ||
    evaluation.modelDiscoveryUrl !== `${directApiOrigin}/v1/models` ||
    evaluation.modelDiscoveryScope !== "explicit_reviewed_model_visibility_only") {
    errors.push(`${label} must define one synchronous fixed-route evaluation and explicit model visibility lookup`);
  }
  const audio = contract.audio;
  if (!audio || !sameArray(audio.commands, ["audio", "audio-verify"]) ||
    !sameArray(audio.urls, ["speech", "transcriptions", "generate"].map(op => `${directApiOrigin}/v1/audio/${op}`)) ||
    audio.method !== "POST" || audio.deadlineSeconds !== 90 || audio.maxRequests !== 1 ||
    audio.synchronous !== true || audio.neverAutomaticallyRetries !== true || audio.neverPoll !== true ||
    audio.verifyLocalOnly !== true || audio.nativeAudioBytesRequired !== true || audio.downloadIsNotDelivery !== true ||
    audio.modelDiscoveryUrl !== `${directApiOrigin}/v1/models` ||
    audio.modelDiscoveryScope !== "explicit_reviewed_model_visibility_only" ||
    audio.requestSchema !== "schemas/audio-request.schema.json" ||
    audio.responseSchema !== "schemas/audio-receipt.schema.json" ||
    audio.verifySchema !== "schemas/audio-artifact-verify-request.schema.json") {
    errors.push(`${label} must define bounded synchronous audio and local artifact verification`);
  }
  for (const kind of ["embedding", "rerank"]) {
    const retrieval = contract[kind];
    if (!retrieval || retrieval.method !== "POST" ||
      retrieval.url !== `${directApiOrigin}/v1/${kind === "embedding" ? "embeddings" : kind}` ||
      retrieval.command !== kind || retrieval.deadlineSeconds !== 90 || retrieval.maxRequests !== 1 ||
      retrieval.requestSchema !== `schemas/${kind}-request.schema.json` ||
      retrieval.responseSchema !== "schemas/retrieval-receipt.schema.json" ||
      retrieval.modelDiscoveryUrl !== `${directApiOrigin}/v1/models` ||
      retrieval.modelDiscoveryScope !== "explicit_reviewed_model_visibility_only" ||
      ["synchronous", "neverAutomaticallyRetries", "onePostPerInvocation", "neverPoll"].some(key => retrieval[key] !== true)) {
      errors.push(`${label} must define one bounded synchronous ${kind} request and explicit reviewed visibility lookup`);
    }
  }
  const music = contract.music;
  if (!music || music.kind !== "music" || music.model !== "stepaudio-3-music-preview" ||
    music.submitUrl !== `${directApiOrigin}/v1/audio/music/submit` ||
    music.statusUrlTemplate !== `${directApiOrigin}/v1/audio/music/tasks/{task_id}` ||
    music.contentUrlTemplate !== `${directApiOrigin}/v1/audio/music/tasks/{task_id}/content` ||
    music.maxOutputBytes !== 33554432 || music.artifactRetentionSeconds !== 86400 ||
    music.retentionStarts !== "gateway_artifact_saved_at" || music.waitWindowMaxReads !== 7 ||
    music.waitWindowDeadlineSeconds !== 300 || music.maxAutomaticWaitWindows !== 2 ||
    ["asynchronous", "onePostPerSubmission", "neverAutomaticallyResubmits", "recordRequiredBySkill",
      "recordExcludesPromptAndLyrics", "apiTaskBoundToOwnerAndOriginatingKey",
      "reconciliationStopsAutomaticContinuation", "downloadIsNotDelivery"].some(key => music[key] !== true)) {
    errors.push(`${label} must define bounded asynchronous music, original-key task recovery and native delivery`);
  }
  const userMediaInput = contract.userMediaInput;
  if (!userMediaInput || userMediaInput.onlyCurrentUserExplicitMedia !== true ||
    userMediaInput.publicUrlRequiresInstalledOrOnDemandProfilePropertyAndTransport !== true ||
    userMediaInput.publicUrlImageEditRequiresExactDeclaredJsonOperation !== true ||
    userMediaInput.requiresInstalledOrOnDemandDeclaredTransport !== true ||
    userMediaInput.skillNeverDownloadsOrRehosts !== true ||
    userMediaInput.executorSendsDeclaredMultipartAttachmentsDirectly !== true ||
    userMediaInput.nativeAttachmentRouteMustMatchDeclaredOperation !== true ||
    userMediaInput.nativeAttachmentMustNeverFallBackToJsonTextSubmission !== true ||
    userMediaInput.untransportableNativeReferenceMustNeverBeConvertedToTextPrompt !== true ||
    userMediaInput.currentRequestScopeOnly !== true ||
    typeof userMediaInput.fallback !== "string" || !userMediaInput.fallback) {
    errors.push(`${label} must define the fail-closed direct user-media input contract`);
  }
  const asyncTaskHandling = contract.asyncTaskHandling;
  if (contract.supportSummary?.format !== "puretokens-support-v1" ||
    contract.supportSummary.noAdditionalIO !== true ||
    contract.supportSummary.noAutomaticExportOrUpload !== true ||
    contract.supportSummary.neverUseUpstreamRequestId !== true ||
    contract.supportSummary.privateSupportOnly !== true ||
    contract.supportSummary.neverUseAsTaskRecordOrIdempotencyKey !== true ||
    contract.supportSummary.source !== "current_invocation_receipt_and_latest_http_attempt_only" ||
    contract.supportSummary.requestIdRule !== "one_canonical_lowercase_uuid_legacy_only_if_standard_absent_omit_conflicts" ||
    !sameArray(contract.supportSummary.requestIdHeaders ?? [], ["X-Request-ID", "X-Oneapi-Request-Id"])) {
    errors.push(`${label} must retain the local-only, allowlisted support summary contract`);
  }
  if (!asyncTaskHandling || asyncTaskHandling.providerProtocolOwner !== "gateway" ||
    asyncTaskHandling.publicImageContract !== "submit_once_receive_durable_task_id_then_read_same_task_status_and_protected_content" ||
    asyncTaskHandling.asyncFlagMeaning !== "product_task_compatibility_not_supplier_async_capability" ||
    asyncTaskHandling.neverSendInternalWorkspaceDeliveryHeaders !== true ||
    asyncTaskHandling.neverSwitchToSynchronousProviderTransport !== true ||
    asyncTaskHandling.reconciliationRequiredPrecedesStateClassification !== true ||
    asyncTaskHandling.reconciliationOutcome !== "stop_ordinary_polling_retain_same_task_id_do_not_submit_replacement_or_infer_refund" ||
    asyncTaskHandling.submissionOutputUnknown !== "treat_submission_outcome_as_unknown_do_not_repeat_post_or_submit_replacement_task" ||
    asyncTaskHandling.statusHttp429 !== "honor_valid_positive_retry_after_then_continue_same_task_if_remaining_budget" ||
    asyncTaskHandling.non429StatusReadFailure !== "stop_bounded_window_without_replacement_task" ||
    asyncTaskHandling.doesNotInventApiErrorCode !== true) {
    errors.push(`${label} must define reconciliation, unknown-submission, status-rate-limit, and API-code handling for the same public task`);
  }
  const mediaDeliveryResourceBounds = contract.mediaDeliveryResourceBounds;
  if (!mediaDeliveryResourceBounds || mediaDeliveryResourceBounds.oneInFlightStatusReadPerTask !== true ||
    mediaDeliveryResourceBounds.noBackgroundPollingOrQueue !== true ||
    mediaDeliveryResourceBounds.readContentOnlyAfterTerminalSuccess !== true ||
    mediaDeliveryResourceBounds.oneInFlightContentReadPerTask !== true ||
    mediaDeliveryResourceBounds.sequentialImageContentIndexes !== true ||
    mediaDeliveryResourceBounds.neverAutomaticallyPrefetchOrRefetchDeliveredContent !== true ||
    mediaDeliveryResourceBounds.handoffNativeBytesBeforeNextRead !== true ||
    mediaDeliveryResourceBounds.doesNotPersistMediaBytesInSkillStateOrLogs !== true ||
    mediaDeliveryResourceBounds.confirmedDeliveryEndsCurrentResponse !== true ||
    typeof mediaDeliveryResourceBounds.fallback !== "string" || !mediaDeliveryResourceBounds.fallback) {
    errors.push(`${label} must define bounded same-task polling and native-media delivery without skill-state or log persistence`);
  }
}

async function validateSpecialistReferences(errors, record) {
  const { directory, skillDir, manifest } = record;
  const required = ["executionContract", "behaviorScenarios"];
  if (directory === "puretokens-image" || directory === "puretokens-video") required.unshift("modelIndex", "taskReceipt");
  const references = {};
  for (const field of required) {
    const relativePath = manifest?.[field];
    if (typeof relativePath !== "string" || !relativePath) {
      errors.push(`${directory}: ${field} must name an installed reference file`);
      continue;
    }
    references[field] = await readSkillReference(errors, skillDir, relativePath, `${directory}: ${field}`);
  }
  validateExecutionContract(errors, directory, references.executionContract);
  validateBehaviorScenarios(errors, directory, references.behaviorScenarios);
  if (references.modelIndex) {
    await validateModelIndex(errors, directory, skillDir, references.modelIndex);
    const defaultModel = references.modelIndex.defaultModel;
    if (manifest.rules?.defaultModel !== defaultModel || references.executionContract?.parameterValidation?.defaultModel !== defaultModel ||
        !record.skillText.includes(`默认 \`${defaultModel}\` 或用户精确 ID`)) {
      errors.push(`${directory}: Skill entry, manifest and execution contract must match the model index defaultModel`);
    }
  }
  if (references.taskReceipt) validateTaskReceipt(errors, directory, references.taskReceipt);
  if (directory === "puretokens-update") {
    if (typeof manifest?.usageGuide !== "string" || !manifest.usageGuide) {
      errors.push(`${directory}: usageGuide must name an installed usage guide`);
    } else {
      await verifySkillFile(errors, skillDir, manifest.usageGuide, `${directory}: usageGuide`);
    }
  }
  if ((directory === "puretokens-image" || directory === "puretokens-video") &&
    (manifest?.rules?.usesLazyModelProfileLoading !== true || manifest?.rules?.loadsBehaviorScenariosOnDemandOnly !== true)) {
    errors.push(`${directory}: rules must require lazy selected-profile and on-demand behavior-scenario loading`);
  }
}

function validateHostDistributions(errors, directory, manifest, hostSupport) {
  const distribution = manifest?.distribution;
  if (!distribution || typeof distribution !== "object") {
    errors.push(`${directory}: distribution is required`);
    return;
  }
  const expectedKeys = new Set();
  for (const host of hostSupport.supported) {
    if (host.delivery !== "native-installer") continue;
    const key = distributionKeyForHost(host.id);
    expectedKeys.add(key);
    const delivery = distribution[key];
    if (!delivery || typeof delivery !== "object") {
      errors.push(`${directory}: ${host.id} manual source distribution is required`);
      continue;
    }
    if (delivery.manualInstallationSupported !== true) {
      errors.push(`${directory}: ${host.id} must support manual source installation`);
    }
    if (delivery.globalSkillDirectory !== host.globalSkillDirectory) {
      errors.push(`${directory}: ${host.id} global Skill directory must match references/host-support.json`);
    }
    if (delivery.workspaceSkillDirectory !== host.workspaceSkillDirectory) {
      errors.push(`${directory}: ${host.id} workspace Skill directory must match references/host-support.json`);
    }
    if (host.id === "codex") {
      if (delivery.requiresPluginFeature !== false) errors.push(`${directory}: Codex delivery must not require the Plugin feature`);
      for (const retiredField of ["managedByDesktop", "managedSkillDirectory", "plugin"]) {
        if (Object.hasOwn(delivery, retiredField)) errors.push(`${directory}: Codex delivery must not declare ${retiredField}`);
      }
    }
  }
  for (const key of Object.keys(distribution)) {
    if (!expectedKeys.has(key)) errors.push(`${directory}: distribution has unsupported host entry ${key}`);
  }
}

function distributionKeyForHost(id) {
  return id.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}

function validateExecutionContract(errors, directory, contract) {
  const label = `${directory}: executionContract`;
  if (!contract || typeof contract !== "object") return;
  if (contract.$schema !== "https://puretokensx.com/schemas/media-execution-contract.schema.json") errors.push(`${label} must declare the execution-contract schema`);
  const kind = skillKind(directory);
  if (contract.schemaVersion !== 1 || contract.kind !== kind) errors.push(`${label} must be schemaVersion=1 for ${kind}`);
  if (kind === "update") {
    validateUpdateContract(errors, label, contract);
    return;
  }
  validateDirectTransport(errors, label, contract.transport, kind === "image" || kind === "video", kind === "balance" ? balanceApiOrigin : directApiOrigin);
  const operations = contract.operations;
  if (!operations || typeof operations !== "object") {
    errors.push(`${label} must define operations`);
    return;
  }
  if (kind === "evaluate") {
    validateRequest(errors, `${label} submit`, operations.submit, "POST", `${directApiOrigin}/typesafe/v1/systemone`);
    validateRequiredBodyFields(errors, `${label} submit`, operations.submit, ["model", "state", "questions"]);
    for (const rule of ["synchronous", "neverAutomaticallyRetries", "onePostPerInvocation", "neverPoll", "typedQuestionMatching", "unknownSubmissionStops", "supportExcludesInputsAndResults", "confidenceIsNotAccuracy"]) {
      if (contract.result?.[rule] !== true) errors.push(`${label} must require ${rule}`);
    }
    return;
  }
  if (kind === "embedding" || kind === "rerank") {
    const embedding = kind === "embedding";
    validateRequest(errors, `${label} submit`, operations.submit, "POST", `${directApiOrigin}/v1/${embedding ? "embeddings" : "rerank"}`);
    validateRequiredBodyFields(errors, `${label} submit`, operations.submit, embedding
      ? ["model", "input", "encoding_format"]
      : ["model", "query", "documents", "top_n", "return_documents"]);
    for (const rule of ["synchronous", "neverAutomaticallyRetries", "onePostPerInvocation", "neverPoll",
      "unknownSubmissionStops", "supportExcludesInputsAndResults", "validatesRequestAndNewOutputBeforeCredentials"]) {
      if (contract.result?.[rule] !== true) errors.push(`${label} must require ${rule}`);
    }
    const limits = contract.parameterValidation;
    const reviewed = embedding ? ["qwen3.7-text-embedding", "qwen3.7-text-embedding-flash"] : ["qwen3.7-text-rerank"];
    const requestFields = embedding ? ["model", "input", "output_file"] : ["model", "query", "documents", "top_n", "output_file"];
    if (contract.result?.deadlineSeconds !== 90 || contract.result?.outputFormat !== (embedding ? "jsonl" : "json") ||
      limits?.maxRequestBytes !== 1048576 || limits?.maxResponseBytes !== 2097152 ||
      limits?.minItems !== 1 || limits?.maxItems !== 64 || limits?.maxTextRunes !== 32768 ||
      !sameArray(limits?.reviewedModelIds, reviewed) || !sameArray(limits?.requestFields, requestFields) ||
      limits?.outputFile !== `explicit_absolute_new_${embedding ? "jsonl" : "json"}_file_existing_writable_parent`) {
      errors.push(`${label} must retain bounded reviewed text inputs and an explicit new output file`);
    }
    if (embedding) {
      if (operations.submit?.fixedBody?.encoding_format !== "float" ||
        !sameArray(contract.result?.outputFields, ["index", "dimensions", "embedding"]) ||
        contract.result?.outputOrder !== "original_zero_based_input_index" ||
        !sameArray(contract.result?.chatSummaryFields, ["item_count", "dimensions", "path"])) {
        errors.push(`${label} must save ordered float embeddings and keep vectors out of chat`);
      }
    } else if (operations.submit?.fixedBody?.return_documents !== false || limits?.maxQueryRunes !== 8192 ||
      limits?.topNDefault !== "document_count" || limits?.topNRange !== "1..document_count" ||
      !sameArray(contract.result?.outputFields, ["rank", "index", "relevance_score"]) ||
      contract.result?.rankBase !== 1 || contract.result?.indexBase !== 0 ||
      contract.result?.scoreIsNotProbability !== true || contract.result?.preservesOriginalFiniteScore !== true ||
      contract.result?.receiptSummaryMatchesFile !== true) {
      errors.push(`${label} must preserve original candidate indexes and finite relevance scores`);
    }
    return;
  }
  if (kind === "audio") {
    for (const [op, suffix] of [["speech", "speech"], ["transcribe", "transcriptions"], ["generate", "generate"], ["music_sync", "generate"], ["voice_change", "generate"]]) {
      validateRequest(errors, `${label} ${op}`, operations[op], "POST", `${directApiOrigin}/v1/audio/${suffix}`);
    }
    validateRequest(errors, `${label} music submit`, operations.musicSubmit, "POST", `${directApiOrigin}/v1/audio/music/submit`);
    validateRequest(errors, `${label} music status`, operations.musicStatus, "GET", `${directApiOrigin}/v1/audio/music/tasks/{task_id}`, true);
    validateRequest(errors, `${label} music content`, operations.musicContent, "GET", `${directApiOrigin}/v1/audio/music/tasks/{task_id}/content`, true);
    if (!sameArray(contract.result?.synchronousOperations, ["speech", "transcribe", "generate", "music_sync", "voice_change"]) ||
      contract.result?.synchronousMusic?.model !== "music_v2_5" ||
      contract.result?.synchronousMusic?.neverUsesAsyncMusicRoute !== true ||
      contract.result?.asynchronousMusic?.neverAutomaticallyResubmits !== true) errors.push(`${label} must separate synchronous audio and async music`);
    for (const rule of ["onePostPerInvocation", "neverAutomaticallyRetries",
      "nativeAudioBytesRequired", "localArtifactVerification", "downloadIsNotDelivery", "supportExcludesInputsAndResults"]) {
      if (contract.result?.[rule] !== true) errors.push(`${label} must require ${rule}`);
    }
    return;
  }
  if (kind === "balance") {
    validateRequest(errors, `${label} read`, operations.read, "GET", balanceUsageUrl);
    if (operations.read?.unitMetadataUrl !== balanceUnitUrl || operations.read?.unitMetadataRequiresConfiguredApiKey !== false ||
      operations.read?.requiresBrowserSession !== false || operations.read?.maxRequests !== 2 || operations.read?.deadlineSeconds !== 30 ||
      operations.read?.requiresConfiguredApiKey !== true || operations.read?.bearerNormalization !== balanceBearerNormalization ||
      operations.read?.responseSchema !== "https://puretokensx.com/schemas/balance-snapshot.schema.json") {
      errors.push(`${label} read must require the configured API key and the balance snapshot schema`);
    }
    if (contract.result?.reportOnlyReturnedFields !== true || contract.result?.neverEstimate !== true || contract.result?.neverRetry !== true ||
      contract.result?.scopeRule !== balanceScopeRule || contract.result?.unit !== "USD" || contract.result?.includesSubscriptionQuota !== false ||
      contract.result?.calculation !== balanceCalculation || contract.result?.legacyBillingFallbackAllowed !== false ||
      contract.result?.defaultPresentation !== "one_line_remaining_with_wallet_or_key_allowance_label" ||
      contract.result?.balanceRejectionDoesNotInvalidateMediaConnection !== true ||
      !sameArray(contract.result?.localFailureCodes, balanceLocalFailureCodes) ||
      contract.result?.fallbackWhenBalanceUnavailable !== "report_balance_specific_failure_without_amount_and_preserve_working_connection") {
      errors.push(`${label} must report only returned balance fields and use the direct-request fallback`);
    }
    return;
  }
  if (kind === "connection") {
    validateRequest(errors, `${label} identity`, operations.identity, "GET", `${directApiOrigin}/v1`);
    if (contract.result?.identitySource !== "api_declared_public_health" || contract.result?.expectedStatus !== "ok" ||
      contract.result?.expectedName !== "Pure Tokens API" || contract.result?.expectedBasePath !== "/v1" ||
      contract.result?.neverRetry !== true || contract.result?.doesNotExposeHostConfiguration !== true ||
      contract.result?.unconfirmedDoesNotProveOtherService !== true) {
      errors.push(`${label} must verify only the fixed /v1 API identity without retrying, exposing host configuration, or inferring another service`);
    }
    return;
  }
  if (kind === "models") {
    validateRequest(errors, `${label} catalog`, operations.catalog, "GET", `${directApiOrigin}/v1/media/models`);
    validateRequest(errors, `${label} evaluation catalog`, operations.evaluationCatalog, "GET", `${directApiOrigin}/v1/models`);
    validateRequest(errors, `${label} audio catalog`, operations.audioCatalog, "GET", `${directApiOrigin}/v1/models`);
    validateRequest(errors, `${label} embedding catalog`, operations.embeddingCatalog, "GET", `${directApiOrigin}/v1/models`);
    validateRequest(errors, `${label} rerank catalog`, operations.rerankCatalog, "GET", `${directApiOrigin}/v1/models`);
    validateRequest(errors, `${label} public group guidance`, operations.groupGuidance, "GET", "https://console.puretokensx.com/api/product/docs/model-catalog");
    if (operations.groupGuidance?.requiresConfiguredApiKey !== false ||
      contract.result?.groupGuidance?.requiresConfiguredApiKey !== false ||
      contract.result?.groupGuidance?.readsHostConfiguration !== false ||
      contract.result?.groupGuidance?.permissionVerified !== false ||
      contract.result?.groupGuidance?.maxRequests !== 1 ||
      contract.result?.modelCatalogRequiresAuthentication !== true || contract.result?.doesNotSubmitMediaTasks !== true ||
      contract.result?.neverRetry !== true || contract.result?.noStaticCatalogFallback !== true ||
      contract.result?.mediaCompatibilityShortlistsRequireDeclaredCapabilityAndInputSchema !== true ||
      contract.result?.mediaFilterMatchesDeclaredSchemaOnly !== true ||
      contract.result?.evaluationDiscovery !== "explicit_reviewed_model_visibility_only" ||
      contract.result?.audioDiscovery !== "explicit_reviewed_model_visibility_only" ||
      contract.result?.embeddingDiscovery !== "explicit_reviewed_model_visibility_only" ||
      contract.result?.rerankDiscovery !== "explicit_reviewed_model_visibility_only") {
      errors.push(`${label} must separate anonymous group guidance from authenticated model visibility, never submit media work or retry, and derive compatibility only from declared profile data`);
    }
    return;
  }
  if (kind === "image") {
    validateRequest(errors, `${label} catalog`, operations.catalog, "GET", `${directApiOrigin}/v1/media/models`);
    validateRequest(errors, `${label} submit`, operations.submit, "POST", `${directApiOrigin}/v1/images/generations`);
    validateRequiredBodyFields(errors, `${label} submit`, operations.submit, ["model", "prompt", "async"]);
    validateRequest(errors, `${label} status`, operations.status, "GET", `${directApiOrigin}/v1/images/{task_id}`, true);
    validateRequest(errors, `${label} content`, operations.content, "GET", `${directApiOrigin}/v1/images/{task_id}/content?index={index}`, true);
    const editPaths = Array.isArray(operations.edit?.allowedPaths) ? operations.edit.allowedPaths : [];
    if (operations.edit?.method !== "POST" || editPaths.length !== 2 || !editPaths.includes("/v1/images/generations") || !editPaths.includes("/v1/images/edits") ||
      operations.edit?.pathSource !== "selected_profile_with_model_index_only_for_selection_or_on_demand_live_profile_operation_path" || operations.edit?.contentType !== "profile_declared_json_or_multipart" ||
      operations.edit?.requiresDeclaredProfileOperation !== "image_edit") {
      errors.push(`${label} must define the fixed-origin declared image-edit operation`);
    }
    validateRequiredBodyFields(errors, `${label} edit`, operations.edit, ["model", "prompt", "async"]);
    if (operations.submit?.fixedBody?.async !== true || operations.edit?.fixedBody?.async !== true ||
      operations.edit?.inputSource !== "current_user_explicit_public_url_or_native_media_only" || operations.edit?.transport !== "profile_declared_json_public_url_or_multipart_file") {
      errors.push(`${label} must fix async and declare the profile-gated JSON-URL or multipart image-edit input`);
    }
    if (contract.parameterValidation?.defaultModel !== mediaDefaultModels.image || contract.parameterValidation?.normalSubmissionUsesSelectedProfileAndIndexWhenNeeded !== true ||
      contract.parameterValidation?.exactModelCoreSubmissionDoesNotRequireCatalogPreflight !== true ||
      contract.parameterValidation?.onDemandLiveCatalogRead !== "only_for_explicit_discovery_installed_profile_gap_or_post_rejection_diagnosis" ||
      contract.parameterValidation?.allOptionalParametersRequire !== "selected_profile_with_model_index_only_for_selection_parameter_schema_or_on_demand_live_input_schema" ||
      contract.parameterValidation?.enforcesDeclaredRequiresTogether !== true ||
      contract.parameterValidation?.sizeExpressionPrecedence !== "send_only_the_highest_precedence_user_supplied_declared_size_expression") {
      errors.push(`${label} must use its installed selection for normal image submissions and limit live catalog reads to on-demand cases`);
    }
    const mediaInput = contract.inputMediaValidation;
    if (!mediaInput || mediaInput.nativeAttachmentRequires !== "selected_profile_with_model_index_only_for_selection_or_on_demand_live_profile_image_edit_operation_for_current_visual_reference_or_explicit_edit" ||
      mediaInput.imageEditOperation !== "image_edit" ||
      mediaInput.nativeVisualReferenceUsesImageEditOperation !== true ||
      mediaInput.nativeVisualReferencePromptSemantics !== "preserve_reference_intent_while_using_the_declared_image_edit_multipart_transport" ||
      mediaInput.nativeReferenceAttachmentRequires !== "selected_profile_with_model_index_only_for_selection_or_on_demand_profile_image_edit_operation_with_multipart_file_transport" ||
      mediaInput.untransportableNativeReferenceMustNeverBeConvertedToTextPrompt !== true ||
      mediaInput.whenNativeReferenceTransportUnavailable !== "stop_before_post_when_no_declared_image_edit_multipart_operation_or_no_native_byte_delivery_and_require_public_https_url_or_explicit_new_text_only_request" ||
      mediaInput.requestScope !== "current_user_request_only_no_unrelated_skill_history_workspace_search_or_quality_review" ||
      contract.result?.afterConfirmedDelivery !== "return_one_completion_receipt_and_end_turn_without_quality_review_history_search_or_new_media_submission") {
      errors.push(`${label} must fail closed for untransportable native reference attachments and end after confirmed delivery`);
    }
    validateTaskIdentityStateAndSubmissionFailure(errors, label, contract, true);
    validateImageRetrieval(errors, label, contract);
    validatePolling(errors, `${label} polling`, contract.polling, { deadlineSeconds: 120, fallbackDelaysSeconds: [0, 3], steadyDelaySeconds: 3, maxAutomaticStatusReads: 40, initialDelaySeconds: 5, initialDelayAnchor: "receipt_of_accepted_pending_submission_only_when_no_api_retry_after", initialDelayPersistence: "existing_retry_not_before_field_in_receipt_request_and_optional_task_record", initialDelayOnContinuation: "wait_only_remaining_retry_not_before_or_read_immediately_if_absent_or_expired" });
    if (contract.result?.successRequires !== "native_image_bytes") errors.push(`${label} must require native image bytes for success`);
  } else {
    validateRequest(errors, `${label} catalog`, operations.catalog, "GET", `${directApiOrigin}/v1/media/models`);
    validateRequest(errors, `${label} submit`, operations.submit, "POST", `${directApiOrigin}/v1/videos`);
    validateRequest(errors, `${label} edit`, operations.edit, "POST", `${directApiOrigin}/v1/videos/edits`);
    validateRequest(errors, `${label} status`, operations.status, "GET", `${directApiOrigin}/v1/videos/{task_id}`, true);
    validateRequest(errors, `${label} content`, operations.content, "GET", `${directApiOrigin}/v1/videos/{task_id}/content`, true);
    validateRequiredBodyFields(errors, `${label} submit`, operations.submit, ["model", "prompt"]);
    if (contract.parameterValidation?.normalSubmissionUsesSelectedProfileAndIndexWhenNeeded !== true ||
      contract.parameterValidation?.exactModelCoreSubmissionDoesNotRequireCatalogPreflight !== true ||
      contract.parameterValidation?.onDemandLiveCatalogRead !== "only_for_explicit_discovery_installed_profile_gap_or_post_rejection_diagnosis" ||
      contract.parameterValidation?.allModelsOptionalParametersRequire !== "selected_profile_with_model_index_only_for_selection_parameter_schema_or_on_demand_live_input_schema" ||
      contract.parameterValidation?.promptRequired !== true ||
      contract.parameterValidation?.resolutionUsesInstalledOrOnDemandModeConstraint !== true || contract.result?.successRequires !== "native_video_bytes") {
      errors.push(`${label} must use its installed selection for normal video submissions, limit live catalog reads to on-demand cases, and require native video bytes`);
    }
    const mediaInput = contract.inputMediaValidation;
    if (!mediaInput || mediaInput.imageToVideoOperation !== "image_to_video" ||
      mediaInput.referenceImageVideoOperation !== "reference_image_video" ||
      mediaInput.referenceVideoOperation !== "reference_video" ||
      mediaInput.referenceAudioOperation !== "reference_audio" ||
      mediaInput.videoEditOperation !== "video_edit" ||
      mediaInput.explicitFirstFrameIntentUses !== "image_to_video" ||
      mediaInput.declaredGenerateAudioIntent !== "map_explicit_generated_sound_to_true_and_silence_to_false_only_for_declared_boolean_generate_audio" ||
      mediaInput.nativeAttachmentRouteMustMatchDeclaredOperation !== true ||
      mediaInput.nativeAttachmentMustNeverFallBackToJsonTextSubmission !== true ||
      mediaInput.nativeAttachmentBodyMode !== "executor_multipart_only" ||
      mediaInput.whenCurrentAttachmentPathUnavailable !== "stop_before_post_and_explain_declared_multipart_requirement") {
      errors.push(`${label} must lock native attachments to their declared multipart operation and stop before POST when a current attachment path is unavailable`);
    }
    validateTaskIdentityStateAndSubmissionFailure(errors, label, contract, false);
    const retrieval = contract.contentRetrieval;
    if (!retrieval || retrieval.fetchOnlyAfterTerminalSuccess !== true || retrieval.oneInFlightContentReadPerTask !== true || retrieval.neverAutomaticallyPrefetchOrRefetchDeliveredContent !== true || retrieval.handoffBeforeNextContentRead !== true) {
      errors.push(`${label} must retrieve video content only after terminal success, one response at a time, without prefetching or duplicate delivery`);
    }
    validatePolling(errors, `${label} polling`, contract.polling, { deadlineSeconds: 300, fallbackDelaysSeconds: [5, 10, 20, 40, 60], steadyDelaySeconds: 60, maxAutomaticStatusReads: 7 });
  }
  validateContinuationIntegrity(errors, label, contract);
  if (contract.result?.sameTaskOnly !== true || contract.result?.neverAutoResubmit !== true || !Array.isArray(contract.unsupportedInput) || !contract.unsupportedInput.length) {
    errors.push(`${label} must stay on the same task without automatic resubmission and declare unsupported input`);
  }
}

function validateContinuationIntegrity(errors, label, contract) {
  const record = contract.continuationRecord;
  if (!record || record.validatesBeforeCreation !== true || record.reuseRequiresMatchingProof !== true ||
      record.unknownSubmissionNeverResubmits !== true ||
      !record.preserves?.includes("last_failure") || !record.preserves?.includes("pending_redelivery_index") ||
      record.failureClassification !== "reviewed_terminal_failure_only_no_raw_errors_legacy_reason_unknown" ||
      record.redelivery !== "explicit_same_task_index_local_proof_first_never_submit_ack_after_new_handoff" ||
      !sameArray(record.downloadProofFields, ["sha256", "bytes", "media_type"]) ||
      !sameArray(record.commands, ["status", "wait", "resume", "content", "redeliver", "delivered"]) ||
      !record.preserves?.includes("original_operation") || !record.preserves?.includes("download_proofs") ||
      record.legacyRecordWithoutProof !== "continue_same_task_but_download_into_another_directory_before_handoff" ||
      contract.contentRetrieval?.existingOutputReuse !== "explicit_record_matching_sha256_bytes_media_type_only_otherwise_preserve_and_choose_another_directory" ||
      contract.inputMediaValidation?.attachmentIntegrity !== "verify_file_identity_size_and_digest_before_and_during_multipart_no_hidden_copy") {
    errors.push(`${label} must preserve validated continuation metadata and require explicit download integrity proofs`);
  }
}

function validateUpdateContract(errors, label, contract) {
  const transport = contract.transport;
  if (!transport || transport.localSkillManager !== true || transport.usesVerifiedStableRelease !== true ||
    transport.doesNotFallBackToSourceArchive !== true ||
    transport.requiresPinnedOfficialSourceRevision !== true || transport.doesNotUseCustomInstallPayload !== true ||
    transport.doesNotReadCredentialsOrHostConfiguration !== true || transport.doesNotSubmitMediaOrUseMcp !== true) {
    errors.push(`${label} must use the local official Skill manager without reading credentials, submitting media, or using MCP`);
  }
  const sync = contract.operations?.sync;
  if (!sync || sync.directorySelector !== "download_checksum_verified_selector_from_the_same_stable_release" || sync.commandTemplate !== "native_fetch install_or_update --host <current-supported-host>" ||
    sync.sourceRepository !== "https://github.com/puretokensx/puretokens-skill.git" ||
    sync.releaseManifest !== "https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json" ||
    sync.validationCommand !== "pin_stable_manifest_verify_selector_and_platform_package_then_native_sync" ||
    sync.sameVersion !== "verify_release_executor_checksum_and_all_managed_inventories_then_no_download_sync_or_init") {
    errors.push(`${label} must define stable platform-only sync and the verified same-version shortcut`);
  }
  const selection = sync?.hostSelection;
  if (!selection || selection.whenNoHostOrTarget !== "bounded_direct_launcher_detection" ||
    selection.maxProcessQueries !== 8 || !sameArray(selection.readProcessFields, ["name", "parentId"]) ||
    !sameArray(selection.automaticHosts, ["codex", "opencode", "grok-build", "qoder"]) ||
    selection.unknownLauncher !== "stop_before_release_download_or_installation_writes_then_confirm_missing_client" ||
    selection.explicitHostOrTarget !== "preserve_without_detection" ||
    selection.doesNotReadConfigurationOrCommandLines !== true ||
    selection.doesNotIdentifyChatModelOrVerifyAPI !== true ||
    selection.diagnosticCommand !== "native_fetch detect-host") {
    errors.push(`${label} must bound launcher-only detection, preserve explicit choices and stop before download on uncertain host evidence`);
  }
  const target = sync?.targetSelection;
  if (!target || target.default !== "verified_release_selector" ||
    !sameArray(target.workspaceTargetRequiredFor, ["octop"]) ||
    target.missingWorkspaceTarget !== "stop_before_release_download_or_installation_writes_with_confirm_workspace_target" ||
    target.confirmedCustomTarget !== "preserve") {
    errors.push(`${label} must resolve installation targets in the installer and request a missing workspace before downloads`);
  }
  const init = contract.operations?.init;
  if (!init || init.commandTemplate !== "native_executor init --host <current-supported-host>" ||
    init.checks !== "public_identity_then_one_authenticated_catalog_check_without_displaying_configuration" ||
    init.totalDeadlineSeconds !== 20 || init.maxRequests !== 2 || init.usageGuide !== "references/usage-guide.md") {
    errors.push(`${label} must define the safe init identity check and usage guide`);
  }
  const result = contract.result;
  if (!result || result.installsMissingOfficialSkills !== true || result.upgradesOnlyManagedMatchingSkills !== true ||
    result.neverOverwritesUnmanagedDirectories !== true ||
    result.installsChecksumVerifiedNativeApiExecutor !== true ||
    result.reportsSynchronizedVersion !== true ||
    result.doesNotClaimSuccessWithoutVersionReceipt !== true ||
    result.neverModifiesOfficialCheckout !== true ||
    result.requiresNewHostConversationAfterSuccess !== true) {
    errors.push(`${label} must preserve the official checkout, install only the checksum-verified native executor, preserve unmanaged directories, report the synchronized version, and require a new host conversation`);
  }
}

function validateDirectTransport(errors, label, transport, requiresNativeMediaByteDelivery, fixedApiOrigin = directApiOrigin) {
  if (!transport || transport.fixedApiOrigin !== fixedApiOrigin || transport.usesFullApiUrls !== true ||
    transport.usesManagedNativeExecutor !== true || transport.usesCurrentHostConfiguredCredential !== true ||
    transport.credentialSourceNeverBecomesRequestTarget !== true ||
    transport.neverExposesCredentialsOrHostConfiguration !== true ||
    transport.doesNotUseMcpOrFallbackTransport !== true ||
    transport.doesNotUseNodeRuntime !== true ||
    transport.doesNotUseComputerUseOrUiAutomation !== true ||
    transport.doesNotInvokeOtherMediaSkillsAsApiFallback !== true ||
    transport.executionStart !== executorRequestStart ||
    transport.whenManagedNativeExecutorUnavailable !== executorRequestActualFailureOnly ||
    (requiresNativeMediaByteDelivery && transport.requiresNativeMediaByteDelivery !== true)) {
    errors.push(`${label} must use the fixed full API origin, current-host credential, managed native executor, and no Node, MCP, UI automation, Computer Use, or fallback transport`);
  }
}

function validateImageRetrieval(errors, label, contract) {
  const retrieval = contract.contentRetrieval;
  if (!retrieval || retrieval.indexBase !== 0 || retrieval.requestedCount !== "explicit_n_or_default_1" || retrieval.allowedIndexes !== "0..requestedCount-1" ||
    retrieval.completeDeliveryRequiresEveryRequestedIndex !== true || retrieval.partialOrMissingContent !== "report_delivered_and_missing_indexes_without_resubmission" ||
    retrieval.fetchOnlyAfterTerminalSuccess !== true || retrieval.sequentialByIndex !== true || retrieval.oneInFlightContentReadPerTask !== true ||
    retrieval.neverAutomaticallyPrefetchOrRefetchDeliveredContent !== true || retrieval.handoffBeforeNextContentRead !== true) {
    errors.push(`${label} must retrieve every zero-based requested image content index sequentially after terminal success without prefetching or duplicate delivery`);
  }
}

function validateTaskIdentityStateAndSubmissionFailure(errors, label, contract, image) {
  const identity = contract.taskIdentity;
  if (!identity || identity.receiptField !== "task_id" || identity.acceptedFormat !== "^[A-Za-z0-9_-][A-Za-z0-9._:-]{0,255}$(?![\\s\\S])" ||
    !sameArray(identity.topLevelResponseFields, ["task_id", "id"]) || identity.neverDerivesIdFromUrlsNestedObjectsOrPrompt !== true ||
    identity.pathEncoding !== "percent_encode_opaque_id_as_one_path_segment" || identity.whenMissing !== "report_task_id_not_returned_and_do_not_poll_download_or_resubmit") {
    errors.push(`${label} must normalize only a declared top-level task ID and fail closed when it is missing`);
  }
  const state = contract.taskState;
  if (!state ||
    state.reconciliationRequiredField !== "reconciliation_required" || state.reconciliationPrecedesLifecycle !== true ||
    state.whenReconciliationRequired !== "stop_ordinary_polling_retain_same_task_id_do_not_submit_replacement_or_infer_refund" ||
    !sameArray(state.pendingStates, ["pending", "queued", "processing", "running", "in_progress"]) ||
    !sameArray(state.successStates, ["completed", "succeeded", "success"]) ||
    !sameArray(state.failureStates, ["failed", "cancelled", "canceled", "expired", "error"]) ||
    state.whenUnrecognized !== "report_unknown_state_stop_automatic_polling_and_require_explicit_same_task_continuation") {
    errors.push(`${label} must use the executor task states and stop on an unrecognized state`);
  }
  const failure = contract.submissionFailure;
  if (!failure || failure.modelParameterOrCapabilityRejection !== "one_on_demand_catalog_read_then_require_explicit_corrected_new_request" ||
    failure.rateLimit !== "report_retry_after_when_returned_and_require_explicit_retry" ||
    failure.serverTransportOrTimeoutWithoutTaskId !== "report_submission_outcome_unknown_without_declaring_task_absent_or_resubmitting" ||
    failure.hostExecutionPolicyBlockedBeforeRequestStarts !== "report_validation_no_api_request_or_task_id_and_require_host_session_with_external_network_permission" ||
    failure.hostInvocationOutputUnknown !== "treat_submission_outcome_as_unknown_do_not_repeat_post_or_submit_replacement_task" ||
    failure.hostUnknownOutcomeResponse !== "one_unknown_receipt_then_only_local_request_cleanup_no_further_api_or_task_work" ||
    failure.noTaskIdLaterContinuation !== "require_explicit_confirmation_of_a_new_billable_request_before_one_new_post") {
    errors.push(`${label} must stop uncertain host submission output without automatic resubmission or continued work`);
  }
  validateFailureReceipt(errors, label, contract.failureReceipt);
  if (image && contract.contentRetrieval?.requestedCount !== "explicit_n_or_default_1") {
    errors.push(`${label} must preserve an explicit image count for same-task delivery`);
  }
}

function validateFailureReceipt(errors, label, receipt) {
  if (!receipt || !sameArray(receipt.requiredFields, failureReceiptRequiredFields) ||
    !sameArray(receipt.allowedFailurePhases, failureReceiptPhases) ||
    receipt.httpStatus !== "numeric_only_when_returned_otherwise_omit" ||
    receipt.apiErrorCode !== "recognized_public_category_code_only_when_explicitly_returned_otherwise_omit" ||
    receipt.errorMessage !== "fixed_public_category_message_or_fixed_local_explanation_never_arbitrary_server_text" ||
    receipt.retryAfterSeconds !== "include_when_a_valid_positive_retry_after_is_returned" ||
    !sameArray(receipt.forbiddenDisclosure, failureReceiptForbiddenDisclosure)) {
    errors.push(`${label} must use the safe structured failure receipt without exposing internal, upstream, credential, request, or user-media data`);
  }
}

function validatePolling(errors, label, polling, expected) {
  if (!polling || polling.serverDelay !== "honor_valid_positive_retry_after_before_fallback" || polling.serverDelayMustFitRemainingAutomaticDeadline !== true ||
    polling.afterHttp429 !== "honor_valid_positive_retry_after_then_continue_same_task_if_remaining_budget" ||
    !sameArray(polling.fallbackDelaysSeconds, expected.fallbackDelaysSeconds) || polling.steadyDelaySeconds !== expected.steadyDelaySeconds ||
    polling.automaticDeadlineSeconds !== expected.deadlineSeconds || polling.maxAutomaticStatusReads !== expected.maxAutomaticStatusReads ||
    polling.initialDelaySeconds !== expected.initialDelaySeconds || polling.initialDelayAnchor !== expected.initialDelayAnchor ||
    polling.initialDelayPersistence !== expected.initialDelayPersistence || polling.initialDelayOnContinuation !== expected.initialDelayOnContinuation ||
    polling.oneInFlightStatusReadPerTask !== true || polling.automaticPollingScope !== "submission_or_explicit_same_task_continuation_turn_only_no_background_timer_or_queue" ||
    polling.afterStatusReadError !== "stop_on_5xx_transport_or_timeout_and_require_explicit_same_task_continuation" ||
    polling.explicitContinuation !== "explicit_user_request_allows_another_bounded_window_without_resetting_automatic_budget" ||
    polling.afterDeadline !== "successful_pending_window_receipt_one_foreground_continuation_then_ask" || polling.maxForegroundWaitWindows !== 2) {
    errors.push(`${label} must use the bounded adaptive same-task polling contract without background work or overlapping reads`);
  }
}

function validateRequest(errors, label, request, method, expectedUrl, usesTemplate = false) {
  if (!request || request.method !== method || request[usesTemplate ? "urlTemplate" : "url"] !== expectedUrl) {
    errors.push(`${label} must be ${method} ${expectedUrl}`);
  }
}

function validateRequiredBodyFields(errors, label, request, required) {
  if (!Array.isArray(request?.requiredBodyFields) || required.some((field) => !request.requiredBodyFields.includes(field))) {
    errors.push(`${label} must require ${required.join(", ")}`);
  }
}

function validateBehaviorScenarios(errors, directory, scenarios) {
  const label = `${directory}: behaviorScenarios`;
  if (!scenarios || typeof scenarios !== "object") return;
  if (scenarios.$schema !== "https://puretokensx.com/schemas/media-behavior-scenarios.schema.json") errors.push(`${label} must declare the behavior-scenarios schema`);
  if (scenarios.schemaVersion !== 1 || scenarios.kind !== skillKind(directory) || !Array.isArray(scenarios.scenarios) || !scenarios.scenarios.length) {
    errors.push(`${label} must be a non-empty schemaVersion=1 scenario list for this Skill`);
    return;
  }
  const ids = new Set();
  for (const scenario of scenarios.scenarios) {
    if (!scenario || !/^[a-z][a-z0-9-]*$/.test(scenario.id || "") || typeof scenario.when !== "string" || !scenario.when || typeof scenario.then !== "string" || !scenario.then) {
      errors.push(`${label} scenarios must have a kebab-case id plus non-empty when and then`);
      continue;
    }
    if (ids.has(scenario.id)) errors.push(`${label} duplicate scenario ${scenario.id}`);
    ids.add(scenario.id);
  }
}

async function validateModelIndex(errors, directory, skillDir, index) {
  const label = `${directory}: modelIndex`;
  if (!index || typeof index !== "object") return;
  const capability = skillKind(directory);
  const expectedDefaultModel = mediaDefaultModels[capability];
  if (index.$schema !== "https://puretokensx.com/schemas/model-index.schema.json") errors.push(`${label} must declare the model-index schema`);
  if (index.schemaVersion !== 1 || index.capability !== capability || typeof index.catalogUpdatedAt !== "string" || typeof index.catalogCapturedAt !== "string" ||
    typeof index.defaultModel !== "string" || !index.defaultModel || !Array.isArray(index.models) || !index.models.length) {
    errors.push(`${label} must be a non-empty schemaVersion=1 ${capability} index`);
    return;
  }
  const ids = new Set();
  const profilePaths = new Set();
  for (const model of index.models) {
    if (!model || !validModelID(model.id) || !Array.isArray(model.aliases) || model.aliases.some((alias) => typeof alias !== "string" || !alias) ||
      typeof model.profile !== "string" || model.profile !== `profiles/${model.id}.json`) {
      errors.push(`${label} models must have a non-empty id, string aliases, and their canonical profile path`);
      continue;
    }
    if (ids.has(model.id)) errors.push(`${label} duplicate model id ${model.id}`);
    ids.add(model.id);
    if (new Set(model.aliases).size !== model.aliases.length) errors.push(`${label} duplicate aliases for ${model.id}`);
    profilePaths.add(model.profile);
    const profile = await readSkillReference(errors, skillDir, `references/${model.profile}`, `${label} profile ${model.id}`);
    if (!profile || profile.$schema !== "https://puretokensx.com/schemas/model-profile.schema.json" || profile.schemaVersion !== 1 ||
      profile.capability !== capability || profile.id !== model.id || profile.catalogUpdatedAt !== index.catalogUpdatedAt ||
      profile.catalogCapturedAt !== index.catalogCapturedAt || !profile.parameterSchema || typeof profile.parameterSchema !== "object" || Array.isArray(profile.parameterSchema)) {
      errors.push(`${label} profile ${model.id} must match the index and declare a parameterSchema object`);
    }
    if ((model.supportScope === "reviewed_source") !== (profile?.sourceReview?.scope === "reviewed_source_not_live_availability")) {
      errors.push(`${label} source-reviewed profiles must preserve their explicit non-live scope in both index and profile`);
    }
  }
  if (!ids.has(index.defaultModel)) errors.push(`${label} defaultModel must be in models`);
  if (index.defaultModel !== expectedDefaultModel) errors.push(`${label} defaultModel must remain ${expectedDefaultModel}`);
  try {
    const entries = await readdir(path.join(skillDir, "references", "profiles"), { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isFile() && entry.name.endsWith(".json") && !profilePaths.has(`profiles/${entry.name}`)) {
        errors.push(`${label} has an unindexed profile profiles/${entry.name}`);
      }
    }
  } catch {
    errors.push(`${label} profiles directory is missing or unreadable`);
  }
}

function validateTaskReceipt(errors, directory, receipt) {
  const label = `${directory}: taskReceipt`;
  if (!receipt || typeof receipt !== "object") return;
  const kind = skillKind(directory);
  if (receipt.$schema !== "https://puretokensx.com/schemas/task-receipt.schema.json" || receipt.schemaVersion !== 1 || receipt.kind !== kind) {
    errors.push(`${label} must be a schemaVersion=1 ${kind} task receipt`);
    return;
  }
  for (const phase of ["submission", "continuation", "reconciliation", "completion", "failure"]) {
    const fields = receipt[phase]?.requiredFields;
    if (typeof receipt.machineReceipt !== "string" || !Array.isArray(fields) || receiptPhaseFields[phase].some((field) => !fields.includes(field)) || typeof receipt[phase]?.nextAction !== "string" || !receipt[phase].nextAction) {
      errors.push(`${label} ${phase} must include only the relevant concise user receipt fields`);
    }
    if (phase === "completion" && !fields?.includes("delivered_count")) {
      errors.push(`${label} completion must include delivered_count`);
    }
    if (phase === "reconciliation" && !fields?.includes("reconciliation_required")) {
      errors.push(`${label} reconciliation must include reconciliation_required`);
    }
  }
}

function skillKind(directory) {
  return directory.replace("puretokens-", "");
}

function sameArray(actual, expected) {
  return Array.isArray(actual) && actual.length === expected.length && actual.every((value, index) => value === expected[index]);
}

async function validateSchemaDocuments(errors) {
  const schemas = [
    "schemas/media-execution-contract.schema.json",
    "schemas/media-behavior-scenarios.schema.json",
    "schemas/model-index.schema.json",
    "schemas/model-profile.schema.json",
    "schemas/host-support.schema.json",
    "schemas/host-delivery.schema.json",
    "schemas/installation-inspection.schema.json",
    "schemas/direct-api-execution-contract.schema.json",
    "schemas/catalog-freshness.schema.json",
    "schemas/balance-snapshot.schema.json",
    "schemas/balance-receipt.schema.json",
    "schemas/task-receipt.schema.json",
    "schemas/executor-request.schema.json",
    "schemas/executor-receipt.schema.json",
    "schemas/support-summary.schema.json",
    "schemas/timings.schema.json",
    "schemas/model-query.schema.json",
    "schemas/model-query-receipt.schema.json",
    "schemas/task-record.schema.json",
    "schemas/init-receipt.schema.json",
    "schemas/doctor-receipt.schema.json",
    "schemas/evaluation-request.schema.json",
    "schemas/evaluation-receipt.schema.json",
    "schemas/audio-request.schema.json",
    "schemas/audio-receipt.schema.json",
    "schemas/audio-artifact-verify-request.schema.json",
    "schemas/embedding-request.schema.json",
    "schemas/rerank-request.schema.json",
    "schemas/retrieval-receipt.schema.json",
    "schemas/embedding-output.schema.json",
    "schemas/rerank-output.schema.json"
  ];
  for (const schema of schemas) await verifyJsonFile(errors, schema, `schema ${schema}`);
}

async function validateInstallerSources(errors) {
  const runtimeDirectory = path.join(repositoryRoot, "runtime");
  const shellInstallerPath = path.join(runtimeDirectory, "puretokens-skill-install.sh");
  const powerShellInstallerPath = path.join(runtimeDirectory, "puretokens-skill-install.ps1");
  const executorManifestPath = path.join(runtimeDirectory, "executor", "manifest.json");
  try {
    if (!(await stat(shellInstallerPath)).isFile()) errors.push("runtime/puretokens-skill-install.sh is required");
    if (!(await stat(powerShellInstallerPath)).isFile()) errors.push("runtime/puretokens-skill-install.ps1 is required");
    for (const extension of ["sh", "ps1"]) {
      if (!(await stat(path.join(runtimeDirectory, `puretokens-skill-fetch.${extension}`))).isFile()) errors.push(`runtime/puretokens-skill-fetch.${extension} is required`);
    }
    const executorManifest = JSON.parse(await readFile(executorManifestPath, "utf8"));
    if (executorManifest?.schemaVersion !== 1 || executorManifest?.name !== "puretokens-api-executor" || !executorManifest?.artifacts || typeof executorManifest.artifacts !== "object") {
      errors.push("runtime/executor/manifest.json must define the managed native executor artifacts");
      return;
    }
    for (const [platform, artifact] of Object.entries(executorManifest.artifacts)) {
      const relativePath = artifact?.path;
      const expectedHash = artifact?.sha256;
      if (!/^(darwin|linux|windows)-(amd64|arm64)$/.test(platform) || typeof relativePath !== "string" || !/^[a-z0-9./-]+(?:\.exe)?$/.test(relativePath) || !/^[a-f0-9]{64}$/.test(expectedHash || "")) {
        errors.push(`runtime/executor/manifest.json has an invalid ${platform || "unnamed"} artifact`);
        continue;
      }
      const file = path.resolve(runtimeDirectory, "executor", relativePath);
      if (!file.startsWith(`${path.join(runtimeDirectory, "executor")}${path.sep}`) || !(await stat(file)).isFile()) {
        errors.push(`runtime/executor/manifest.json is missing ${platform} artifact`);
        continue;
      }
      const actualHash = createHash("sha256").update(await readFile(file)).digest("hex");
      if (actualHash !== expectedHash) errors.push(`runtime/executor/manifest.json checksum mismatch for ${platform}`);
    }
  } catch {
    errors.push("native Pure Tokens Skill installer sources are missing or invalid");
  }
}

function readFrontmatter(text) {
  const match = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/);
  if (!match) return {};
  const result = {};
  for (const line of match[1].split(/\r?\n/)) {
    const separator = line.indexOf(":");
    if (separator <= 0) continue;
    result[line.slice(0, separator).trim()] = line.slice(separator + 1).trim();
  }
  return result;
}

async function verifyFile(errors, relativePath) {
  const absolutePath = path.resolve(repositoryRoot, relativePath);
  if (!absolutePath.startsWith(`${repositoryRoot}${path.sep}`)) {
    errors.push(`skills/index.json: path escapes repository: ${relativePath}`);
    return;
  }
  try {
    if (!(await stat(absolutePath)).isFile()) errors.push(`skills/index.json: missing file ${relativePath}`);
  } catch {
    errors.push(`skills/index.json: missing file ${relativePath}`);
  }
}

async function verifySkillFile(errors, skillDir, relativePath, label) {
  const absolutePath = path.resolve(skillDir, relativePath);
  if (!absolutePath.startsWith(`${skillDir}${path.sep}`)) {
    errors.push(`${label}: path escapes Skill directory`);
    return undefined;
  }
  try {
    if (!(await stat(absolutePath)).isFile()) {
      errors.push(`${label}: missing file ${relativePath}`);
      return undefined;
    }
    return absolutePath;
  } catch {
    errors.push(`${label}: missing file ${relativePath}`);
    return undefined;
  }
}

async function readSkillReference(errors, skillDir, relativePath, label) {
  const absolutePath = await verifySkillFile(errors, skillDir, relativePath, label);
  if (!absolutePath) return undefined;
  try {
    return JSON.parse(await readFile(absolutePath, "utf8"));
  } catch {
    errors.push(`${label}: must be valid JSON`);
    return undefined;
  }
}

async function verifyJsonFile(errors, relativePath, label) {
  const absolutePath = path.resolve(repositoryRoot, relativePath);
  try {
    JSON.parse(await readFile(absolutePath, "utf8"));
  } catch {
    errors.push(`${label}: missing or invalid JSON`);
  }
}
