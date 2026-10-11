# Native video executor commands

Task IDs are opaque and must be preserved exactly, including dots in model-labelled IDs and colons in legacy output IDs. Never remove punctuation, derive a model from the ID, or construct a replacement ID. The executor accepts 1-256 ASCII characters: a letter, digit, `_` or `-` first, then those characters plus `.` and `:`. URLs, paths, whitespace and percent-encoded input are not task IDs; only the executor encodes the validated ID for the request path.

Resolve the executable from this Skill's absolute directory: sibling `.puretokens-executor/puretokens-api` on macOS/Linux; `.puretokens-executor/puretokens-api.exe` on Windows. Use the current host ID. All commands are one-shot; no user runtime is required.

```sh
"/absolute/skills/.puretokens-executor/puretokens-api" submit --host codex --request "/absolute/request.json"
```

```powershell
& "C:\absolute\skills\.puretokens-executor\puretokens-api.exe" submit --host codex --request "C:\absolute\request.json"
```

Create UTF-8 JSON using the host file tool; use escaped backslashes or forward slashes for Windows paths. Remove the request file after the command. Never put credentials in requests or prompts on the command line.

Generation:

```json
{"kind":"video","operation":"generate","model":"minimax_h3","prompt":"A slow camera movement over a mountain lake","parameters":{"duration":5}}
```

Current local first-frame image:

```json
{"kind":"video","operation":"generate","media_operation":"image_to_video","model":"minimax_h3","prompt":"A gentle camera movement","attachments":[{"field":"first_frame_image","path":"/absolute/current.png"}]}
```

Public HTTPS first-frame URL for this model (JSON, without the multipart-only `media_operation`; do not download or rehost it):

```json
{"kind":"video","operation":"generate","model":"minimax_h3","prompt":"A gentle camera movement","parameters":{"first_frame_image":"https://example.com/current.png"}}
```

`submit` returns the task ID immediately; it never polls or downloads. Do not repeat a submission with unknown output. `status` reads once and `wait` performs one bounded window. Carry the original operation as `original_operation`, model, confirmed count and safe parameters into a same-task request:

```json
{"kind":"video","original_operation":"generate","task_id":"RETURNED_ID","model":"minimax_h3","requested_count":1,"parameters":{"duration":5}}
```

After a completed same-task receipt, `content --host codex --request <file>` downloads one index:

```json
{"kind":"video","original_operation":"generate","task_id":"RETURNED_ID","model":"minimax_h3","requested_count":1,"parameters":{"duration":5},"task_status":"completed","index":0,"output_dir":"/absolute/existing/output-directory"}
```

A video task has one output, index 0. Attach the returned `downloaded_paths` file; video has no next index. Downloading is not delivery. Reuse requires a matching SHA-256, byte count and media type in an explicit task record. Without that proof, preserve any existing file and select another output directory for this same task index.

## Choose the next command

| Latest result | Next action |
| --- | --- |
| Accepted and pending | Return the receipt, then use `wait` (`resume` with a record); preserve the returned wait timestamp. |
| Completed | Use `content` for the confirmed index and an existing absolute output directory. |
| Downloaded | Hand the file to the user; with a record, acknowledge using `delivered` only after actual handoff. |
| Normal window ended | `ok=true`, still processing. Preserve `wait_windows_completed`; `next_step=wait` permits one more foreground window while delivery is authorized. At two windows, pause and ask. |
| Deferred retry or actual status failure | Preserve the ID and retry timestamp; offer explicit later continuation. Do not POST again. |
| Reconciliation required | Stop automatic waiting; an explicit recorded `resume` reads status once. |
| Output conflict or missing proof | Preserve files and fetch the same index into another output directory. |
| Submission unknown with no ID | Stop; records cannot discover the missing task and must not trigger another POST. |

## Continuation record

Use `--record <absolute-task-json>` in an explicit workspace/user location by default for video. Choose before submitting; an existing unrecorded task may continue with `--request` alone, never a second submission to switch modes. Choose the output location with `content --output-dir <existing-absolute-directory>`; it can also be retained from an explicit submission output_dir. The record contains task ID, kind, model, original operation, requested count, validated safe parameters, progress and per-index download proofs (`sha256`, `bytes`, `media_type`); no prompt, credentials, reference URL or media bytes. Do not edit it by hand.

```text
<executor> submit --host codex --request <request-file> --record <absolute-task-json>
<executor> resume --host codex --record <absolute-task-json>
<executor> content --host codex --record <absolute-task-json> --index 0 --output-dir <existing-absolute-output-directory>
<executor> redeliver --host codex --record <absolute-task-json> --index 0
<executor> delivered --record <absolute-task-json> --index 0
```

`status`, `wait` and `resume` accept an existing `--record` instead of `--request`; never combine both. `resume` uses only the recorded task: reconciliation requires one status read; a completed record without reconciliation verifies local delivery files; otherwise it performs a bounded wait. Call `delivered` only after the host actually hands off that downloaded file. Records and output files are explicit user artifacts, retained or cleaned according to the user's chosen location. A record cannot recover an unknown submission that returned no task ID.

