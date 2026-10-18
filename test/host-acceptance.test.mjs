import assert from "node:assert/strict";
import { test } from "node:test";
import { acceptanceCases, acceptanceGroups, acceptancePlatforms, hostAcceptanceLevel, hostCapabilityAcceptance, validateAcceptanceMeasurements, validStablePublicationException, validateHostAcceptance, validateStableHostAcceptance } from "../scripts/validate-host-acceptance.mjs";
import { createAcceptanceEvidence } from "../scripts/prepare-host-acceptance.mjs";
const version = "0.17.1";
const sha = "a".repeat(64);
const manifest = { artifacts: { "darwin-arm64": { sha256: sha } } };
const supportedHosts = [{ id: "codex", credentialAdapter: "fixture-tested" }];
const installationOnlyHosts = [{ id: "trae", credentialAdapter: "pending" }];
function fixture() {
  const checks = () => ({ installation: "pending", authenticatedAPI: "pending", sameTaskResume: "pending", nativeAttachmentDelivery: "pending" });
  return { schemaVersion: 1, version, checkedAt: "2026-09-06", hosts: [{ host: "codex", credentialFixtures: "passed", macOS: checks(), windows: checks(), linux: checks() }], evidence: [] };
}
function installationOnlyFixture() {
  const record = fixture();
  record.hosts[0].host = "trae";
  record.hosts[0].credentialFixtures = "unavailable";
  for (const os of acceptancePlatforms) {
    Object.assign(record.hosts[0][os], {
      authenticatedAPI: "unavailable", sameTaskResume: "unavailable", nativeAttachmentDelivery: "unavailable",
      reason: "This release has no credential adapter; API commands stop before a request.",
    });
  }
  return record;
}
test("fixture success cannot mark real host checks passed", () => {
  const record = fixture();
  assert.deepEqual(validateHostAcceptance(record, version, manifest, supportedHosts), []);
  record.hosts[0].macOS.installation = "passed";
  assert.ok(validateHostAcceptance(record, version, manifest, supportedHosts).length);
});
test("host evidence must identify actual versions and release bytes", () => {
  const record = fixture();
  record.hosts[0].macOS.installation = "passed";
  const evidence = { method: "real-host", host: "codex", os: "macOS", hostVersion: "fixture-host-version", osVersion: "fixture-os", osArchitecture: "arm64", executionMode: "local", shellVersion: "fixture-shell", executorPlatform: "darwin-arm64", executorSha256: sha, version, checkedAt: "2026-09-06", artifact: "references/fixture.md", cases: Object.fromEntries(acceptanceCases.map(id => [id, id === "installation" ? "passed" : "pending"])) };
  record.evidence.push(evidence);
  assert.deepEqual(validateHostAcceptance(record, version, manifest, supportedHosts), []);
  for (const [key, value] of [["method", "fixture"], ["version", "0.16.0"], ["executorSha256", "b".repeat(64)], ["hostVersion", ""], ["osArchitecture", ""], ["shellVersion", ""], ["executionMode", "unknown"], ["executionMode", "remote"]]) {
    const changed = structuredClone(record);
    changed.evidence[0][key] = value;
    assert.ok(validateHostAcceptance(changed, version, manifest, supportedHosts).length, key);
  }
});

test("Linux evidence requires Linux bytes and cannot masquerade as a different OS", () => {
 const record = fixture(); const linuxManifest = { artifacts: { "linux-arm64": { sha256: sha } } };
 record.hosts[0].linux.installation = "passed";
 record.evidence.push({ method:"real-host",host:"codex",os:"linux",hostVersion:"fixture",osVersion:"fixture",osArchitecture:"arm64",executionMode:"local",shellVersion:"fixture",executorPlatform:"linux-arm64",executorSha256:sha,version,checkedAt:"2026-09-08",artifact:"references/fixture.md",cases:Object.fromEntries(acceptanceCases.map(id => [id,id==="installation"?"passed":"pending"])) });
 assert.deepEqual(validateHostAcceptance(record,version,linuxManifest,supportedHosts),[]);
 record.evidence[0].os="macOS";
 assert.ok(validateHostAcceptance(record,version,linuxManifest,supportedHosts).length);
});

