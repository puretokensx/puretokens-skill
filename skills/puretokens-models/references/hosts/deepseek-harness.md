# deepseek-harness

Bound host: `--host deepseek-harness`.

Use `--host deepseek-harness` only in official DeepSeek Harness's default local Desktop session. This is distinct from community DSH Desktop (`dsh-desktop`). Install to absolute `DSH_HOME/skills`, otherwise `~/.dsh/skills`; project `.dsh/skills`, project `.agents/skills` and custom roots can shadow it. Verify the loaded Skill path. No Node installation is required by Skills.

The adapter reads the default Desktop Cordis patch and the selected version-1 credential reference. Higher-priority home patches, custom composition, multiple matching connections, model route/auth overrides and inherited credential overrides stop. CLI/remote sessions or session-specific provider selection not represented by this default profile are unsupported: stop before invoking the executor; never substitute another host. Saved default selection is not proof of current chat identity. API permission and native attachment handoff remain pending real-host acceptance.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
