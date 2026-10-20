# Credential adapter scope

Only the native executor reads the documented connection records. It verifies the configured Pure Tokens hostname, HTTPS and path before using the matching credential for a fixed API request. Credentials are not arguments, request-file fields, receipts, logs or persistent executor state. The configured origin never replaces the fixed API origin.

Balance uses the same verified credential on the fixed `https://console.puretokensx.com/api/product/console/api-keys/usage` route used by the official CC Switch integration. This is an API-key route, not a browser-session route. Its public `/api/product/console/status` metadata read carries no credential. The console-origin exception applies only to balance; media, models and connection remain on the fixed API origin.

Before the balance usage request only, add `sk-` in memory if absent, matching the official CC Switch import normalization and console route admission rule. Preserve an existing prefix and all key suffixes. Never rewrite configuration or alter credentials sent to media/model/connection endpoints. Redact both original and normalized forms from the response. A balance-specific rejection does not establish that a working media connection is invalid.

| Host | Explicit record read by this release | Evidence / boundary |
| --- | --- | --- |
| Codex | `CODEX_HOME/config.toml`, otherwise `~/.codex/config.toml`; configured `profile` and `model_provider`; provider `experimental_bearer_token`, explicit `env_key`, or sibling `auth.json` only when that provider declares `requires_openai_auth = true` | Standard TOML / Switch fixture, selected profile, directory override and both declared credential sources covered. CLI-only `-c`/`--profile` overrides not persisted in the record are not discoverable. |
| Claude Code | `CLAUDE_CONFIG_DIR/settings.json`, otherwise `~/.claude/settings.json`; `env.ANTHROPIC_BASE_URL` and `env.ANTHROPIC_AUTH_TOKEN` | Switch-format user settings fixture. Project/session override precedence has not been verified. |
| Gemini CLI | `~/.gemini/.env`; `GOOGLE_GEMINI_BASE_URL` and `GEMINI_API_KEY` | Switch-format fixture; project/session overrides and alternate auth modes not verified. |
| WorkBuddy | `WORKBUDDY_CONFIG_DIR/models.json`, then `CODEBUDDY_CONFIG_DIR/models.json`, otherwise `~/.workbuddy/models.json`; `url` + `apiKey` records | Requires one unique matching credential. Zero matching records, missing credential and multiple matching credentials have distinct outcomes. Saved matching records are resolved independently of the selected chat model; no chat-model switch is required for init. Missing/unreadable records, unsupported formats, no matches, missing credentials and ambiguity produce distinct workbuddy_* diagnostics. This is not proof of a separately selected UI model. |
| Grok Build | `~/.grok/config.toml`; `models.default` selects `model.<id>.base_url/api_key` | Standard TOML fixture. Unpersisted session selection not verified. |
| OpenCode | Declared JSON/JSONC layers and the selected native `auth.json` entry described below; root `model`, when present, selects `provider.<id>.options.baseURL/apiKey` | Optional default, layer ordering, provider filters and native API-key precedence covered by fixtures. Without a default, requires exactly one enabled endpoint-matching saved service. Not proof of current chat selection. |
| TRAE / TRAE SOLO | Default local `User/globalStorage/state.vscdb` under macOS `~/Library/Application Support/{Trae,Trae CN,TRAE SOLO}` or Windows `%APPDATA%/{Trae,Trae CN,TRAE SOLO}`; exact `puretokens.traework.status.v1` and its two declared account-scoped model-cache keys | Source adapter for a single Switch-managed saved connection. Multiple installed state stores, distinct credential fingerprints, unfinished transactions, edited/missing models, masked keys or unsupported formats stop before networking. No version whitelist; no claim of active chat/account selection. |
| Claude Desktop | macOS `~/Library/Application Support/Claude-3p`, Windows `%APPDATA%/Claude-3p`; `claude_desktop_config.json` must select `deploymentMode: 3p`; `configLibrary/_meta.json.appliedId` selects exactly one UUID-named JSON profile. Read its `inferenceGatewayBaseUrl` / `inferenceGatewayApiKey` only for `gateway`, `static`, `bearer`. | Mirrors Switch's `crates/client-adapters/claude-desktop/src/lib.rs`. No profile enumeration, label matching or Claude Code fallback. Only the local desktop filesystem is supported; cloud/remote/Cowork access is not implied. |
| DSH Desktop | Explicit absolute `DSH_HOME`, otherwise macOS `~/Library/Application Support/dsh-desktop/harness`, Windows `%APPDATA%/dsh-desktop/harness`; `settings.yaml.agent-default-model.provider/model` selects `llm-pi-ai.providers.<id>` with a declared matching model. Its `apiKeyEnv` selects one `refs` value in sibling `.credentials.yaml` version 1. | Mirrors Switch's `crates/client-adapters/dsh-desktop/src/lib.rs`. Validate `openai-completions` and the Pure Tokens endpoint before opening the credential store. No environment-key fallback, provider-label identity, profile enumeration or unrelated store access. YAML rejects duplicate keys, aliases, merge keys, multiple documents and excessive nesting; quoted/block/flow scalars are parsed normally. |

