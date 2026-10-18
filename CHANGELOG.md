<p align="center">
  <strong>English</strong> · <a href="./CHANGELOG.zh-CN.md">中文</a>
</p>

# Changelog

## 0.19.5 - 2026-10-18

- Add text embedding and reranking Skills that save vectors as JSONL and relevance rankings as JSON.
- Install all ten Skills together and include the new capabilities in the installation completion summary.
- Add both Wan 3.0 video models and Grok generation ratios `21:9` and `5:2`, keeping the current image and video defaults.
- Remove `grok-imagine-image-2.0` from new model selection while preserving recovery and delivery of existing tasks.
- Correct Grok video editing options so unsupported models and parameters stop before submission.
- Update client compatibility and Switch setup guidance for dedicated models.

## 0.19.4 - 2026-10-15

- Keep installation prompts short while the installer handles client/platform selection, checks, initialization and result guidance.
- Detect supported native client launchers without reading configuration; preserve confirmed clients and stop for one selection when the host is ambiguous.
- Report sanitized download/write failures and keep completed installation separate from connection verification.
- Restore interrupted Windows installations correctly with both Windows PowerShell 5.1 and PowerShell 7 transaction-plan parsing.
- Isolate installer tests from inherited OpenCode directories and cover legacy JSON-array behavior offline.

## 0.19.3 - 2026-10-15

- Extend bounded host detection to native Grok Build and Qoder CLI launchers. Preserve confirmed clients; ambiguous launchers require one explicit selection.
- Report sanitized local download/write/sync failures with their phase, installation state and next step; distinguish verified transaction recovery from unknown file state.
- Keep committed installation successful when init cannot launch, times out or cannot capture output. Accept verification only with successful identity and credential evidence.
- Keep installation prompts short and synchronize update instructions, contracts and offline Shell/PowerShell regression coverage.

## 0.19.2 - 2026-10-15

- Reduce the copyable installation prompts to three lines containing only the task, official stable entry and install operation. Keep client/platform selection, verification, initialization and failure guidance in the installer and installed instructions.
- Request Octop's missing workspace target before release downloads or installation writes, preserving explicit client and directory choices.

## 0.19.1 - 2026-10-15

- Shorten both installation prompts and download the official stable fetch directly, without first fetching README.
- Let fetch attempt current-client detection through at most eight direct launcher names and parent IDs. Explicit native Codex/OpenCode launchers are recognized; unknown interpreters/editors stop for client confirmation before release downloads or installation writes.
- Preserve explicit host/custom-target selection, platform checksum verification, safe updates and the separate connection-verification result.
- Use a new distribution version without overwriting 0.19.0.

## 0.19.0 - 2026-10-14

- Restore stable installation and update downloads from `puretokensx/puretokens-skill`, including upgrades from the previous repository, with verified packages for macOS, Windows and Linux.
- Add local installation inspection and explicitly requested repair while preserving customizations and unknown files.
- Offer compatible image/video models after eligible failures, with current-connection evidence and confirmation before any new paid task.
- Show exact-model group guidance using live Switch names while preserving existing group selections.
- Preserve terminal failure reasons across sessions and support explicit delivery of an existing task's result again, reusing verified local files when available.
- Answer product introductions and getting-started questions from local help, and distinguish connection verification from public API identity checks.
- Provide client-specific loading steps, concise usage examples and clearer delivery guidance for images, video, speech and music.
- Refresh 29 image/video profiles against the current public catalog, including updated GPT Image limits, H3 parameters and same-task delivery of up to eight images for models that support it.

## 0.18.9 (migration draft) - 2026-10-12

- Prepare official install and update URLs for `puretokensx/puretokens-skill`, retaining pinned-version downloads and checksum verification.

## 0.18.8 - 2026-10-04

- Explain command, HTTPS and directory permissions before the initial download in the complete installation prompts. Preserve the one-line installation entry and do not require Full Access by default.
- Distinguish Windows socket access denial from unavailable HTTPS security context, ordinary TLS errors and HTTP rejection. Report sanitized failure stages and next steps without exposing exception details or retrying automatically.
- Resume installation only after explicit continuation and current authorization. If synchronization succeeded but connection verification failed, continue init alone without reinstalling; uncertain synchronization still requires diagnosis.
- Keep installer guidance, update Skill scenarios and regression coverage aligned, including preservation of existing files and init-only continuation.

## 0.18.7 - 2026-10-04