test("installation and API success never imply complete media delivery", () => {
  const evidence = {cases: Object.fromEntries(acceptanceCases.map(id => [id, "pending"]))};
  assert.equal(hostAcceptanceLevel(evidence), "unverified");
  evidence.cases.installation = "passed";
  assert.equal(hostAcceptanceLevel(evidence), "installation-verified");
  evidence.cases.authenticatedAPI = "passed";
  assert.equal(hostAcceptanceLevel(evidence), "api-verified");
  for (const id of acceptanceCases) evidence.cases[id] = "passed";
  assert.equal(hostAcceptanceLevel(evidence), "media-verified");
  evidence.cases["video-generation"] = "unavailable";
  assert.equal(hostAcceptanceLevel(evidence), "api-verified");
});

test("WSL evidence belongs to Linux and cannot accept a local host summary", () => {
  const record = fixture();
  const evidence = {
    method: "real-host", host: "codex", os: "linux", executionMode: "wsl",
    hostVersion: "fixture", osVersion: "fixture", osArchitecture: "amd64", shellVersion: "fixture",
    executorPlatform: "linux-amd64", executorSha256: sha, version, checkedAt: "2026-09-12",
    artifact: "references/fixture.md", cases: Object.fromEntries(acceptanceCases.map(id => [id, "passed"])),
  };
  record.evidence.push(evidence);
  const artifacts = Object.fromEntries(["linux-amd64", "windows-amd64", "darwin-amd64"].map(id => [id, {sha256: sha}]));
  assert.deepEqual(validateHostAcceptance(record, version, {artifacts}, supportedHosts), []);
  for (const [os, platform] of [["windows", "windows-amd64"], ["macOS", "darwin-amd64"]]) {
    evidence.os = os;
    evidence.executorPlatform = platform;
    assert.ok(validateHostAcceptance(record, version, {artifacts}, supportedHosts).some(error => error.includes("WSL")));
  }
  evidence.os = "linux";
  evidence.executorPlatform = "linux-amd64";
  record.hosts[0].linux.installation = "passed";
  assert.ok(validateHostAcceptance(record, version, {artifacts}, supportedHosts).some(error => error.includes("local real-host")));
});
test("unavailable host OS combinations require an explanation", () => {
 const record=fixture(); record.hosts[0].linux.installation="unavailable";
 assert.ok(validateHostAcceptance(record,version,manifest,supportedHosts).length);
 record.hosts[0].linux.reason="No verified client target for this release";
 assert.deepEqual(validateHostAcceptance(record,version,manifest,supportedHosts),[]);
});

test("hosts without adapters keep installation pending and API summary checks unavailable", () => {
  const record = installationOnlyFixture();
  assert.deepEqual(validateHostAcceptance(record, version, manifest, installationOnlyHosts), []);
  assert.ok(validateStableHostAcceptance(record).length, "classification does not accept installation or media");
  for (const os of acceptancePlatforms) for (const id of ["authenticatedAPI", "sameTaskResume", "nativeAttachmentDelivery"]) {
    for (const state of ["pending", "failed", "passed"]) {
      const changed = structuredClone(record);
      changed.hosts[0][os][id] = state;
      assert.ok(validateHostAcceptance(changed, version, manifest, installationOnlyHosts)
        .some(error => error.includes(`${os}/${id}`) && error.includes("no credential adapter")), `${os}/${id}/${state}`);
    }
  }
  delete record.hosts[0].macOS.reason;
  assert.ok(validateHostAcceptance(record, version, manifest, installationOnlyHosts).some(error => error.includes("must explain")));
});

