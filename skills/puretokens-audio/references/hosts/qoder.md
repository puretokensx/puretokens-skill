# qoder

Bound host: `--host qoder`.

Use `--host qoder` locally, distinct from QoderWork. Use `QODER_CONFIG_DIR/skills`; otherwise `QODER_CLI_HOME` or home plus `QODER_CONFIG_DIR_NAME` (default `.qoder`), then `skills`. Supported directory overrides must be absolute. Saved connection verification does not identify current chat selection. Unrepresented command-line or session overrides stop instead of reading another connection. Missing native attachment capability reports downloaded but not delivered; never substitute a browser or another transport.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
