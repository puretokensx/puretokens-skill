# minimax-code

Bound host: `--host minimax-code`.

Use `--host minimax-code` only in MiniMax Code Desktop 3.1.0 macOS/Windows local conversations confirmed to use the saved default connection. CLI profiles, remote execution and session/request-specific model selection are unsupported. Desktop priority is explicit request → session override → saved default; a saved default does not identify an overridden conversation. If that binding is unknown, stop before the executor. Never borrow the official MiniMax login or another host's connection.

Desktop loads `<runtime-data>/skills`. Use the explicitly supplied `MINIMAX_DATA_DIR`, then `MAVIS_DATA_DIR`; without overrides or Desktop directory preferences the default is `~/.minimax`. The standalone installer does not parse Desktop settings: if either regional preferences store exists, provide the already confirmed absolute Skill target or run within the current Desktop runtime with its declared data root. It refuses to guess or migrate `.mavis`. The credential adapter separately resolves only the documented preferences files and the saved default in `<runtime-data>/config.yaml`. Verify the loaded Skill path because project, agent and shared roots may shadow it; doctor only examines the selected global root and loaded installation, never session stores. Real Skill loading, API and attachment delivery remain pending.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