- Refresh 29 image/video model profiles and GPT Image parameters. Set the default image model to `gpt-image-2.5-flare` and video model to `minimax_h3`; explicit model selections remain unchanged.
- Add official DeepSeek Harness Desktop and MiniMax Code Desktop installation and connection adapters, with documented local-session and custom-directory support.
- Fix VS Code connection resolution for Switch configurations containing multiple models that share one supported connection and Key. Conflicting connections stop with a clear error.
- Correct H3 first/last-frame requests to use `image_to_video` with both frame inputs.
- Add `puretokens-audio` command handling for speech, recording transcription, sound generation and asynchronous music, plus `puretokens-evaluate` for Jev Choice/Score/Noul text evaluation.
- Improve image/video intent selection, parameter guidance and composite workflows. Preserve original prompts, lyrics and attachment bytes.
- Improve task-response diagnostics, same-task recovery and local artifact reattachment. Unknown submissions are not automatically retried.
- Improve bilingual installation guidance and synchronize eight Skills with one native platform executor. No additional user runtime is required.
- Add optional anonymous operational completion reports, disabled by default, without account, configuration or content data.
- Extend automated coverage for client connections, model permissions, attachment handling, task recovery and dedicated APIs.

## 0.18.6 - 2026-09-19

- Accept public task IDs containing dotted model labels and legacy output suffixes without rewriting their identity. Apply the same bounded rules to submission, status/wait/content, task records and support schemas; continue rejecting paths, URLs, encoded input and control characters.
- Replace generic lifecycle test IDs with gateway-shaped image/video IDs and add shared boundary fixtures plus native receipt/schema round trips. The regression reproduced the old false "no usable task ID" failure before the fix.
- Synchronize the paired Web task-identity contract and rebuild all six native executors. No gateway routing or credential changes; unknown submissions are never automatically retried. Existing discarded IDs cannot be recovered by this fix.
- The maintainer approved stable publication of 0.18.6 after the offline regression and build checks, with real-host media acceptance still pending. Record a new version-bound exception without changing the 150 pending host/OS checks; do not reuse the 0.18.5 exception.

## 0.18.5 - 2026-09-18

- Refresh GPT Image profiles from the public catalog: `gpt-image-2`, `gpt-image-2.5` and `gpt-image-2(Sub)`; retire Flare/Sunburst entries. Enforce 1K, ten declared ratios, one output, up to ten references, declared quality and output format, and size/ratio constraints.
- Preserve exact parenthesized model IDs safely across discovery, profiles, submission, receipts and task records. Keep the gateway-owned asynchronous task contract unchanged.
- Register Hermes, EvoX, VS Code and Octop with bounded file/SQLite credential adapters, installer roots and isolated fixtures. Octop requires an explicit local workspace Skill target; its SQLite reader is compiled into the executor.
- Fail closed on Hermes credential pools/overrides, inconsistent EvoX selection, multiple VS Code matching connections and unsupported Octop storage or ambiguous selection. Fixtures do not establish real-host acceptance.
- Align Web/Switch/Skill ownership and support boundaries. The maintainer approved a one-time direct stable publication exception for 0.18.5; the 150 pending host/OS checks remain pending and are not real-host acceptance.

## 0.18.4 - 2026-09-17

- Preserve completed Shell installations when cleanup is denied; release the owned lock when possible, report cleanup separately, and continue init without repeating denied deletion.
- Validate PNG/JPEG pixel payloads with bounded decoding and reject empty/truncated WebP containers; retain same-task failure and download-proof checks.
- Share model parameter and cross-field constraints between catalog filtering and submission, including required reference inputs and mutually exclusive frame modes.
- Align OpenCode installer and doctor directory selection with explicit/XDG roots and report known shared Skill copies without deleting them.
- Unify connection/init/doctor intent guidance, generate shared host instructions from maintained sources, and test offline multi-image/edit/video command trajectories.
- Retain 0.18.3 real-host evidence as history without carrying its passes into 0.18.4. The maintainer approved a one-time stable publication exception for 0.18.4 with 118 real-host summary checks still pending; engineering, reproducible-build and archive checks remain required. Later releases retain the normal acceptance gate.

## 0.18.3 - 2026-09-16

- Align Image 2.5 Flare and Sunburst with the current catalog: allow `3:1` and `1:3` for generation and editing, and reject the removed `4:5` and `5:4` ratios without converting them.
- Refresh the 23-model catalog while preserving Image 2 and all video parameters, including the reviewed Seedance first/last-frame operations.
- Add an allowlisted support block to executor receipts with current-invocation diagnostics and validated gateway request correlation; no extra requests, config reads, exports or task-record persistence.
- Shorten image/video entrypoints, keep recovery details on demand, and add practical examples and symptom-based troubleshooting without changing host binding, one-submit behavior or native attachment delivery.
- Document paired gateway ownership for synchronous image suppliers and backward-compatible request-ID headers; a Skill update alone does not deploy gateway changes.

