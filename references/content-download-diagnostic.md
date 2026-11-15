# Same-machine content diagnostics (maintainers only)

Use this instrument only for an explicitly authorized investigation of an
existing completed image whose download failed. It is not a new Skill command,
installer component, production fallback, result-delivery tool, or stable release.
Normal executables exclude the `content_diagnostic` test build tag.

## Boundaries

- Run in the affected client's terminal on the affected computer and network.
  Keep its launch context and configuration unchanged between probes. Do not
  borrow another client's Key or copy configuration into a test script.
- Normally read one existing absolute task record; require a confirmed completed image,
  known index, no unknown submission, reconciliation, deferred retry, downloaded
  output or delivered output. Invalid records stop before credentials/networking.
  Never insert a guessed task ID or edit a record to make it eligible.
- Codex-only missing-record investigation may instead read an existing sanitized
  successful `status` support receipt with `--status-receipt` and an explicitly
  matching `--task-id`. Require the same host/OS/architecture, HTTP 200, image,
  `completed`, `next_step=content`, and a canonical request ID. Accept only the
  `support` block and optional `timings_ms` in a bounded regular JSON file; reject
  duplicate/unknown fields, mixed record input and indexes other than zero.
  An exact copy of an already captured receipt is allowed; inventing or changing
  its values is not. No extra status request is made.
  This historical receipt is not signed evidence, does not prove a link to an
  unknown submission, and does not prove the historical and current Key match.
  The content GET still uses only the current Codex connection and the gateway's
  normal task authorization. A mapped connection cannot use this mode without
  its original task binding. Never create or modify a task record from it.
- Resolve only the active host's existing credential adapter and connection
  binding. Make one fixed same-task content GET per authorized invocation.
  No status/model/preflight requests, POST, redirect, retry loop, credential
  export, URL extraction, custom origin, or alternate authentication. The Go
  probes cancel a second transport connection assignment. The Windows native
  probe disables failed-connection resends, redirects, cookies and automatic
  authentication; unsupported policy options stop before sending the GET.
- Default and plain probes use the same 120-second HTTP total, 30-second
  response-header and 30-second read-idle budgets. These are diagnostic limits,
  not changes to production's 10-minute total or 90-second header budget.
- Output numeric timings/bytes, sanitized error, negotiated protocol and
  canonical gateway request ID only, plus the diagnostic build identity.
  No media bytes, digest of user media, URLs, raw headers/errors, credentials,
  provider details, configuration or local paths enter the receipt.
  `http_stack` distinguishes `go_net_http` and `windows_winhttp`. Native
  `proxy_mode` contains only `direct` or `environment_http_proxy`, never an address.
- Use a private temporary directory, validate the native file, then remove it
  on normal exit. Do not persist task changes or mark attachment handoff.
  `ok=true` means this probe read and validated bytes, not user delivery.
  The affected user's explicit workspace outputs remain untouched. A forced
  process kill cannot guarantee temporary-directory cleanup.
- Receipt mode reports `target_source=status_receipt`. It can establish native
  raster-byte validity only, not the original requested format/count or delivery;
  SVG remains unavailable without the original record's explicit format policy.

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

Before a real Windows native probe, run this binary's local synthetic fixtures
on that Windows computer. Stop if they fail. These tests use loopback only,
synthetic credentials and generated PNG bytes; they never read host configuration
or call the production service. They check DLL callbacks, full and chunked
content, cancellation, idle/header/total budgets, truncated and oversized media,
redirect/auth/throttle rejection, no POST/replay, redaction, buffer cleanup and
rejection of an untrusted TLS certificate. This is local transport verification,
not a passed real-host media delivery case.

```powershell
& $diagnostic -test.run '^TestContentWinHTTPNative' -test.timeout 30s
if ($LASTEXITCODE -ne 0) { throw 'Native HTTP self-test failed; do not run the content request.' }
```

Do not run the full compiled test suite on a user's computer. Use the exact
anchored native-only selection above. Do not change proxy/certificate settings
or disable TLS verification to make these tests pass.

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

If only a successful status receipt is available, use the newly built diagnostic
binary (older copies do not have this option), preserve the unknown record and
select the copied receipt instead:

```powershell
& $diagnostic content-diagnose --host codex --status-receipt $statusReceipt --task-id $confirmedTaskId --index 0 --probe executor --confirm-one-get
```

This replaces the record-based baseline, not an additional probe. Use the same
evidence file, task ID, host context and index for any separately authorized
comparison. It is not a new stable executor command or recovery feature.

Change only `executor` to `plain` for the second authorized invocation. This
command exists only in the diagnostic binary, not in released Skills v1.0.5.
Do not run four commands in a loop or change network settings between a pair.

Only after reviewing that pair, choose at most one additional probe at a time:

| Probe | Only changed condition | What a success can suggest |
| --- | --- | --- |
| `http1` | Executor path restricted to HTTP/1.1 | Protocol/path interoperability, if baseline actually used HTTP/2 |
| `long-idle` | Executor read-idle limit becomes 90 seconds; total stays 120 seconds | The 30-second stop policy prevented completion in this sample |
| `winhttp` | Windows WinHTTP owns connections/TLS/HTTP instead of Go's HTTP transport; uses the plain reader and matched budgets | A genuinely independent transport comparison, not a proven fix |

The native probe is a separately selected comparison, never an automatic fallback
and never a fifth request after exhausting the agreed limit. If executor, plain
and http1 have already run, select winhttp instead of long-idle. It still uses the
same active-host credential adapter, fixed task/index and original status evidence.
The CLI exposes no origin or proxy override. Resolve the same environment proxy
choice as Go: direct stays direct; an unauthenticated HTTP proxy is reused only
in memory. SOCKS, HTTPS proxies and proxy credentials are not silently converted
or ignored; unsupported cases stop before the content GET. Windows system proxy,
PAC, default logon credentials and certificate exceptions are not used.

The native request permits HTTP/2 and reports the actual protocol. It uses the
same user agent and gzip advertisement as the Go probe; an encoded response is
currently unsupported and stops rather than treating compressed bytes as an image.
TLS/certificate trust and DNS can still differ between the native and Go stacks.
Sequential samples and those differences prevent a single success from proving
the Go implementation is faulty or establishing general stability. Windows
versions unable to disable internal failed-connection resends return
diagnostic_transport_unsupported before any content request.

After the native-only self-test passes, the separately authorized content
invocation changes only the selected probe:

```powershell
& $diagnostic content-diagnose --host codex --status-receipt $statusReceipt --task-id $confirmedTaskId --index 0 --probe winhttp --confirm-one-get
```

The Windows DLL fixture suite is excluded on other operating systems. A passed
macOS fixture suite and successful Windows cross-build do not prove these DLL
fixtures, the affected network, or actual attachment delivery passed.

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