test("support metadata and fixture classification cannot be missing, unknown or stale", () => {
  for (const support of [undefined, ["codex"], [{id:"codex"}], [{id:"codex",credentialAdapter:"unknown"}], [...supportedHosts, ...supportedHosts]]) {
    assert.ok(validateHostAcceptance(fixture(), version, manifest, support).some(error => error.includes("support metadata")));
  }
  const record = fixture();
  record.hosts[0].credentialFixtures = "unavailable";
  assert.ok(validateHostAcceptance(record, version, manifest, supportedHosts).some(error => error.includes("credentialFixtures")));
  const installationOnly = installationOnlyFixture();
  installationOnly.hosts[0].credentialFixtures = "passed";
  assert.ok(validateHostAcceptance(installationOnly, version, manifest, installationOnlyHosts).some(error => error.includes("credentialFixtures")));
  const upgradedSupport = [{id:"trae",credentialAdapter:"fixture-tested"}];
  assert.ok(validateHostAcceptance(installationOnlyFixture(), version, manifest, upgradedSupport).some(error => error.includes("credentialFixtures")));
});

test("hosts without adapters accept installation-only evidence but never API or media case outcomes", () => {
  const record = installationOnlyFixture();
  record.hosts[0].macOS.installation = "passed";
  record.evidence.push({
    method: "real-host", host: "trae", os: "macOS", hostVersion: "fixture-host-version",
    osVersion: "fixture-os", osArchitecture: "arm64", executionMode: "local", shellVersion: "fixture-shell",
    executorPlatform: "darwin-arm64", executorSha256: sha, version, checkedAt: "2026-09-12",
    artifact: "references/fixture.md",
    cases: Object.fromEntries(acceptanceCases.map(id => [id, id === "installation" ? "passed" : "unavailable"])),
  });
  assert.deepEqual(validateHostAcceptance(record, version, manifest, installationOnlyHosts), []);
  assert.equal(hostAcceptanceLevel(record.evidence[0]), "installation-verified");
  for (const id of acceptanceCases.filter(id => !["installation", "help-no-api"].includes(id))) for (const state of ["pending", "failed", "passed"]) {
    const changed = structuredClone(record);
    changed.evidence[0].cases[id] = state;
    assert.ok(validateHostAcceptance(changed, version, manifest, installationOnlyHosts)
      .some(error => error.includes(id) && error.includes("no credential adapter")), `${id}/${state}`);
  }
  const renamedSupport = [{id:"new-host",credentialAdapter:"pending"}];
  record.hosts[0].host = record.evidence[0].host = "new-host";
  assert.deepEqual(validateHostAcceptance(record, version, manifest, renamedSupport), []);
  record.hosts[0].macOS.authenticatedAPI = record.evidence[0].cases.authenticatedAPI = "passed";
  assert.ok(validateHostAcceptance(record, version, manifest, renamedSupport).some(error => error.includes("no credential adapter")));
});

test("image/video acceptance does not imply audio, music, evaluation or complete stable acceptance", () => {
  const record = fixture();
  const evidence = {
    method: "real-host", host: "codex", os: "macOS", executionMode: "local",
    cases: Object.fromEntries(acceptanceCases.map(id => [id, acceptanceGroups.media.includes(id) ? "passed" : "pending"])),
  };
  record.evidence.push(evidence);
  for (const id of ["installation", "authenticatedAPI", "sameTaskResume", "nativeAttachmentDelivery"]) record.hosts[0].macOS[id] = "passed";
  assert.equal(hostAcceptanceLevel(evidence), "media-verified");
  assert.deepEqual(hostCapabilityAcceptance(evidence), {
    media: "passed", audio: "pending", music: "pending", evaluation: "pending", embedding: "pending", rerank: "pending", workflows: "pending", help: "pending",
  });
  assert.ok(validateStableHostAcceptance(record).some(error => error.includes("audio, music, evaluation")));
  const separate = structuredClone(evidence);
  for (const id of acceptanceCases) separate.cases[id] = acceptanceGroups.media.includes(id) ? "pending" : "passed";
  record.evidence.push(separate);
  assert.equal(hostCapabilityAcceptance(separate).audio, "pending", "cannot join separate incomplete environments");
  assert.ok(validateStableHostAcceptance(record).some(error => error.includes("audio, music, evaluation")));
  evidence.cases["music-generation"] = "passed";
  assert.equal(hostCapabilityAcceptance(evidence).music, "pending", "music resume still needs evidence");
  evidence.cases["music-resume"] = "failed";
  assert.equal(hostCapabilityAcceptance(evidence).music, "failed");
});

