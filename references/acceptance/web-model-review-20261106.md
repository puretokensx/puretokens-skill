# Web model compatibility review, publication follow-up 2026-11-07

Status: bounded local model adaptation implemented; refreshed engineering
verification passed on 2026-11-07, with four PowerShell tests skipped because
PowerShell is unavailable on this machine. This is not a release,
paid provider acceptance, real Key entitlement or client attachment handoff.
Only puretokens-skill is modified. Pre-existing Claude mapped-connection changes
remain intact and are not claimed as part of this model work.

## Evidence

- Skills baseline: main, a87f113, package version 0.19.8, six commits ahead of
  local origin/main, plus pre-existing uncommitted mapped-connection changes.
- Web source inspected: 87d26e9a, 2026-11-07. Local source identity does not
  independently prove the exact production deployment commit.
- Anonymous fixed-console catalog GET: HTTP 200, response Date
  Sat, 07 Nov 2026 08:16:26 GMT, 149 rows, no authentication or paid request.
  Raw SHA-256: fe7edd581320626dc8d29e95cfc21d48b5f4f5c435f0d62256dd70cda79f831c.
- Web metadata SHA-256:
  14dc7eff07a890d47b04090b66da5be2546fb517edbc03539123bf7697d75ab1.
- Dedicated audio and image request contracts were reviewed. Private
  implementation identities and digests are not part of this public record.
- The saved catalog is a reviewed projection, not the untouched response.
  scripts/reviewed-media-supplements.mjs applies exact-model restrictions.
  capturedAt is import time; the response Date and digest identify the GET.

## Superseded findings

The November 6 response had 134 rows and conflicting Wan visibility. That
conflict is resolved: wan3.0-video and wan3.0-video-prime are present, not
wan-3.0. The old snapshot was not installed. The previous statement that all
GPT 2.6 edits exclude GIF is superseded: flare/sunburst permit GIF edits at 1K;
only base 2.6 excludes them. Vidu currently retains generate_audio and seed.
The later November 7 snapshot includes six dedicated kling-video-* schemas.
The old claim that these IDs lack dedicated declarations is superseded.
The reviewed retirement contract also retires kling-v3
and kling-v3-turbo for new submissions; they are not retained as source-only
selection entries. Their existing task IDs and recovery remain unchanged.
The 07:56 UTC snapshot had 151 rows/49 media declarations. The final 08:16 UTC
capture has 149 rows/48 media declarations; muse-video is absent and leaves
new selection. Its absence does not establish permanent retirement. No protocol
changes were found in the remaining 48 declarations before local restrictions.

## Changes implemented

- Refresh 38 to 48 dedicated public media profiles: 22 image, 26 video.
  Installed selection contains 22 image and 27 video profiles, including
  source-reviewed viduq4-preview absent from this snapshot. Nine video profiles
  retain separately reviewed source restrictions; only viduq4-preview adds an
  ID not present in the public capture. This is not a public availability claim.
- Add 15 exact IDs: four Seedream IDs, two Gemini image IDs, three GPT 2.6
  variants, Grok image 2.0 and video 1.5 preview, Omni Flash and three Veo 3.1 IDs.
  Absent public IDs leave new selection only; no silent aliases or changes
  to old task identity/recovery. Historical fixtures remain test-only.
- Add six exact kling-video-* profiles from the later public capture. Validate
  V4 reference-only adaptive ratio, distinct bounded existing voice IDs, total
  material counts, prompt-mode limits and Omni feature/base audio/multi_shot.
  Motion control permits an empty prompt only with the required image/video
  and orientation. Exact-model guards cannot be bypassed by missing profiles.
  Retired kling-v3 and kling-v3-turbo stop new submissions and are filtered from
  discovery, even if a stale directory still lists them. No replacement aliases.
- This initial adaptation kept gpt-image-2.5-flare and MiniMax-H3; the later
  user-selected default update below supersedes the image default.
- Validate operation-specific fields, image format/size combinations, Wan
  standard/sequential limits, Seedream dimensions, combined input/output limits
  and explicit sequence count agreement. The global output cap remains eight.
  Discovery applies that same cap. Wan mode limits apply only to named resolution
  tiers; explicit pixel sizes are not incorrectly compared with those tier names.
