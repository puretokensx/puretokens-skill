# Claude Desktop mapped Skills connection v1

Status: unreleased source support. This contract is owned jointly with Switch
`docs/decisions/ADR-0008-claude-desktop-model-mapping.md`. Both projects must ship
compatible versions; installing the existing 0.19.8 release cannot enable it.

## One application action

Switch's existing Apply action configures chat and Skills together. It discloses
the local plaintext Key in the existing confirmation, without an additional
permission dialog or asking the user to copy credentials. Skills install remains
separate. The chat profile keeps its local token; dedicated requests never use
the mapper, another host, a credential exchange or a new background service.

## Exact file binding

Only local Desktop Code sessions confirmed to use the saved Desktop 3P connection
are in scope. Cloud/Cowork, remote execution or unrepresented session overrides
stop; an installed Skill alone does not establish this binding.

The root is macOS `~/Library/Application Support/Claude-3p`, or Windows
`%APPDATA%/Claude-3p`. Read only:

1. `claude_desktop_config.json`: `deploymentMode=3p`.
2. `configLibrary/_meta.json`: active `appliedId` equals the fixed managed UUID
   `70757265-746f-4000-8000-636c61756465`.
3. That UUID's profile: gateway/static/bearer, the exact
   `http://127.0.0.1:15722/claude-desktop` endpoint and a `pts-local-` token.
4. `puretokens-skills-connection.json`, with exactly eight fields:
   `version=1`, `host=claude-desktop`, `profile_id`, `state=active`,
   `profile_sha256`, `api_base_url`, `api_key`, `connection_id`.

The hash binds actual profile bytes, including line endings. The API origin
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

Like other applied client Keys, this file is independent of Switch login.
Logout stops chat mapping and clears Switch memory, but does not remove the
applied client connection. Switching accounts alone does not rebind it. Applying
another account's configuration replaces it through the normal transaction.
Direct reconfiguration removes the companion's Key by writing `state=inactive`;
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
