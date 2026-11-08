# Desktop host execution

## paseo

Use the actual host ID `paseo`, not Codex because it shares an engine. Install to `~/.agents/skills`, shared with Codex; `doctor` checks declared roots only. This conditional support shipped in Skills 1.0.0; real-host API/attachment acceptance remains pending.

Local Paseo is restricted to its Codex engine and a confirmed saved custom provider. Require the existing absolute PASEO_HOME plus inherited connection matching; if unavailable, stop with a sanitized selection error. Do not set environment credentials to manufacture a match, read independent Codex auth, or access a remote daemon. An inner Codex process is not sufficient to classify the host.

Unknown project/session/runtime overrides stop; do not guess a selected model or read another client. Reuse existing dedicated APIs and same-task recovery. Native media handoff requires an actual current-session capability; a path or successful init is not attachment delivery. Never manufacture a root override to bypass a mismatch.

## oh-my-pi

Oh My Pi uses `--host oh-my-pi`, never the Pi entry. Installation, doctor and credentials share the selected profile: validated `PI_CONFIG_DIR` (default `.omp`), then `OMP_PROFILE` before `PI_PROFILE`; an explicitly empty canonical profile means default. Named profiles use `<home>/<config>/profiles/<profile>/agent/skills` and ignore `PI_CODING_AGENT_DIR`; only the default may use that absolute override. The executor accepts up to four explicitly declared absolute `PI_CONFIG_FILES` overlays in order, using the resulting exact `modelRoles.default` and the selected profile's models file. Before invoking it, confirm the current local session uses this saved/file-overlay selection; absence of a visible override is not proof. Project settings, dotenv scopes and unrepresented session/runtime/extension or compatibility-provider selection remain unsupported. Do not inspect another client or profile to establish the connection.

This support shipped in Skills 1.0.0. Standalone installation fetches the published stable release; fixture coverage does not establish real-host API or attachment delivery acceptance.

## minimax-code-cli

MiniMax Code CLI (`mcode`) uses `--host minimax-code-cli`; Desktop retains `--host minimax-code`. CLI resolves `MINIMAX_DATA_DIR`, then `MAVIS_DATA_DIR`, then `~/.minimax`, without Desktop preferences. Its installer loads `<root>/skills`. Only a confirmed local saved-default conversation is supported; profiles and unrepresented overrides stop. This support shipped in Skills 1.0.0. Standalone installation fetches the published stable release; real-host API and attachment delivery acceptance remain pending.

## claude-desktop

Claude Desktop direct profiles retain existing support. Mapped-mode Skills support shipped in 1.0.0 and requires Switch's same-host applied Skills connection, not its local chat token. Read [the exact binding and lifecycle contract](claude-mapped-connection.md) only for mapped mode. Missing legacy records return `active_connection_endpoint_unsupported` without an API request; compatible Switch configuration application is required, not reinstallation or another host's Key.

Use --host claude-desktop in a local Desktop Code session. Local Code sessions discover ~/.claude/skills (or the explicitly configured CLAUDE_CONFIG_DIR/skills); this directory is shared with Claude Code. The executor uses the active Desktop third-party gateway profile, never Claude Code settings as a fallback. Restart/start a new local session after installation.

For mapped mode, confirm that this local session uses the saved Desktop 3P connection, then use an explicit absolute task record for every asynchronous media submission. The executor binds recovery to the same random connection identity. A changed connection stops remote recovery, never creates a replacement; verified local files can still be delivered. No extra consent prompt is needed for an already requested task within the applied connection.

Cloud chats and Cowork isolated environments may not have access to the macOS/Windows connection records or executable. Do not copy credentials into them, infer availability from a successful installation, or use a browser/MCP transport. Report the actual local failure. Host attachment handoff must be verified in the selected execution mode.

## dsh-desktop

Use --host dsh-desktop. The local Harness discovers DSH_HOME/skills. Without an explicit DSH_HOME, the installer uses ~/Library/Application Support/dsh-desktop/harness/skills on macOS or %APPDATA%/dsh-desktop/harness/skills on Windows. Quote paths containing spaces. The desktop supplies DSH_HOME to Harness; explicit absolute overrides select that one profile, never a directory search.

Project .dsh/skills, project .agents/skills and custom roots can shadow user skills. Check the loaded SKILL.md location; doctor checks the selected user root and the documented shared Agents root, not arbitrary project/plugin roots. Use the local session's attachment handoff after content; downloaded_awaiting_host_delivery is not a successful handoff.

