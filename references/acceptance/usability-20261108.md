# Installation and execution usability, 2026-11-08

## Scope

Only puretokens-skill is changed. Preserve the existing uncommitted client,
credential, model and multimedia work. This task does not install into a real
client, read production credentials, submit paid requests, modify Switch/Web,
bump the version, push or publish. Source version remains 0.19.8.

## Implemented

- Same-version integrity verification remains archive-free, write-free and
  init-free. It now reports init_status=not_run and the current host's compact
  activation guidance with one example. Already-loaded Skills are usable in the
  current conversation; refreshing/enabling or a new conversation is conditional.
- Installation init keeps its separate 20-second budget. Successful file
  installation and connection verification remain distinct outcomes.
- Fetch uses 45-second metadata/selector and 180-second archive limits, safe
  stage-start progress and no retries. Shell curl also has a 15-second connection
  limit and 30-second low-speed detection. PowerShell uses TimeoutSec and disables
  expensive default progress rendering; it has no separate stalled-body timer.
- The update entrypoint is shorter. GIF/SVG and Grok editing detail are linked
  references, not repeated in ordinary media flow. Video index 0 and an explicitly
  requested, reviewed tail frame at index 1 retain independent proof and handoff.
- Embedding/rerank may select a unique filename within an already explicit,
  authorized writable directory. An unclear directory still requires a question;
  absolute output paths and non-overwrite checks remain mandatory.
- Fresh image/video property failures expose only controlled field, reason and
  bounded public choices. correct_request requires validation + not_submitted +
  media_request_invalid, no task identity or reconciliation, and permits one
  agent-authored assembly correction preserving user intent. Existing/unknown
  tasks, attachment/permission/transport failures and another failure stop.
- Shared workflow guidance, receipt schemas, support allowlists, usage docs and
  execution contracts agree. Detailed validation never enters support exports.
- Offline list scoring permits the narrow correction only with private fixture
  evidence of the intended parameter and unchanged host, record and other request
  fields. It rejects repeats, uncertainty, changed intent and continuation after
  another failure. Default-image evaluation fixtures now use gpt-image-2.6.

## Verification

- Completed npm run check, exit 0: repository/generated guidance validation,
  acceptance consistency, source/build proof and all six artifact hashes passed;
  Go tests and go vet passed; Node reported 220 tests, 216 passed, 0 failed and
  4 skipped because PowerShell is unavailable on this macOS host.
- The final default-model fixture alignment was subsequently verified with
  node --test test/agent-eval.test.mjs: 21 passed, 0 failed.
- New correction tests first reproduced rejection of a valid unsubmitted
  correction, then passed with the bounded evaluator change. Negative cases
  cover unknown/transmitted/existing tasks, changed model/content/attachments/
  count/format/output/record/host, extra commands, repeats and a failed correction.
- Installed shared guidance and nested references are checked by the engineering
  gate. git diff --check passed. No changes were staged or committed because
  several touched files also contain pre-existing uncommitted work.

## Evidence Boundaries

These are source, fixture, isolated installation and evaluator results, not real
host loading, authenticated API, billing, attachment opening or playback proof.
The existing host-acceptance registry still has 268 pending host/OS summary
checks and zero detailed real-host results; this task does not promote them.

No Windows execution or real slow-network benchmark was performed. Download
timeout values and reduced reading work do not establish measured end-to-end
speedups. Media validation may follow credential resolution, and a missing
installed profile may require a catalog GET: not_submitted means no generation
request, not necessarily no API request or credential read.

The runtime adds no Node/Python requirement, background service, alternate
credential source, automatic resubmission or cross-root migration. Necessary
interrupted-transaction protection and local customizations are preserved.
