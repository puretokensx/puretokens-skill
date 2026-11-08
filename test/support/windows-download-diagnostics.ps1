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
    $node.Name -in @('Fail', 'Read-Json', 'Get-OfficialFile', 'Get-OfficialAsset')
}, $true))
if ($parseErrors.Count -ne 0 -or $functions.Count -ne 4) { throw 'Download fixture could not load production functions' }
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
  param($Uri, $OutFile, $TimeoutSec, $Headers, $UserAgent, [switch]$UseBasicParsing, [switch]$PassThru, $MaximumRedirection)
  $script:calls++
  $script:observedTimeout = $TimeoutSec
  $script:observedProgress = $ProgressPreference
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
    'asset_api' {
      $script:apiRequests.Add([string]$Uri)
      if ($Uri -like 'https://github.com/*') { return [PSCustomObject]@{ StatusCode = 504 } }
      if ($Uri -eq 'https://api.github.com/repos/puretokensx/puretokens-skill/releases/tags/v1.2.3') {
        $script:apiMetadata | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $OutFile -Encoding UTF8
      } elseif ($Uri -eq 'https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/123') {
        if ($Headers.Accept -cne 'application/octet-stream') { throw 'Expected binary asset request' }
        if ($script:apiAssetFails) { return [PSCustomObject]@{ StatusCode = 504 } }
        [IO.File]::WriteAllBytes($OutFile, [byte[]]@(1,2,3))
      } else { throw 'Unexpected official API fixture URL' }
      return [PSCustomObject]@{ StatusCode = 200 }
    }
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
foreach ($stage in @('read_release_manifest', 'download_selector', 'download_platform_archive')) {
  $script:calls = 0
  $result = Get-OfficialFile 'https://example.invalid/private' 'unused' $stage
  $expectedTimeout = if ($stage -eq 'download_platform_archive') { 180 } else { 45 }
  if ($result -ne 200 -or $script:calls -ne 1 -or $script:observedTimeout -ne $expectedTimeout -or $script:observedProgress -ne 'SilentlyContinue') {
    throw "Unexpected timeout or progress policy for $stage"
  }
}
$apiRoot = Join-Path ([IO.Path]::GetTempPath()) ('pt-api-fixture-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $apiRoot | Out-Null
try {
  $script:fixtureKind = 'asset_api'
  $script:downloadRoot = $apiRoot
  foreach ($mode in @('valid','draft','prerelease','wrong-tag','duplicate','wrong-origin','bad-digest','wrong-size','unavailable')) {
    $script:officialReleaseMetadata = $null
    $script:apiRequests = New-Object 'System.Collections.Generic.List[string]'
    $script:apiAssetFails = $mode -eq 'unavailable'
    $name = 'puretokens-skill-install.ps1'
    $asset = @{ id=123; name=$name; state='uploaded'; size=3; digest='sha256:039058c6f2c0cb492c533b0a4d14ef77cc0f78abccced5287d84a1a2011cfb81'; url='https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/123'; browser_download_url="https://github.com/puretokensx/puretokens-skill/releases/download/v1.2.3/$name" }
    $script:apiMetadata = @{tag_name='v1.2.3'; draft=$false; prerelease=$false; assets=@($asset)}
    switch ($mode) {
      'draft' { $script:apiMetadata.draft=$true }
      'prerelease' { $script:apiMetadata.prerelease=$true }
      'wrong-tag' { $script:apiMetadata.tag_name='v9.9.9' }
      'duplicate' { $script:apiMetadata.assets=@($asset,$asset) }
      'wrong-origin' { $asset.url='https://example.invalid/asset' }
      'bad-digest' { $asset.digest='sha256:' + ('0' * 64) }
      'wrong-size' { $asset.size=4 }
    }
    $failed=$false
    try { $null=Get-OfficialFile "https://github.com/puretokensx/puretokens-skill/releases/download/v1.2.3/$name" (Join-Path $apiRoot 'asset') download_selector }
    catch {
      $failed=$true
      if ($_.Exception.Message -notlike '*Pure Tokens Skill download:*' -or $_.Exception.Message -match 'example.invalid|private-fixture') { throw 'Unsafe fallback failure' }
    }
    if ($failed -ne ($mode -ne 'valid')) { throw "Wrong fallback outcome: $mode" }
    if ($script:apiRequests.Count -gt 3) { throw 'Official fallback looped' }
    if ($mode -eq 'valid' -and ((Get-FileHash -LiteralPath (Join-Path $apiRoot 'asset') -Algorithm SHA256).Hash.ToLowerInvariant() -ne $asset.digest.Substring(7))) { throw 'Downloaded wrong asset bytes' }
  }
  Write-Output 'Official asset fallback: 9 bounded offline cases passed.'
} finally { Remove-Item -LiteralPath $apiRoot -Recurse -Force }
