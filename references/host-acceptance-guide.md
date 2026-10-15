# Real-host acceptance

Run this guide on the selected release bytes, one host and OS at a time. Isolated
executor/installer tests are recorded separately and never mark real-host cells
passed. `references/host-acceptance.json` is the single acceptance record.

Prepare an offline entry for the current release using:

```sh
node scripts/prepare-host-acceptance.mjs --host opencode --output /absolute/workspace/acceptance-entry.json
```

The parent directory must exist. Optional `--os macOS|windows|linux`,
`--arch amd64|arm64` and `--mode local|wsl|remote|sandbox` describe the actual
execution environment. The command reads only repository release metadata and
writes a new private file; it never installs Skills, reads host configuration,
calls an API, overwrites evidence or copies old passes. Host/OS/shell versions,
date and artifact location are deliberately blank. Complete those from the actual run, retain
untested cases as pending, and add the entry to the record's `evidence` array
only when the evidence artifact exists. A template is not acceptance evidence.

Before testing, record the release version, actual host version, OS version and
architecture, shell version, execution mode, executor platform and SHA-256,
date, and a workspace-relative evidence file.
Keep evidence free of credentials, connection configuration, prompts, reference
URLs and private media. Record case results and failure phase/code only; screenshots
may show only the relevant safe UI. An API task ID is not required in the checked-in
evidence.

Real-host installation changes the selected host's Skill directory; authenticated
checks use its current connection; media submissions are billable. Run each only
within authorization for that host and effect. Do not substitute a production
account, another host, or a mock result when an authorized environment is absent.
Never add paid tests to CI.

| Case | Action | Required evidence |
| --- | --- | --- |
| installation | Install or upgrade in the host, start a new conversation, invoke the loaded Skill | Actual loaded directory/version and executor hash; matching ownership inventory; unrelated files retained |
| authenticatedAPI | Query connection, models and balance once each | Separate identity/authentication outcomes; balance scope and currency; no configuration disclosed |
| image-generation | Generate one authorized image | One submission receipt; same-task completion; image opens after host attachment handoff |
| uploaded-image-edit | Edit a current local attachment | One declared edit submission; attachment preserved as input; resulting image opens |
| generated-image-edit | Explicitly edit the image just delivered | That image is the selected current attachment; one new edit task; result opens |
| multiple-images | Request variants with a model declaring `n` | Accepted count; each zero-based index handed off before fetching the next; all images open |
| video-generation | Generate one authorized video | Same task throughout; delivered video plays; no replacement submission |
| sameTaskResume | Resume an existing task after an interrupted wait/new conversation | Same ID, original operation/count retained; no new POST; reconciliation checked once on explicit resume |
| nativeAttachmentDelivery | Deliver image and video through the actual host | User can open the attached image and play the attached video; a local path or status text alone does not pass |
| speech-generation | Read short synthetic text using a reviewed voice and supported style/speed | One synchronous POST; exact text preserved; real MP3/WAV attachment opens and plays |
| audio-transcription | Transcribe one explicitly selected synthetic recording | Original bytes sent once; validated transcript displayed without replacing it with a summary; no task polling |
| sound-generation | Generate one short sound scene | One synchronous POST; supported fields only; audio attachment plays; no music substitution |
| audio-reattachment | Reattach the same already-downloaded audio after interrupted handoff | Local audio-verify validates original proof; no credential/network request or regeneration; actual attachment opens |
| music-generation | Generate one authorized instrumental piece or song | Explicit instrumental intent and exact supplied lyrics; submit receipt before wait/content; same task and playable MP3/WAV |
| music-resume | Continue that music task in a new conversation, before its output is acknowledged delivered | Same record/ID, wait budget and file proof retained; no replacement POST, synchronous audio command or unnecessary content GET |
| jev-evaluation | Evaluate synthetic text with explicit choice, score and noul questions | One synchronous request; question/option order retained; validated typed results shown; no polling or external action |
| image-to-video-workflow | Explicitly use an image just delivered as a video's first frame | Exact selected local image bytes passed through a declared multipart operation; distinct records; each actual attachment delivered |
| video-audio-workflow | Request video and independent narration or music assets | User accepted separate assets; each delivered once; no claim of composition, mixing or duration alignment |
| help-no-api | Ask for local help, then ask only to revise a narration/lyric/prompt | Correct guidance/text; no Skill executor API call, init, credentials read, model discovery or paid media/evaluation submission |

The original nine image/video cases remain the `media` group. New cases form
`audio`, `music`, `evaluation`, `workflows` and `help` groups. Report these
separately: `media-verified` does not verify the new groups. A candidate with
only offline request tests or an unavailable gateway deployment keeps those
real-host cases pending; absence of credentials is not evidence of API success.
The help case excludes the host's own conversational model traffic: its zero
request claim is specifically about Skill API work and credential access.

