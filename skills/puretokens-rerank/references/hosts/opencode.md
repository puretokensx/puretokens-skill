# opencode

Bound host: `--host opencode`.

Use `--host opencode`. The executor merges declared global, explicit-file, project, `.opencode` and system-managed JSON/JSONC layers. It honors XDG roots, absolute `OPENCODE_CONFIG`/`OPENCODE_CONFIG_DIR`, and `OPENCODE_DISABLE_PROJECT_CONFIG`. A configured default must select an enabled provider; an absent default permits exactly one endpoint-matching saved Pure Tokens service. Multiple matches stop without comparing keys. Do not add a default model or change the user's chat selection to make init pass.

After verifying the endpoint, the executor checks the matching native API-key entry in OpenCode's XDG data-root `auth.json`; static inline provider credentials take precedence. No reference expansion, environment-key fallback, configuration changes or session/history inspection. Plugins, dynamic/remote authentication, managed preferences and unrepresented session overrides are unsupported. Saved service verification is not current chat identity and is not a routing signal by itself; route here only when host context selects Pure Tokens or the user explicitly requests it. Installation, API verification and native attachment delivery are separate checks.

Installation selects `OPENCODE_CONFIG_DIR/skills` when explicitly set, otherwise `XDG_CONFIG_HOME/opencode/skills`, defaulting to `~/.config/opencode/skills`. These overrides must be absolute without parent traversal. A user-specified absolute target remains available; it does not establish host discovery.

Doctor checks the loaded root, selected/global OpenCode roots and the documented `~/.agents/skills` and `~/.claude/skills` compatibility roots. It inspects only the ten managed Skill names and the executor, not credentials in those shared roots. It does not enumerate projects or arbitrary `skills.paths`. Duplicate detection does not establish which copy the host selected and never deletes copies. Verify the actual loaded Skill location and version in a fresh local session; installing new bytes alone does not prove that OpenCode loaded them.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