Desktop path selection is tested for macOS and Windows with synthetic files. DSH's YAML parser is compiled into the executable and adds no user runtime requirement. Unpersisted session model/provider changes are not discoverable from the configured default; real-host acceptance must verify that the local session uses the same selection.

Source evidence checked on 2026-09-07: Switch's independent desktop adapter source files above; Claude's official `https://code.claude.com/docs/en/desktop.md` (“Use skills”, personal skills apply to local sessions); DSH's official `dataelement/dsh-desktop` source revision `dd5ebe99c2ddadd36ac0b6d94e6dee2f65ac85cb`, `src/main/runtime/harness-runtime.ts` (explicit DSH_HOME) and bundled `@deepseek-ai/dsh-skill-filesystem` 0.1.2-rc.1 (user Harness and shared Agents roots). These are implementation evidence, not real-host acceptance.

The README's fixture-tested status refers only to these exact records. Complete host acceptance also requires the real host to execute the installed binary, identify the current connection, resume a task and deliver its file. Record those checks per OS in `host-acceptance.json`; do not mark pending cells passed from compilation or mock API tests.

`doctor` inspects only the loaded installation and documented Skill directories for the selected host before the existing init check. These local manifest checks neither resolve project/session credential overrides nor prove host attachment delivery. Help and update-version checks never invoke a credential adapter.

## OpenCode

Version-scoped source evidence: official `anomalyco/opencode` v1.18.23, revision `ef2880f379129aa048be9e9353e30aa168d42c17`, checked 2026-09-12. Relevant sources are `packages/opencode/src/config/{config,paths,managed}.ts`, `provider/provider.ts`, `auth/index.ts` and the core global-path resolver. Fixture coverage does not establish real-host acceptance.

Read only these named files, in merge order: `config.json`, `opencode.json`, `opencode.jsonc` under `$XDG_CONFIG_HOME/opencode` (default `~/.config/opencode`); an explicit absolute `OPENCODE_CONFIG`; project `opencode.json`/`opencode.jsonc` from the nearest Git root to the current working directory; the same pair in declared `.opencode` directories from the working directory upward to that root, then `~/.opencode`, then explicit absolute `OPENCODE_CONFIG_DIR`; finally system-managed `opencode.json`/`opencode.jsonc`. Without a Git marker, the bounded ancestor walk ends at the filesystem root. Honor `OPENCODE_DISABLE_PROJECT_CONFIG`; never enumerate directories or read recent models, session history or another client's records. System-managed roots are `/Library/Application Support/opencode` on macOS, `%ProgramData%/opencode` on Windows (default `C:\ProgramData\opencode`) and `/etc/opencode` on Linux. Directory overrides must be absolute without parent traversal or control characters.

Deep-merge objects; higher-priority fields replace lower values, including empty/null values. Respect `enabled_providers` and `disabled_providers`. An explicit `model` must identify an enabled configured provider and a nonempty model ID; failure never falls back to another provider. Only an absent `model` permits selection of exactly one endpoint-matching saved service. Do not compare keys to resolve ambiguity. This verifies a declared API capability, not the active chat model; host routing must already select Pure Tokens or the user must explicitly request this service.