For synchronous audio, unknown responses and accepted-but-unsaved files have no
task lookup; do not rerun a paid request to manufacture recovery evidence.
For async music and composite flows, errors/unknown acceptance stop automatic
continuation. Test those branches, malformed audio, unsupported fields, repeated
submission and no-ID recovery using existing isolated executor fixtures, not
production failures. A successful live sample cannot accept those fault cases.
When the host cannot expose bytes or attach audio, record that limitation instead
of marking a file path as playback. Reuse the same authorized assets for
reattachment, continuation and workflow checks where their lifecycle permits.

Use an existing safe task for continuation when possible. Network failure,
reconciliation and malformed media cases are fault-injected only into isolated
fixtures, never by changing production services. A real reconciliation outcome
can be recorded if naturally encountered; do not manufacture one with paid tasks.

For Windows, run the native installer fixtures with `powershell.exe` (5.1),
`pwsh`, and the 32-bit SysWOW64 PowerShell when present. The repository tests
these entrances on its Windows runner; this is not a record of real-host execution. That evidence is
separate from actual host discovery, authenticated API use and attachment handoff.

Installation permission cases are described in
`skills/puretokens-update/references/install-permissions.md`. In isolated tests,
cover socket access denial, granted networking followed by a Windows security-context
failure, ordinary TLS/DNS/HTTP errors, temporary-file write denial and explicit
continuation after the blocker changes. Assert one request per failed invocation,
sanitized phase/category/next step, no automatic retry or permission changes, and
preservation of the old installation. Unknown synchronization output is not a
permission-retry case; successful sync plus failed init must not trigger reinstall.
The fixtures simulate failures, not a real approval system. For real hosts, record
the actual execution mode and only host-reported permission/approval outcomes,
including scope; unknown stays unknown. Exercise denied approval and later
user-approved continuation only in an authorized test session; do not change a
user's normal security settings for testing. Do not mark a sandbox case passed
from an external-terminal or Full Access run. Full Access is not required by this
guide. Bootstrap failure before the script starts must also be observed through
the host's installation instructions, not claimed as covered by fetch alone.

For each host/OS, attach an evidence entry with `host`, `os`, `hostVersion`,
`osVersion`, `osArchitecture` (`amd64` or `arm64`), `shellVersion`,
`executionMode` (`local`, `wsl`, `remote` or `sandbox`),
`executorPlatform`, `executorSha256`, `version`, `checkedAt`,
`artifact` and `cases`. `cases` uses the table's IDs and values `passed`, `failed`,
`pending` or `unavailable`. Set the four existing host/OS summary cells passed
only when a corresponding real-host case has passed. `authenticatedAPI` does
not imply media permissions or wallet coverage of subscriptions.

The host/OS summary cells summarize local evidence only; consult each evidence
entry for its exact architecture and versions. A pass on one architecture or
execution mode does not accept another. In WSL, record the Linux execution
environment, not the surrounding Windows desktop. Never copy desktop credentials
into a remote or sandbox environment to make a case pass.

Acceptance levels are derived from each individual evidence entry, not declared
manually or assembled from unrelated versions: `installation-verified` requires
the installation case; `api-verified` additionally requires authenticated API
checks; `media-verified` requires the original nine media cases, including playable video and
actual attachment handoff. No evidence is `unverified`. Registration and
credential fixtures do not establish any real-host level; Trae remains without
an API adapter.

Run `npm run acceptance:validate` after editing evidence. It rejects passed
summary cells without matching real-host evidence, mismatched release versions,
missing evidence artifacts and executor hashes that do not match the release.
It also checks credential fixture outcomes and API-dependent acceptance against
the adapter state in `references/host-support.json`.
The check reports pending cells; it does not convert them to passes.

## Platform matrix and counts

Each host has macOS, windows and linux summary slots. Pending means an in-scope
acceptance check remains to be done, not proven client availability. Unavailable
means the check is outside this release's supported scope and needs an explicit
reason; it is neither a pass nor missing test evidence.

An adapter marked `pending` in the host registry is not implemented in this
release. Its `credentialFixtures` must be `unavailable`, as must the
`authenticatedAPI`, `sameTaskResume` and `nativeAttachmentDelivery` summaries
on every OS and all API-dependent detailed cases. Local `installation` and
`help-no-api` can still be accepted independently. Trae currently has
this classification: macOS and Windows installation remain pending, but API
and media acceptance are unavailable. Installation-only real-host evidence is
valid and establishes only `installation-verified`; a safe stop before a
request is not a failed or passed authenticated API case.

For a `fixture-tested` adapter, `credentialFixtures` must be `passed`, but
real-host checks stay pending until tested. When adding an adapter, update its
fixture classification and reassess each unavailable check and reason; never
carry forward the old no-adapter exclusion or mark real-host checks passed
from fixtures.

This release schedules Linux targets for Claude Code, Codex, Gemini CLI, Grok Build and OpenCode; other Linux combinations are unavailable with an explicit reason until a client execution target is verified. Record Linux evidence with `os: linux` and a matching `linux-amd64` or `linux-arm64` executor. Do not substitute Windows/macOS hashes.

Four summary checks and the nineteen detailed cases above are distinct. The validator reports pending summary cells, detailed outcomes and separate capability groups; absence of evidence is never a pass. Do not rewrite historical release records to add new passes.