## 0.18.2 - 2026-09-12

- Fix OpenCode rejecting a valid saved service when no default chat model is configured. Resolve exactly one enabled matching service only when the default is absent; explicit selections never fall back.
- Support bounded JSONC configuration merging, declared XDG/project/managed layers and OpenCode's native API-key store. Reject ambiguous connections, unsupported hooks, dynamic authentication and invalid overrides without changing user configuration.
- Separate saved service verification from current chat identity. Rebuild all platform executors; retain 0.18.1 acceptance as history instead of carrying its passes into the new candidate.

## 0.18.1 - 2026-09-12

- Default installation now resolves a verified stable release manifest and downloads only the current OS/architecture archive. Missing releases never fall back to main or a full source archive; explicit local source sync remains available.
- Same-version install/update verifies the selected executor and every managed inventory, then skips archive download, writes and init. Changed/missing files stop without automatic repair.
- Give installation init one 20-second total deadline across both read-only checks. Preserve successful file installation when connection verification fails.
- Package only the current system's scripts; validate all six archives and uploaded draft assets before explicit stable publication. Pending real-host acceptance blocks stable promotion, not local development checks.

- Complete Pi registration, media entry guidance and installation lifecycle coverage. Resolve endpoint-verified inline authentication in its effective order; reject unsupported references and unrepresented higher-priority authentication without falling back.
- Preserve completed delivery on task resume. Accept copied Windows/POSIX task metadata while requiring local absolute paths and matching byte proofs for attachment handoff; rebind downloads only through an explicit output directory.
- Select Windows executors by OS architecture, including 32-bit PowerShell on 64-bit Windows. Align Pi root validation across installers and executor; add Unicode/space-path and invalid-root regressions.
- Enforce immutable distribution versions in the maintainer release gate and disable cached Go test results in the engineering gate. Real-host installation, API and full media delivery remain separate evidence levels, not inferred from fixtures.

## 0.18.0 — 2026-09-07

- Refresh the official media catalog to 23 models, adding grok-imagine, grok-imagine-1 and grok-video-1.5; retain separately sourced, previously approved Nano Banana ratios and Seedance frame operations.

- Remove obsolete Node installation/uninstall commands, legacy migration ZIPs, retired Skill/runtime cleanup and embedded historical installation snapshots. Keep one native installation route and current managed-update integrity checks.

- Remove Codex legacy plugin inspection and migration from all installations; synchronize Skills, run the selected host init and report its result.

- Fix shared-directory installs invoking Codex plugin migration for other hosts. Installation receipts identify the selected host and retain init outcomes; Qoder guidance uses the native host-specific installer instead of legacy payload copying.

- Consolidate host-specific troubleshooting into on-demand guidance, reconcile stale maintainer state, derive adapter contract checks from the host matrix, and reject unknown hosts before path resolution. Share strict configuration JSON parsing across adapters.

- Add Kimi Code and Qoder local host installation, fixed-API credential adapters, diagnostics and isolated coverage; real-host API and attachment acceptance remains pending.

- Query newly accepted images after a default 5 seconds while retaining the 120-second window; compute download digests during transfer. Completed records can verify local files for attachment redelivery, while reconciliation still queries the original task separately.

- Shorten media entry instructions with on-demand failure guidance and direct exact-model profiles. Treat normal wait-window exhaustion as successful pending with one bounded foreground continuation, preserve the budget in records, and return machine next-step hints. Classify invalid/oversized media accurately.

- Support Image 2.5 flare/sunburst reference edits with up to six images, and add 4:5 / 5:4 aspect ratios to all three Nano Banana profiles while keeping lite limited to 1K.

- Add approved Image 2.5 flare/sunburst profiles and safe quality persistence; add Seedance first/last-frame inputs with mutually exclusive reference modes. Preserve the existing gpt-image-2 profile. New metadata follows the approved Web source; live availability remains API-controlled.

- Launch Windows installer executor calls without extra console windows, capture UTF-8 output and retain exit-code checks without shell or elevation.

- Preserve completed Windows installs when host cleanup is denied, verify installed inventories before success, and avoid blaming working WorkBuddy connections for adapter mismatches.

- Limit website guidance to responsive conversations: wallet recharge after explicit quota/balance errors, usage records for user-requested spending details, and wallet information for allowance-scope questions; guidance-only questions issue no API requests.

- Quota and balance failures now direct users to the official Wallet & Recharge page; quota guidance also retains the API-key allowance distinction.