After endpoint verification, read only `$XDG_DATA_HOME/opencode/auth.json` (default `~/.local/share/opencode/auth.json`). Associate the exact selected provider ID, never its label, with a `type: api` entry. Inline provider `options.apiKey` takes precedence over that entry's `key`; empty, null or unsupported inline values stop without fallback. Absent inline credentials require the selected native API-key entry. OAuth, unknown selected authentication, malformed/unreadable stores, any `wellknown` remote-config entry, dynamic `{env:...}`/`{file:...}`/`${...}` values and credential-bearing custom headers are unsupported. Never expand references or try process-environment keys.

Configuration is JSONC with duplicate-key rejection, a 64-level nesting guard, 1 MiB per record and a 4 MiB total configuration budget; the native auth store is strict JSON. Reject substitution markers throughout every raw layer and parsed keys/values before merging, not only recognized credential fields; OpenCode substitutes raw configuration text before parsing. Non-regular or symlinked leaf records stop. Nonempty plugin declarations in any layer, and `plugin`/`plugins` entries in the declared configuration directories, stop without reading plugin code. The legacy global `config` entry and macOS managed-preference files at `/Library/Managed Preferences/ai.opencode.managed.plist` and `/Library/Managed Preferences/<current OS username>/ai.opencode.managed.plist` are existence-checked only and stop as unsupported. No migration or configuration changes.

`OPENCODE_CONFIG_CONTENT`, `OPENCODE_AUTH_CONTENT`, `OPENCODE_TEST_HOME`, `OPENCODE_TEST_MANAGED_CONFIG_DIR` and `OPENCODE_CONSOLE_TOKEN` are unsupported when nonempty; do not inspect or persist their values. Unrepresented CLI, agent, session, plugin or remote-organization overrides remain unsupported, not silently inferred from saved defaults. Do not read account/session databases to guess them. An adapter failure does not establish that the user's working connection is absent or invalid.

## ZCode

Verified against Switch `crates/client-adapters/zcode/src/lib.rs` and the installed ZCode application runtime on 2026-09-08. Read only `~/.zcode/v2/config.json`, or `<ZCODE_DATA_BASE_DIR>/.zcode/v2/config.json` for an explicit absolute base. Do not read the legacy model-providers store, settings, sessions, or another client. Under `provider`, require one unique entry with `enabled: true`, a matching `options.baseURL` and `kind: openai-compatible`; use its inline `options.apiKey` in memory. Labels and the legacy root model field do not select the connection. Duplicate JSON keys, multiple matches, malformed data, unsupported credential references and missing credentials fail closed. This verifies configured API capability rather than current chat model identity. Fixtures are separate from real-host acceptance.

## Kimi Code and Qoder — 2026-09-10

Kimi Code reads only `KIMI_CODE_HOME/config.toml` or `~/.kimi-code/config.toml`. Resolve `default_model` through `models.<id>.provider`, then validate that provider's `base_url` before reading its inline `api_key`. Supported provider types are openai, openai_responses and anthropic. Names are lookup references, never Pure Tokens identity evidence. No legacy `.kimi`, login store, secondary model or environment credential fallback. Skills use the same root plus `skills`; doctor also checks the documented shared `.agents/skills` discovery root.

Qoder reads only its resolved user `settings.json`: absolute `QODER_CONFIG_DIR` takes priority, otherwise absolute `QODER_CLI_HOME` or home plus `QODER_CONFIG_DIR_NAME` (default `.qoder`). Select exactly one endpoint-matching entry from `providers`, require openai-compatible/openai and one inline `apiKey`; reject multiple matches without comparing keys. No daemon, account database or QoderWork access. Duplicate JSON keys and excessive depth are rejected. Its configured connection does not prove current chat selection.

Evidence: Switch client adapter source at commit 0ccc470 (`crates/client-adapters/kimi-code/src/lib.rs`, `qoder/src/lib.rs`); installed Kimi Code executable static source (no execution or configuration reads), sections `features/skill/catalog/skillRoots.ts` and data-home resolver, examined 2026-09-10; Qoder official Skills and CLI reference documentation. The older MoonshotAI/kimi-cli Python source uses different directories and was rejected as evidence for this new Kimi Code host. Qoder official user Skill directory: https://docs.qoder.com/cli/Skills . These observations and fixtures are not real-host end-to-end acceptance.

