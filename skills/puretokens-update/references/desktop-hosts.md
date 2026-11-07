# Host reference index

宿主绑定与本地失败停止：首次调用前按当前应用或用户明确指定绑定一个 host，全部命令沿用，不从安装目录推断。`active_connection_*`、`workbuddy_*` 或 `host_credential_adapter_unavailable` 本地失败后停止 API 流程，保留用户原任务和已有 task ID；不得自动调用 puretokens-connection、init、doctor、models 或其他 Skill 探测，不得枚举、更换 `--host` 或借用凭据。仅用户明确重查或有可验证的新依据时，在原 host 检查一次；换宿主须用户明确指定。“继续生成”本身不是重查、换宿主或重提授权；本次未发送不代表此前未提交。

Only for a host-specific limitation or failure: read `hosts/<bound-host>.md` below. If the host is already bound, go directly to that file; do not read other clients or repeat this lookup during the same unchanged session. Ordinary requests need no extra preflight.

Shared installation files never select the host. The current session or explicit user choice supplies `--host`; connection presence alone is not a routing signal. Do not trim shared files to one client or replace another client's configuration.

For a local connection failure, see [sanitized diagnostics](host-errors.md). An unlisted host must stop; see [unsupported boundaries](unsupported-hosts.md), never substitute another client.

- [claude-code](hosts/claude-code.md)
- [codex](hosts/codex.md)
- [workbuddy](hosts/workbuddy.md)
- [gemini-cli](hosts/gemini-cli.md)
- [grok-build](hosts/grok-build.md)
- [opencode](hosts/opencode.md)
- [trae](hosts/trae.md)
- [claude-desktop](hosts/claude-desktop.md)
- [dsh-desktop](hosts/dsh-desktop.md)
- [deepseek-harness](hosts/deepseek-harness.md)
- [minimax-code](hosts/minimax-code.md)
- [zcode](hosts/zcode.md)
- [kimi-code](hosts/kimi-code.md)
- [qoder](hosts/qoder.md)
- [pi](hosts/pi.md)
- [hermes](hosts/hermes.md)
- [evox](hosts/evox.md)
- [vscode](hosts/vscode.md)
- [octop](hosts/octop.md)
- [oh-my-pi](hosts/oh-my-pi.md)
- [minimax-code-cli](hosts/minimax-code-cli.md)
- [qwen-code](hosts/qwen-code.md)
- [codebuddy-code](hosts/codebuddy-code.md)
- [copilot-cli](hosts/copilot-cli.md)
- [factory-droid](hosts/factory-droid.md)
- [mimo-coder](hosts/mimo-coder.md)
- [paseo](hosts/paseo.md)