- Add curated error-specific next actions across all Skills and executor receipts, preserving same-task recovery, uncertain submissions and balance/public-identity exceptions; avoid diagnosing missing models as key-group errors.

- Align continuation, download-integrity and pinned-selector contracts with runtime behavior; clarify recorded/unrecorded flows, legacy file conflicts and host routing; enforce complete failure receipts.

- Harden attachment streaming against changed/truncated input; verify recorded download digests before reuse and delivery.
- Validate model parameters before persisting task records and restrict safe string projections.
- Always use the pinned installer for host directory selection, including new hosts beside old installers.
- Bind executors to source/build inputs and require reproducible six-platform release verification; add native artifact smoke tests.
- Add Linux host acceptance slots and distinguish summary counts from detailed evidence.

- Add ZCode local host installation, update, unique enabled connection adapter and diagnostics; real-host acceptance remains pending.

- Add Claude Desktop and DSH Desktop to all six Skills, native installation/update selection and diagnostics. Claude Desktop uses its selected 3P gateway profile; DSH uses its selected Harness provider and versioned credential reference, with no cross-host or environment-key fallback.
- Add bounded YAML parsing to the compiled executor, desktop credential regressions and macOS/Windows installation-path fixtures. No new user runtime is required.
- Document local execution boundaries, shared/overriding Skill directories and pending real-host API/attachment acceptance for both clients. Include the previously unreleased 0.17.1 fixes below.

## 0.17.1 — 2026-09-06

- Schedule the first pending-image status read 20 seconds after the accepted response, then query every 3 seconds, bounded by 40 reads and a 120-second wait window. Reuse the existing retry_not_before field so tool handoff/continuation waits only the remaining time; older records need no migration. API Retry-After takes precedence and local delays are not reported as API headers. Video polling and single-submission behavior are unchanged.
- Normalize the balance bearer with the existing CC Switch `sk-` prefix rule before its first request. Bare keys accepted by media no longer fail the console usage route's prefix check; configured credentials, suffixes and other API authentication stay unchanged.
- Redact both original and normalized credentials from balance errors. Distinguish usage rejection from public conversion-metadata failure, and stop treating a balance-only 401/403 as proof that a working media connection or Key is invalid.
- Add regressions for bare and prefixed keys, no retry, unchanged media authentication, and escaped credential echoes. Rebuild all six platform executors and migration archives.
- Protect updates with full managed-file inventories and exact historical migration inventories. Same-name directories, added/modified files and symlinks no longer qualify for blind replacement; interrupted recovery retains changed directories and backups.
- Reject header-only/truncated GIF and WebM output before saving or reuse, with bounded GIF frame decoding and WebM container/track/block checks.
- Replace arbitrary server error text with controlled public categories; retain only recognized codes actually returned by the API. Explicit resume of a reconciliation record performs one same-task status read.
- Fix the managed provenance helper, add regression/fuzz cases and Windows ownership fixtures, and validate real-host acceptance evidence separately from isolated tests.

## 0.17.0 — 2026-09-05

- Fix balance queries to use the existing CC Switch API-key usage route and public USD conversion metadata. Report actual wallet balance or Key allowance concisely, exclude subscription quotas, and reject legacy unlimited placeholders or missing units without inventing money. No browser login or media preflight is added.
- Preserve original task counts, parameters, operations and IDs through continuation failures; retain Retry-After across commands without adding automatic submission retries.
- Add explicit task records, same-task resume and host-confirmed delivery acknowledgement. Records exclude prompts, reference media and credentials.
- Add optional doctor, parameter preflight and declared-capability model filtering; ordinary generation still uses only its selected installed profile, refreshing a missing declaration at most once.
- Give media downloads a separate bounded transfer window and actionable local failure guidance. Redact credentials after JSON decoding and reject normalized local-only reference URLs.
- Add lightweight update checks, pinned-source retrieval and checksum-verified platform package selection when matching assets are published; handle Gemini discovery precedence and Windows PowerShell UTF-8.
- Align installed examples, public contracts and response schemas; retain the legacy migration bridge and keep real-host acceptance distinct from fixture tests.

## 0.16.0 — 2026-09-05

- Parse standard TOML and explicit Codex profile, directory and credential sources.
- Return submitted task IDs immediately; separate submit/status/wait/content, preserve identity on failures, cancel requests at deadlines and reject redirects.
- Validate selected model profiles, counts, JSON URL references and multipart operations before billing; preserve string fields correctly.
- Validate binary media and container boundaries, remove failed partial files, protect concurrent output files, support output filesystems without hard links, and distinguish downloads from host attachment delivery.
- Delegate the legacy CLI to platform installers; add update locking, rollback and interrupted-transaction recovery without requiring a plugin CLI for clean Codex installs.
- Authenticate init independently from public identity and use existing bearer billing endpoints with explicit unspecified-unit/scope semantics.
- Add behavioral regressions, separate host acceptance evidence from credential fixtures, and simplify media Skills with cross-platform command examples.