DSH settings select agent-default-model.provider/model and llm-pi-ai.providers.<selected>. Its apiKeyEnv references exactly one entry in the sibling version-1 .credentials.yaml refs mapping or a strictly validated legacy flat mapping. Flat values must all be nonempty strings with POSIX reference names; reserved version/refs/records fields, mixed layouts and missing references stop. The executor checks the selected endpoint before reading that file; no process-environment credential fallback or migration.

## deepseek-harness

Use `--host deepseek-harness` only in official DeepSeek Harness's default local Desktop session. This is distinct from community DSH Desktop (`dsh-desktop`). Install to absolute `DSH_HOME/skills`, otherwise `~/.dsh/skills`; project `.dsh/skills`, project `.agents/skills` and custom roots can shadow it. Verify the loaded Skill path. No Node installation is required by Skills.

The adapter reads the default Desktop Cordis patch and the selected version-1 credential reference. Higher-priority home patches, custom composition, multiple matching connections, model route/auth overrides and inherited credential overrides stop. CLI/remote sessions or session-specific provider selection not represented by this default profile are unsupported: stop before invoking the executor; never substitute another host. Saved default selection is not proof of current chat identity. API permission and native attachment handoff remain pending real-host acceptance.

## unsupported

Antigravity CLI remains unsupported because its process-only credential and Skill execution binding are not verified. Rakazo, OpenMausBot and Cursor are not registered Skill hosts. Do not substitute a supported host ID.

Switch's `deepseek-bot` is a separate community client, not `deepseek-harness`
or `dsh-desktop`. It currently has no registered Skills host, installation
target or credential adapter. Do not pass either Harness host ID or read its
connection as a substitute. Switch configuration alone does not enable Skills.

Its per-Bot model selection must be bound to the current execution session
before an adapter can be enabled. Skill discovery, effective connection and
native result delivery still need independent implementation and acceptance;
a saved default or successful chat does not establish those capabilities.

## minimax-code

Use `--host minimax-code` only in MiniMax Code Desktop 3.1.0 macOS/Windows local conversations confirmed to use the saved default connection. CLI profiles, remote execution and session/request-specific model selection are unsupported. Desktop priority is explicit request → session override → saved default; a saved default does not identify an overridden conversation. If that binding is unknown, stop before the executor. Never borrow the official MiniMax login or another host's connection.

Desktop loads `<runtime-data>/skills`. Use the explicitly supplied `MINIMAX_DATA_DIR`, then `MAVIS_DATA_DIR`; without overrides or Desktop directory preferences the default is `~/.minimax`. The standalone installer does not parse Desktop settings: if either regional preferences store exists, provide the already confirmed absolute Skill target or run within the current Desktop runtime with its declared data root. It refuses to guess or migrate `.mavis`. The credential adapter separately resolves only the documented preferences files and the saved default in `<runtime-data>/config.yaml`. Verify the loaded Skill path because project, agent and shared roots may shadow it; doctor only examines the selected global root and loaded installation, never session stores. Real Skill loading, API and attachment delivery remain pending.

## codex

Switch 和 Web 的客户端展示名为 ChatGPT；原生宿主入口仍是 Codex。
绑定 `--host codex`，安装目录保持 `~/.agents/skills`，适配器仍读取声明的
`.codex` 有效连接文件。显示名称不能用来猜测当前会话或更换凭据来源；
自动检测仍只认已验证的原生启动器，未知 Electron／浏览器边界须明确选择。

## zcode

Use host `zcode` in a local execution environment. Install into `~/.zcode/skills`, or `ZCODE_DATA_BASE_DIR/.zcode/skills` when that explicit base directory is absolute. Refresh and enable the installed Skills in ZCode. The executor reads only the v2 connection store and uses one uniquely enabled matching Pure Tokens connection. This confirms API capability, not the current conversation model. Route media here only when host context explicitly selects Pure Tokens or the user explicitly requests it; connection presence alone is not a routing signal. Multiple matching enabled connections stop before a request. Legacy migration belongs to ZCode; no legacy-store fallback is attempted. Remote workspaces need their own accessible executor and connection; syncing Skills alone does not supply either. Native image/video attachment delivery remains pending real-host acceptance.

## kimi-code