- GIF byte validation already existed; add receipt/task-record preservation
  and enforcement of the explicit requested format on download and proof reuse.
  GIF requests cannot accept PNG/JPEG as completed output. Preserve raw GIF
  bytes, extension and frames; native file attachment may be used when preview
  cannot animate. Attachment handoff is not animation playback acceptance.
  Old wrong-format files remain untouched; explicit same-index content/redeliver
  into another directory can recover without POST or interference from another
  bad index. next_step never expands the user's currently authorized subset.
  The later scoped follow-up adds explicit safe SVG output for exact GPT 2.6 IDs;
  see Operation and output coverage below.
- Recognize Wan boolean audio, declared automatic duration and flat HTTPS
  references, not unreviewed nested inputs or idempotency claims.
- Generic JSON operations may use multipart only with explicit per-role file
  declarations and complete required inputs. Seven exact Gemini/Nano 2.1, Veo
  and Omni Flash profiles instead use reviewed local_file_data_url transport:
  original PNG/JPEG/WebP bytes are validated and encoded in memory into the
  existing JSON API, preserving images/reference_images/first/last frame roles.
  Validate the bytes actually sent rather than reopening a path. Reject changed,
  incomplete, linked, corrupt or oversized files; 14 MiB per file and 20 MiB
  encoded request. No upload service, rehosting or caller-supplied data URLs.
- Add exact GPT 2.6 JSON masked_image_edit with 1-10 declared HTTPS original
  images and one local PNG mask. Validate immutable mask bytes in memory and
  preserve the existing multipart edit. Missing installed profiles use the same
  reviewed transport projection after one catalog GET. Do not expose data URLs
  or persist originals/masks in task records.
- Merge confirmed Vidu mode restrictions while retaining previous boundaries.
  Separate Super Vega last_frame delivery is not implemented: reject
  return_last_frame=true before submission in the initial pass. The scoped
  follow-up below supersedes that restriction for four reviewed Pro models.
- StepFun TTS 2.5 accepts volume 0.1-2. ASR 2.5 / step-asr accept hotwords arrays
  and explicit text response. Other models do not inherit these extensions.
  Preserve bounded validation, credential redaction, one POST and no repeat.
- Implement separate asynchronous audio_task for gemini-3.8-flash-tts, lyria-3.5
  and lyria-3-pro-preview using the dedicated submission route in the audio
  execution contract, GET /v1/tasks/{id}, and
  GET /v1/tasks/{id}/artifacts/audio/content. Verify original ID and status
  platform, distinguish SUCCESS/FAILURE/NOT_START/SUBMITTED, preserve the task
  through waits/resume/redeliver and validate WAV/MP3 bytes up to 32 MiB.
  Gemini input, speaker names, instructions and Lyria lyrics remain private;
  task records retain no body text. These models cannot use synchronous audio
  or StepFun music routes. Audio selection now contains 21 reviewed exact IDs.

## Remaining boundaries

1. Local typed-image and asynchronous audio adapters have source and fixture
   coverage, not real Key/group, billable provider or host attachment acceptance.
   Their implementation uses existing gateway APIs; no Web, Switch or gateway
   code/configuration/deployment change was made or is requested by this work.
2. Kling advanced contents/settings/options now use a separately validated
   native_video operation; flat/native mixing is rejected. Existing resources
   may be referenced, but resource creation, voice cloning and realtime voice
   are deliberately out of scope. No server idempotency claim is introduced.
3. No paid request or installed-client execution has been performed for the new
   model IDs. Catalog visibility, installation and init do not prove these.
4. Explicit SVG and separate video tail frames have source/fixture support, not
   real-host handoff evidence. Audio translation has a generic gateway route
   but lacks a verified currently public exact-model protocol; it is not exposed.
5. Three Jev and three embedding/rerank reviewed IDs remain visible. No dedicated
   protocol change was identified; no paid acceptance is implied.

## Earlier verification, before scoped extension

These results cover the earlier GIF and JSON-mask follow-up, not the later SVG,
tail-frame and Kling native extension. Fresh extension results are recorded
under the scoped follow-up below; neither is a published or installed version.

- New validation/audio regressions failed before implementation. GIF preservation
  also failed before its fix. The explicit-pixel and discovery-count regressions
  each reproduced their failure before correction.