## 0.15.2 — 2026-09-05

- Made `init` failures actionable without exposing connection data. The executor and both platform installers now distinguish active-connection resolution failures, missing credentials, ambiguous WorkBuddy credentials, network failures, rejected identity checks with a returned HTTP status, unreadable responses, and identity mismatches. Installation still succeeds independently of a non-billable `init` check.

## 0.15.1 — 2026-09-05

- Corrected credential resolution to read the active, documented connection record that Pure Tokens Switch writes for Claude Code, Codex, WorkBuddy, Gemini CLI, Grok Build, and OpenCode. Codex now reads the active provider token in `config.toml` rather than `auth.json`; provider labels are not used. Each adapter verifies the Pure Tokens endpoint before retaining one credential only in memory for the fixed request. Trae remains manual-only because Switch deliberately does not write or inspect a managed credential record there.

## 0.15.0 — 2026-09-04

- Added a checksum-verified, one-shot native API executor shipped with the Skills. Users do not need Node, npm, Python, Go, MCP, a proxy, sidecar, or a background service for media, balance, connection, or model requests.
- Images and Videos now invoke the managed executor for fixed Pure Tokens API requests, including declared multipart attachments, bounded same-task polling, and native media-byte delivery. Each billable submission is issued at most once and is never automatically replaced after an uncertain result.
- Rebuilt the platform installers and migration archives to synchronize the executor alongside the six Skills. Verified credential adapters now cover Claude Code, Codex, WorkBuddy, Gemini CLI, Grok Build, and OpenCode; Trae remains manual-only because Switch does not write a managed credential record for it.
- Added bounded multipart streaming, per-media attachment limits, response-size checks, reconciliation handling, continuation tests, non-media rejection tests, and checksum validation for all published platform binaries.
- Added an explicit `init` command. Installation and updates run it automatically to perform one safe fixed-API identity check and print the versioned usage guide with image, video, model, balance, reference-media, and update examples.

## 0.14.3 — 2026-09-04

- Corrected the direct-execution contract: Image, Video, Models, Connection, and Balance must resolve the current matching connection credential privately in memory and issue the fixed API request through the host terminal or native HTTPS capability. They may not first reject a request because a separate “authenticated media interface” is not exposed.
- Made image editing explicit for the default `gpt-image-2` profile: a current user attachment is sent directly as multipart to `POST /v1/images/edits` with the declared `image` field, rather than producing a fictional-interface validation message.
- Restored the exact legacy update archive expected by published 0.13.x updaters, so an old install can reach this release. It contains only a non-functional compatibility marker for that updater's static check; current installs do not use or require a Node media runtime.

## 0.14.2 — 2026-09-04

- Made the fixed host-native API path exclusive for Images, Videos, Models, Connection, and Balance. These Skills now explicitly forbid Computer Use, browser or desktop UI automation, opening Pure Tokens Switch/Desktop, and using another image or video Skill as a fallback.
- Added validation contracts and scenario coverage for a missing native API executor: stop before any request with a local validation failure instead of searching for a visual generation interface or requesting Computer Use permission.
- Made `puretokens-image` and `puretokens-video` the declared first-choice specialists for generic image and video requests on a current Pure Tokens connection, ahead of generic `imagegen`, Imagen, or video Skills. The contract explicitly documents the host-side metadata boundary: a Skill cannot override a host that ignores installed-Skill priority.

## 0.14.1 — 2026-09-04

- Reduced normal media-request context: Image and Video now read a compact model index and only the selected model profile. Full behavior scenarios are loaded only for matching exceptional paths.
- Replaced the monolithic installed model-selection files with generated per-model profiles, while retaining one controlled base catalog and release freshness checks.
- Rebuilt the one-time legacy migration archive from the current source-only installer and Skills. It no longer carries a Node runtime, direct API executor, or retired model-selection files.

## 0.14.0 — 2026-09-04

- Replaced the custom direct-API Node runtime with the host's native authenticated HTTPS/API capability for every supported host. Images, Videos, Models, Connection, and Balance now use the fixed `https://api.puretokensx.com` API URLs without MCP, a proxy, sidecar, or local execution helper.
- Defined the connection contract for third-party CC Switch and manual host configuration: a current matching credential may be used only in memory for the fixed request; the Skill never checks a provider label or exposes connection configuration.
- Updated installation and update flows to install no API runtime and to remove only the verifiably retired official Node runtime when present. Updated user documentation, contracts, scenarios, and validation to match this path.

