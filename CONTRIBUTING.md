# Maintainer Guide

End users follow README installation guidance and receive a native executable. Node and Go are repository development tools only.

Use CHANGELOG.md for shipped and candidate changes, and references/host-acceptance.json for outstanding real-host acceptance. Keep local handoff notes separate from public release documentation.

## Repository hygiene

Keep current contracts, source generators, regression fixtures and version-bound
acceptance evidence. Historical evidence is not a current capability declaration;
use `references/switch-skill-compatibility.md` for the release boundary and
`references/host-acceptance.json` for real-host results. Archived task progress and
one-time authorization remain in Git history, not competing handoff documents.

Local `dist/releases/` should retain only the current verified release/candidate's
six archives, four installer scripts and manifest. Before deleting superseded
packages, verify the retained manifest/checksums and that no packaging or
verification process uses the old files. Remove completed redundant local task
drafts and system metadata; preserve unique research, incident evidence and local
customizations. Do not delete Git tags, releases or history as build residue.

The six committed executor binaries and self-contained generated Skill references
are required distribution inputs, not disposable duplicates. The README brand
image and `runtime/executor/testdata/tiny.webm` are active documentation/test assets.
Do not rewrite published Skill bytes during repository-only cleanup; version-bound
wording in installed copies requires the normal next-version synchronization.

## Validation

Run the authoritative engineering gate from the repository root:

```sh
npm run check
```

This gate must cover repository contracts, generated-document drift, executor behavior and actual request/receipt schema conformance. Test with isolated fixtures and local HTTP servers; never use real credentials or paid media requests as routine validation. Before release, additionally run `npm run release:validate` for catalog freshness and reproducible six-platform binary verification. Fixture success does not complete real-host API, continuation or attachment-delivery acceptance; record those separately in `references/host-acceptance.json`.

## Gateway Media Contract Ownership

The paired gateway repository is `PureTokenPlus_web`. Its
`docs/architecture/media-request-contract.md` owns supplier adaptation and the
public task lifecycle. A synchronous supplier does not make the Skill
synchronous: the gateway persists work, returns a public task ID, and executes
with the original API token's authorization and billing scope. The executor
keeps the public `async: true` compatibility field and never sends Pic's private
`X-PTS-Image-Managed-Delivery` header.

Cross-repository media changes must update that gateway document, its replayable
New API patch/register and tests, and this repository's
`references/direct-api-execution-contract.json`, matching schema/validator and
executor regressions together. Do not revert to raw provider responses, longer
client waits, or internal headers to work around a gateway regression. The
2026-09-16 gateway fix requires its reviewed provider-mapping metadata projection
to be deployed as well as code; Skill-only release cannot activate that fix.
Local fixture acceptance is not production rollout or real-provider acceptance.

Gateway correlation and the Skill `support` block are a paired contract.
`PureTokenPlus_web` patch `037-public-support-request-id.patch` exposes the
existing canonical gateway ID as `X-Request-ID` while retaining
`X-Oneapi-Request-Id`. The executor accepts one lowercase canonical UUID,
uses legacy only when the standard header is absent, and omits duplicates,
malformed or conflicting values. Never read upstream request IDs or add client
headers to simulate server correlation.

`runtime/executor/support_summary.go` projects an explicit allowlist from the
current receipt and latest HTTP attempt into `puretokens-support-v1`, checked by
`schemas/support-summary.schema.json`. It adds no network/config/filesystem
collection, automatic export/upload or persistent task-record fields. It excludes
prompts, parameters, paths, credentials, configuration, raw errors and media.
Request correlation does not prove task acceptance, billing or idempotency.
A subsequent HTTP failure must clear earlier request IDs; local-only recovery
must not restore previous HTTP evidence. Regression coverage includes header
validation, actual receipt schemas, no additional requests, same-task recovery
and absence from persisted records. Sharing requires user-directed private
support, not public logs or a full receipt dump.

After changing executor source, dependencies, build inputs or package metadata, use the toolchain in `runtime/executor/build-config.json` and run `npm run executor:build` to regenerate all six artifacts and their checksums before packaging. `npm run release:package` prepares one-platform archives and a manifest pinned to the source commit.

