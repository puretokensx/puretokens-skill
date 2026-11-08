# Dedicated capability coverage update

Scope: Skills only; add Voyage float embeddings, reviewed Wan native inputs,
and refresh the anonymous media catalog. Preserve existing defaults and task
recovery. No paid API calls, host credentials, Web/Switch changes or publication.

## Implemented source changes

| Capability | Implementation and contract evidence | Evidence level |
| --- | --- | --- |
| `voyage-4-large` embeddings | `runtime/executor/retrieval.go`, embedding request schema and installed embedding guide: optional query/document input type, 256/512/1024/2048 dimensions, fixed float output and no truncation, returned-dimension validation and ordered JSONL | Source and offline fixtures; not released or real-host accepted |
| Wan 3.0 native input | `runtime/executor/wan_native.go`, shared reviewed policy and `wan-native-usage.md`: the two exact IDs, native input/media roles and nested settings, public HTTPS file/link with explicit prompt extension | Source and offline fixtures; not released or real-host accepted |
| Existing Gemini task recovery | `TestUnlistedImageModelKeepsSameTaskRecovery`: original model/task retained, one content GET, verified native PNG, no discovery or new POST | Offline fixture only; not real delivery |
| Installed model selection | Anonymous catalog captured at `2026-11-08T13:27:39.712Z`, 21 image and 26 video models; this snapshot omits `gemini-3.1-flash-image` without declaring permanent retirement | Public catalog evidence, not account permission or generation acceptance |

Image, video and embedding defaults remain `gpt-image-2.6`, `MiniMax-H3` and
`qwen3.7-text-embedding`. No new host adapter, credential source, user runtime,
preflight or retry path was added. Local Wan files, voice cloning, realtime
voice and subject/resource creation remain outside this change.

## Protocol and review evidence

- Voyage controls were checked against the official text-embedding contract
  (`https://docs.voyageai.com/docs/embeddings`) and the gateway embedding DTO.
  Source compatibility is not evidence of deployed route availability.
- Six pure-function cases passed against the current gateway Wan normalizer:
  file and link for each model, plus each model's maximum reviewed mixed
  reference/document envelope. These checks made zero network or paid calls.
- Read-only guidance review applied four scenarios: Voyage 512-dimensional
  document vectors, public Wan file/link input, rejection of local PDF upload,
  and recovery of an unlisted Gemini task. The one finding, an outdated model
  list in local help, was corrected and covered by a regression assertion.
  Guidance review is not an executed host session.

## Engineering verification

- Positive Voyage/Wan tests first reproduced the missing capability before
  implementation. Added tests cover bounds, invalid fields, dimension mismatch,
  permission rejection, no retries, task identity and sanitized records.
- The six platform executors were rebuilt; source/build identities and artifact
  hashes passed the repository verifier. Cross-compilation does not verify
  execution on all platforms.
- The first complete check caught the video entrypoint exceeding the existing
  lightweight budget. The entrypoint was shortened without raising the limit;
  the final `npm run check` completed with exit code 0: Go tests and vet passed;
  Node reported 268 tests, 264 passed, zero failed and four skipped because
  PowerShell is unavailable on this host. The skips remain unverified.
- Repository, generated guidance/profile consistency, host-evidence integrity
  and six executor build proofs passed. `git diff --check` also passed.

## Acceptance and publication boundary

No real credentials were read, no paid request was submitted, and no actual
client attachment handoff was tested. Existing host acceptance remains at
268 pending host/OS summary checks and zero recorded real-host case passes.
No acceptance state is promoted by these fixtures.

This capability work initially ended as an unreleased local source update.
It was subsequently included in stable 1.0.3 under a separate version-specific
publication decision and complete release checks; see 1.0.3-release.md. No Web,
Switch or gateway change, paid acceptance or real-host delivery is implied.
