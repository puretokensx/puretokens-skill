# octop

Bound host: `--host octop`.

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

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
