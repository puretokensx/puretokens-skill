import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
export const acceptanceGroups = {
  media: ["installation", "authenticatedAPI", "image-generation", "uploaded-image-edit", "generated-image-edit", "multiple-images", "video-generation", "sameTaskResume", "nativeAttachmentDelivery"],
  audio: ["speech-generation", "audio-transcription", "sound-generation", "sync-music-generation", "voice-change", "audio-reattachment"],
  music: ["music-generation", "music-resume"],
  evaluation: ["jev-evaluation"],
  embedding: ["text-embedding"],
  rerank: ["text-rerank"],
  workflows: ["image-to-video-workflow", "video-audio-workflow"],
  help: ["help-no-api"],
};
export const acceptanceCases = Object.values(acceptanceGroups).flat();
export const acceptancePlatforms = ["macOS", "windows", "linux"];
export const acceptanceExecutionModes = ["local", "wsl", "remote", "sandbox"];
const summaryCases = ["installation", "authenticatedAPI", "sameTaskResume", "nativeAttachmentDelivery"];
const states = new Set(["passed", "failed", "pending", "unavailable"]);

export function validStablePublicationException(document) {
  const exception = document?.stablePublicationException;
  return exception?.kind === "maintainer-approved-pending-acceptance" &&
    exception.version === document?.version &&
    exception.approved === true &&
    exception.approvedAt === document?.checkedAt &&
    typeof exception.reason === "string" &&
    exception.reason.trim().length >= 20;
}

export function hostAcceptanceLevel(evidence) {
  if (evidence.cases?.installation !== "passed") return "unverified";
  if (evidence.cases?.authenticatedAPI !== "passed") return "installation-verified";
  if (acceptanceGroups.media.every(id => evidence.cases?.[id] === "passed")) return "media-verified";
  return "api-verified";
}

// These are per-environment results, not a union of different hosts or releases.
export function hostCapabilityAcceptance(evidence) {
  return Object.fromEntries(Object.entries(acceptanceGroups).map(([group, cases]) => {
    const results = cases.map(id => evidence.cases?.[id]);
    const prerequisites = evidence.cases?.installation === "passed" &&
      (group === "help" || evidence.cases?.authenticatedAPI === "passed");
    const status = results.every(state => state === "unavailable") ? "unavailable" :
      results.includes("failed") ? "failed" :
      prerequisites && results.every(state => state === "passed") ? "passed" : "pending";
    return [group, status];
  }));
}

export function validateAcceptanceMeasurements(measurements, cases) {
  const errors = [];
  if (!Array.isArray(measurements) || measurements.length > acceptanceCases.length) {
    return ["Acceptance measurements must be a bounded list of client-observed cases."];
  }
  const seen = new Set();
  for (const measurement of measurements) {
    if (!measurement || typeof measurement !== "object" ||
        Object.keys(measurement).some(key => !["case", "basis", "durationsMs", "requests"].includes(key)) ||
        !acceptanceCases.includes(measurement.case) || seen.has(measurement.case) ||
        measurement.basis !== "client-observed" || !["passed", "failed"].includes(cases?.[measurement.case])) {
      errors.push("Measurements need one observed passed/failed case each; no private data or fixture timings.");
      continue;
    }
    seen.add(measurement.case);
    const durations = measurement.durationsMs;
    if (!durations || typeof durations !== "object" || Array.isArray(durations) ||
        !Object.keys(durations).length || Object.keys(durations).some(key =>
          !["installation", "init", "preparation", "request", "wait", "download", "handoff", "total"].includes(key)) ||
        Object.values(durations).some(value => !Number.isSafeInteger(value) || value < 0)) {
      errors.push("Measured durations must be nonnegative integer milliseconds with known phase names.");
    }
    const requests = measurement.requests;
    if (requests !== undefined && (!requests || typeof requests !== "object" || Array.isArray(requests) ||
        !Object.keys(requests).length || Object.keys(requests).some(key => !["submit", "status", "content", "other"].includes(key)) ||
        Object.values(requests).some(value => !Number.isSafeInteger(value) || value < 0))) {
      errors.push("Measured request counts must use known counters; omit unobserved counts.");
    }
    if (measurement.case === "help-no-api" && cases?.[measurement.case] === "passed" &&
        (["submit", "status", "content", "other"].some(key => requests?.[key] !== 0))) {
      errors.push("A passed local-help measurement needs observed zero Skill API requests in every counter.");
    }
    if (cases?.[measurement.case] !== "passed") continue;
    if (measurement.case === "audio-reattachment" &&
        ["submit", "status", "content", "other"].some(key => requests?.[key] !== 0)) {
      errors.push("Audio reattachment is local-only; a passed measurement needs observed zero API requests.");
    }
    if (["sameTaskResume", "music-resume"].includes(measurement.case) && requests?.submit !== undefined && requests.submit !== 0) {
      errors.push("Same-task recovery cannot include a new submission.");
    }
    const synchronous = ["speech-generation", "audio-transcription", "sound-generation", "sync-music-generation", "voice-change", "jev-evaluation", "text-embedding", "text-rerank"].includes(measurement.case);
    if (synchronous && (["status", "content"].some(key => requests?.[key] !== undefined && requests[key] !== 0) ||
        ["wait", "download"].some(key => Object.hasOwn(durations ?? {}, key)))) {
      errors.push("Synchronous cases have one request interval, not separate task polling or content download.");
    }
    if (synchronous && requests?.submit !== undefined && requests.submit !== 1) {
      errors.push("A successful synchronous case requires exactly one submit attempt.");
    }
  }
  return errors;
}

