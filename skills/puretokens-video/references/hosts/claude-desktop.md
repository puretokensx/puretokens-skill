# claude-desktop

Bound host: `--host claude-desktop`.

Claude Desktop direct profiles retain existing support. Mapped-mode Skills support shipped in 1.0.0 and requires Switch's same-host applied Skills connection, not its local chat token. Read [the exact binding and lifecycle contract](../claude-mapped-connection.md) only for mapped mode. Missing legacy records return `active_connection_endpoint_unsupported` without an API request; compatible Switch configuration application is required, not reinstallation or another host's Key.

Use --host claude-desktop in a local Desktop Code session. Local Code sessions discover ~/.claude/skills (or the explicitly configured CLAUDE_CONFIG_DIR/skills); this directory is shared with Claude Code. The executor uses the active Desktop third-party gateway profile, never Claude Code settings as a fallback. Restart/start a new local session after installation.

For mapped mode, confirm that this local session uses the saved Desktop 3P connection, then use an explicit absolute task record for every asynchronous media submission. The executor binds recovery to the same random connection identity. A changed connection stops remote recovery, never creates a replacement; verified local files can still be delivered. No extra consent prompt is needed for an already requested task within the applied connection.

Cloud chats and Cowork isolated environments may not have access to the macOS/Windows connection records or executable. Do not copy credentials into them, infer availability from a successful installation, or use a browser/MCP transport. Report the actual local failure. Host attachment handoff must be verified in the selected execution mode.

Read [sanitized diagnostics](../host-errors.md) only after a local failure.
