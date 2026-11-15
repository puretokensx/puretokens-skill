# Same-machine content diagnostics (maintainers only)

Use this instrument only for an explicitly authorized investigation of an
existing completed image whose download failed. It is not a new Skill command,
installer component, production fallback, result-delivery tool, or stable release.
Normal executables exclude the `content_diagnostic` test build tag.

## Boundaries

- Run in the affected client's terminal on the affected computer and network.
  Keep its launch context and configuration unchanged between probes. Do not
  borrow another client's Key or copy configuration into a test script.
- Read one existing absolute task record; require a confirmed completed image,
  known index, no unknown submission, reconciliation, deferred retry, downloaded
  output or delivered output. Invalid records stop before credentials/networking.
  Never insert a guessed task ID or edit a record to make it eligible.
- Resolve only the active host's existing credential adapter and connection
  binding. Make one fixed same-task content GET per authorized invocation.
  No status/model/preflight requests, POST, redirect, retry loop, credential
  export, URL extraction, custom origin, or alternate authentication. A second
  underlying transport connection assignment is cancelled to block replay.
- Default and plain probes use the same 120-second HTTP total, 30-second
  response-header and 30-second read-idle budgets. These are diagnostic limits,
  not changes to production's 10-minute total or 90-second header budget.
- Output numeric timings/bytes, sanitized error, negotiated protocol and
  canonical gateway request ID only, plus the diagnostic build identity.
  No media bytes, digest of user media, URLs, raw headers/errors, credentials,
  provider details, configuration or local paths enter the receipt.
- Use a private temporary directory, validate the native file, then remove it
  on normal exit. Do not persist task changes or mark attachment handoff.
  `ok=true` means this probe read and validated bytes, not user delivery.
  The affected user's explicit workspace outputs remain untouched. A forced
  process kill cannot guarantee temporary-directory cleanup.

## Build and offline checks

Maintainers need the repository's pinned Go version and Node 20 or later.
Recipients need only their existing supported host, matching native diagnostic
binary and task record; no Node, Python, Go, new service or reinstallation.

From the executor directory, run:

```sh
go test -tags content_diagnostic -run '^TestContent(Diagnostic|TransportParity)' -count=1 -race ./...
```

From the repository root, build only the affected platform:

```sh
node scripts/build-content-diagnostic.mjs windows-amd64
```

The ignored `dist/diagnostics/` output contains the native executable and its
build identity/SHA-256 JSON. Verify the recipient's bytes against that checksum.
Do not replace the installed `puretokens-api.exe`, publish the instrument as a
stable version, or claim cross-compilation is Windows acceptance.
`--build-info` prints build identity without reading credentials or networking.

## Controlled comparison

Agree on a maximum of two initial GETs and four total GETs, at most two minutes
of HTTP time per invocation. Nothing automatically launches the next probe.
Retrieval does not submit generation or regenerate the image; do not infer
billing policy from this fact. Stop on authentication rejection, throttling,
record/connection mismatch, cancellation or any unexpected task identity.

Run **executor**, review the receipt, then explicitly run **plain** if still
needed. Both use the same standard Go transport/protocol negotiation and budgets.
The plain path uses an independent idle supervisor and ordinary response-body
read, followed by the existing native-file validator. It bypasses the production
progress reader, buffering and file-finalization pipeline, not authentication
or TLS. Thus it isolates that pipeline, not the entire Go HTTP implementation.

PowerShell template, after selecting the actual binary and unchanged record:

```powershell
& $diagnostic content-diagnose --host codex --record $record --index 0 --probe executor --confirm-one-get
```

Change only `executor` to `plain` for the second authorized invocation. This
command exists only in the diagnostic binary, not in released Skills v1.0.5.
Do not run four commands in a loop or change network settings between a pair.

Only after reviewing that pair, choose at most one additional probe at a time:

| Probe | Only changed condition | What a success can suggest |
| --- | --- | --- |
| `http1` | Executor path restricted to HTTP/1.1 | Protocol/path interoperability, if baseline actually used HTTP/2 |
| `long-idle` | Executor read-idle limit becomes 90 seconds; total stays 120 seconds | The 30-second stop policy prevented completion in this sample |

A matched-budget plain success with executor failure is evidence to investigate
the executor read/file pipeline, not a conclusive causal proof from sequential
samples. An HTTP/1.1 success does not establish a general network failure; a
long-idle success does not justify unlimited waits. If both readers fail, retain
the possibility of shared Go transport, proxy, server/edge, or intermittent path
behavior. Chat/status success does not prove full binary-response delivery.
Use request IDs to correlate existing server evidence; HTTP 200 without written
byte evidence cannot establish complete downstream receipt.

Local TLS fixtures cover complete/chunked/paused data under HTTP/1.1 and HTTP/2,
matching idle limits, truncation, oversize data, rejection/redirect, active-host
binding, unchanged records and output redaction. These are offline evidence,
not production or affected-Windows acceptance. Record real probe outcomes
separately before changing a production timeout, protocol or downloader.