- Kling adaptive/reference ratio, duplicate voice IDs, prompt limits and retired
  model selection regressions failed before correction. Typed-reference tests
  cover immutable byte validation, role preservation, modified files, incomplete
  markers and symlinks. Async audio fixtures cover all three models, missing IDs,
  timeout/503 unknown submissions, status/content failure, platform mismatch,
  two bounded wait windows, local recovery/redelivery and protocol isolation.
  Both installed async audio JSON examples execute against fixtures; actual
  emitted receipts, records and support summaries pass schema validation.
- New GIF/mask regression fixtures cover bytes, wrong-format refusal, old proof
  rejection, same-index new-directory recovery, independent-index isolation,
  unchanged legacy format behavior, live profile gaps and JSON/multipart roles.
  Multi-index recovery, live mask projection and undeliverable tail-frame tests
  reproduced their failure before correction.
- `npm run executor:build`: exit 0; rebuilt darwin, linux and windows for both
  amd64 and arm64. The engineering gate verified all six artifact hashes and
  embedded source/toolchain identities against the current build proof.
- `npm run check`: exit 0. Repository/schema/generated-guidance validation passed;
  Go tests passed (9.741 seconds), and `go vet ./...` passed. Node ran 201 tests:
  197 passed, zero failed and four skipped (75.456 seconds). The skipped tests
  require unavailable PowerShell, so Windows execution remains unverified.
- The first full check found the validator's ordered acceptance-scenario list
  missing the new format/mask cases. The list and regression assertion were
  synchronized without relaxing the gate; the complete rerun passed.
- `git diff --check` passed.
- Existing real-host evidence remains unchanged: 196 summary checks pending and
  zero detailed real-host cases recorded. Cross-compilation is not Windows or
  Linux runtime acceptance. Release validation was not requested or run.
- No real credential reads, host configuration writes, paid requests, version
  bump, push or production deployment in this follow-up. Changes remain local;
  shared build and documentation files also contain pre-existing mapped-connection
  changes, which were not bundled into a new commit by this follow-up.

## Operation and output coverage

### Follow-up scope, 2026-11-07

User authorized SVG, separate video tail-frame delivery, reviewed Kling native
parameters and verification of native audio translation. Realtime speech, voice
cloning and resource creation are explicitly out of scope, not planned gaps.
Only this Skills repository may change; no paid requests or publication.
Implementation complete: bounded static SVG files, original-task tail-frame
identity and independent recovery, six-model Kling native validation. New
regressions first failed, then passed after implementation. Fresh checks:

- `npm run executor:build` passed for darwin/linux/windows, amd64 and arm64.
- `npm run check` passed: source/build-proof and generated-document gates; Go
  tests (8.261 seconds) and vet; Node 201 tests, 197 passed, 4 skipped, 0 failed
  (93.253 seconds). PowerShell is unavailable, so those four Windows execution
  tests remain unverified. Cross-compilation does not replace OS acceptance.
- SVG fixtures cover original bytes, exact-model/explicit-format authorization,
  full-200/no-Content-Range, hostile XML/CSS/resources, complexity bounds and
  verified local redelivery without networking.
- Tail-frame fixtures verify original task ID, unique key/type, ignored returned
  URL, two exact GETs, no POST, independent per-index handoff, requested_count=1
  after both outputs, and local redelivery without credential resolution.
- Kling fixtures cover all six native operations, malformed envelopes, roles,
  limits, native/flat conflicts, discovery, one fixed POST, preserved false
  settings, and removal of private contents/IDs/URLs from receipts and records.
- Actual emitted SVG/tail receipts and records, and native requests without
  duplicated prompt, pass schema validation.
- No real-host case was added: the existing registry still has 196 pending
  summary checks and zero detailed real-host results. No paid request, host
  credential read, version bump, push, release or other-repository write occurred.
  Shared files retain pre-existing work; no combined commit was created.

Audio translation audit: gateway router/relay-router.go registers translations;
ElevenLabs ValidateAudioRequest permits transcription only, and StepFun ASR
uses audio.input.transcription. The anonymous public catalog contains no
confirmed exact translation model. No translation command or ASR-plus-chat
fallback was added. No Web or gateway changes are required by this implementation.

