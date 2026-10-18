# Media Parameter Review

## Current review - 2026-10-18

Anonymous production catalog capture at 2026-10-18 12:14:13 UTC: 114 models,
including 15 image and 15 video models. Compared with the earlier 02:05 UTC
capture, add `wan3.0-video` and `wan3.0-video-prime`, remove
`grok-imagine-image-2.0` from installed selection, and refresh the two Grok
generation ratio enums. All other retained media schemas are unchanged.
Existing tasks retain their exact model IDs. Defaults remain
`gpt-image-2.5-flare` and `minimax_h3`. Evidence:
`acceptance/client-model-sync-20261018.json`.

- Web source `b3c481d7` closes `video_edit` for exact
  `grok-imagine-video-1.5` and `grok-imagine-video-1.5-preview`. The standard
  Grok edit inherits the source timeline and geometry, caps output at 720p and
  accepts a source no longer than 8.7 seconds. Its provider body omits
  `duration`/`seconds`, `aspect_ratio`/`aspectRatio` and `resolution`.
  Source: `PureTokenPlus_web/docs/architecture/video-provider-adapters.md`
  and the official xAI video-editing contract.
- `runtime/executor/reviewed-media-policy.json` is the shared, embedded,
  exact-model restriction source for installed profiles, catalog projection,
  filtered discovery and pre-submission validation. These restrictions only
  narrow capabilities and override stale live declarations. They do not
  override authentication, add a preflight, or affect existing-task recovery.
- Reject conflicting user parameters locally before any catalog read or POST;
  explain the limitation rather than discarding requested controls. Source
  byte inspection stays in the gateway; Skills do not download or rehost input.
- The post-deployment public schema now declares Grok generation ratios
  `21:9` and `5:2` for `grok-imagine-video` and `grok-imagine-video-1.5`.
  Synchronize these exact enums into the candidate profiles. This resolves
  the earlier metadata rollout dependency; it does not enable video editing
  for 1.5 or allow ratio controls on standard Grok edits.
- Both Wan profiles declare duration 2–30 seconds, `480p`/`720p`/`1080p`,
  the returned ratios including `adaptive`, and fixed `/v1/videos` operations.
  JSON `video_generation` permits mixed public HTTPS image/video/audio
  references, at most 10/5/5. Separate declared multipart operations carry
  the current first-frame, image, video or audio attachment bytes. Last-frame
  input, automatic duration `-1`, supplier `media[]` and `prompt_extend`
  controls are not part of these installed public operations. Supplier routing
  and prompt expansion stay in the gateway.
- Reviewed audio, Jev, embedding and rerank IDs remain publicly listed; no
  dedicated request-contract expansion was found in this comparison. General
  chat-model changes need no static media profile or chat-model alias here.
  Web/Switch still list the same 22 client identities. The published Skills
  release is still 0.19.4; these updates belong to source candidate 0.19.5.

This review records source and synthetic compatibility only. It includes no
authenticated model-permission check, paid call or real-host attachment proof.

## Historical review - 2026-10-14

The anonymous production catalog returned 111 models, including 16 image and
13 video models. The refreshed selection contains 29 media profiles; source
hash and comparison details are in `acceptance/client-model-sync-20261014.json`.

- Add exact `gemini-nano-banana-2.1` and `muse-video` selection profiles with
  their declared dedicated APIs and attachment operations. Remove
  `wan3.0-video` and `wan3.0-video-prime` from installed selection only; existing
  task records retain their original model IDs.
- All five GPT Image variants now declare only `image_size=1K`. Remove the
  previous 2K/4K values and their ratio mappings rather than silently converting
  a user's size request.
- H3 retains `adaptive` and its existing frame/reference operations. Remove
  undeclared resolution, seed and watermark fields, and retain the newly
  declared JSON generation/reference operation.
- Synchronize Nano Banana and Seedance parameters and operations from this
  capture. Explicit live definitions continue to take precedence over the
  separately reviewed first/last-frame supplement.
- Keep user-approved defaults `gpt-image-2.5-flare` and `minimax_h3`.

This is public contract synchronization. It adds no evidence of authenticated
model permission, paid generation, real-client acceptance or native delivery.

## Historical review - 2026-10-04

2026-10-04复核：公开目录112项，29个媒体profile（15图片、14视频）。仅移出未再列出的 `seedream-5.0-pro`，不据此推断永久下线；其余schema经现有审核补充后未变。默认图片 `gpt-image-2.5-flare`、视频 `minimax_h3` 保持；H3首尾帧使用 `image_to_video` 与声明的first/last字段。当前所有图片profile仅声明n=1；多结果执行机制用仓库合成profile测试，不作为线上批量能力声明。脱敏证据见 `acceptance/client-model-sync-20261004.json`。

## Historical review - 2026-10-03