Use `--host kimi-code` in the local execution environment, distinct from legacy Kimi CLI. Use `KIMI_CODE_HOME/skills` or `~/.kimi-code/skills`; supported directory overrides must be absolute. Unrepresented command-line or session configuration overrides stop instead of reading another connection. Missing native attachment capability reports downloaded but not delivered; do not open a browser or substitute another transport. Real API and native attachment delivery remain separate acceptance checks.

## qoder

Use `--host qoder` locally, distinct from QoderWork. Use `QODER_CONFIG_DIR/skills`; otherwise `QODER_CLI_HOME` or home plus `QODER_CONFIG_DIR_NAME` (default `.qoder`), then `skills`. Supported directory overrides must be absolute. Saved connection verification does not identify current chat selection. Unrepresented command-line or session overrides stop instead of reading another connection. Missing native attachment capability reports downloaded but not delivered; never substitute a browser or another transport.

## workbuddy

WorkBuddy 适配器无法匹配连接时，不据此断言当前聊天未使用 Pure Tokens；若用户确认已配置并成功使用 Pure Tokens，保留可用连接，按具体诊断检查适配；不能把自带模型聊天正常当作 Pure Tokens 配置有效的证据。排查仅提供客户端版本及脱敏诊断码，不读取或索取配置原文。

`init` 按当前宿主适配器验证已声明的有效连接，不要求用户为检查而切换聊天模型；配置选择规则依宿主而定，不能将此结果当作当前会话选择证明。WorkBuddy 使用 `workbuddy_record_missing`、`workbuddy_record_unreadable`、`workbuddy_record_format_unsupported`、`workbuddy_connection_not_found`、`workbuddy_credential_missing`、`workbuddy_connection_ambiguous` 区分本地原因；这些状态不代表安装失败，也不证明用户从未配置过。只反馈脱敏状态，不读取配置原文。

## common

各系统的 `cleanup_status: pending` 表示清理未完成，不等于文件同步失败；按独立 init 结果说明连接状态，不绕过删除守卫或自动重装。锁文件存在不等于被占用；缺少管理清单不能归因于清理失败，也不能称为无害。

连接验证失败必须按脱敏状态解释，适用于所有宿主。`active_connection_record_missing` 只说明执行环境中未找到适配器声明的记录；`active_connection_unavailable` 表示读取或解释未完成；`active_connection_selection_unconfirmed` 表示无法确认有效选择；`active_connection_format_unsupported`、`active_connection_endpoint_missing`、`active_connection_endpoint_unsupported` 表示适配器支持范围或字段识别问题；历史码 `active_connection_not_puretokens` 仅表示未通过固定端点识别，不证明用户从未配置。`active_connection_credential_missing` 表示适配器未取得可用凭据，不等于服务端判定凭据无效。`active_connection_ambiguous` 表示无法安全选择，不要求删除其他连接。保留现有可用连接，反馈宿主版本和诊断码；不展示配置，不要求重装、切换聊天模型、重配或更换凭据。`api_request_executed: false` 时明确本次未发送 API 请求、未认证凭据。`host_credential_adapter_unavailable` 表示执行环境不受支持，不是用户没配置。身份或网络检查失败也不能归因于未配置。

<!-- host-binding -->

## opencode

Use `--host opencode`. The executor merges declared global, explicit-file, project, `.opencode` and system-managed JSON/JSONC layers. It honors XDG roots, absolute `OPENCODE_CONFIG`/`OPENCODE_CONFIG_DIR`, and `OPENCODE_DISABLE_PROJECT_CONFIG`. A configured default must select an enabled provider; an absent default permits exactly one endpoint-matching saved Pure Tokens service. Multiple matches stop without comparing keys. Do not add a default model or change the user's chat selection to make init pass.

After verifying the endpoint, the executor checks the matching native API-key entry in OpenCode's XDG data-root `auth.json`; static inline provider credentials take precedence. No reference expansion, environment-key fallback, configuration changes or session/history inspection. Plugins, dynamic/remote authentication, managed preferences and unrepresented session overrides are unsupported. Saved service verification is not current chat identity and is not a routing signal by itself; route here only when host context selects Pure Tokens or the user explicitly requests it. Installation, API verification and native attachment delivery are separate checks.

Installation selects `OPENCODE_CONFIG_DIR/skills` when explicitly set, otherwise `XDG_CONFIG_HOME/opencode/skills`, defaulting to `~/.config/opencode/skills`. These overrides must be absolute without parent traversal. A user-specified absolute target remains available; it does not establish host discovery.

