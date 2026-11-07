# Host-specific guidance and shared installation, 2026-11-07

## Scope

Optimize installed guidance without splitting executors by client or changing
credential resolution. Only puretokens-skill is edited. Existing multimedia
and new-client changes in this worktree are retained, not bundled into a commit.
No production credentials, real client installation, paid API calls, Switch/Web
changes, version bump, push or release are part of this task.

## Implementation

- A short desktop-hosts.md index selects hosts/<bound-host>.md. Diagnostics
  and unsupported boundaries are separate references, loaded only when needed.
- Four media/evaluation Skills use common host-delivery.md rules plus the bound
  host's delivery/<host>.md. The default native protocol is rendered once per
  Skill; host files contain only their additional conditional protocols.
- All ten Skills remain self-contained and host-neutral. Shared directories
  keep one set of Skills and one current-platform executor. Different roots
  are not merged, linked or migrated automatically. Installation location never
  determines a credential source or proves the current chat connection.
- The development generator rejects unknown/duplicate sections, unknown
  delivery mappings and stale host files. Generated-file drift and nested
  reference links are tested. It adds no user runtime dependency.
- Native media, GIF/SVG originals, independent tail-frame handoff, same-task
  recovery and no automatic resubmission remain unchanged.

## Fresh targeted verification

- 19 tests passed across host-reference-guidance, shared-guidance, host-delivery,
  contract-continuity and multimedia-guidance tests.
- Shared-root same-version regression passed: declared shared roots retain
  identical file hashes and mtimes across host updates, two mocked metadata
  requests, no archive download, no init, one executor and no transaction residue.
- Shared-root upgrade regression passed: an older managed installation upgrades
  with both Codex/Paseo references intact, removes an obsolete managed reference,
  preserves additions/modifications and refuses overwrite without damaging the
  peer host. All installs run under isolated temporary HOME and mocked downloads.
- The new stale-file test first failed with a missing rejection, then passed
  after generator validation. Link checking excludes illustrative media embeds
  while resolving documentation links within each installed Skill.

## Size evidence

UTF-8 file bytes, not model tokens, elapsed time or allocated filesystem blocks:

| Measurement | Bytes |
| --- | ---: |
| Previous complete host guide, recorded before this split | 22,257 |
| Current short host index | 2,311 |
| Current Codex-only execution reference | 491 |
| Current common delivery guide | 4,439 |
| Current Codex-only delivery reference | 1,126 |
| All ten Skill trees before this split | 1,090,926 |
| All ten Skill trees after this split | 1,179,572 |
| Darwin arm64 executor, unchanged by this task | 12,027,250 |

Initial Codex lookup reads 2,802 bytes instead of the prior 22,257-byte complete
manual (about 87% fewer source bytes); a bound host can go directly to its file.
Error diagnostics are additional only after a relevant failure. The total Skill
tree grows by 88,646 bytes because references remain self-contained across ten
Skills. This is a reading-context optimization, not a package-size reduction or
a measured latency improvement. No per-client binary is introduced.

## Completion gate

- Final `npm run check`: exit 0. Repository and generated-content validation,
  acceptance-record consistency, six executor artifact hashes, Go tests and vet
  passed. Node: 213 tests, 209 passed, 0 failed, 4 skipped because PowerShell is
  unavailable on this macOS host.
- The initial full run had one concurrent stale-lock fixture failure. Both lock
  modes passed a focused replay. Review found the fixture's 8-second marker
  deadline could release peers early, and its cleanup assertion could mask the
  original error. Only this test's bounded marker budget was raised to 60 seconds
  and its original failure preserved; production lock behavior was not changed.
  Both focused cases and the subsequent full engineering run passed.
- Generated consistency and `git diff --check` passed. No files are staged or
  committed for this task because several edited sources already contain prior
  uncommitted work. No push, version change or publication was performed.

These checks do not establish Windows execution, real-client loading,
authentication, billing, attachment opening or playback. Existing real-host
acceptance levels remain unchanged.
