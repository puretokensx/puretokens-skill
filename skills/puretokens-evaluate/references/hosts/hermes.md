# hermes

Bound host: `--host hermes`.

Use `--host hermes` locally. Use `HERMES_HOME/skills`, default `~/.hermes/skills` on macOS/Linux. Windows uses `%LOCALAPPDATA%/hermes/skills`; the legacy home root is used only if the current root does not exist. Relative or parent-traversing overrides stop.

Use only the selected inline custom provider without competing model routes, dynamic key sources or credential pools. Unknown formats and unrepresented profile/session overrides stop; do not migrate or rewrite configuration.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
