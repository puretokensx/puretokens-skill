# Claude Desktop mapped Skills connection

Skills support shipped in 1.0.0. This contract is owned jointly with Switch
`docs/decisions/ADR-0008-claude-desktop-model-mapping.md`. Both projects require
compatible versions and a valid same-host connection written by Switch Apply.
Skill installation alone does not create that record or prove real-host delivery.
Version 2 below is source-only until matching executor and Switch releases ship.

## One application action

Switch's existing Apply action configures chat and Skills together. It discloses
the local plaintext Key in the existing confirmation, without an additional
permission dialog or asking the user to copy credentials. Skills install remains
separate. The chat profile keeps its local token; dedicated requests never use
the mapper, another host, a credential exchange or a new background service.

## Minimal connection handoff

Only local Desktop Code sessions confirmed to use the saved Desktop 3P connection
are in scope. Cloud/Cowork, remote execution or unrepresented session overrides
stop; an installed Skill alone does not establish this binding.

The root is macOS `~/Library/Application Support/Claude-3p`, or Windows
`%LOCALAPPDATA%/Claude-3p`. Read only:

1. `claude_desktop_config.json`: `deploymentMode=3p`.
2. `configLibrary/_meta.json`: one valid active UUID in `appliedId`.
3. That UUID's profile: gateway/static/bearer, the exact
   `http://127.0.0.1:15722/claude-desktop` endpoint and a `pts-local-` token.
4. `puretokens-skills-connection.json`: the same-host dedicated connection.

Version 2 has exactly seven active fields: `version=2`, `host=claude-desktop`,
`state=active`, `local_gateway_token`, `api_base_url`, `api_key`, `connection_id`.
It contains no `profile_id`, profile hash or device fingerprint. Compare
`local_gateway_token` with the already configured active profile's token only
in memory. This is the existing mapper token, not a new credential or handshake;
never send it to the public API or export it in diagnostics. It confirms which
applied connection owns the dedicated Key. No manual user configuration is added.
Any active UUID may resolve the same supported connection. JSON formatting,
model lists and unrelated preferences do not affect it. The final concurrent
check compares only deployment mode, active selection, routing/auth fields and
the complete parsed dedicated record; it rejects a changed connection but
allows unrelated settings and serialization changes. It never adopts new
credentials mid-command. Arbitrary endpoints/auth schemes and unrepresented
session overrides remain unsupported; custom preferences are not such overrides.

Legacy version 1 retains its eight fields: `version=1`, `host=claude-desktop`,
`profile_id`, `state=active`, `profile_sha256`, `api_base_url`, `api_key`,
`connection_id`. Its fixed managed UUID is
`70757265-746f-4000-8000-636c61756465`; the hash binds actual profile bytes,
including line endings. Keep strict v1 checks and byte-exact concurrent reads
until Switch Apply upgrades the record. A legacy mismatch cannot prove that
the saved Key is still current, so never ignore or recompute it in the reader.

The API origin
must be an existing supported Pure Tokens origin; it never becomes the request
target. The Key must not be a local mapper token. Reject duplicate JSON fields,
symlinks/reparse points, hardlinks, inconsistent snapshots and unknown records.
Unix requires a current-user-owned private file. Windows additionally requires
current-user ownership and a non-null DACL granting access only to that user,
SYSTEM or local administrators; unrecognized grants fail closed. The writer
uses the existing AppData transaction permissions. Windows ACL tests compile
but real-device execution remains pending; storage is plaintext, not encrypted.

Switch writes all four files in its existing CAS/backup/rollback transaction.
Only Apply may enroll a missing record; login rehydration may maintain an
already enrolled connection. Reapplying the same upstream/Key preserves the
random UUID, even when chat mappings change. A different upstream/Key creates
a new UUID. This equality check is confined to the same-host Switch writer,
with no persisted Key hash or comparison output.

The new reader accepts valid v1 and v2 records. It must not reinterpret a v1
byte hash as a local-token binding or bypass a mismatch. Only an explicit Switch Apply
creates/upgrades v2; background rehydration preserves v1 for old readers and
retains the existing exact journal/CAS checks. Same-connection migration preserves
`connection_id` and existing task recovery. Old readers reject v2 safely: release
the v1/v2 reader before the v2 writer, then upgrade Skills before applying the
new Switch configuration. Do not manually edit records, downgrade v2 to v1,
restore stale credentials or automatically regenerate a task. No automatic v2
downgrade is implemented: retain a compatible reader while v2 is active. An
older reader or writer may stop safely; returning to it needs a separately
reviewed migration, not hash guessing or an assumed old-version Apply path.

Like other applied client Keys, this file is independent of Switch login.
Logout stops chat mapping and clears Switch memory, but does not remove the
applied client connection. Switching accounts alone does not rebind it. Applying
another account's configuration replaces it through the normal transaction.
Direct reconfiguration removes the companion's Key and local token by writing
`state=inactive` (v2 contains only `version`, `host`, `state`);
official restore removes only an owned record. Existing private transaction
backups can contain old client credentials under Switch's backup policy.

## Requests and recovery

All six dedicated capabilities use the existing executor, exact model IDs and
fixed routes. No model-directory/health/connection preflight is added. API
permission and real attachment delivery remain independent acceptance checks.

Mapped image/video/async music require an explicit absolute `--record` even
for one result. Submit persists only the opaque `connection_id`, never the Key
or its hash. Remote status/wait/content/resume/redeliver must match that ID.
On `task_connection_changed`, preserve the task and files, stop without a
request, and obtain support for original-connection recovery. Never edit the
binding, try another Key, resubmit or automatically switch models. Pure local
terminal recovery, delivery acknowledgment and verified local redelivery remain
available. Synchronous audio/Jev/embedding/rerank have no remote task recovery.

Missing legacy companion returns `active_connection_endpoint_unsupported`: no
request was sent and installation did not fail. Explain that a compatible
Switch must apply this client's configuration; do not reinstall to repair it or
promise an unpublished stable update. Inactive, changed or ambiguous records
stop with sanitized connection diagnostics.
Connection mismatch retains `active_connection_selection_unconfirmed`, but its
safe message distinguishes a legacy byte-bound record from a v2 local-connection
mismatch. Direct users to compatible Switch Apply, not manual fingerprint or
configuration edits, reinstallation or regeneration. Neither message exposes
configuration, hash, token, file path or API Key, nor claims to identify the
failing branch of an older generic receipt. This v2 supersedes the unpublished
five-field semantic-hash draft; that draft is not a supported wire version.
