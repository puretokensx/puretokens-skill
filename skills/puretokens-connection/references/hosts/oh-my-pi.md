# oh-my-pi

Bound host: `--host oh-my-pi`.

Oh My Pi uses `--host oh-my-pi`, never the Pi entry. Installation, doctor and credentials share the selected profile: validated `PI_CONFIG_DIR` (default `.omp`), then `OMP_PROFILE` before `PI_PROFILE`; an explicitly empty canonical profile means default. Named profiles use `<home>/<config>/profiles/<profile>/agent/skills` and ignore `PI_CODING_AGENT_DIR`; only the default may use that absolute override. The executor accepts up to four explicitly declared absolute `PI_CONFIG_FILES` overlays in order, using the resulting exact `modelRoles.default` and the selected profile's models file. Before invoking it, confirm the current local session uses this saved/file-overlay selection; absence of a visible override is not proof. Project settings, dotenv scopes and unrepresented session/runtime/extension or compatibility-provider selection remain unsupported. Do not inspect another client or profile to establish the connection.

This support shipped in Skills 1.0.0. Standalone installation fetches the published stable release; fixture coverage does not establish real-host API or attachment delivery acceptance.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