Unrepresented per-session configuration-file or model overrides must not be inferred from disk defaults: run only when the host's effective connection is represented by the documented file, otherwise stop with active_connection_unavailable.

Pi: Switch source `crates/client-adapters/pi/src/lib.rs` defines `PI_CODING_AGENT_DIR` or `~/.pi/agent`, and `models.json.providers` with `baseUrl`, `api`, `apiKey`, `models`. The root must be absolute without parent traversal. After finding one unique Pure Tokens endpoint using `openai-completions`, the executor may read only the sibling `auth.json`, associating its exact provider ID with that endpoint-verified entry. The ID is a lookup reference, not service identity evidence. A present entry takes precedence and must contain `type: api_key` and a supported inline `key`; malformed, unreadable, empty, OAuth or dynamic authentication stops without falling back. Only an absent file or absent matching entry allows consideration of the saved `models.json.apiKey`.

Pi provider IDs with built-in environment or platform authentication are blocked from the lower-priority models fallback unless a supported higher-priority auth entry exists. This is a rejection set, not an environment-reading allowlist. Other custom IDs may use their saved inline value. No commands, interpolation or escaped values are expanded; `$`, command forms and control characters stop locally. No other stores, arbitrary environment variables or fallback credentials are read. Never remove an active authentication entry to force fallback.

Authentication precedence and the rejection set were checked on 2026-09-12 against Pi revision `71dca871bc80b6bc97be37f0ca3189399d651fff`, `packages/coding-agent/docs/providers.md`, `docs/models.md`, `src/core/auth-storage.ts` and `packages/ai/src/env-api-keys.ts` in the official `badlogic/pi-mono` repository. These are version-scoped implementation sources, not real-host acceptance. CLI, extension and session overrides not represented by the declared files remain unsupported; stop when the effective connection cannot be confirmed. Saved availability does not establish current chat selection.

The raw `env-api-keys.ts` source has SHA-256 `6876b915ffe1342f8be69ef1f75721c8a265a332a1bbbf5afe36b6f80b332552`. All 36 mapped IDs plus the three special authentication IDs are covered by the Pi precedence regression; do not substitute a model-vendor list or a differently cached documentation result for this source.

## Hermes, EvoX, VS Code and Octop (2026-09-18)

Implementation sources: Switch `332959e`, `crates/client-adapters/{hermes,evox,vscode,octop}`;
official Hermes `hermes_cli/runtime_provider_custom.py`, `agent/credential_pool.py`,
`hermes_cli/auth.py`, `tools/skills_tool.py`; official VS Code Agent Skills
documentation; installed Octop 1.0.0 distribution source
`octop/infra/{utils/paths.py,agents/workspace_dir.py,db/repos/settings.py,agents/providers/store.py}`.
EvoX skill-root evidence is the installed application's embedded skill importer
and the canonical Switch adapter, not a claim of complete runtime acceptance.

- Hermes: resolve absolute `HERMES_HOME`, otherwise `~/.hermes`; Windows uses
  `%LOCALAPPDATA%/hermes`, with legacy HOME fallback only when absent.
  Read only `config.yaml`; `model.provider=custom:<id>` and `model.default`
  select `providers.<id>`. Verify its `base_url` before inspecting sibling
  `auth.json`. Support inline `api_key` and the three documented API modes.
  Reject model route overrides, selected aliases, dynamic key commands/env,
  alternate endpoint/header fields and any nonempty `credential_pool`.
  This conservative pool restriction includes pools not currently selected;
  do not remove or modify them to enable a lower-priority inline credential.
- EvoX: absolute `EVOX_AGENT_DIR` / `EVOX_CODING_AGENT_DIR` must agree;
  default `~/.evox/agent`. Read `settings.json` and, after endpoint recognition,
  the exact provider entry in `auth.yaml`. Require consistent
  `defaultProvider/defaultModel/defaultInstanceId` and
  `config.modelPolicy.userPreference`, one declared selected model, and
  matching endpoint/protocol with inline `type=api_key`.
  Only existence-check legacy `config.yaml/models.json/auth.json`; reject
  instead of migrating. Session/instance routing not represented here stops.
