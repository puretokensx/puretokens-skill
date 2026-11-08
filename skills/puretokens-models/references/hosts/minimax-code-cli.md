# minimax-code-cli

Bound host: `--host minimax-code-cli`.

MiniMax Code CLI (`mcode`) uses `--host minimax-code-cli`; Desktop retains `--host minimax-code`. CLI resolves `MINIMAX_DATA_DIR`, then `MAVIS_DATA_DIR`, then `~/.minimax`, without Desktop preferences. Its installer loads `<root>/skills`. Only a confirmed local saved-default conversation is supported; profiles and unrepresented overrides stop. This support shipped in Skills 1.0.0. Standalone installation fetches the published stable release; real-host API and attachment delivery acceptance remain pending.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
