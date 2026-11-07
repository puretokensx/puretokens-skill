# trae

Bound host: `--host trae`.

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

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
