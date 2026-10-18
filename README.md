<p align="center">
  <img src="./assets/brand/puretokens-skill-hero.png" alt="Pure Tokens Official Skills" width="100%" />
</p>

# Pure Tokens Skills

Official Skills for checking a Pure Tokens connection, balance and model catalog, generating images, videos or audio, evaluating text with Jev, and producing text embeddings or relevance rankings.

Version 0.19.0 restores stable downloads from `puretokensx/puretokens-skill` and adds local installation recovery, model alternatives, live group guidance and product help.

Version 0.19.5 adds `puretokens-embedding` and `puretokens-rerank`, and synchronizes reviewed Grok editing restrictions and the Switch client compatibility inventory. Installation uses the version selected by the published stable manifest and includes ten Skills. Client/platform selection, verification and initialization remain installer responsibilities.

Version 0.18.7 adds MiniMax Code Desktop for local macOS/Windows default connections. Standalone installation with Desktop preferences requires a confirmed target or the client runtime data-directory environment. See `references/switch-skill-compatibility.md` for the connection and session requirements.

## Agent-assisted installation

The one-line compatibility entry is preserved below and requires access to the repository instructions. Use the short [Chinese](#chinese-installation-prompt) or [English](#english-installation-prompt) prompt to identify the task and official stable entry without a README or `raw.githubusercontent.com` prerequisite. The installer owns client/platform selection, verification, safe synchronization, initialization and result guidance; ask for missing information only when its receipt requires confirmation.

Before installation, the current client session needs permission to run local commands, download over HTTPS, and write the temporary download and target Skill directories. Use the client's supported approval controls; a network-only grant does not establish command or filesystem access. Do not require Full Access by default or change permissions automatically. Use permissions already reported by the host; do not add diagnostic probes to a normal installation.

If downloading the entry script fails, the installer has not started. Report that stage and use the actual error and host-reported permissions; do not infer an API-key problem or reinstall. A failed README download does not establish that release assets are unreachable. If an older prompt already stopped at that step, use a complete prompt only after the user explicitly asks to continue; do not retry the documentation download. See the [permission and continuation guide](skills/puretokens-update/references/install-permissions.md) only when troubleshooting.

### Copy this to a terminal-capable local agent

```text
Install or update the official Pure Tokens Skills from https://github.com/puretokensx/puretokens-skill.
```

### Chinese installation prompt

```text
请为当前客户端安装或更新 Pure Tokens 官方 Skills，并告知结果。
从以下官方稳定发布地址下载 puretokens-skill-fetch.sh 或 puretokens-skill-fetch.ps1，保存到本地后执行 install：
https://github.com/puretokensx/puretokens-skill/releases/latest/download/
```

### English installation prompt

```text
Install or update the official Pure Tokens Skills for this client and report the result.
Download puretokens-skill-fetch.sh or puretokens-skill-fetch.ps1 from the stable release address below, save it locally and run install:
https://github.com/puretokensx/puretokens-skill/releases/latest/download/
```

The agent saves the latest official stable release's `puretokens-skill-fetch.sh` (Windows: `.ps1`) locally and runs install. The fixed bootstrap URL is `https://github.com/puretokensx/puretokens-skill/releases/latest/download/puretokens-skill-fetch.sh`; Windows uses the `.ps1` suffix. Installed copies in `.puretokens-executor` provide check-update and update. Never pipe remote content into a shell or fall back to main, source archives or mirrors when stable assets are absent.

With fetch 0.19.1, omit host/target when neither is already confirmed. Fetch examines at most eight direct launcher processes, reading names and parent IDs only; a direct native `codex` or `opencode` can select its registered host. Unknown interpreters/editors stop detection, including Node, Electron/VS Code and ambiguous Claude launchers. `detect-host` is an offline, read-only diagnostic, not a generation preflight. Unconfirmed detection stops before release downloads or installation writes with `current_host_unconfirmed`; ask only for the missing client and run once with `--host` / `-Host`. Older fetch versions may also require that explicit selection. Preserve any confirmed host and custom target; Octop requires an explicit workspace target. Detection does not verify credentials, API support, a selected chat model or attachment delivery.

Installation boundary: save the official fetch script locally and run it once. Let it resolve the client where possible, pin the commit, download, verify, sync and initialize. Do not add full-script audits, PowerShell capability probes, duplicate inventory checks or doctor to ordinary installation. Respect required host approvals. On launch denial, unavailable output, timeout, download or verification failure, report the completed stage and sanitized failure, then stop. Do not create probes, shims or Python patches, modify official scripts, switch launch paths or retry automatically. Debugging requires a separate user request.

After the user resolves a permission issue and explicitly asks to continue, resume according to the last confirmed phase. Before synchronization, use the same official fetch workflow once; it resolves and verifies the stable release again, without reusing partial downloads. If synchronization succeeded and only init failed, perform one init only when the user asks to continue connection verification; do not reinstall. Unknown output or interrupted synchronization needs a separately authorized local installation diagnosis, not an automatic rerun. Never use this installation recovery to resubmit a media task.

For missing files, local changes or interrupted installation, request a read-only installation inspection. `install-inspect` classifies missing, modified, additional and transaction entries. Only an explicit repair request runs the official fetch `repair` path, verifies the current stable package and preserves customizations and unknown files. Ordinary update does not repair automatically. See [installation inspection and recovery](skills/puretokens-update/references/installation-recovery.md).

## What is installed

| Skill | Purpose |
| --- | --- |
| `puretokens-connection` | Verify the fixed Pure Tokens API identity without exposing configuration. |
| `puretokens-balance` | Check account wallet balance or the current Key's remaining allowance in USD. |
| `puretokens-models` | Read the authenticated current model catalog and declared capabilities. |
| `puretokens-image` | Generate images and perform declared image edits. |
| `puretokens-video` | Generate videos and use declared image/video/audio references or video edits. |
| `puretokens-audio` | Turn text into speech, transcribe recordings, generate sound scenes, or create music and songs. |
| `puretokens-evaluate` | Classify text, score against ordered criteria, or make yes/no judgments with Jev. |
| `puretokens-embedding` | Save embeddings for explicitly supplied texts into a new JSONL file. |
| `puretokens-rerank` | Save query-to-candidate relevance rankings into a new JSON file with original indexes and scores. |
| `puretokens-update` | Local product introduction, getting-started and usage help, connection verification, and safe Skill synchronization. |

## How direct API generation works

The media Skills always use full fixed URLs under `https://api.puretokensx.com`: Images uses `/v1/images/generations`; Videos uses `/v1/videos`. They do not send media work to an arbitrary configured Base URL, MCP server, local proxy, sidecar, or a second endpoint.

Each request is performed by the single-file native executor installed with the Skills. It runs only for that command, connects to the fixed API, then exits; it does not start a port, background service, proxy, sidecar, or desktop automation. The executor reads only the active connection record at the supported host's documented configuration path, verifies that record targets Pure Tokens, and keeps one matching credential in memory for the fixed request. Pure Tokens Switch, CC Switch and manual configurations can use the exact records listed in `references/credential-adapters.md`; unpersisted project/session overrides are not verified. Neither the Skill nor the user passes a key or Base URL. It never scans home directories, checks a provider label, asks for, displays, stores, logs, or reports an API key, Base URL, or host configuration.

No Node, npm, Python, Go, Pure Tokens Desktop, MCP, or upload relay is required to generate media. The installer verifies and places the platform-native executor, which performs fixed requests, multipart attachments, same-task polling, and bounded native-media delivery. If validation or attachment preparation fails before POST starts, no task was submitted. Once POST may have started, a network or unreadable-response failure leaves the outcome unknown; the Skill preserves that uncertainty and never resubmits automatically.

Computer Use, browser automation, and opening or clicking Pure Tokens Switch/Desktop are not fallback execution paths. The Skills never use them to find a visible generation interface, obtain a credential, submit media, or deliver a result; they also never invoke another image or video Skill as a fallback.

## Audio

Ask “Read this text into an MP3,” “Transcribe this recording,” or “Generate rain
and footsteps.” `puretokens-audio` reuses the current connection for one
synchronous request, with a 90-second deadline and no polling or automatic retry.
It supports speech up to 1000 characters, MP3/WAV/OGG recordings up to 8 MiB, and
sound instructions up to 500 characters. Voices are reviewed per model; generated
MP3/WAV files require actual attachment handoff. Voice cloning, translation
and real-time conversation are not integrated. An unknown response does not prove
no charge; failed attachment handoff can reuse the locally verified original file.
Production calls, billing and real-host playback are tracked separately in the
maintenance acceptance record.

Music uses `stepaudio-3-music-preview` through the same Skill: describe an
instrumental piece or provide lyrics for a song. The executor submits once and
immediately returns a task receipt, then queries and downloads that same task
through separate commands. Descriptions are limited to 1000 characters, lyrics
to 4000; output is MP3/WAV up to 32 MiB. Explicit local task records support
continuation without storing descriptions or lyrics. Gateway files remain
available for 24 hours after storage; downloaded originals can be reattached
after local verification. Deploy the paired Web gateway music API-key patch
before enabling music requests. See `references/audio-execution-contract.md`.

Audio [scenario guidance](skills/puretokens-audio/references/prompt-guide.md)
separates spoken text, sound descriptions, music style and exact lyrics.
Explicit combined requests can use [bounded workflows](skills/puretokens-video/references/workflows.md):
generate an image and pass its unchanged local bytes as a supported video's first
frame, or generate video and independent narration/music assets. Each step has its
own progress and delivery; a later failure never restarts earlier tasks. These
workflows do not compose, edit or mix a finished video. Simple requests do not load
the workflow guide or add preflight calls.

## Jev text evaluation

Ask “Use Jev to classify this ticket as billing, technical, or other” or “Use Jev to judge urgency and score impact against these three levels.” The new `puretokens-evaluate` Skill uses one synchronous native request to the fixed `/typesafe/v1/systemone` endpoint. It reuses the current host's Pure Tokens connection; no TypeSafe SDK or additional key is needed. Independent questions over the same text share one request.

The default is `jev-latest`; reviewed exact alternatives are `jev-1.13.0` and `jev-preview`. Choice, Score and Noul return typed options/numbers and probabilities. Confidence is not validated accuracy or authorization for external actions. Inputs are text/structured text, not images/audio/video. Timeouts or invalid responses stop without automatic retries, polling or claims about charges. Explicit evaluation model discovery lists only reviewed IDs returned by `/v1/models`; listing does not prove the native endpoint is callable. See [evaluation contract](references/jev-evaluation-contract.md). Production and real-host evaluation acceptance remains unverified.

## Text Embeddings And Reranking

Ask to embed the supplied texts into an
explicit new absolute `.jsonl` file, or rerank supplied candidates against a
query into a new absolute `.json` file. The native commands are
`embedding|rerank --host <current-host> --request <absolute-json-file>`.
They validate the request and new output path before reading credentials, reuse
the current host's verified Pure Tokens connection in memory, and make one
synchronous 90-second request without polling or automatic retry.

Embedding uses reviewed exact IDs `qwen3.7-text-embedding` and
`qwen3.7-text-embedding-flash` at fixed `POST /v1/embeddings`, with
`encoding_format=float`. JSONL rows contain `index`, `dimensions`, `embedding`
in original input order; chat reports only count, dimensions and the user path.
Rerank uses `qwen3.7-text-rerank` at fixed `POST /v1/rerank`, with
`return_documents=false` and `top_n` defaulting to all candidates. JSON contains
`results` with one-based rank, zero-based original index and the raw finite
relevance score. A matching receipt summary may be displayed; the score is not
a probability or calibrated accuracy.

Both accept 1–64 texts with at most 32768 Unicode characters each, with a
nonempty rerank query of at most 8192 characters. Requests are limited to 1 MiB,
responses to 2 MiB. Existing files are preserved; there is no implicit file or
directory ingestion, database creation, automatic batching or overwrite.
`accepted` with `output_verified=false` means output delivery failed after
acceptance and does not prove no charge. Unknown or accepted failures never
trigger another request. Explicit `models` queries for `kind=embedding` or
`kind=rerank` intersect one `GET /v1/models` with reviewed exact IDs; they are
not ordinary preflights or evidence of route, permission, billing or real-host
acceptance. See the [execution contract](references/embedding-rerank-execution-contract.md).

Switch guides the user to select the required model groups, apply the client
chat connection, and separately install the Skills. Dedicated models need API
permission, not a place in the chat model list. Jev, audio, embedding and rerank
guidance uses the same connection boundary; configuration, a visible model or
an example does not establish a successful dedicated call.

## Media routing

Image understanding, OCR, video analysis, screenshot troubleshooting, and prompt-only writing do not create media tasks. Model capability questions use model discovery; progress checks and result retrieval continue the existing task. The image and video entries include intent-to-operation tables, with scene prompt guides read only when needed. Verbatim prompts remain separate from actual output parameters, and attachment count alone does not imply first/last frames.

When the current host uses a Pure Tokens connection, ordinary image requests select `puretokens-image` before generic `imagegen`, Imagen, or other image Skills; ordinary video requests select `puretokens-video` before generic video Skills. This applies even when the user does not explicitly say “Pure Tokens.” Once selected, the Pure Tokens specialist either executes its fixed API path or stops safely; it never falls back to a generic media Skill.

The priority is carried by the installed Skill metadata and the host's current connection context. Only the native executor resolves one matching current-connection credential privately in memory for a fixed request; it never exposes or reports connection configuration. A host that ignores installed-Skill selection metadata must correct its own selection policy; a Skill cannot force another host or third-party Skill to change priority at runtime.

## Supported hosts

| Host | Skill directory | Direct execution |
| --- | --- | --- |
| Claude Code | `~/.claude/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| Codex | `~/.agents/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| WorkBuddy | `~/.workbuddy/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| Gemini CLI | `~/.gemini/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| Grok Build | `~/.grok/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| OpenCode | `~/.config/opencode/skills` | Credential fixtures tested; host end-to-end acceptance pending |
| Trae / TraeWork (`trae`) | `~/.trae/skills` (legacy installation only) | No credential adapter; current TraeWork discovery is not verified |
| Claude Desktop | `~/.claude/skills` (shared with Claude Code) | Local Code sessions; Desktop credential fixtures tested, end-to-end acceptance pending |
| DSH Desktop | macOS: `~/Library/Application Support/dsh-desktop/harness/skills`; Windows: `%APPDATA%\dsh-desktop\harness\skills` | Credential fixtures tested; host end-to-end acceptance pending |
| Official DeepSeek Harness | `~/.dsh/skills` or absolute `DSH_HOME/skills` | Default local Desktop connection in 0.18.7 |
| MiniMax Code Desktop | `~/.minimax/skills` or the explicit runtime data directory | Default local session; confirm the install target when Desktop preferences exist |
| ZCode | `~/.zcode/skills` | Local connection adapter; real API and attachment delivery acceptance pending |
| Kimi Code | `~/.kimi-code/skills` | Credential fixtures covered; real API and attachment delivery pending |
| Qoder | `~/.qoder/skills` | Local IDE/CLI execution; real API and attachment delivery pending |
| Pi | `~/.pi/agent/skills` | Inline authentication precedence covered by fixtures; real API and attachment delivery pending |
| Hermes | `~/.hermes/skills`; Windows: `%LOCALAPPDATA%/hermes/skills` | Selected inline custom provider; pooled/dynamic authentication unsupported |
| EvoX | `~/.evox/agent/skills` | Canonical default-instance settings/auth fixtures; legacy stores unsupported |
| VS Code | `~/.copilot/skills` | Default local profile; one supported multi-model connection sharing one Key in 0.18.7 |
| Octop | Explicit current workspace `.octop/skills` target | Read-only local SQLite connection; no assumed global Skill installation |

Connection adapters have fixture coverage; real-host acceptance is recorded separately.
The [Switch × dedicated-task matrix](references/switch-skill-compatibility.md)
separates Skill execution, connection resolution, model permissions, API routing
and actual delivery. Audio/Jev command entries are included in 0.18.7;
embedding/rerank have dedicated Skill entries starting with 0.19.5.
Configuring a client or listing a model does not verify a task.
Hermes and EvoX honor their documented absolute data-root overrides. Octop
requires `--host octop --target <absolute-workspace>/.octop/skills` (PowerShell:
`-HostId octop -Target <absolute-workspace>/.octop/skills`); do not enumerate
agents or write the global skill-package database. Cursor is not registered.

`references/host-support.json` defines the registered hosts. The table shows defaults; Claude/WorkBuddy honor explicit configuration-directory overrides, Pi honors an absolute `PI_CODING_AGENT_DIR` without parent traversal, and DSH honors the local Harness's explicit `DSH_HOME`. Gemini updates an existing higher-priority `.agents/skills` installation and reports managed duplicates. Provider labels never determine support.

Registration, installation, authenticated API access and complete media delivery are distinct. Real-host evidence records the client/OS/shell versions, architecture and execution mode; only a complete case set establishes image opening, video playback and attachment handoff for that exact environment. Consult `references/host-acceptance.json` for current per-case evidence; partial passes are not complete media acceptance. Local results do not accept WSL, remote or sandbox modes. See `references/host-acceptance-guide.md` for the procedure.

For Claude Desktop, use a local Code session and host ID `claude-desktop`; its active Desktop 3P connection is separate from Claude Code authentication. Cloud, SSH, WSL and Cowork isolated environments are not the local desktop; unavailable connection records or executors stop execution. DSH uses `dsh-desktop`; project/custom Skill roots may override user roots, so verify the loaded location. See the [desktop host guide](skills/puretokens-update/references/desktop-hosts.md).

## Images and videos

Normal generation reads only the selected profile for a default or exact model ID; the small index is needed only for model selection or alias resolution. It does not load every model or fetch the catalog before every task. `puretokens-image` defaults to `gpt-image-2.5-flare`; `puretokens-video` defaults to `minimax_h3`. An explicit user model choice takes precedence; an existing task retains its original model. The live catalog is read only when a user explicitly asks for current models, requests an option or media operation absent from the selected profile, or needs a post-rejection diagnosis.

A normal wait window ending means the task is still processing, not failed. The active authorized delivery can continue for one more bounded foreground window, then pauses for user input. Task records preserve this budget; errors, reconciliation and deferred retries stop automatic continuation. Videos, multiple outputs and cross-session work use explicit workspace task records by default. A failed attachment handoff reuses the verified existing file rather than submitting or downloading again.

Image and video tasks are asynchronous. Each submission creates at most one POST. Once a returned top-level task ID is known, the Skill polls and retrieves only that task. If a POST may have started but the task ID is missing, acceptance is unknown: it never submits a duplicate task or claims that no charge occurred. Image content is delivered sequentially from zero-based indexes `0..n-1`; video content is delivered only after terminal success.

For a local image, video, or audio attachment, the Skill sends the current attachment only in the exact declared multipart Images or Videos API operation. It never uploads to a separate service, rehosts media, turns attachment bytes into a prompt, or silently downgrades an image/video reference request to text generation. Public HTTPS URLs are passed only in profile-declared fields; file/voice IDs are not supported.

The selected installed profile governs ordinary requests. An explicit authenticated catalog query reports current declarations; catalog membership is not a required preflight or a substitute for the media API’s access decision. If the media API denies access, report that rejection and guide the user to check the connection’s model permissions without automatically resubmitting.

<!-- media-model-catalog:start -->
## Media model catalog

Synchronized with the base model catalog: 2026-10-18T12:22:09.975Z.

This list is an installed selection aid, not a per-request authorization check. Ordinary generation reads only the selected profile; live discovery is limited to explicit requests, profile gaps or rejection diagnosis. Reviewed local compatibility supplements are separately sourced.

README is generated only from base-catalog models with explicit image/video capabilities; it never infers capability from a model name. The installed model index selects a model and only that model's profile carries known parameters; the live catalog is read on demand only for explicit discovery, an installed-profile gap, or post-rejection diagnosis. Before release, refresh from the controlled base catalog and run `npm run release:validate`; the release gate fails when the snapshot is over seven days old.

### Image models

| Model ID | Provider | You can also say | Good for | Example |
| --- | --- | --- | --- | --- |
| `gemini-nano-banana-2.1` | Google | Exact ID only | Image generation | `Use gemini-nano-banana-2.1 to generate an image.` |
| `gpt-image-2` | OpenAI | `image2` | Image generation | `Use gpt-image-2 to generate an image.` |
| `gpt-image-2.5` | OpenAI | Exact ID only | Image generation | `Use gpt-image-2.5 to generate an image.` |
| `gpt-image-2.5-flare` | OpenAI | Exact ID only | Image generation | `Use gpt-image-2.5-flare to generate an image.` |
| `gpt-image-2.5-sunburst` | OpenAI | Exact ID only | Image generation | `Use gpt-image-2.5-sunburst to generate an image.` |
| `gpt-image-2(Sub)` | OpenAI | Exact ID only | Image generation | `Use gpt-image-2(Sub) to generate an image.` |
| `grok-imagine-image` | xAI | `grok image` | Image generation | `Use grok-imagine-image to generate an image.` |
| `grok-imagine-image-quality` | xAI | Exact ID only | Image generation | `Use grok-imagine-image-quality to generate an image.` |
| `nano-banana-2` | Google | `nano banana 2` | Image generation | `Use nano-banana-2 to generate an image.` |
| `nano-banana-2-lite` | Google | Exact ID only | Image generation | `Use nano-banana-2-lite to generate an image.` |
| `nano-banana-pro` | Google | `nano banana pro` | Image generation | `Use nano-banana-pro to generate an image.` |
| `qwen-image-3.0` | Qwen | Exact ID only | Image generation | `Use qwen-image-3.0 to generate an image.` |
| `qwen-image-3.0-pro` | Qwen | Exact ID only | Image generation | `Use qwen-image-3.0-pro to generate an image.` |
| `wan2.7-image` | Qwen | Exact ID only | Image generation | `Use wan2.7-image to generate an image.` |
| `wan2.7-image-pro` | Qwen | Exact ID only | Image generation | `Use wan2.7-image-pro to generate an image.` |

### Video models

| Model ID | Provider | You can also say | Good for | Example |
| --- | --- | --- | --- | --- |
| `grok-imagine-video` | xAI | `grok video` | Video generation | `Use grok-imagine-video to generate a video.` |
| `grok-imagine-video-1.5` | xAI | Exact ID only | Video generation | `Use grok-imagine-video-1.5 to generate a short video.` |
| `minimax_h3` | MiniMax | Exact ID only | Video generation | `Use minimax_h3 to generate a short video.` |
| `muse-video` | Meta | Exact ID only | Video generation | `Use muse-video to generate a short video.` |
| `omni` | Google | Exact ID only | Video generation | `Use omni to generate a short video.` |
| `seedance-2.0` | ByteDance | Exact ID only | Video generation | `Use seedance-2.0 to generate a video.` |
| `seedance-2.0-fast` | ByteDance | Exact ID only | Video generation | `Use seedance-2.0-fast to generate a video.` |
| `seedance-2.0-mini` | ByteDance | Exact ID only | Video generation | `Use seedance-2.0-mini to generate a video.` |
| `seedance-2.5` | ByteDance | Exact ID only | Video generation | `Use seedance-2.5 to generate a video.` |
| `seedance2.0` | ByteDance | Exact ID only | Video generation | `Use seedance2.0 to generate a short video.` |
| `veo_fast` | Google | Exact ID only | Video generation | `Use veo_fast to generate a short video.` |
| `veo_lite` | Google | Exact ID only | Video generation | `Use veo_lite to generate a short video.` |
| `veo_quan` | Google | Exact ID only | Video generation | `Use veo_quan to generate a short video.` |
| `wan3.0-video` | Qwen | Exact ID only | Video generation | `Use wan3.0-video to generate a short video.` |
| `wan3.0-video-prime` | Qwen | Exact ID only | Video generation | `Use wan3.0-video-prime to generate a short video.` |

<!-- media-model-catalog:end -->

## Errors and receipts

Unreadable submission responses, missing task IDs at the declared location, and unsupported ID types or formats have distinct local categories. None alone proves that the server omitted an ID or failed to generate media. Diagnosis uses the executor version in the affected client's original receipt, not another computer's version, and never repeats a paid request just to collect diagnostics.

Machine receipts retain available model, task ID, original operation, state, safe parameters and progress. User-facing replies show only the useful status, actual attachment or actionable failure. Failures include a safe phase, public API code only when explicitly returned, HTTP status when returned, a sanitized message, and an action the user can take. The Skills never expose raw response bodies, request headers/bodies, internal URLs, credentials, or user media.

Error text comes from controlled categories; only recognized public codes
actually returned by the API are retained. Unknown codes and arbitrary server
error text are omitted. GIF validation checks complete frame data; WebM checks
container boundaries, video tracks and blocks. Damaged output is not reused.
Container checks do not replace opening/playing the actual host attachment.
Explicit resume of a reconciliation record reads the same task once, then uses
the newly confirmed status.

## Updating

`puretokens-update` resolves the latest published stable manifest and pins its version, source commit, selector and platform archive checksums. It downloads only the current OS/architecture archive; version 0.19.5 contains ten shared Skills, one executor and only the current system's scripts. Missing assets never fall back to main or a source archive. Check-update does not write files or run init. Same-version install/update verifies the selected release's executor checksum and all eleven managed inventories, then returns without archive download, writes or init. Missing/modified files or unresolved transactions stop without automatic repair. Explicit local source sync remains available for maintainers.

A fresh installation or actual update still runs init automatically, with at most two read-only requests sharing one 20-second deadline and no automatic retry. File synchronization and connection verification are reported separately; an init timeout does not roll back installation, trigger reinstallation or prove an invalid credential. A versioned synchronization receipt confirms a completed installation; a same-version verification receipt confirms the existing installation without changing files.

The source sync scripts are `runtime/puretokens-skill-install.sh` for macOS/Linux and `runtime/puretokens-skill-install.ps1` for Windows. They install, verify, and place the platform executor; users do not need Node, npm, Python, Go, or a package manager.

Each managed directory has a `.puretokens-managed.json` inventory of files and
checksums. Updates and interrupted recovery stop on added, changed or missing
files and symlinks, preserving existing contents. An unmarked directory
must match the current official source exactly; a matching name or self-reported version/hash is
insufficient. Inventories detect accidental edits, not malicious local tampering.

After a fresh installation or actual version update, the installer automatically runs `init`; an intact same-version shortcut does not. It performs a non-billable fixed `/v1` identity check followed by one authenticated `/v1/media/models` request within a shared 20-second deadline without displaying credentials or host configuration, then prints a compact result, host-specific examples and the local help entry. If verification does not complete, it reports a sanitized reason such as no active matching connection, missing credential, API rejection with its HTTP status, network failure, or an unconfirmed API identity; it never prints the configured URL, provider, or key. Ask to initialize Pure Tokens Skills or verify authentication to run `init` again explicitly. A general current-connection or authentication check also uses `init` once; only an explicit fixed-API public identity request uses `connection`. Installation diagnostics use `doctor`. Product introductions, getting-started questions and usage help read the [local guide](skills/puretokens-update/references/usage-guide.md) without requiring a host or configured connection, running the executor, reading credentials, using the network, installing anything or creating a paid task. Static help does not promise current models or prices. Choose one intent without chaining redundant checks.

## Development validation

Optional anonymous completion reporting is disabled by default. Explicit
`PTP_OPERATIONS_RECEIPTS=1` enables bounded installation, verified connection
and execution event counts, without account IDs, credentials, prompts or
configuration. Failed reporting never changes the operation result. See
[the receipt contract](references/operations-receipts-contract.md) for the
one-second send budget and the distinction between submission and delivery.

Maintainers can run:

```bash
npm run check
npm run release:validate
```

Maintain shared guidance in `references/desktop-hosts.md`,
`references/skill-fragments/host-binding.md` and the host registry, then run
`npm run docs:sync-guidance`. It produces ten self-contained installed copies
and entrypoint hashes; the engineering gate rejects drift. No user runtime is added.

PNG/JPEG download and reuse validate pixel data with a 33,554,432-pixel decode
limit. WebP checks complete containers and nonempty image chunks, not full
codec decoding. Model queries and submissions share combination rules; live
queries do not rewrite profiles or automatically relax installed enum/range
limits. Changed declarations require a compatible stable update.

## Executor receipts and acceptance

The 0.17 command flow is submit → immediate task ID → bounded wait/status → one-index content download → host attachment delivery. No per-generation balance, init, or catalog preflight is added. A downloaded file is not yet a delivered attachment. Request JSON files replace interactive stdin; see each media Skill’s executor-usage reference.

Balance uses the same API-key route as the official CC Switch integration: `GET https://console.puretokensx.com/api/product/console/api-keys/usage`, followed by one public `/api/product/console/status` read for the USD conversion ratio. No browser login is required. An unlimited Key returns account wallet balance; a limited Key returns its own remaining allowance. The default reply is one concise amount with its scope, without subscription quotas. Legacy billing placeholders are never used as money. Queries stop after at most two GETs within 30 seconds; failures show an actionable reason without inventing an amount.

The executor adds a missing `sk-` prefix in memory for this balance request, as CC Switch does during import. It leaves user configuration and media authentication unchanged. If balance returns 401/403 while other calls work, the Skill reports a balance-specific rejection and preserves the working connection.

`references/host-acceptance.json` distinguishes tested credential fixtures from real host acceptance. Local automated checks do not prove Windows/macOS host attachment delivery.

Follow `references/host-acceptance-guide.md` for the full host acceptance cases.
`npm run acceptance:validate` checks host/OS versions, executor hashes, evidence
files and outcomes. Checks without actual evidence remain pending.

Explicit request checks use `preflight` without submitting media. `doctor` combines local installation diagnostics with read-only connection checks; help-only questions read the installed usage guide without network access. Neither is a routine generation preflight. Optional task records in a user/workspace location support same-task resume and delivered-index tracking; they contain no credentials, prompt, reference URLs or media bytes.

Exact media quotations, server idempotency guarantees and lookup of unknown submissions require server capabilities not implemented here. Local validation and task records do not provide them.

ZCode: `--host zcode`; an absolute `ZCODE_DATA_BASE_DIR` selects `<base>/.zcode/skills`. One uniquely enabled Pure Tokens connection is required. This does not identify the conversation model; remote workspace Skill sync does not supply the executor or connection.

Maintainer build verification uses the exact toolchain in `runtime/executor/build-config.json`. `npm run executor:build` records source/build input identities and six artifacts; `npm run validate` checks them. `npm run release:validate` also rebuilds all six platforms into temporary directories and compares bytes. End users need neither Go nor Node.

Verified output reuse requires an explicit task record with matching file SHA-256, byte count and media type. Legacy records can resume the same task, but files without proofs must be preserved and fetched again into another output directory. Native image/video handoff remains a separate real-host acceptance item.