## 0.13.26 — 2026-09-04

- Removed the custom temporary install-payload ZIPs and their download chain. Installation and updates now obtain a fresh official `main` checkout, then run the source-only native sync script from that checkout. The script no longer downloads anything itself. One unlinked legacy archive remains solely to migrate the already-published 0.13.25 updater, whose update path is otherwise hard-coded to that exact file.

## 0.13.25 — 2026-09-04

- Fixed direct Images and Videos submissions that could stall before any API call in Codex and other managed hosts. Every managed runtime POST now uses only a bounded canonical Base64 request-body argument; standard-input request bodies are rejected immediately instead of waiting for an EOF that a host may never send.
- Added a 90-second total runtime deadline and a sanitized structured JSON runtime-failure envelope with an execution phase. Skills now distinguish a local pre-submit failure from a submission whose acceptance is genuinely unknown, without repeating a billable request.
- Clarified host-policy blocks before the runtime starts: they are local `validation` failures with no API request, task ID, HTTP response, or billing claim. Image and video Skills now direct users to retry only in a session where external-network execution can be approved.

## 0.13.24 — 2026-09-03

- Fixed a legacy Codex-plugin migration hole: the installer now removes the exact listed `puretokens-media` plugin selector, lists plugins again to verify its absence, and stops without a synchronized-version receipt if that cannot be proven. After verified removal, it explicitly requires a full Codex restart because already-open conversations retain loaded legacy Skill instructions.

## 0.13.23 — 2026-09-03

- Simplified the Agent install prompt to the repository URL and moved all bootstrap detail into one official install bundle. Its native installer now accepts a host ID, derives the documented Skill directory, validates the unpacked bundle, and synchronizes without a second package download.

## 0.13.22 — 2026-09-03

- Replaced the two-download Agent bootstrap with a one-download compact-payload flow. The unpacked latest native installer now accepts that validated source explicitly, so it synchronizes without first fetching a runtime installer and then fetching the payload again.

## 0.13.21 — 2026-09-03

- Made the copyable Agent installation prompt explicitly bootstrap the latest platform installer from this repository's `main/runtime` on updates, instead of letting an already-installed older updater run. This closes the old-updater self-update loop.

## 0.13.20 — 2026-09-03

- Made native Skill updates fit short host command windows more reliably. The compact payload now gets one 20-second attempt through GitHub's official API raw-content endpoint and, only if that fails, one further 20-second attempt from GitHub raw content. Each update also bootstraps the latest platform installer before execution, so an older installed installer cannot retain an obsolete network path. No third-party mirror, repeated source retry, or target synchronization occurs before static payload validation succeeds.

## 0.13.19 — 2026-09-03

- Strengthened direct credential binding. A runtime now uses a configured key only when its documented host configuration explicitly identifies the fixed Pure Tokens endpoint; it never sends a key from a local, client-managed, or unrelated connection to the Pure Tokens API. The configured URL only establishes credential provenance—every balance, connection, models, Images, and Videos request still uses the fixed public API URL.

## 0.13.17 — 2026-09-03

- Optimized `puretokens-update` for hosts with short command windows. Native installers now download a compact, static-validated official install payload that excludes documentation and brand assets, instead of the full repository archive; each payload download has a 45-second hard limit. A host timeout or interruption is now explicitly an unconfirmed result, never a claimed upgrade or guessed version.

## 0.13.16 — 2026-09-03

- Added a Codex-only legacy-plugin migration to `puretokens-update`. When synchronizing Codex's `~/.agents/skills`, it uses only the official Codex plugin interface to detect and remove the exact installed `puretokens-media` plugin after Skill synchronization. If the plugin cannot be inspected or removed, including workspace-managed cases, it reports the required Plugins or administrator action instead of claiming success.

## 0.13.15 — 2026-09-03

- A successful `puretokens-update` now reports the exact synchronized official Skill version after listing its installation, upgrade, and retired-Skill cleanup results.

## 0.13.14 — 2026-09-03

- Changed `puretokens-update` and both native installers to remove verified retired managed Skill directories after all current Skills have synchronized successfully. It also removes hidden retired backups left by earlier installers, so hosts cannot keep discovering obsolete `puretokens_media` instructions. Any unmanaged current, retired, or backup directory remains a preflight conflict and is never removed.

## 0.13.13 — 2026-09-03

- Replaced the user-facing Node-based Skill updater with official native platform installers: Shell on macOS/Linux and PowerShell on Windows. Installation and upgrades now download and statically validate the official `main` payload before synchronization, without requiring users to install Node, npm, Git, a package manager, or dependencies.