For installation, retain separate outcomes for a clean install and an upgrade
from the preceding distributed version, default and explicit directories, spaces
and non-ASCII names, and a new conversation loading the updated instructions.
Include x86 PowerShell on amd64 Windows, native ARM64 and emulated processes as
distinct environments. For recovery, include interruption before handoff,
already-complete handoff, and a copied Windows/POSIX record with explicit output
rebinding. A copied record never supplies authentication on the new machine.

## Delivery latency measurements

Measure installation separately: stable manifest/selector resolution, archive
transfer, verification/synchronization, and the shared init deadline. Retain
same-version install and update cases with zero archive reads, zero installed
file changes and no init. A changed/missing file or unresolved transaction must
stop without a repair download. Include absent stable assets, selector/archive
checksum failures, a newer installed version and simultaneous updates.

Exercise a replacement failure after at least one existing component has moved,
and verify restoration, file identity, transaction cleanup and no init under both
Windows PowerShell 5.1 and PowerShell 7. The offline legacy-array fixture also
reproduces 5.1 JSON output semantics on modern PowerShell; it does not replace
native Windows execution.

Use only the current platform archive. Validate both Shell and PowerShell entry
paths and the absence of the other system's scripts in the archive. At least one
fresh install and repeat invocation must use an actual built ZIP rather than a
source-directory fixture. Init's two read-only checks share a 20-second deadline;
time out each phase separately without any retry or rollback of installed files.

For the selected authorized host/release, record elapsed milliseconds for:
request preparation to accepted receipt; accepted receipt to observed completed
status; observed completion to verified download; verified download to actual
openable/playable attachment handoff. Record status/content request counts and
whether the user had to repeat a continuation request. These are client-observed
intervals, not server generation-time estimates.

Keep timing evidence with the existing per-host acceptance artifact, using only
case names, durations, request counts, release/host identity and safe failure
codes. Do not store prompts, credentials, reference URLs or media bytes. Compare
single-image, local-image edit, video and interrupted-handoff recovery. Include a
normal first-window expiry, an actual network error and an interrupted process in
isolated fault-injection tests; never create production failures for measurement.
A recovery passes only when the same task is retained, no replacement POST is
sent, and the existing verified download is handed off without another content
GET. Local fixtures and virtual-clock timings remain separate from real-host
acceptance. Start with the user's two most-used accessible hosts; do not infer
which hosts or billable tasks are authorized from this guide.

Each real-host evidence entry may include `measurements`, a bounded list with
one row per actually observed case. A row contains `case`,
`basis: "client-observed"`, `durationsMs` and optional `requests`. Allowed duration keys
are `installation`, `init`, `preparation`, `request`, `wait`, `download`,
`handoff`, `total`; values are nonnegative integer milliseconds. Request counters
are `submit`, `status`, `content`, `other`. Count attempts, not just successful
responses. Omit unmeasured counters/phases; never fill missing observations with
zero. A passed `help-no-api` measurement requires all four counters observed zero.
Host chat model traffic is outside these Skill counters.

For synchronous audio/evaluation, `request` covers the entire command until
its receipt (including audio response saving); there is no separately observed
server-generation interval or task wait. Do not manufacture a breakdown.
Audio/music handoff requires the real host, and transcript/Jev handoff means
the result was displayed. Measurements must correspond to passed or failed
cases; measuring a failed local validation does not establish network latency.
No prompt, transcript, lyrics, media, headers, configuration, path, task ID or
raw response may be added to measurement rows. Offline/virtual-clock values
remain in test reports, never in real-host measurements.

## Stable publication

The manual stable publication gate requires every declared available host/OS
summary check to pass with matching real-host evidence. An API-capable target
also needs all nineteen detailed cases passed together in one local environment,
including the new capability groups. Older media-only evidence cannot satisfy
this requirement by combining different hosts or unrelated runs.
Hosts without an adapter require only installation acceptance on their declared
installation targets; their API-dependent cases must remain unavailable.
Unavailable targets must have an explicit reason; they are not advertised as
accepted. At least one complete media environment is required, so an empty or
all-unavailable matrix cannot authorize a stable release. These gates do not
run paid tests automatically and never fabricate missing evidence.

A maintainer may record a one-time, version-bound `stablePublicationException`
for a direct publication decision. The exception must identify the exact version,
share the acceptance record date, and explain the approval. It only bypasses the
stable promotion gate; it never changes pending, unavailable or failed evidence,
does not count as real-host acceptance, and must not be copied to a later version.

For 0.18.8, the maintainer explicitly requested publication after the installation
permissions readiness review on 2026-10-04. This independent version-specific
decision retains the previously requested maintenance-only gap reporting. The
0.18.7 record is archived at `acceptance/0.18.7-host-acceptance.json`; its approval
was not copied forward. The maintainer requested
that its gaps remain in the maintenance record, not in user-facing release,
installation or update notices. `host-acceptance.json` retains all 166 pending
summary checks and the empty real-host evidence array. The exception does not
waive engineering tests, committed-source CI, reproducible builds or uploaded
asset verification. Product restrictions and actual command failures retain
their normal behavior; publication does not create real-host evidence.