Doctor checks the loaded root, selected/global OpenCode roots and the documented `~/.agents/skills` and `~/.claude/skills` compatibility roots. It inspects only the ten managed Skill names and the executor, not credentials in those shared roots. It does not enumerate projects or arbitrary `skills.paths`. Duplicate detection does not establish which copy the host selected and never deletes copies. Verify the actual loaded Skill location and version in a fresh local session; installing new bytes alone does not prove that OpenCode loaded them.

## pi

Use `--host pi`. Skills use `~/.pi/agent/skills`, or absolute `PI_CODING_AGENT_DIR/skills` without parent traversal. The executor first verifies one matching `models.json` endpoint using `openai-completions`, then checks only sibling `auth.json` for the same provider ID. Its inline `api_key` entry takes precedence; unsupported or unreadable active authentication stops without fallback. With no matching auth entry, only custom providers without built-in environment authentication may use the saved inline value. Commands, `$` interpolation/escapes and unrepresented CLI, extension or session overrides are unsupported. Never remove active authentication to force fallback. Saved connection verification does not prove current chat selection. Real API and attachment delivery remain unverified.

## hermes

Use `--host hermes` locally. Use `HERMES_HOME/skills`, default `~/.hermes/skills` on macOS/Linux. Windows uses `%LOCALAPPDATA%/hermes/skills`; the legacy home root is used only if the current root does not exist. Relative or parent-traversing overrides stop.

Use only the selected inline custom provider without competing model routes, dynamic key sources or credential pools. Unknown formats and unrepresented profile/session overrides stop; do not migrate or rewrite configuration.

## evox

Use `--host evox` locally. Use `EVOX_AGENT_DIR/skills`, or `EVOX_CODING_AGENT_DIR/skills`, default `~/.evox/agent/skills`. Conflicting, relative or parent-traversing overrides stop. Require consistent provider/model/default-instance settings and matching canonical authentication. Unknown formats, legacy stores and unrepresented profile/session overrides stop; do not migrate or rewrite configuration.

## vscode

Use `--host vscode` in the local default profile; Skills install to
`~/.copilot/skills`. Named profiles, portable mode, SSH/WSL and remote hosts
are outside this adapter's verified scope. The native customendpoint file must
have one uniquely matching group; all its models must use supported Pure Tokens
routes and one identical inline Bearer credential. The distributed adapter verifies
this only in executor memory without reporting comparison data. Different
Keys, multiple matching groups, dynamic authentication, discovery/route
overrides or mixed-service groups stop. Do not remove models, alter chat
selection or inspect configuration to force init to pass. This identifies
the saved connection, not the selected chat model; route here only when
the user or current host context explicitly requests Pure Tokens.
Unrepresented session/profile overrides remain unsupported and cannot be
inferred from this file. The multi-model implementation has been distributed
since 0.18.7; fixtures do not establish real API or attachment delivery.

## octop

Use `--host octop` and an explicit absolute `--target` (PowerShell `-Target`)
for the current local agent workspace's `.octop/skills` directory. Obtain
the workspace from the host's current context, not an agent/database scan.
There is no assumed global Skill root. Do not install into `skill-packages`
or change Octop's package database. Doctor checks only the loaded installation.

Connection reads use `OCTOP_HOME` or `~/.octop`, the documented local SQLite
database, and a read-only transaction including committed WAL. A saved default
selects the exact provider; without one, only a unique enabled matching
service is supported. This is saved API capability, not current conversation
identity. PostgreSQL, alternate database paths and session/backend overrides
stop. Sandboxed workspaces without access to the same host connection must
stop; never copy credentials into the workspace.

Installation, native execution, API authorization and host attachment handoff remain separate acceptance.

## trae

TRAE supports Skills. International installations use `~/.trae/skills`, domestic
installations use `~/.trae-cn/skills`; project Skills use `.trae/skills`.
Run the installer with the bound `--host trae`. With both global roots present,
confirm the directory for the current edition and pass an explicit absolute
target; do not install duplicate copies or delete the other edition's files.
Refresh/enable the installed Skills in TRAE's Skill settings and start a new
conversation. Requires a Skills release containing the new TRAE adapter.

Only the native executor resolves the single Switch-managed saved connection
through its declared read-only adapter. It neither claims the current chat/account
selection nor requires changing that selection. Missing/edited receipts,
multiple desktop state stores or different saved group credentials stop safely.
Do not decrypt login/recovery data, read other account caches, switch hosts or
fall back to another credential. Native Skill loading, fixed-API authentication
and file handoff remain separate real-host acceptance steps.