- VS Code: default local `Code/User/chatLanguageModels.json` under macOS
  Application Support or Windows APPDATA; parse native JSONC with unique keys.
  Recognize the exact full Chat/Responses/Messages URL before using a model's
  inline `requestHeaders.Authorization`. The distributed adapter requires one
  matching `customendpoint` group whose models all use supported Pure Tokens
  routes and the same inline Bearer credential. Compare only in executor
  memory; never expose values, fingerprints or comparison results. Distinct
  matching groups, different Keys, duplicate model IDs, mixed-service groups,
  dynamic credentials, discovery URLs and group authentication/route overrides
  stop without choosing a model or changing configuration. Names and ownership
  markers are not service identity evidence. No SecretStorage, extension,
  process control, named/portable profile, conversation history or inferred
  chat selection. This multi-model implementation has been distributed since 0.18.7.
  Switch's real writer and the adapter share a synthetic three-route
  fixture; this is not real-host or current-session verification. See
  `switch-skill-compatibility.md` for all five acceptance stages.
- Octop: absolute `OCTOP_HOME` or `~/.octop`; only `config.json` (storage
  selection), `octop.db` and its SQLite coordination files. Reject alternate
  database environment options, nondefault paths and PostgreSQL before opening
  storage. Open an existing database with `mode=ro`, `query_only=ON`,
  `trusted_schema=OFF`, bounded busy/total time and one read transaction.
  Include committed WAL; never use `immutable=1`, copy the database, migrate,
  create a missing DB, or invoke a host CLI. `settings.active_model` links the
  saved default by name only as a lookup reference; without a default require
  one unique enabled matching provider. Inspect endpoint/format/model first,
  then project only that row's `api_key`. No sessions/agents/tokens table
  reads. Compiled SQLite is an executor dependency, not a user runtime.

Paths reject traversal and non-regular/symlinked records; JSON rejects duplicate
keys and excessive depth, YAML additionally rejects aliases and multiple
documents. Unsupported effective overrides stop without environment credential
fallback. No fixture establishes the active conversation or real media delivery.

## Official DeepSeek Harness — 2026-10-03

The `deepseek-harness` adapter is independent from `dsh-desktop`. Reviewed official source revision: `deepseek-ai/deepseek-harness@da00f7f5358f2949383b35c14f548bc20187d80c`, especially `apps/desktop/src/paths.ts`, `packages/boot/app-boot/src/profile-context.ts`, `packages/credentials/credentials-local/src/index.ts`, `packages/llm/llm-pi-ai/src/{index,auth}.ts` and `docs/subsystems/skills.md`. Switch writer: `crates/client-adapters/deepseek-harness/src/lib.rs`.

Resolve absolute DSH_HOME without dot/parent traversal, otherwise ~/.dsh. Read only the optional home cordis.patch.yml (nonempty content stops) and profiles/desktop/cordis.patch.yml. The supported default patch has one enabled llm-pi-ai row and one agent-default-model row, without arbitrary plugin rows; YAML aliases, duplicate keys/rows and extra documents stop. Resolve selected provider/model structurally, verify the exact Pure Tokens /v1 endpoint and openai-completions protocol, and require the selected model in the declared list. Multiple matching providers and model/credential/header overrides stop; labels never identify the service.

Only then read .credentials.yaml version 1 and the exact apiKeyEnv reference under refs. A nonempty inherited value for that exact reference shadows the store in the official host, so this adapter refuses it without comparing or using either value. No .env, OAuth records, community settings.yaml, arbitrary environment fallback or conversation database is read. Unknown credential-store fields stop. CLI launch overlays, custom bundles and session overrides cannot be inferred from saved defaults and remain outside support.

Shared synthetic sample: test/fixtures/switch-deepseek-harness-connection.json; Switch validates it against actual writer output on macOS/Windows layouts. The adapter has been distributed since 0.18.7. Fixtures are not a real-host loading, current-session identity, API or attachment-delivery pass.

## MiniMax Code Desktop — distributed since 0.18.7

