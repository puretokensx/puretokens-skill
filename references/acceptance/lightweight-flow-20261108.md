# Lightweight installation and media flow, 2026-11-08

## Scope

This follow-up starts from main at fd40a14, after the v1.0.2 release. Only
puretokens-skill changes. No version bump, push, publication, real-client
installation, credential access, paid request or Switch/Web/gateway change
is part of this task. Package version remains 1.0.2; these distribution
changes require a new version before any future release.

## Changes and bounded benefits

- After an explicit browser-route HTTP 502/503/504 and one verified official
  asset API fallback, remaining pinned assets use that route directly. In
  the three-asset Shell fixture this reduces requests from seven to five.
  Every asset retains unique identity, exact URL, size and digest checks.
  A failed API request stops without another route or retry.
- New fetch accepts the same invocation's officially obtained bootstrap
  metadata, validates its bound and its own released bytes, and downloads
  the three remaining assets without another metadata lookup or browser
  probe. This count excludes the caller's metadata and fetch downloads.
  Supplied metadata is not a historical cache or a substitute for
  pre-execution verification. Published v1.0.2 and older fetch do not
  accept the new option; the documented old invocation remains valid.
- Windows metadata paths reject drive-relative, root-relative, UNC and
  device-namespace forms before reading. Regular-file, non-link and
  2 MiB bounds still apply.
- Image/video/audio entry instructions shrink from 37,989 to 28,939 UTF-8
  bytes (23.82%). The update entry shrinks from 11,873 to 9,456 bytes.
  These are reading-size measurements, not latency or token benchmarks.
- Normal requests load only needed model and operation guidance, without
  update/init/balance/diagnostic preflights. Replies prioritize actual
  artifacts or text; failures provide one evidence-based next action.
  Preserve host binding, explicit paid intent, unknown-submission stops,
  bounded waiting, same-task recovery and handoff-before-acknowledgment.
- Video recovery documentation now preserves an explicitly requested,
  reviewed tail-frame index instead of incorrectly limiting every task
  to index zero. No execution protocol or supported model is added.

## Verification

- Final npm run check exited 0: repository/generated guidance validation,
  acceptance consistency, six executor artifact identities, Go tests and
  go vet passed. Node: 266 tests, 262 passed, zero failed, four skipped
  because pwsh was not on the normal test PATH.
- Separately downloaded official PowerShell 7.6.9 for macOS arm64 to a
  task-owned temporary directory, verified asset size and SHA-256, and
  ran test/support/windows-download-diagnostics.ps1 successfully. This
  covers 13 path cases, 14 failure categories, explicit continuation,
  timeout selection, nine fallback cases including route reuse/failure,
  and six bootstrap cases. The temporary package/runtime were removed.
  This is PowerShell-on-macOS fixture evidence, not Windows 5.1 or real
  Windows filesystem/network acceptance. Other skipped tests stay skipped.
- Regression tests observed the old seven-request path and unsupported
  bootstrap option fail before implementation, then pass after the fix.
  Invalid metadata, damaged bytes, missing/modified installed files and
  incomplete transactions remain fail-closed.
- Independent read-only installer review has no remaining actionable
  finding after the Windows path correction.
- Offline fresh-context agent scenarios passed: image-short-path used
  two instruction reads, no questions, one simulated POST, two simulated
  API reads and one attachment handoff. redeliver-completed-image used
  two instruction reads, no questions, no POST/API reads and one handoff.
  Their entry snapshot SHA-256 was
  857110c4f6aa54e45bfff72bd7c897a31a3f50d90a911be9e335d17a8fc72186.
  These scenarios precede the final recovery-reference wording refinement;
  final-source static regressions pass. Task-owned scenario copies were
  removed. They are not live model, billing or attachment-playback proof.

## Remaining boundaries

The 268 pending host/OS acceptance summaries are unchanged. No real-host
generation or end-to-end speed benchmark was run. No user runtime, hidden
credential cache, background service, automatic resubmission, additional
user confirmation or cross-root migration was introduced.