## 0.13.11 — 2026-09-03

- Made a current local visual-reference image use the selected model's declared multipart `image_edit` operation directly, while preserving the user's reference-image intent in the prompt. It never uploads, rehosts, invents a URL, or turns an unavailable attachment into text-only generation.
- Enforced declared image size-expression constraints: required pairs such as `width` plus `height` must be complete, and competing size expressions submit only the highest-precedence user-supplied form.
- Made video operation selection explicit for first-frame requests (`image_to_video` and its declared field, including Wan `first_frame_image`) and for declared `generate_audio` intent. An attached image with no stated first-frame/reference role now asks before billing.

## 0.13.10 — 2026-09-03

- Synchronized the installed media profiles with the current Pure Tokens base catalog. `gpt-image-2`, `seedream-5.0-pro`, `nano-banana-2`, `nano-banana-2-lite`, and `nano-banana-pro` now expose their declared native `image_edit` operations.
- Made `gpt-image-2` the default for an unspecified current local image-edit request as well as normal image generation. The Skill still sends attachments only through the declared operation and never switches models automatically.

## 0.13.9 — 2026-09-02

- Made an untransportable current local reference image fail closed. If the selected model does not declare a compatible native multipart reference transport, the Skill stops before any paid request and requires either the user's public HTTPS URL or a new explicit text-only request that ignores the attachment. It never inspects, summarizes, recreates, uploads, searches for, or silently converts that image into a text prompt.
- Made confirmed native-media delivery a hard turn boundary. After delivering the requested result, the Skill returns one completion receipt and stops; it does not inspect the output, search history or the workspace, invoke unrelated Skills, or start another media task without a new explicit user request.

## 0.13.8 — 2026-09-01

- Closed the WorkBuddy standard-input bypass in the direct runtime: every WorkBuddy POST now rejects `--json-stdin` and `--multipart-stdin` at argument validation, so only bounded Base64 request-body arguments are accepted.
- Made native video attachments route-locked. A declared image/video/audio operation must use its declared multipart path and field; it cannot degrade into a text-only JSON video request. For Seedance 2.0 Mini reference-image video, the required route is `reference_image_video` with 1–9 `reference_images` multipart files. If the host session cannot provide the current attachment bytes, the Skill stops before billing.

## 0.13.7 — 2026-09-01

- Fixed WorkBuddy media submissions that stalled before any API call: its Bash route now passes each bounded JSON request body or multipart descriptor as a canonical Base64 argument, not through standard input. The runtime decodes it only in memory and continues using the same fixed authenticated API path.

## 0.13.6 — 2026-09-01

- Made an unknown media submission a hard response boundary: after a started POST has unusable host output and no verified task ID, the Skill returns one receipt and ends the response without additional tools, status reads, polling, or prolonged follow-up work. A later continuation without an ID now requires explicit confirmation of a new paid request.

## 0.13.5 — 2026-09-01

- Simplified the client-download Agent prompt to one copyable sentence pointing to the official Pure Tokens Skill repository.
- Hardened the installation contract: the freshly cloned official `main` checkout is read-only. If validation fails or the checkout becomes dirty, installation stops; it must never run generators, docs sync, formatters, repair commands, Git writes, or any other source-mutating command to make the checkout pass.
- Prevented duplicate paid media submissions after a started POST has an empty, truncated, malformed, or otherwise unusable runtime result. The Skill now treats that outcome as unknown, retains only a verified returned task ID, and never creates a replacement task automatically.
- Fixed WorkBuddy credential resolution for its per-model Pure Tokens `/v1/...` resource URLs. The runtime still accepts only the fixed API origin with no query or fragment, requires one unambiguous credential, and sends requests only to fixed allowed API paths.

## 0.13.4 — 2026-09-01

- Added a deterministic model-access-group recovery path. When an exact requested image or video model is not returned by the current authenticated catalog after the permitted diagnosis, the Skill directs the user to select a group containing that model in Pure Tokens client configuration, ensure the managed key covers the selected groups, verify and apply it, then start a new host conversation and explicitly retry.
- The Skill does not guess a group name or claim model-to-group membership unless an authenticated API response explicitly provides that mapping.

## 0.13.3 — 2026-09-01

- Extended the managed fixed-endpoint credential runtime to every supported host with an approved, auditable local connection contract: Claude Code, Codex, WorkBuddy, Gemini CLI, Grok Build, and OpenCode. Each resolver accepts only an exact Pure Tokens configuration match, keeps one unambiguous credential only in memory for an allowed fixed API request, and never displays, persists, or requests it.
- Updated all API-facing Skills, installed contracts, host matrix, and validation to use those six managed runtimes consistently for API identity, catalog, balance, Images, Videos, same-task status, and native-media delivery.
- Marked Trae explicitly as the single manual-credential-setup exception. Its product contract has no approved local credential resolver, so the Skill stops safely instead of scanning or inferring Trae user state.

