# Client/model synchronization, 2026-11-03

Status: source changes only, not publication or real-host acceptance. No real
credentials, client configurations, media, prompts, paid requests or deployment
were used. The existing 0.19.8 publication approval is historical, not approval
for these source changes.

## Evidence

- Switch source base: `712df5285eb016205d86e7a48c35d242c850df9c` plus existing
  local changes. Its 25-client catalog includes Oh My Pi and Antigravity CLI.
- Oh My Pi upstream sample: 18.13.1, revision
  `fbc3ac6a148d24d50f573e9b531999e32bae42eb`; Switch writer and dual-platform
  `tests/fixtures/client-configs/oh-my-pi` are synthetic, not installed-host proof.
- Production media snapshot: `references/media-model-catalog.json`, captured
  `2026-11-03T02:14:24.310Z` from anonymous console model-catalog. Media IDs
  remain 38; both Wan 3.0 models now allow only `16:9`, `9:16`, `1:1`,
  `4:3`, `3:4`, `21:9`, `adaptive`. Defaults are unchanged.
- DSH version-1 refs and strict legacy flat references are tested without
  migration, fallback credentials or writes.
- Oh My Pi rejects bare environment-reference names and exact-name shadowed
  literals without using an environment credential. Unrepresented session or
  extension overrides require host confirmation; file parsing cannot detect them.
- New CLI command fixtures exercise exact image/video/audio/Jev API routes,
  refreshed visible/absent model catalogs, 401/403 rejection and no resubmission.
  Embedding/rerank evidence remains shared executor coverage, not host E2E.
- Installer tests use temporary isolated homes, install ten Skills and one
  executor, and verify manifests. Windows test cases are added; execution on
  macOS does not constitute a Windows installation result.
- Switch focused model/guide/summary tests: 56 passed on the pinned toolchain,
  including mapped/unconfirmed/source-only badges and the shared MiniMax ID's
  Desktop-only scope. No unqualified install or paid example is offered for it.
- Anonymous public catalogue rechecked at `2026-11-03T03:18:38.901Z`: 133 rows;
  all 18 installed audio IDs, 3 Jev IDs and 3 embedding/rerank IDs remain listed.
  `stepaudio-3-realtime-preview` is listed but remains outside the one-shot
  executor contract. Generic `unknown`/`openai` metadata does not establish
  dedicated API parameters, account permission or real-call acceptance.

## Local verification

- Rebuilt all six native executor targets and verified source/artifact hashes.
- `npm run check`: Go tests and vet passed; Node 193 cases, 189 passed,
  4 skipped, 0 failed. Skips require PowerShell, unavailable on this Mac.
- Switch desktop TypeScript check passed; no native UI screenshot or real
  configuration write was performed.
- `npm run release:validate` intentionally remains blocked by the immutable
  distributed-version gate: changed source cannot ship again as 0.19.8.
  No version bump, tag, push, publication or old approval reuse occurred.

## Mapping simplification, 2026-11-03

Historical intermediate decision. The no-exchange/no-proxy boundary remains;
the no-connection-file and unsupported-mode conclusion was superseded by the
approved same-host Apply implementation recorded below.

The uncommitted mapped-credential exchange experiment was withdrawn in both
repositories. There is no credential endpoint, encryption protocol, new Key file
or dedicated-API proxy. Existing chat mapping and the prior client/model sync
remain intact. The Skills change is limited to a specific local diagnostic for
the exact documented mapped endpoint, using the existing
`active_connection_endpoint_unsupported` status without an API request.
Switch labels this mode as requiring a direct connection, not source-only support.

The new diagnostic regression failed before the change and passed afterward.
Switch's focused UI suite passed 56 tests and its current-contract suite passed
32 tests. The restored chat mapper passed all 7 tests, including unchanged-stream
continuity and rejection of credential-export and dedicated-API routes. Generated
guidance is synchronized. All six executor targets were rebuilt and their embedded
identities and hashes verified. Final `npm run check` passed: Go tests and vet,
then 193 Node cases with 189 passed, 4 PowerShell-only skips and 0 failures.
An earlier packaged-binary check ran before the platform rebuild completed and
failed on the old proof; the final gate verified the completed artifact set.