test("older complete media acceptance cannot accept unverified embedding or rerank execution", () => {
  const record = fixture();
  const evidence = {
    method: "real-host", host: "codex", os: "macOS", executionMode: "local",
    cases: {
      ...Object.fromEntries(acceptanceCases.map(id => [id, "passed"])),
      "text-embedding": "pending", "text-rerank": "pending",
    },
  };
  record.evidence.push(evidence);
  for (const id of ["installation", "authenticatedAPI", "sameTaskResume", "nativeAttachmentDelivery"]) record.hosts[0].macOS[id] = "passed";
  assert.ok(validateStableHostAcceptance(record).some(error => error.includes("no complete local")));
  assert.equal(hostCapabilityAcceptance(evidence).embedding, "pending");
  evidence.cases["text-embedding"] = "passed";
  assert.equal(hostCapabilityAcceptance(evidence).embedding, "passed");
  assert.equal(hostCapabilityAcceptance(evidence).rerank, "pending");
});

test("local help can be verified without a credential adapter", () => {
  const record = installationOnlyFixture();
  record.evidence.push({
    method: "real-host", host: "trae", os: "macOS", hostVersion: "fixture",
    osVersion: "fixture", osArchitecture: "arm64", executionMode: "local", shellVersion: "fixture",
    executorPlatform: "darwin-arm64", executorSha256: sha, version, checkedAt: record.checkedAt,
    artifact: "references/fixture.md",
    cases: Object.fromEntries(acceptanceCases.map(id => [id, ["installation", "help-no-api"].includes(id) ? "passed" : "unavailable"])),
  });
  assert.deepEqual(validateHostAcceptance(record, version, manifest, installationOnlyHosts), []);
  assert.equal(hostCapabilityAcceptance(record.evidence[0]).help, "passed");
  assert.equal(hostCapabilityAcceptance(record.evidence[0]).audio, "unavailable");
});

test("stage timings reject private fields, invented phases, fixture data and invalid counters", () => {
  const cases = {"speech-generation": "passed", "help-no-api": "passed"};
  const valid = {case: "speech-generation", basis: "client-observed",
    durationsMs: {request: 1250, handoff: 80}, requests: {submit: 1, status: 0, content: 0, other: 0}};
  assert.deepEqual(validateAcceptanceMeasurements([valid], cases), []);
  for (const invalid of [
    {...valid, prompt: "private"}, {...valid, task_id: "private"},
    {...valid, basis: "fixture"}, {...valid, durationsMs: {serverGeneration: 1000}},
    {...valid, durationsMs: {request: -1}}, {...valid, durationsMs: {request: Infinity}},
    {...valid, durationsMs: []}, {...valid, durationsMs: {}},
    {...valid, requests: {submit: 1.5}}, {...valid, requests: {headers: {}}},
    {...valid, requests: []}, {...valid, case: "unregistered"},
  ]) assert.ok(validateAcceptanceMeasurements([invalid], cases).length);
  assert.ok(validateAcceptanceMeasurements([valid, valid], cases).length);
  assert.ok(validateAcceptanceMeasurements([valid], {"speech-generation": "pending"}).length);
  const help = {...valid, case: "help-no-api"};
  assert.ok(validateAcceptanceMeasurements([help], cases).length, "help must not make paid requests");
  help.requests = {submit: 0, status: 0, content: 0, other: 0};
  assert.deepEqual(validateAcceptanceMeasurements([help], cases), []);
  delete help.requests.other;
  assert.ok(validateAcceptanceMeasurements([help], cases).length, "unmeasured is not zero");
  for (const invalid of [
    {...valid, requests: {...valid.requests, submit: 2}},
    {...valid, requests: {...valid.requests, status: 1}},
    {...valid, durationsMs: {request: 50, wait: 100}},
    {...valid, durationsMs: {request: 50, download: 10}},
  ]) assert.ok(validateAcceptanceMeasurements([invalid], cases).length, "sync measurement must preserve single-request lifecycle");
  for (const id of ["sameTaskResume", "music-resume", "audio-reattachment"]) {
    assert.ok(validateAcceptanceMeasurements([{...valid, case: id}], {[id]: "passed"}).length, "recovery must not recreate");
  }
});