Before running the release-preparation workflow, rebuild and commit the executor artifacts, checksums and matching source inputs in the selected revision. That workflow tests and validates the revision, then packages its checksum-verified committed artifacts; it rebuilds all six platforms into temporary directories with the exact toolchain in `runtime/executor/build-config.json` and compares bytes before packaging the committed artifacts. Local packaging from modified or untracked source inputs produces draft candidates with `sourceCommit: null`, which cannot be promoted as pinned release assets.

Installation ownership uses complete per-directory inventories, including empty
directories. Added/modified/missing entries and symlinks block replacement and
recovery. An unmarked directory may be adopted only when its full contents match the current official source. No historical snapshots or retired-installation migration are maintained.
Inventory markers detect accidental edits, not malicious local tampering.

`npm run acceptance:validate` checks the real-host record and release hashes;
follow `references/host-acceptance-guide.md` for actual host acceptance. Pending
cells remain pending without real evidence. Windows installer fixtures already
invoke both Windows PowerShell 5.1 and PowerShell 7; preserve both entrances.

For parser changes, run the relevant bounded Go fuzz targets from
`runtime/executor`, for example `go test -run '^$' -fuzz FuzzMediaContainers
-fuzztime 10s -parallel 2`. Seed cases also run in ordinary Go tests. Do not
install a codec runtime for users: container checks are bounded, and actual video
playback remains a host acceptance item.

## Contract changes

Keep request/receipt schemas, executable behavior, installed Skill instructions and command examples in the same change. Ordinary generation reads one selected profile and submits once; explicit validation, catalog filtering and diagnostics must not become mandatory preflights. Use concise user receipts while retaining safe machine metadata. Task records are optional user/workspace artifacts without prompts, credentials, reference URLs or media bytes.

Media entrypoints retain host binding, submission/stop conditions and native
handoff. Keep recovery details in each installed `references/executor-usage.md`,
symptom/support guidance in `references/failure-guide.md`, and user examples in
the update Skill's usage guide. Load those sections only when relevant.
Instruction fixtures protect these boundaries but do not prove an actual host
or model followed them; do not convert byte savings into claimed latency gains.

Align package, Skill index, all registered Skill manifests, executor version/manifest and host-acceptance release versions. Refresh each manifest's sourceSha256 after editing SKILL.md. Update both changelogs. Preserve the exact first text installation block and its heading in both READMEs; the client download page extracts them.

Shared guidance has one maintained source: `references/desktop-hosts.md`,
`references/skill-fragments/host-binding.md` and `references/host-support.json`.
Run `npm run docs:sync-guidance` after editing these or any Skill entrypoint.
It expands self-contained installed copies, host lists and entrypoint hashes;
`npm run check` rejects drift. This generator is a development tool only.

Operation summaries are maintained in `references/operation-guidance.json`.
The same generator checks their command/lifecycle/recovery/delivery against the
owned execution contracts and expands marked sections in the media/evaluation
entrypoints and usage guide. `references/skill-fragments/receipt-guide.md` and
`media-workflows.md` are the only maintained sources of their installed copies.
Do not edit generated sections/copies or mix synchronous audio recovery with
asynchronous music. Local help and on-demand reference reads add no API calls.

Multi-step guidance orchestrates existing commands, not a new background
executor. Explicitly authorized generated artifacts may become the next step's
input only through the selected profile's existing multipart representation;
retain bytes and task identity, never synthesize a URL or silently drop an input.
There is no composition/mixing, cross-step server artifact reference, authoritative
quote, unknown-submit lookup or model-retirement interface added by this guidance.
Those require a separately reviewed gateway contract, authorization/expiry/error
semantics, replayable patch and paired documentation before Skill adaptation.
Do not infer support from private Web workspace routes or expose their transport.

The paired gateway contract's “Skill recovery and selection extensions” section
(2026-09-27) records the next implementation requirements, not available APIs.
First establish durable async image/video/music submission identity and read-only
lookup with account/original-token scope, fingerprint conflicts, crash uncertainty
and expiry tombstones. Do not reuse support request IDs or infer idempotency from
the Web workspace UUID. Synchronous audio/Jev recovery remains out of scope until
result retention exists. Quotes/retirement need authoritative revision, scope and
expiry semantics; server artifact references need authorization, expiry pinning
and destination input validation. Keep `unimplementedServerDependencies` and the
installed instructions unchanged until actual gateway implementation, replayable
patches, cross-database verification and compatible feature negotiation exist.
No ordinary generation gains an extra capability preflight from this backlog.