export function validateHostAcceptance(document, version, manifest, supportedHosts) {
  const errors = [];
  if (document?.schemaVersion !== 1 || document.version !== version || !Array.isArray(document.hosts) ||
      !Array.isArray(document.evidence) || !/^\d{4}-\d{2}-\d{2}$/.test(document.checkedAt ?? "")) {
    return ["Host acceptance must identify the release/date, host matrix and real-host evidence array."];
  }
  if (!Array.isArray(supportedHosts) || supportedHosts.some(host =>
    typeof host?.id !== "string" || !/^[a-z][a-z0-9-]*$/.test(host.id) ||
    !["fixture-tested", "pending"].includes(host.credentialAdapter)) ||
    new Set(supportedHosts.map(host => host.id)).size !== supportedHosts.length) {
    return ["Host acceptance requires unique host support metadata with a known credential adapter state."];
  }
  if (document.stablePublicationException !== undefined && !validStablePublicationException(document)) {
    errors.push("Stable publication exception must be explicit, version-bound and dated with the acceptance record.");
  }
  const supportById = new Map(supportedHosts.map(host => [host.id, host]));
  const hostIds = [...supportById.keys()];
  const ids = document.hosts.map(host => host.host);
  if (new Set(ids).size !== ids.length || ids.length !== hostIds.length || hostIds.some(id => !ids.includes(id))) {
    errors.push("Host acceptance must cover each supported host exactly once.");
  }
  for (const evidence of document.evidence) {
    const artifact = manifest.artifacts?.[evidence.executorPlatform];
    if (!ids.includes(evidence.host) || !acceptancePlatforms.includes(evidence.os) ||
        evidence.method !== "real-host" || evidence.version !== version ||
        !artifact || evidence.executorSha256 !== artifact.sha256 ||
        !["hostVersion", "osVersion", "shellVersion", "artifact"].every(field => typeof evidence[field] === "string" && evidence[field].trim()) ||
        !["amd64", "arm64"].includes(evidence.osArchitecture) ||
        !acceptanceExecutionModes.includes(evidence.executionMode) ||
        !/^\d{4}-\d{2}-\d{2}$/.test(evidence.checkedAt ?? "") ||
        !evidence.cases || Object.keys(evidence.cases).some(id => !acceptanceCases.includes(id)) ||
        acceptanceCases.some(id => !states.has(evidence.cases[id]))) {
      errors.push("Real-host evidence requires release bytes, host/OS/shell versions, OS architecture, execution mode, date, artifact and every case outcome.");
    }
    if ((evidence.os === "macOS" && !evidence.executorPlatform?.startsWith("darwin-")) ||
        (evidence.os === "windows" && !evidence.executorPlatform?.startsWith("windows-")) ||
        (evidence.os === "linux" && !evidence.executorPlatform?.startsWith("linux-"))) {
      errors.push("Real-host evidence executor platform must match its OS.");
    }
    if (evidence.executionMode === "wsl" && evidence.os !== "linux") {
      errors.push("WSL evidence must identify its Linux execution environment.");
    }
    if (evidence.measurements !== undefined) {
      errors.push(...validateAcceptanceMeasurements(evidence.measurements, evidence.cases));
    }
    if (supportById.get(evidence.host)?.credentialAdapter === "pending") {
      for (const check of acceptanceCases.filter(id => !["installation", "help-no-api"].includes(id))) {
        if (evidence.cases?.[check] !== "unavailable") {
          errors.push(`${evidence.host}/${evidence.os}/${check} evidence must be unavailable: this release has no credential adapter.`);
        }
      }
    }
  }
  for (const host of document.hosts) {
    const support = supportById.get(host.host);
    if (support && host.credentialFixtures !== (support.credentialAdapter === "fixture-tested" ? "passed" : "unavailable")) {
      errors.push(`${host.host}/credentialFixtures must match its registered credential adapter state; fixtures are not real-host evidence.`);
    }
  }
  for (const host of document.hosts) for (const os of acceptancePlatforms) for (const check of summaryCases) {
    const state = host[os]?.[check];
    if (supportById.get(host.host)?.credentialAdapter === "pending" && check !== "installation" && state !== "unavailable") {
      errors.push(`${host.host}/${os}/${check} must be unavailable: this release has no credential adapter.`);
    }
    if (state === "unavailable" && !host[os]?.reason?.trim()) errors.push(`${host.host}/${os} must explain why this release cannot accept that target.`);
    if (!states.has(state)) errors.push(`Invalid ${host.host}/${os}/${check} acceptance state.`);
    if (state === "passed" && !document.evidence.some(item => item.method === "real-host" && item.host === host.host && item.os === os && item.executionMode === "local" && item.cases?.[check] === "passed")) {
      errors.push(`${host.host}/${os}/${check} has no matching local real-host evidence.`);
    }
  }
  return errors;
}

