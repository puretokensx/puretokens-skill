# Audio Skill and gateway contract

Reviewed 2026-10-21. Owners: `puretokens-skill` owns `puretokens-audio`, the
one-shot executor, reviewed model/voice profiles, local validation and host
attachment handoff. `PureTokenPlus_web` owns API routing, upstream adapters,
authorization, billing and its separate account-session audio workspace.
The paired contract is `docs/product/audio-workspace-integration.md` in Web.

## Fixed API-key protocol

| Operation | Fixed POST URL | Reviewed exact model IDs | Response |
| --- | --- | --- | --- |
| `speech` | `https://api.puretokensx.com/v1/audio/speech` | `stepaudio-2.5-tts`, `step-tts-2`, `step-tts-mini` | Complete MP3/WAV bytes |
| `transcribe` | `https://api.puretokensx.com/v1/audio/transcriptions` | `stepaudio-2.5-asr`, `step-asr`, `stepaudio-3-asr-max`, `stepaudio-2-asr-pro` | Final JSON `text` |
| `generate` | `https://api.puretokensx.com/v1/audio/generate` | `stepaudio-3-gen-preview` | Complete MP3/WAV bytes |

One invocation makes one synchronous POST with a 90-second deadline, no redirect,
retry, task polling or substitute model. It privately reuses the bound host's
verified Pure Tokens connection. API-key `/v1` routes must not be replaced by
account-session `/pg` or `/api/product/audio` routes. SSE adaptation for ASR
Max/Pro belongs to the gateway: the client requires one final JSON response.

StepFun speech sends `model`, unchanged `input`, a model-specific reviewed `voice`,
`response_format`, optional `speed` (0.5–2), and optional `instruction` only for
TTS2.5. The local text limit is 1000 Unicode characters, instruction limit 500.
StepFun sound sends only `model`, `instruction` (max 500), `response_format`, fixed
`task=text_to_audio` and `stream_format=audio`. Transcription sends only multipart
`model`, `response_format=json`, and the original MP3/WAV/OGG file, at most 8 MiB.
It does not send local paths, original filenames or upload to another service.
The request file is bounded to 64 KiB; no implicit paid chunking or conversion.

Local request errors include a controlled `validation` object (`field`, `reason`,
optional numeric `limit`). It never echoes input values, paths or unknown field
names. `not_submitted` permits correcting a construction error before the first
POST, but never silent truncation, voice substitution or extra paid chunks.
Combining speech and music produces independent files; a required mixed track
needs clarification before either paid operation because mixing is unsupported.

`runtime/executor/audio-profiles.json` is the reviewed source embedded in the
executor. `npm run docs:sync-guidance` generates the installed compact index and
one file per model. Changes to voices, fields, limits or IDs must synchronize
that source, schemas, native validation and the paired gateway contract/tests.
TTS3 has no reviewed preset voice mapping and remains excluded. Cloning,
translation, streaming input and real-time conversation are not implemented.

## ElevenLabs synchronous expansion (candidate 0.19.6)

These are source-reviewed API-Key contracts, not production availability,
model/voice entitlement, paid acceptance or host playback evidence. New IDs are
absent from the anonymous production snapshot dated 2026-10-21. Ordinary calls
still read only the selected profile, never a catalog preflight.

| Exact IDs | Executor operation | Fixed route and fields |
| --- | --- | --- |
| `eleven_v3`, `eleven_v4`, `eleven_multilingual_v2`, `eleven_flash_v2_5` | `speech` | `/v1/audio/speech`: model, original input (4096 Unicode characters), explicit entitled voice ID, response_format=mp3; no speed/instruction/WAV |
| `scribe_v2`, `scribe_v2_medical` | `transcribe` | `/v1/audio/transcriptions`: original multipart file, model, response_format=json |
| `eleven_text_to_sound_v2` | `generate` | `/v1/audio/generate`: instruction (1000 characters), task=text_to_audio, stream_format=audio, MP3, required duration_seconds 0.5–30 |
| `music_v2_5` | `music_sync` | `/v1/audio/generate`: instruction (1000 characters, including exact lyrics), task=text_to_music, stream_format=audio, MP3, required duration_seconds 3–600 with millisecond precision, explicit instrumental boolean |
| `eleven_multilingual_sts_v2` | `voice_change` | `/v1/audio/generate`: explicit entitled voice ID, task=voice_change, stream_format=audio, MP3; executor encodes original bounded local file as raw Base64 audio plus audio_format |

Voice IDs use the gateway's ASCII `[A-Za-z0-9_-]{1,120}` syntax; syntax is not
entitlement. Never inherit a StepFun default voice or guess permission from a
display name. Transcription/voice-change inputs retain the 8 MiB MP3/WAV/OGG
boundary; the gateway measures the five-minute limit from original bytes.
No audio URLs, caller-provided Base64, implicit conversion or separate upload
service. All outputs retain the existing 16 MiB structure verification,
exclusive file save, artifact receipt and local-only audio-verify semantics.

Synchronous music never enters kind=music, music task polling or the asynchronous
submit route. It has no task record or remote recovery. The 90-second deadline
may expire on long music; report unknown and never resubmit automatically.
No BFF UUID deduplication/history guarantee is inherited. Explicit false and
fractional durations are preserved. Every model operation is one POST with the
same no-retry policy; lyrics are never silently truncated or split.