Across computers or operating systems, keep the original record unchanged and
use the current explicitly selected host's own connection. Never copy credentials.
If old paths are unavailable, choose an existing absolute directory on this
computer with `content --output-dir`; retrieve only the undelivered index 0 of
the same task. A fully delivered record returns `done` without downloading again. When the user explicitly asks to receive that file again, use `redeliver --host <host> --record <record> --index <index>`; ordinary `content` does not re-deliver a historically delivered index.
Missing legacy download proofs do not authorize overwriting old files or
inventing proof: preserve them and retrieve the same index into another directory.

## Explicit validation

`preflight --host codex --request <file>` checks the requested model parameters and attachment representation without a POST. Use it only for an explicit check; normal generation validates during submit. It is not a price quote or a guarantee of permission, balance or media delivery. A profile gap may require one catalog GET.

Existing local type/range/enum restrictions are not automatically relaxed by a live catalog read. To inspect a changed declaration, explicitly query `models` for that exact ID. Catalog queries do not refresh installed files; install a compatible stable release only when the user requests an update, then validate against its profile. If no compatible release exists, report the pending adaptation. Never edit managed profiles or resubmit an existing task to bypass validation.

Machine receipts preserve available context with `original_operation` separate from the invoked command. `retry_not_before` is an RFC3339 lower bound for the next same-task read; preserve it across continuation. An omitted continuation count is unknown, not one; video tasks only allow one output. Report `submission_outcome: unknown` as uncertainty; never automatically resubmit. Show API codes only if actually returned, and retain only sanitized error detail. Keep user-facing updates to task ID/status, actual artifact delivery or the needed corrective action.

A record requiring reconciliation is an exception to bounded waiting: an explicit `resume` reads its status once, keeping the same task ID and honoring Retry-After. Continue from the new receipt; never clear the flag by hand or resubmit.

Always retain `reconciliation_required` and the exact `retry_not_before` when present. Without a record, carry these fields from the latest receipt into each same-task request. A request file is temporary; the receipt or optional task record supplies continuation metadata after that file is removed.

## Foreground continuation and delivery

Machine `next_step` is one of `wait`, `content`, `deliver`, `await_user`, `done`. It never authorizes a new submission or bypasses user cancellation or host policy. A normal bounded wait ending returns `ok=true`, the actual pending status and `wait_outcome=window_ended`. Carry `wait_windows_completed` (0..2, saturated) into every same-task request; records preserve it automatically. At count 1, the active authorized delivery may continue one more foreground window without asking again. At count 2, pause and offer explicit continuation. An explicit later resume allows one bounded window but does not reset that counter. No background timers or queues.

`wait_outcome=retry_deferred` means an existing retry timestamp cannot fit the current window: preserve it and offer later continuation. It does not identify whether the timestamp came from the API or the executor's initial delay. A 429 inside a window may recover only when its positive Retry-After fits the remaining budget. A returned failure receipt (including exhausted 429), 5xx/network timeout, unknown status or reconciliation stops host automatic continuation. Older receipts/records may omit these fields; retain the same task and never submit to upgrade its state.

A failed attachment handoff does not call content again: for a completed record with `reconciliation_required` not true, run `resume --host <host> --record <file>` to verify and expose its existing file. This completed-record path is local-only and does not resolve credentials or read the API. Then hand off the returned verified file. If no valid file is exposed, preserve existing files and retrieve the same index into a different directory. Across commands, use the recorded digest proof; without a record, reattach only the unchanged file from this active conversation's download receipt. If that identity is uncertain, preserve it and fetch the same index into another directory. Never mark delivered before the actual host handoff.

If submit output is unknown, only local cleanup of this command's temporary request file is allowed before stopping. Cleanup must never issue another API command, repeat POST or discard the task record.

## Support evidence

Normal JSON receipts include an allowlisted `support` block for this invocation.
It adds no request, config read, export, upload or persistent record. Only share
that block when the user requests troubleshooting information for private
support; do not publish the full receipt. See [failure-guide.md](failure-guide.md).
Its `request_id`, when present, correlates the latest HTTP response, not the
whole task. It is neither a task ID nor an idempotency key and cannot recover
an unknown submission. Older executors may omit this block; do not rerun a
paid request to obtain it.

## Explicit re-delivery and historical failures

`redeliver` requires an existing absolute record and explicit user intent to receive
one previously delivered index again. It verifies the recorded local file first,
without credentials or network. If missing, it may retrieve that same task/index;
changed or conflicting files are preserved, so select another existing directory
with `--output-dir`. Never resubmit or edit record proofs to recover a file.
`pending_redelivery_index` preserves this request across interruption; `resume`
continues it without inventing another one. Historical delivered indexes remain
unchanged. A `redelivery_index` receipt still needs actual new host handoff before
`delivered --record <record> --index <index>` clears the pending marker.

Terminal failure records retain only allowlisted classification in `last_failure`.
A recovered `failure_source=task_record` describes that historical failure, not a
fresh API request. Legacy records with no evidence remain cause-unknown; do not
assume they lack a permission, quota or content restriction to recommend a model
switch. A transient failed status GET must not replace terminal cause evidence.