Model queries and submissions share `runtime/executor/model_constraints.go`.
Queries infer required operation inputs without claiming attachment-byte
validation. Local profiles remain conservative: gaps may fetch a catalog once,
but existing enum/range/type failures never trigger automatic relaxation.
An explicit exact-model query can identify drift; adapting and releasing the
profile is a maintainer action, not a mutable user-side profile cache.

`workflow_trace_test.go` checks offline command, HTTP, receipt and simulated
handoff ordering for multiple images, local edits and videos. These tests do
not prove an actual client follows Skill instructions. PNG/JPEG integrity uses
standard decoders with a 33,554,432-pixel cap; WebP remains structural validation.
Keep positive fixtures, corrupted-payload cases and bounded-resource tests.

## Distribution

This repository does not publish an npm package. Native fetch resolves the latest stable release manifest, pins its version/source commit and verifies its selector and current-platform ZIP. No ordinary install follows main or falls back to a source archive. Explicit local source sync remains available for maintainers. Keep the registered shared Skills; do not fork payloads per client or create a global shared-executor service.

`npm run release:package` produces six OS/architecture-only archives, four bootstrap/selector script assets and the schema-v2 release manifest. `npm run release:verify` checks all archive members against current source, checksums, lengths, platform-only contents and script hashes; `--publishable` additionally requires clean committed distribution sources. Dirty candidates cannot be promoted. The same checks run on the downloaded draft assets immediately before publication.

Release preparation remains read-only with respect to GitHub releases. The separate manual `Publish verified stable release` workflow accepts an existing matching version tag, tests the sources, prepares and verifies all assets, creates a draft, uploads and re-downloads the complete asset set, and only then promotes it to latest. Failure leaves the previous public release unchanged and any new draft for maintainer review; never overwrite an existing tag/release or silently downgrade installations. Publishing is an explicit action, not a side effect of pushing main. Never delete unknown Skill directories or replace unrelated configuration.

From 0.18.1 onward, each distributed version is immutable. Increment the version before changing installed Skills, profiles, executors or installation/build inputs; same-version fast verification and downgrade protection rely on this. `release:validate` checks complete first-parent Git history, including same-version changes that were later reverted, and includes untracked/deleted distribution files. A version-bumped dirty tree is only a candidate, never proof of publishable assets. Executor test-only changes do not require a new version. Keep the gate before push and publication; main changes are no longer the default installation source.

Stable promotion also runs `node scripts/validate-host-acceptance.mjs --stable`: every declared available target needs real evidence, API-capable targets need all cases declared by that validator passed in one local environment, and an empty/all-unavailable matrix is rejected. An explicit version-bound publication exception may permit pending results without changing their status; it is neither real-host proof nor authorization for paid tests or another release. The original media level does not accept audio, music, Jev, composite workflows or help; capability groups are reported separately. Prepare pending-only entries with `node scripts/prepare-host-acceptance.mjs --host <id> --output <new-file>` and follow the guide for optional observed stage timings. Regular development gates still accept honest pending evidence. Run real-host acceptance only with explicit authorization; do not turn it into paid CI traffic.

Acceptance classification must match the credential adapter state in `references/host-support.json`. Hosts without an adapter keep installation acceptance separate and mark API-dependent summaries and detailed cases `unavailable` with an explicit reason, not `pending` or `passed`. Credential fixtures never substitute for real-host evidence.

Exact media quotes, server idempotency and discovery of unknown submissions are server-side dependencies not implemented here. Do not claim these guarantees from validation, local records or catalog data. Keep keys, cookies and user media out of source, fixtures and release artifacts.

Build identity is recorded in `runtime/executor/build-proof.json` and embedded in each executable. `npm run validate` checks inputs and artifact identities; `npm run executor:verify` independently rebuilds and compares all six outputs. Current-platform Go tests also execute the packaged binary offline. These development checks add no user runtime dependency.