## Asynchronous music

The same Skill additionally supports `stepaudio-3-music-preview` through the
executor's task commands, not synchronous `audio` or `audio-verify`.
Web patch `073-api-key-music-tasks.patch` must be deployed before enabling it:

- `POST https://api.puretokensx.com/v1/audio/music/submit` accepts exactly
  `model`, `caption` (nonempty, up to 1000 Unicode characters), explicit boolean
  `instrumental`, optional `lyrics` (up to 4000), and `format` (`mp3`/`wav`).
  Instrumental true excludes nonempty lyrics. Preserve caption/lyrics exactly.
  Reject unknown, duplicate and null fields and bodies over 64 KiB. No client
  group, duration, voice, reference attachment or batch-count field is supported.
- TokenAuth and normal model/channel admission select the authorized group.
  Music retains the existing fixed-price or literal request-priced expression,
  task reservation, settlement and reconciliation; the API adds no price override.
- Successful submission returns only public `id` and `status`. An ambiguous
  submission with durable identity returns `status=unknown` and
  `reconciliation_required=true`. Stop automatic continuation and never retry
  submission, including alternate channels. Without a returned ID there is no
  lookup or idempotency guarantee.
- `GET https://api.puretokensx.com/v1/audio/music/tasks/{id}` returns public
  identity and status; `/content` returns private MP3/WAV bytes, at most 32 MiB.
  Both check account ownership, originating token ID and current model limits.
  Another key, including another key on the same account, cannot read the task.
  Status refresh failure returns 503; unavailable/expired content returns 410.
  Content is retained for 24 hours from artifact storage, not submission.

Executor requests use `kind=music`, `operation=generate`, `prompt` for caption,
`parameters={instrumental,response_format,lyrics?}`, and an existing absolute
output directory. Local validation precedes credential resolution. Submit once
with a new explicit absolute task record, return immediately, then use
status/wait/content/resume/delivered. Each wait window allows at most seven reads
and 300 seconds; automatic delivery permits two windows. Pending exhaustion is
not failure. Unknown/reconciliation/HTTP failures stop automatic continuation.
Only index0 exists; records contain safe model/format/instrumental/progress and
download proof, never caption or lyrics. Resume verifies and reuses a downloaded
file without credentials/network; mark delivered only after actual host handoff.
Changed/missing files and expiry stop without regeneration or refund claims.

API-key callers do not inherit BFF UUID deduplication or account-session history.
The existing `/pg` music workflow remains available to the Web product. Both
repositories must update this contract, profiles, route tests and the replayable
patch when changing public behavior. Authorization checks follow OWASP API
Security Top 10 (2023), API1 Broken Object Level Authorization; fixtures cover
cross-user, cross-token, absent authentication and model-limit rejection.

## Synchronous result, failure and delivery

Generated audio is bounded to 16 MiB, checked for MIME and complete MP3 frames or
PCM/float WAV structure, and saved without overwriting a user file in an explicit
output directory. OGG input requires a single complete Opus (mono/stereo mapping
family0) or Vorbis stream with page CRC/sequence and codec headers. These checks
do not decode playback or prove audible quality. Transcription requires final
JSON (max 128 KiB), a nonempty `text` (max 100000 Unicode characters), and no error.
Only the safe text projection is returned; arbitrary upstream fields are omitted.

Binary receipt: `submission_outcome=accepted`, `next_step=deliver`,
`delivery_status=downloaded`, and `artifact={path,sha256,bytes,media_type}`.
Host attachment handoff remains required. The local `audio-verify` command checks
an explicitly saved artifact proof and returns the same file for attachment;
it never reads credentials, uses network or generates new audio. Missing or
changed files stop. No synchronous-response recovery or unknown-request lookup
exists. Receipts must not claim charge amounts or refunds.

`not_submitted` means local validation/credential resolution stopped before HTTP.
A complete 4xx response is `rejected`; network/read/timeout/5xx/invalid 2xx response
is `unknown`. Valid audio received but not saved is `accepted` with a content
failure. All stop automatic continuation; no duplicate POST. Support summaries
exclude input, transcript, file paths and output bytes. Explicit artifact receipts
are user/workspace files, not a credential or media cache.

## Model discovery and evidence

Only explicit audio discovery uses one authenticated `GET /v1/models`, intersected
with exact reviewed IDs. Operation filtering uses reviewed local profiles; the
returned directory supplies visibility only, not input schema, live route
acceptance, price or authorization proof. Ordinary generation has no discovery
preflight. Offline usage/voice questions read local profiles without credentials.

Protocol sources: Web `upstream-new-api-src/router/relay-router.go`,
`relay/channel/openai/adaptor.go`, `relay/channel/openai/stepfun_asr.go`,
`relay/helper/valid_request.go`, and its paired audio-workspace document;
StepFun official TTS guide and audio generate/transcription references listed in
`runtime/executor/audio-profiles.json` and the Web contract.

Local fixture tests, request/receipt schema conformance, behavior scenarios and
platform builds establish engineering behavior only. Paid provider success,
rejection/billing, production deployment and actual host attachment/playback
remain unverified. Do not relabel them passed based on this integration.