This deliberately does not claim to enable dedicated APIs in mapped mode.

## Non-mapping support follow-up, 2026-11-03

- Oh My Pi named profiles now align installation, doctor and credential roots.
  Canonical-empty profile selection, legacy fallback, bounded config names,
  named-profile isolation and ignored derived agent directories are covered.
- Explicit `PI_CONFIG_FILES` supports up to four local absolute YAML files, in
  declared order. Nested partial settings preserve unaffected roles; missing,
  malformed, duplicate-key and linked layers stop without another connection.
  Different inline Keys on the same service are selected by exact role only,
  never compared or borrowed. Profile/overlay fixtures also exercise image,
  video, audio and Jev routes, refreshed model visibility and 401/403 rejection.
- Project `.env` existence was previously missed. New Go and Shell regressions
  failed before the fix and passed afterward; no dotenv contents are read.
  Windows cases are added but are not Windows execution evidence.
- The upstream settings migration and deep-merge implementation were reviewed
  at the pinned revision above; this does not support arbitrary CLI overrides,
  model-role arrays, compatibility providers or host extensions.
- MiniMax installed-runtime review confirms request/session selection can
  override saved defaults; no verified current-session file contract was found.
  Antigravity remains process-only authentication. Neither is implemented by
  guessing, reading session databases or adding another authentication path.
- Follow-up verification: `npm run check` passed with full Go tests and vet,
  193 Node cases (189 passed, 4 PowerShell-only skips, 0 failures). All six
  executors were rebuilt and their embedded identities/source hashes checked.
  Named-profile source installation and repeated synchronization passed in an
  isolated macOS home; this is not an installed Oh My Pi session.
- Switch changes are limited to the product contract, changelog and client
  compatibility matrix. Its current-contract documentation suite passed 34
  cases, and the three edited documents passed `git diff --check`. Existing
  unrelated Switch work is retained and is not part of this Skills commit.
- No real user configuration, credentials or paid endpoint was used. No push,
  version bump or publication is included. Windows installer execution and
  actual client loading/API/attachment acceptance remain unverified.

## MiniMax directory and six-route follow-up, 2026-11-03

- Read the installed `@minimax-ai/code` 0.7.0 distribution, not user files.
  `chunks/chunk-4HNPBKOB.js` selects trimmed `MINIMAX_DATA_DIR`, then trimmed
  `MAVIS_DATA_DIR`; `chunk-MNYDNW2K.js` owns legacy `.mavis` migration and
  gated runtime profile selection; `chunk-O3CZWGXJ.js` retains session-model
  precedence. This is static source evidence, not a running host acceptance.
- SHA-256 evidence, relative to that package:
  - `package.json`: `b0b59fc707c6c3ec81dcf3faf75739f6d127d40fc9bbe3350b5a055cbab5604b`
  - `chunks/chunk-4HNPBKOB.js`: `808b86cc3f8ce3a8cae353096f96fd2972c350659fae385b601323eb3495aa7e`
  - `chunks/chunk-MNYDNW2K.js`: `765e9bf898ee5d6499634bf16999653265a0a6f0e4d73b93c4f811cdad02c0fc`
  - `chunks/chunk-O3CZWGXJ.js`: `d389a90e3bfaee16196788626bca4416eca5a32d5dc2e8b17f70bd967d6028bb`
- Fixed installer/executor disagreement on surrounding directory whitespace.
  CLI default resolution now shares Desktop's pending-migration guard; a
  linked default path or legacy-only root cannot silently become a fresh
  installation. Explicit supported roots remain independent. Regression tests
  failed before the changes and passed afterward using isolated homes.
