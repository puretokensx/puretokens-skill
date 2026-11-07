# codex

Bound host: `--host codex`.

Switch 和 Web 的客户端展示名为 ChatGPT；原生宿主入口仍是 Codex。
绑定 `--host codex`，安装目录保持 `~/.agents/skills`，适配器仍读取声明的
`.codex` 有效连接文件。显示名称不能用来猜测当前会话或更换凭据来源；
自动检测仍只认已验证的原生启动器，未知 Electron／浏览器边界须明确选择。

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