The public catalog snapshot captured at 02:02 UTC on 2026-10-03 declares 30 media models. See `acceptance/client-model-sync-20261003.json` for source hash and exact additions/removals; the September findings below are historical and superseded where they conflict.

- GPT Image adds exact `gpt-image-2.5-flare` and `gpt-image-2.5-sunburst`. All five current variants declare 1K/2K/4K, ten ratios, quality/output formats and at most ten references. Public tasks remain asynchronous.
- New image profiles: qwen-image-3.0, qwen-image-3.0-pro, wan2.7-image, wan2.7-image-pro. Their edit operations declare JSON `images` HTTPS URLs, not local file upload. Do not upload/rehost user attachments to manufacture this transport.
- User-approved defaults are `gpt-image-2.5-flare` for images and `minimax_h3` for videos; both exact IDs and their full input schemas were rechecked against the public catalog at 02:40 UTC on 2026-10-03. Removed IDs are removed only from installed selection, not rewritten in existing task records. minimax_h3 is an exact distinct ID, never an implicit alias of minimax-h3.
- New video profiles: seedance2.0, omni, veo_fast, veo_lite, veo_quan, minimax_h3. Seedance2.0 and current seedance-2.5 require explicit duration; operation inputs are JSON HTTPS URLs. omni video_edit also declares JSON URL transport. Veo profiles omit native attachment operations, so do not claim that local attachments work.
- Seedance-2.5 no longer declares frame fields; it no longer receives the reviewed first/last-frame supplement. Seedance-2.0/Fast/Mini still declare both fields and retain that separately sourced supplement.
- Executor recognizes `https_url` and `public_https_url` as the same bounded public-only reference transport, checks declared input counts including scalar URLs, and enforces Minimax last-frame/first-frame and reference exclusivity. Added video_urls to reference validation so private URLs cannot pass as ordinary parameters.
- Reviewed Step audio and Jev IDs are publicly listed now, but dedicated API permissions and native output remain unverified. Realtime/Qwen audio, embeddings and rerank are not enabled by a generic OpenAI catalog label; Web audio contract dated 2026-10-01 explicitly withdraws Qwen workspace support.

No real-host, paid task, account entitlement or attachment acceptance is claimed.

## Historical review — 2026-09-18

Reviewed 2026-09-18. Public source: the production console's
`/api/product/docs/model-catalog`. This is selection metadata, not authenticated
model access or paid generation proof.

- GPT Image: exact IDs are `gpt-image-2`, `gpt-image-2.5` and
  `gpt-image-2(Sub)`. Retire Flare/Sunburst selection profiles without adding
  replacement aliases or rewriting existing task identities.
- All three declare only `image_size=1K`, `n=1`, at most ten URL references
  or multipart image-edit files, `quality=auto|low|medium|high`,
  `output_format=png|jpeg|webp`, and `response_format=url`.
  Ratios are `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`,
  `16:9`, `21:9`; do not retain the retired variants' `3:1`/`1:3`.
  Pricing rows are not authority to enable 2K/4K.
- Enforce `aspect_ratio_by_image_size` in both query filters and submission,
  using declared defaults for omitted fields. Model IDs permit one bounded
  parenthesized suffix while still rejecting path separators and traversal.
  Submission, query projection, safe receipts and continuation retain the
  exact ID. Supplier routing stays inside the gateway; public task lifecycle
  remains asynchronous.
- The catalog still contains 12 image and 11 video models. All other public
  parameter and operation declarations match the previous snapshot. Backend
  adapter additions alone do not expand the installed public API contract.
- Nano Banana 2, Lite and Pro: the live aspect-ratio enums no longer declare
  `4:5` or `5:4`. Synchronize those enums exactly; do not silently convert ratios.
- Seedance 2.0, Fast, Mini and 2.5: the live schema still declares
  `first_frame_image` and `last_frame_image`, but omits their operation entries.
  Retain the previously supported frame operations using an explicit reviewed
  supplement first reviewed on 2026-09-12, not a merge of arbitrary stale
  metadata.
- Local Web source evidence:
  `upstream-new-api-src/relay/channel/task/seedance/adaptor.go`,
  `normalizeRequestForModel`, reads both frame fields and rejects mixed
  frame/general references. `adaptor_test.go`,
  `TestNormalizeRequestMapsFirstLastFramesToDocumentedPayload`, verifies that
  mapping and rejection. These were read in `PureTokenPlus_web`; that repository
  and production settings were not modified.

`scripts/reviewed-media-supplements.mjs` records local compatibility additions
separately from the public capture. It only fills absent operation/constraint
entries for the four exact models while both frame fields remain declared.
Explicit live definitions take precedence. Missing fields do not receive a
supplement. All other parameters come directly from the public schema.

No new real-host, authenticated catalog, paid task, or media delivery acceptance
is claimed by this refresh. Existing frame regression tests remain in place.