- Extended the command-dispatch fixture to all six dedicated capabilities for
  VS Code, MiniMax Desktop/CLI, Oh My Pi default/profile-overlay, DSH
  versioned/flat stores, and TRAE international/domestic/SOLO contexts.
  Each verifies model visibility present/absent/present, exact model and fixed
  route, selected synthetic Key, 401/403 rejection and a single submission.
  Audio bytes, ordered vector rows, ranking JSON and typed Jev answers are
  checked. Jev's `jev-latest` may correctly resolve to a versioned response;
  this is not a reason to rewrite the selected request ID.
- DSH and TRAE fixtures also verify that calls do not modify their synthetic
  credential file/database. No real user store was opened.
- These fixtures save local files but do not perform client attachment handoff.
  Windows cases are included, not executed on this macOS machine. No client
  login/configuration was changed and no paid or production request was made.
- Rebuilt all six executors and verified their embedded source identities and
  checksums. `npm run check` passed Go tests/vet and 194 Node tests (190 passed,
  4 PowerShell-only skips, 0 failures). Additional DSH/TRAE route tests passed
  afterward, followed by a passing final full Go test/vet run. Switch's
  existing current-contract documentation suite passed 34 tests.

## Same-host mapped Skills connection, 2026-11-03

The existing Switch Apply action now enrolls the same Claude Desktop connection
for Skills, without another consent dialog or another client's credentials.
See [the shared connection contract](../claude-mapped-connection.md) and Switch
ADR-0008. This is unreleased source support, not an installed 0.19.8 feature.
No real client configuration was read or written for these checks.

- Switch adapter: 19 tests passed, including strict record ownership, duplicate
  and unknown fields, dual-platform four-file plans, direct-mode deactivation,
  restore and stable connection identity across same-Key reapplication.
- Switch native configuration: 52 tests passed; shared transaction crate:
  20 passed. A failed fourth-file write restores both existing and first-time
  configurations without an unfinished journal. Rehydration does not enroll
  legacy installations; logout retains applied client credentials.
- Frontend model/guide/summary: 56 tests passed; current-contract documentation:
  35 passed. Existing chat mapper: 7 passed. TypeScript, Rust formatting, adapter
  Clippy and repository checks passed. Final schema/guidance assertions passed
  23 tests, including allowed task bindings and excluded support-export IDs.
- Fresh synthetic Switch writer output for macOS and Windows was exported into
  an isolated temporary directory and accepted by the actual Go reader. This
  proves serialized format interoperability, not a Windows runtime result.
- The six-capability dispatcher fixture verifies current mapped credentials,
  present/absent/refreshed model lists, exact model IDs/routes, 401/403 rejection
  and one submission. Missing async task records stop before networking.
- Changed connection identities block remote status/wait/content/resume and
  redelivery without modifying task identity or resubmitting. Verified local
  delivery remains available. Support exports omit connection identities.
- Final full `npm run check`: Go tests and vet passed; Node 195 cases,
  191 passed, 4 PowerShell-only skips, 0 failures. Six native artifacts match
  the source/build proof. The first full attempt exposed a missing new contract
  in the isolated guidance fixture; it was corrected before the passing rerun.
- Windows ACL tests cross-compiled, but were not executed on Windows. No real
  client loading, paid calls, native attachment handoff, push or publication
  occurred. The two projects must release compatible versions before uptake.

## Remaining acceptance

Real Skill loading, effective-session confirmation, actual Key group permissions,
dedicated paid API results and native attachments/playback are pending. OMP
dotenv/project/session/runtime/extension overrides, undeclared or nonabsolute
config overlays and MiniMax non-default sessions are unsupported.
Claude Desktop mapping now has the source/fixture path above; Windows ACL and
actual session/API/delivery acceptance remain pending. Antigravity process-only
credentials still have no supported Skills auth path.

Do not turn catalog visibility, init, fixtures, generated binaries or installation
success into a paid-call or delivery claim. Paid acceptance needs a separate
plan, cost estimate and authorization; submit at most once per authorized case.
