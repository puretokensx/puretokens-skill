# dsh-desktop

Bound host: `--host dsh-desktop`.

Use --host dsh-desktop. The local Harness discovers DSH_HOME/skills. Without an explicit DSH_HOME, the installer uses ~/Library/Application Support/dsh-desktop/harness/skills on macOS or %APPDATA%/dsh-desktop/harness/skills on Windows. Quote paths containing spaces. The desktop supplies DSH_HOME to Harness; explicit absolute overrides select that one profile, never a directory search.

Project .dsh/skills, project .agents/skills and custom roots can shadow user skills. Check the loaded SKILL.md location; doctor checks the selected user root and the documented shared Agents root, not arbitrary project/plugin roots. Use the local session's attachment handoff after content; downloaded_awaiting_host_delivery is not a successful handoff.

DSH settings select agent-default-model.provider/model and llm-pi-ai.providers.<selected>. Its apiKeyEnv references exactly one entry in the sibling version-1 .credentials.yaml refs mapping or a strictly validated legacy flat mapping. Flat values must all be nonempty strings with POSIX reference names; reserved version/refs/records fields, mixed layouts and missing references stop. The executor checks the selected endpoint before reading that file; no process-environment credential fallback or migration.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