test("release evidence checks measurements without promoting outcomes or rejecting emulation", () => {
  const record = fixture();
  const evidence = {
    method: "real-host", host: "codex", os: "macOS", hostVersion: "fixture", osVersion: "fixture",
    osArchitecture: "arm64", executionMode: "local", shellVersion: "fixture", executorPlatform: "darwin-arm64",
    executorSha256: sha, version, checkedAt: record.checkedAt, artifact: "references/fixture.md",
    cases: Object.fromEntries(acceptanceCases.map(id => [id, id === "authenticatedAPI" ? "failed" : "pending"])),
    measurements: [{case: "authenticatedAPI", basis: "client-observed", durationsMs: {init: 50},
      requests: {submit: 0, status: 0, content: 0, other: 0}}],
  };
  record.evidence.push(evidence);
  assert.deepEqual(validateHostAcceptance(record, version, manifest, supportedHosts), []);
  assert.equal(hostAcceptanceLevel(evidence), "unverified");
  evidence.osArchitecture = "amd64";
  assert.deepEqual(validateHostAcceptance(record, version, manifest, supportedHosts), [], "OS architecture can differ from emulated executor architecture");
  evidence.osArchitecture = "arm64";
  evidence.measurements[0].rawResponse = "private";
  assert.ok(validateHostAcceptance(record, version, manifest, supportedHosts).some(error => error.includes("private data")));
});

test("offline evidence templates never inherit passes or invent observed identity and durations", () => {
  const document = fixture();
  document.hosts[0].macOS.installation = "passed";
  const inputs = {document, manifest: {...manifest, version}, support: {supported: supportedHosts}};
  const evidence = createAcceptanceEvidence({host: "codex", platform: "macOS", architecture: "arm64"}, inputs);
  assert.ok(Object.values(evidence.cases).every(state => state === "pending"));
  assert.deepEqual(evidence.measurements, []);
  assert.equal(evidence.executorSha256, sha);
  for (const key of ["hostVersion", "osVersion", "shellVersion", "checkedAt", "artifact"]) assert.equal(evidence[key], "");
  document.evidence.push(evidence);
  assert.ok(validateHostAcceptance(document, version, manifest, supportedHosts).length, "blank template cannot count as evidence");
  assert.throws(() => createAcceptanceEvidence({host: "unknown", platform: "macOS", architecture: "arm64"}, inputs));
  assert.throws(() => createAcceptanceEvidence({host: "codex", platform: "macOS", architecture: "arm64", executionMode: "wsl"}, inputs));
  assert.throws(() => createAcceptanceEvidence({host: "codex", platform: "macOS", architecture: "arm64"},
    {...inputs, manifest: {...manifest, version: "different"}}));
  const installationOnly = createAcceptanceEvidence({host: "trae", platform: "macOS", architecture: "arm64"},
    {...inputs, document: installationOnlyFixture(), support: {supported: installationOnlyHosts}});
  assert.equal(installationOnly.cases["help-no-api"], "pending");
  assert.equal(installationOnly.cases["speech-generation"], "unavailable");
});

