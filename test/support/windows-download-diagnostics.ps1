# Maintainer-only, offline fixture. Invoked from Node and every Windows test engine.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
$repository = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$source = Get-Content -LiteralPath (Join-Path $repository 'runtime/puretokens-skill-fetch.ps1') -Raw -Encoding UTF8
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseInput($source, [ref]$tokens, [ref]$parseErrors)
$functions = @($ast.FindAll({
  param($node)
  $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and
    $node.Name -in @('Fail', 'Get-OfficialFile')
}, $true))
if ($parseErrors.Count -ne 0 -or $functions.Count -ne 2) { throw 'Download fixture could not load production functions' }
. ([ScriptBlock]::Create(($functions | ForEach-Object { $_.Extent.Text }) -join "`n"))

if (-not ('PureTokensDownloadFixtureException' -as [type])) {
  Add-Type -TypeDefinition @'
public sealed class PureTokensDownloadFixtureResponse {
    public int StatusCode { get { return 403; } }
}
public sealed class PureTokensDownloadFixtureException : System.Exception {
    public PureTokensDownloadFixtureResponse Response { get; private set; }
    public PureTokensDownloadFixtureException(System.Exception inner) : base("private-fixture", inner) {
        Response = new PureTokensDownloadFixtureResponse();
    }
}
'@
}
function Invoke-WebRequest {
  param($Uri, $OutFile, $TimeoutSec, $Headers, $UserAgent, [switch]$UseBasicParsing, [switch]$PassThru)
  $script:calls++
  $private = 'private-fixture https://example.invalid/private token-fixture'
  switch ($script:fixtureKind) {
    'timeout' { throw [System.Net.WebException]::new($private, [System.Net.WebExceptionStatus]::Timeout) }
    'cancelled' { throw [System.OperationCanceledException]::new($private) }
    'dns' { throw [System.Net.WebException]::new($private, [System.Net.WebExceptionStatus]::NameResolutionFailure) }
    'trust' { throw [System.Net.WebException]::new($private, [System.Net.WebExceptionStatus]::TrustFailure) }
    'tls' { throw [System.Security.Authentication.AuthenticationException]::new($private) }
    'socket_denied' { throw [System.Net.Http.HttpRequestException]::new($private, [System.Net.Sockets.SocketException]::new(10013)) }
    'nested_socket_denied' { throw [System.Security.Authentication.AuthenticationException]::new($private, [System.Net.Sockets.SocketException]::new(10013)) }
    'security_context' {
      $native = [System.ComponentModel.Win32Exception]::new(-2146893042, $private)
      throw [System.Net.Http.HttpRequestException]::new($private, [System.Security.Authentication.AuthenticationException]::new($private, $native))
    }
    'local_io' { throw [System.UnauthorizedAccessException]::new($private) }
    'http' { return [PSCustomObject]@{ StatusCode = 403 } }
    'http_with_inner' { throw [PureTokensDownloadFixtureException]::new([System.ComponentModel.Win32Exception]::new(-2146893042, $private)) }
    'unrelated_native' { throw [System.ComponentModel.Win32Exception]::new(5, $private) }
    'message_only' { throw [System.Exception]::new("$private 10013 0x8009030E SEC_E_NO_CREDENTIALS") }
    'deep_chain' {
      $nested = [System.ComponentModel.Win32Exception]::new(-2146893042, $private)
      for ($i = 0; $i -lt 12; $i++) { $nested = [System.Exception]::new($private, $nested) }
      throw $nested
    }
    'success' { return [PSCustomObject]@{ StatusCode = 200 } }
    default { throw 'Unknown download fixture' }
  }
}
$cases = @(
  @{ Kind = 'timeout'; Category = 'timeout' },
  @{ Kind = 'cancelled'; Category = 'timeout' },
  @{ Kind = 'dns'; Category = 'dns_failure' },
  @{ Kind = 'trust'; Category = 'tls_failure' },
  @{ Kind = 'tls'; Category = 'tls_failure' },
  @{ Kind = 'socket_denied'; Category = 'network_access_denied'; Next = 'review_host_permissions' },
  @{ Kind = 'nested_socket_denied'; Category = 'network_access_denied'; Next = 'review_host_permissions' },
  @{ Kind = 'security_context'; Category = 'tls_security_context_unavailable'; Next = 'review_host_permissions' },
  @{ Kind = 'local_io'; Category = 'local_io_failure'; Next = 'review_download_directory_permissions' },
  @{ Kind = 'http'; Category = 'http_error'; Http = 403 },
  @{ Kind = 'http_with_inner'; Category = 'http_error'; Http = 403 },
  @{ Kind = 'unrelated_native'; Category = 'transport_failure' },
  @{ Kind = 'message_only'; Category = 'transport_failure' },
  @{ Kind = 'deep_chain'; Category = 'transport_failure' }
)
foreach ($case in $cases) {
  $script:fixtureKind = $case.Kind
  $script:calls = 0
  $failed = $false
  $next = if ($case.Next) { $case.Next } else { 'review_download_failure' }
  try { Get-OfficialFile 'https://example.invalid/private' 'unused' read_release_manifest | Out-Null }
  catch {
    $failed = $true
    $message = $_.Exception.Message
    $expected = "stage=read_release_manifest error_code=$($case.Category) http_status=$([int]$case.Http) installation_status=not_completed installed_files_changed=false next_step=$next;"
    if (-not $message.Contains($expected)) { throw "Wrong download classification for $($case.Kind)" }
    if ($message -match 'private-fixture|example.invalid|token-fixture|SEC_E_NO_CREDENTIALS') { throw 'Private exception data leaked' }
    if ($message -notlike '*no automatic retry*' -or $message -notlike '*explicitly choosing to continue*') { throw 'Explicit continuation requirement lost' }
    if ($next -eq 'review_host_permissions' -and $message -notlike '*Full Access is not a default installation requirement*') { throw 'Unexpected default full-access guidance' }
  }
  if (-not $failed -or $script:calls -ne 1) { throw "Failure retried or suppressed: $($case.Kind)" }
}
# This is a new explicit invocation after a synthetic blocker is removed, not an automatic retry.
$script:fixtureKind = 'success'
$script:calls = 0
$result = Get-OfficialFile 'https://example.invalid/private' 'unused' read_release_manifest
if ($result -ne 200 -or $script:calls -ne 1) { throw 'Explicit continuation did not make exactly one successful request' }
Write-Output "Download diagnostics: $($cases.Count) failure cases and one explicit successful continuation passed (offline)."