Independent `minimax-code` adapter, reviewed against installed Desktop 3.1.0 application code (not a running client or user configuration): `@mavis/config/dist/{config,byok-config,credential-reference}.js`, `@mavis/local-runtime-v2/dist/service/skill/registry/roots.js`, `service/model-system/resolution/model-resolver-byok.js`, and `service/turn-system/agent-host/preparation/config/session-model-selection.js`. Static file digests are recorded in `acceptance/client-model-sync-20261004.json`. CLI documentation alone is not Desktop evidence.

The root follows Switch writer `crates/client-adapters/minimax-code/src/paths.rs`: nonempty `MINIMAX_DATA_DIR`, then `MAVIS_DATA_DIR`, then exactly the `MiniMax Code`, `MiniMax`, `MiniMax Agent` directories under macOS `~/Library/Application Support` or Windows `%APPDATA%`, and exactly `minimax-agent-cn-config.json` / `minimax-agent-config.json`. Only `config.localRuntimeDataParentDir` selects a parent plus `.minimax`; conflicting selections, duplicate/unreadable JSON and unsafe paths stop. Otherwise use `~/.minimax`. Do not migrate `.mavis`, inspect sessions, or scan other files. Supported data-directory values are local absolute paths; declared CLI/runtime-profile overrides stop.

Read only `<root>/config.yaml`, with bounded duplicate-free YAML, no aliases/documents, symlinks/reparse points or hard links. `defaultModel` must select `custom_provider:<provider>/<exact-model>`; split once so model IDs containing `/` survive. Require the selected declared enabled model and enabled custom provider, a supported `openai-completions`, `openai-responses` or `anthropic-messages` protocol, and fixed Pure Tokens `/v1` connection identity before taking its inline Key. Labels do not identify a connection. Switch may write three protocol providers: the saved default selects one; never compare their Keys or select a sibling on failure. Foreign siblings and distinct sibling Keys do not authorize using them. Unsupported routing/header/auth overrides, dynamic references and OAuth stop; no official `minimax_api` credential or environment-Key fallback.

The adapter cannot observe `explicit request > session override > defaultModel` through this file. Host binding must confirm the default local conversation; unknown/overridden sessions stop. Shared synthetic `test/fixtures/switch-minimax-code-connection.json` matches Switch writer output on both OS layouts. Fixtures and Desktop static inspection are not real loading, current-session identity, dedicated API or native delivery acceptance.


## TRAE saved connection (source addition, 2026-10-20)

Official Skill discovery: international `~/.trae/skills`, domestic
`~/.trae-cn/skills`, and project `.trae/skills`; see
`https://docs.trae.cn/ide_skills` and `https://docs.trae.ai/ide/skills`.
The installed TRAE 3.6.4 distribution confirms peer-product Skill discovery
controls and the account-scoped model-cache storage format. These sources are
format evidence, not a version gate or real-host execution acceptance.

This adapter uses the saved connection created by Switch's native model service.
Only existence-check the three declared desktop databases. When more than one
exists, stop instead of picking an edition or borrowing another connection.
Read only the active receipt and the two same-account cache keys it declares,
in a bounded read-only SQLite transaction including committed WAL. Accept exactly the saved `/v1`, `/v1/` or `/v1/chat/completions` endpoint
forms written by Switch. Require one
credential fingerprint across the receipt's models and verify every saved model
against the native receipt before using its inline `ak`. Fingerprints remain
private integrity metadata in memory; do not emit them. Labels are not endpoint
identity, and unowned models cannot supply a credential.

Never decrypt TRAE login/SecretStorage or Switch recovery records, enumerate
other account caches, create a credential copy, connect over CDP, invoke a host
CLI, change a chat model or use a fallback connection. A missing receipt or
ambiguous edition/connection is an adapter diagnostic, not evidence that the
user has never configured TRAE. This supports the Switch-managed saved API
connection independently of current chat selection; manual/portable/remote
profiles and unpersisted session overrides are outside this contract.
Fixture coverage and local builds do not establish public release, actual
TRAE Skill loading, authenticated API access or attachment delivery.

The independent Switch writer contract is mirrored in
`runtime/executor/testdata/trae-switch-skill-connection.json`; keep it byte-identical
with Switch `tests/fixtures/client-configs/traework/skill-connection.json`. The Go
adapter consumes the actual receipt/cache shape, including the full Chat endpoint,
through a synthetic read-only SQLite store.