export function validateStableHostAcceptance(document) {
  const errors = [];
  if (validStablePublicationException(document)) return errors;
  let mediaEnvironments = 0;
  for (const host of document.hosts ?? []) for (const os of acceptancePlatforms) {
    const target = host[os] ?? {};
    if (summaryCases.some(id => !["passed", "unavailable"].includes(target[id]))) {
      errors.push(`${host.host}/${os} has pending or failed real-host acceptance.`);
    }
    if (target.authenticatedAPI === "passed") {
      const complete = document.evidence?.some(item => item.method === "real-host" && item.host === host.host && item.os === os &&
        item.executionMode === "local" && acceptanceCases.every(id => item.cases?.[id] === "passed"));
      if (complete) mediaEnvironments++;
      else errors.push(`${host.host}/${os} has no complete local media acceptance including audio, music, evaluation, embedding, rerank, workflows and local help.`);
    }
  }
  if (mediaEnvironments === 0) errors.push("Stable publication requires at least one complete real-host media environment.");
  return errors;
}

export async function checkHostAcceptance({stable = false} = {}) {
  const read = async file => JSON.parse(await readFile(path.join(root, file), "utf8"));
  const [document, pkg, manifest, support] = await Promise.all([
    read("references/host-acceptance.json"), read("package.json"),
    read("runtime/executor/manifest.json"), read("references/host-support.json"),
  ]);
  const errors = validateHostAcceptance(document, pkg.version, manifest, support.supported);
  if (stable) errors.push(...validateStableHostAcceptance(document));
  for (const evidence of document.evidence ?? []) {
    if (typeof evidence.artifact !== "string") continue;
    const artifact = path.resolve(root, evidence.artifact);
    if (!artifact.startsWith(root + path.sep) || !(await stat(artifact).catch(() => null))?.isFile()) {
      errors.push("Acceptance evidence must reference an existing file inside this repository.");
    }
  }
  if (errors.length) throw new Error(errors.join("\n"));
  const pending = document.hosts.reduce((count, host) => count + acceptancePlatforms.reduce((n, os) => n + summaryCases.filter(check => host[os][check] === "pending").length, 0), 0);
  const details = (document.evidence ?? []).flatMap(item => Object.values(item.cases ?? {}));
  console.log(`Host acceptance record is consistent; ${pending} host/OS summary checks remain pending. Detailed real-host case results recorded: ${details.length} (${details.filter(x => x === "passed").length} passed); ${acceptanceCases.length} distinct cases are required per accepted host/OS.`);
  const levels = ["installation-verified", "api-verified", "media-verified"];
  console.log("Evidence levels (each applies only to its recorded host/OS/architecture/mode): " +
    levels.map(level => `${level}=${document.evidence.filter(item => hostAcceptanceLevel(item) === level).length}`).join(", ") + ".");
  console.log("Capability groups passed in individual environments: " +
    Object.keys(acceptanceGroups).map(group => `${group}=${document.evidence.filter(item =>
      hostCapabilityAcceptance(item)[group] === "passed").length}`).join(", ") +
    ". media-verified alone does not verify audio, music, evaluation, embedding or rerank.");
  if (validStablePublicationException(document)) {
    console.log("Stable publication exception: maintainer-approved pending acceptance for this exact version; pending evidence remains unmodified.");
  }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await checkHostAcceptance({stable:process.argv.includes("--stable")});