Video requested_count remains the number of generated videos (one). An explicit
reviewed return_last_frame adds output index 1; index 0 remains the original
video. Recovery never generates or extracts a replacement frame locally.

This table describes the working tree, not a published release or a per-host
acceptance matrix. New rows have source/fixture evidence only; the existing
host acceptance registry is unchanged.

| Capability | Input and native route | Output and recovery | Current boundary |
| --- | --- | --- | --- |
| Image generation/edit | Exact installed profiles; JSON or declared multipart/local_file_data_url; Images API | Verified PNG/JPEG/WebP/GIF and explicitly reviewed SVG; task/index recovery | SVG exact GPT 2.6, complete 200, 8 MiB static validation; real-host attachment not verified |
| GPT 2.6 JSON masked edit | 1-10 public HTTPS originals + one local PNG mask; POST /v1/images/edits | Same image task contract; explicit format retained | No caller data URL or mixed local originals; no new upload service |
| Video generation/reference/edit | Exact installed profile and role; /v1/videos or /edits | Video index 0; explicit reviewed Vega Pro tail frame index 1; independent recovery | Manifest task/key/type verified; no returned URL, local frame extraction or replacement task |
| Kling advanced generation | Six exact IDs, native_video contents/settings/options; /v1/videos | Same asynchronous video record and native file | Strict roles/counts/settings; existing IDs only; no resource creation or flat mixing |
| Synchronous audio | Reviewed speech/transcription/sound/voice-change/music_sync; /v1/audio/* | Verified audio or transcription text; local artifact re-delivery | One POST/90 seconds; no remote polling/retry or inferred model entitlement |
| StepFun music | /v1/audio/music/submit and same-task GETs | MP3/WAV up to 32 MiB; same-task recovery | No protocol mixing with synchronous music |
| Gemini TTS/Lyria | Dedicated audio submission contract and original ID/platform GETs | WAV/MP3 up to 32 MiB; same-task recovery | Three exact reviewed models; no realtime, PCM or voice cloning claim |
| Jev | /typesafe/v1/systemone | Typed choice/score/noul text | Synchronous; no media polling or invented remote recovery |
| Embedding/rerank | Explicit bounded text; /v1/embeddings or /v1/rerank | Ordered JSONL vectors or ranking JSON file | Dedicated tasks, not chat-model aliases |
| Audio translation | Generic gateway route, no verified exact public model | Not exposed | Await native model contract, not ASR plus chat |
| Voice cloning/realtime/resource creation | Excluded by user scope | No Skill command | Not a planned compatibility gap |

SVG validation and separate video artifact identity, task-record and index
semantics are implemented as described above; real-host handoff remains
unverified. Resource creation, voice cloning and realtime interaction remain
explicitly out of scope. No other repository was changed or presumed in need
of redeployment.

## Live catalog recheck, 2026-11-07 12:17-12:19 UTC

This follow-up used anonymous read-only GETs without credentials, cookies,
redirects, submissions or retries. No models, defaults or runtime parameters
needed changing. The existing installed selection was retained instead of
rewriting identical generated files or advancing the catalog import timestamp.

| Fixed console path | Response Date (UTC) | Rows | Raw SHA-256 |
| --- | --- | --- | --- |
| `/api/product/docs/model-catalog` | 2026-11-07 12:17:00 | 149 | `38b7cdeebfec64008f69c1a489aa99b5f30bc5748992d43250a389ca92098d2f` |
| `/api/product/audio/transcriptions/models` | 2026-11-07 12:18:47 | 6 | `3617884c529c07a21bb934e419f8a2e87da3f5a447926055e94664de10bcb249` |
| `/api/product/audio/generations/models` | 2026-11-07 12:18:48 | 3 | `dbb29b622f637a3568584d730e6894847f86102a795358d72e4cb5bdb99cdb55` |
| `/api/product/audio/music/models` | 2026-11-07 12:18:48 | 2 | `8deb736cc715d0e8380b80616231d5b2ffc9f697dd90770e67d534160d379d81` |
| `/api/product/drama/speech/models` | 2026-11-07 12:18:49 | 3 | `7a904c3dc457088d49c64aaa71623d364b0348406e490e9251d3cfadd43224c5` |

All five requests returned HTTP 200. The base catalog's 48 image/video models
(22 image, 26 video) produced no added IDs, removed IDs or field/schema changes
after the existing reviewed projection, compared with the saved import at
2026-11-07T09:20:36.335Z. This is a comparison of reviewed installed parameters,
not an assertion that unsupported raw fields or all catalog metadata are equal.
The source-only `viduq4-preview` remains separate from public availability.

An additional base GET at 12:19:41 UTC (SHA-256
`06b0f78bc5984006ae72afe1e0d3864da22d52895b22aa31a10b6a74b437e3a8`)
confirmed visibility of all 21 reviewed audio IDs, all three Jev IDs and the
three reviewed embedding/rerank IDs. The four public audio workspace lists
exposed 14 distinct reviewed IDs with no missing Skill profile. They did not
expose the four ElevenLabs speech IDs or the three Gemini TTS/Lyria IDs, even
though those seven remain in the base catalog. Neither list establishes
current-Key entitlement, active provider routing, billing or playable output.

The speech catalog was re-read at 12:19:42 UTC (SHA-256
`27cffebe08d12bb484d4068bfae6c57ea9d6b817379cdcb6b4192f8bba178a6b`).
Its three exact StepFun IDs match installed formats, 1000-character input
limits, speed 0.5-2 and all preset voice IDs (36/36/27); TTS 2.5 volume 0.1-2
also matches. `instructionMaxLength=200` belongs to Web's account-session
composer. Source tracing confirms the dedicated API-key contract still states
the Skill's 500-character instruction limit; no API change is inferred from
that BFF-only restriction. Only `stepaudio-2.5-asr` and `step-asr` advertise
hotwords, matching the installed exact-model guard.

Web source HEAD is now `c01f3b5b`; this does not prove the production deployment
commit. Compared with the previously reviewed `87d26e9a`, the relevant new
documentation/shared classifier centralizes the same three asynchronous audio IDs
and clarifies queued submit receipts versus uppercase task lookup states. The
Skill already implements and tests those states/routes. Model metadata retains
SHA-256 `14dc7eff07a890d47b04090b66da5be2546fb517edbc03539123bf7697d75ab1`.
The reviewed dedicated request contracts are unchanged from that source
review. The public audio lists do not expose every dedicated API
parameter; source-reviewed fields retain that evidence boundary.

This follow-up corrects stale statements about tail-frame implementation and
ElevenLabs visibility. It does not add real-host or paid-provider acceptance,
publish a release, read host credentials, or modify Web/Switch/gateway files.
Verification: `npm run validate` passed, including generated guidance/profile
consistency and all six executor build hashes. The focused production-model,
reviewed-supplement, audio-contract and multimedia-guidance suite passed all
13 tests with no skips. `git diff --check` passed. No runtime source changed,
so the prior full engineering run remains separately recorded above.

## User-selected defaults, 2026-11-07

The user selected exact `gpt-image-2.6` as the image default and
`qwen3.7-text-embedding` as the embedding default. These affect new tasks only
when the user has not specified a model. Explicit alternatives remain installed
and existing tasks retain their original model. Video remains `MiniMax-H3`.
The embedding Skill writes its default into the request; the executor still
requires an explicit model and never falls back after failure.

Anonymous base catalog GET at 2026-11-07 12:29:24 UTC returned HTTP 200 and
149 rows. Both selected exact IDs are listed. Name/endpoint/capability inspection
found only `qwen3.7-text-rerank` as a rerank entry; generic OpenAI endpoint labels
do not establish additional rerank support. `voyage-4-large` declares embeddings,
not rerank, and is not newly adapted by this default-selection change.

Default-selection regression cases failed before the change and now pass.
The complete `npm run check` passed: repository/generated/build-proof validation,
Go tests (8.342 seconds), Go vet, and 203 Node tests (199 passed, four PowerShell
skips, zero failures; 84.869 seconds). The explicit image generation/edit and
same-task recovery examples were then aligned with the new default; existing
task model preservation remains required. No executor source or binary changed
for these default choices, and no paid request or publication was performed.

## Paid acceptance plan

Before testing, enumerate exact model/operation, references, dimensions, count
and applicable live cost for approval. Submit each approved case once; unknown
results stop without retry. Verify actual Key/model permissions, outside-group
and rejected-Key behavior, native host delivery and same-task recovery separately
from listing, installation and init.
