# pi

Bound host: `--host pi`.

Use `--host pi`. Skills use `~/.pi/agent/skills`, or absolute `PI_CODING_AGENT_DIR/skills` without parent traversal. The executor first verifies one matching `models.json` endpoint using `openai-completions`, then checks only sibling `auth.json` for the same provider ID. Its inline `api_key` entry takes precedence; unsupported or unreadable active authentication stops without fallback. With no matching auth entry, only custom providers without built-in environment authentication may use the saved inline value. Commands, `$` interpolation/escapes and unrepresented CLI, extension or session overrides are unsupported. Never remove active authentication to force fallback. Saved connection verification does not prove current chat selection. Real API and attachment delivery remain unverified.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
