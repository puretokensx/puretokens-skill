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

## Remaining acceptance

Real Skill loading, effective-session confirmation, actual Key group permissions,
dedicated paid API results and native attachments/playback are pending. Named
OMP profiles, dotenv/config/project/session/runtime overrides and MiniMax
non-default sessions are unsupported. Claude Desktop loopback mappings and
Antigravity process-only credentials have no supported Skills auth path.

Do not turn catalog visibility, init, fixtures, generated binaries or installation
success into a paid-call or delivery claim. Paid acceptance needs a separate
plan, cost estimate and authorization; submit at most once per authorized case.