test("stable acceptance permits installation-only hosts only with evidence and another complete media host", () => {
  const record = fixture();
  record.hosts.push(installationOnlyFixture().hosts[0]);
  const support = [...supportedHosts, ...installationOnlyHosts];
  for (const host of record.hosts) for (const os of ["windows", "linux"]) {
    host[os] = {installation:"unavailable",authenticatedAPI:"unavailable",sameTaskResume:"unavailable",nativeAttachmentDelivery:"unavailable",reason:"No target in this test fixture."};
  }
  for (const id of ["installation", "authenticatedAPI", "sameTaskResume", "nativeAttachmentDelivery"]) record.hosts[0].macOS[id] = "passed";
  const evidence = {
    method:"real-host",host:"codex",os:"macOS",hostVersion:"fixture",osVersion:"fixture",
    osArchitecture:"arm64",executionMode:"local",shellVersion:"fixture",executorPlatform:"darwin-arm64",
    executorSha256:sha,version,checkedAt:"2026-09-12",artifact:"references/fixture.md",
    cases:Object.fromEntries(acceptanceCases.map(id => [id,"passed"])),
  };
  record.evidence.push(evidence);
  assert.deepEqual(validateHostAcceptance(record, version, manifest, support), []);
  assert.ok(validateStableHostAcceptance(record).some(error => error.includes("trae/macOS")));
  record.hosts[1].macOS.installation = "passed";
  assert.ok(validateHostAcceptance(record, version, manifest, support).some(error => error.includes("trae/macOS/installation")));
  record.evidence.push({
    ...evidence, host:"trae",
    cases:Object.fromEntries(acceptanceCases.map(id => [id,id === "installation" ? "passed" : "unavailable"])),
  });
  assert.deepEqual(validateHostAcceptance(record, version, manifest, support), []);
  assert.deepEqual(validateStableHostAcceptance(record), []);
  record.evidence[0].cases["video-generation"] = "pending";
  assert.ok(validateStableHostAcceptance(record).some(error => error.includes("complete local media acceptance")));
});

test("stable publication cannot promote pending, empty or partially delivered evidence", () => {
  const record = fixture();
  assert.ok(validateStableHostAcceptance(record).length);
  for (const os of ["macOS","windows","linux"]) for (const id of ["installation","authenticatedAPI","sameTaskResume","nativeAttachmentDelivery"]) record.hosts[0][os][id] = "unavailable";
  assert.ok(validateStableHostAcceptance(record).length, "all unavailable is not acceptance");
  for (const id of ["installation","authenticatedAPI","sameTaskResume","nativeAttachmentDelivery"]) record.hosts[0].macOS[id] = "passed";
  record.evidence.push({method:"real-host",host:"codex",os:"macOS",executionMode:"local",cases:Object.fromEntries(acceptanceCases.map(id=>[id,"passed"]))});
  assert.deepEqual(validateStableHostAcceptance(record),[]);
  record.evidence[0].cases["video-generation"] = "pending";
  assert.ok(validateStableHostAcceptance(record).length);
  record.evidence[0].cases["video-generation"] = "passed";
  record.evidence[0].executionMode = "remote";
  assert.ok(validateStableHostAcceptance(record).length);
});

test("stable publication exception is explicit and version-bound", () => {
  const record = fixture();
  assert.equal(validStablePublicationException(record), false);
  record.stablePublicationException = {
    kind: "maintainer-approved-pending-acceptance",
    version,
    approved: true,
    approvedAt: record.checkedAt,
    reason: "Maintainer approved direct stable publication before real-host acceptance."
  };
  assert.equal(validStablePublicationException(record), true);
  assert.deepEqual(validateStableHostAcceptance(record), []);
  assert.deepEqual(validateHostAcceptance(record, version, manifest, supportedHosts), []);
  record.stablePublicationException.version = "0.0.0";
  assert.ok(validateHostAcceptance(record, version, manifest, supportedHosts).some(error => error.includes("version-bound")));
});