## 0.13.2 — 2026-09-01

- Added the verified WorkBuddy managed direct-runtime binding. It narrowly resolves only a matching `https://api.puretokensx.com/v1` model credential from WorkBuddy's model configuration in memory, uses it only for the fixed allowed API paths, and never displays, persists, or requests it.
- Made the host matrix explicit: all seven hosts remain installable, while only Grok Build and WorkBuddy are verified for authenticated direct media execution. Other hosts now stop safely instead of implying that generic HTTP requests inherit their chat-model credentials.
- Added bounded declared multipart attachment submission for the verified runtime and hardened native media delivery: fixed request-path allowlists, regular-file-only attachments, attachment count and size limits, content-type checks, bounded output, no overwrite, and cleanup of partial failed downloads.
- Removed contradictory wording that said a Skill could not read configuration while its managed runtime performed the narrow in-memory credential binding. The Skill still never displays, compares, or reports host configuration.

## 0.13.1 — 2026-09-01

- Added the managed Grok Build direct API runtime. It narrowly resolves the configured Pure Tokens model credential only in memory, sends it only to fixed allowed Pure Tokens API paths, and never prints, stores, or requests the credential.
- `sync`, `install`, and `upgrade` now install or atomically upgrade the managed `.puretokens-runtime` beside the specialist Skills, refusing an unmanaged runtime conflict before changing Skill directories.
- Replaced the incorrect assumption that every host automatically injects a configured model credential into generic HTTP requests. Grok Build Skills now use the managed direct runtime rather than generic Fetch/WebFetch for API identity, media catalog, balance, Images, Videos, task status, and media content reads.

## 0.13.0 — 2026-09-01

- Standardized the six installable Skill names and directories to Agent Skills kebab-case: `puretokens-balance`, `puretokens-connection`, `puretokens-models`, `puretokens-image`, `puretokens-video`, and `puretokens-update`.
- `sync` now moves verified retired managed Skill directories into a recoverable hidden backup before installing the current directories; it still refuses to alter an unmanaged conflict.
- Declared and validated the current seven supported hosts only: Claude Code, Codex, WorkBuddy, Gemini CLI, Grok Build, OpenCode, and Trae. Each uses a manual source install in its documented global Skill directory.
- Removed retired delivery mechanisms and combined-media-Skill descriptions from the active product surface.
- Made the direct execution contract explicit for every supported host: existing runtime authentication, fixed full-URL HTTPS requests, JSON task responses, same-task status reads, and native media-byte delivery. Images and videos have no alternate execution path.

## 0.12.3 — 2026-08-31

- Aligned image and video task handling with the public media contract: `reconciliation_required: true` takes precedence over lifecycle state, stops ordinary polling, retains the same task ID, and never triggers a replacement task or inferred refund.
- Status-read HTTP `429` honors a valid in-budget `Retry-After` and continues only the same task; non-429 server, transport, and timeout reads remain bounded stop conditions.
- Split profile-gated image editing by transport: public URL edits require an exact declared JSON `image_edit` operation, while native attachments require the exact declared multipart operation. A reference field alone never authorizes an edit or silent downgrade.

## 0.12.2 — 2026-08-31

- Standardized safe image and video failure receipts with phase, returned HTTP status when available, sanitized user-facing message, actionable next step, and rate-limit wait time when supplied.
- Explicitly prohibit raw error bodies, upstream identifiers, internal URLs, stacks, request data, credentials, and user media from failure output.

## 0.12.1 — 2026-08-31

- Improved image request interpretation: distinguish text-to-image, visual-reference, and edit requests; ask when an input image role is ambiguous; and preserve the user's stated intent when preparing a concise generation brief.
- Clarified that `n` is for variants of one brief only. Distinct requested assets are not silently packed into `n`, dropped, or split into multiple paid submissions.

## 0.12.0 — 2026-08-30

- Normal text-to-image and text-to-video submissions use installed versioned model selection and submit directly without a media-catalog preflight request.
- The live media catalog is read only on demand for explicit discovery, an installed-profile gap, or one post-rejection diagnosis. A failed catalog read cannot block valid core generation or trigger automatic resubmission.
- Hardened asynchronous task handling: normalize only declared top-level task IDs, stop on unknown states, preserve an uncertain submission outcome, and never automatically resubmit.
