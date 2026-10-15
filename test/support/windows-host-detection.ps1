$ErrorActionPreference = "Stop"
$fetch = Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) "runtime/puretokens-skill-fetch.ps1"
# The fetch has its own script scope. Mocks share this isolated test state,
# removed in finally; they never query actual Windows processes.
$global:ptHostDetectionFixture = @{
  Queries = New-Object 'System.Collections.Generic.List[string]'
  Requests = New-Object 'System.Collections.Generic.List[string]'
  Launcher = "codex.exe"; OuterLauncher = "codex.exe"; Mode = "valid"
  Root = $null; HostId = $null; Target = $null
  DenyTempWrite = $false
}
function New-Item {
  [CmdletBinding()]
  param($Path, $ItemType, [switch]$Force)
  if ($global:ptHostDetectionFixture.DenyTempWrite) { throw [UnauthorizedAccessException]::new("private-directory-error") }
  Microsoft.PowerShell.Management\New-Item @PSBoundParameters
}
function Get-CimInstance {
  param($ClassName, $Filter, $Property, $ErrorAction)
  if ($ClassName -ne "Win32_Process" -or @($Property).Count -ne 2 -or
      $Property[0] -ne "Name" -or $Property[1] -ne "ParentProcessId" -or
      $Filter -notmatch '^ProcessId=\d+$') { throw "Unexpected process query" }
  $state = $global:ptHostDetectionFixture
  $state.Queries.Add($Filter)
  switch ($state.Mode) {
    "throw" { throw "private-process-query-error" }
    "empty" { return $null }
    "multiple" { return @([PSCustomObject]@{ Name = "codex.exe" }, [PSCustomObject]@{ Name = "opencode.exe" }) }
    "missing-name" { return [PSCustomObject]@{ ParentProcessId = 71 } }
    "invalid-parent" { return [PSCustomObject]@{ Name = "codex.exe"; ParentProcessId = "private-invalid-parent" } }
  }
  if ($Filter -eq "ProcessId=71") { return [PSCustomObject]@{ Name = $state.Launcher; ParentProcessId = 72 } }
  if ($Filter -eq "ProcessId=72") { return [PSCustomObject]@{ Name = $state.OuterLauncher; ParentProcessId = 1 } }
  return [PSCustomObject]@{ Name = "powershell.exe"; ParentProcessId = 71 }
}
function Invoke-WebRequest {
  param([switch]$UseBasicParsing, $Uri, $OutFile, [switch]$PassThru, $TimeoutSec, $Headers, $UserAgent)
  $state = $global:ptHostDetectionFixture
  $state.Requests.Add($Uri)
  if (-not $state.Root) { throw "Host detection must not request the network" }
  $name = switch -Exact ($Uri) {
    "https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json" { "release.json" }
    "https://github.com/puretokensx/puretokens-skill/releases/download/v0.17.0/puretokens-skill-install.ps1" { "selector.ps1" }
    default { throw "Unexpected release download" }
  }
  Copy-Item -LiteralPath (Join-Path $state.Root $name) -Destination $OutFile
  return [PSCustomObject]@{ StatusCode = 200 }
}
try {
  $state = $global:ptHostDetectionFixture
  $rejected = $false
  try { & $fetch install -Host octop | Out-Null }
  catch {
    $rejected = $_.Exception.Message -like "*stage=select_target error_code=workspace_target_required*" -and
      $_.Exception.Message -like "*next_step=confirm_workspace_target*"
  }
  if (-not $rejected -or $state.Queries.Count -ne 0 -or $state.Requests.Count -ne 0) { throw "Missing workspace was not confirmed before downloads or detection" }
  foreach ($case in @(
    @{ Name = "codex.exe"; Host = "codex" },
    @{ Name = "opencode.exe"; Host = "opencode" },
    @{ Name = "grok.exe"; Host = "grok-build" },
    @{ Name = "qodercli.exe"; Host = "qoder" },
    @{ Name = "qoder.exe"; Host = "qoder" }
  )) {
    $state.Launcher = $case.Name
    $state.Queries.Clear()
    $result = & $fetch detect-host
    if ($result -cne $case.Host -or $state.Queries.Count -ne 2) { throw "Direct native launcher was not detected" }
  }
  foreach ($name in @("node.exe", "python.exe", "Code.exe", "claude.exe", "dsh.exe", "pi.exe", "hermes.exe")) {
    $state.Launcher = $name
    $state.Queries.Clear()
    $rejected = $false
    try { & $fetch install | Out-Null }
    catch {
      $rejected = $_.Exception.Message -like "*stage=detect_host error_code=current_host_unconfirmed*" -and
        $_.Exception.Message -like "*next_step=confirm_current_client*"
    }
    if (-not $rejected -or $state.Queries.Count -ne 2) { throw "Unknown launcher inherited an outer host or continued installation" }
  }
  $state.Launcher = "cmd.exe"
  $state.OuterLauncher = "powershell.exe"
  $state.Queries.Clear()
  $rejected = $false
  try { & $fetch detect-host | Out-Null } catch { $rejected = $_.Exception.Message -like "*current_host_unconfirmed*" }
  if (-not $rejected -or $state.Queries.Count -gt 8) { throw "Host detection was not bounded" }
  foreach ($mode in @("throw", "empty", "multiple", "missing-name", "invalid-parent")) {
    $state.Mode = $mode
    $state.Queries.Clear()
    $rejected = $false
    try { & $fetch install | Out-Null }
    catch {
      $rejected = $_.Exception.Message -like "*stage=detect_host error_code=current_host_unconfirmed*" -and
        $_.Exception.Message -notlike "*private-*"
    }
    if (-not $rejected -or $state.Queries.Count -ne 1) { throw "Unavailable process evidence did not stop safely" }
  }
  if ($state.Requests.Count -ne 0) { throw "Unconfirmed host attempted a release download" }
  $state.Mode = "valid"
  $state.Launcher = "codex.exe"
  $state.DenyTempWrite = $true
  $rejected = $false
  try { & $fetch install | Out-Null }
  catch {
    $rejected = $_.Exception.Message -like "*stage=prepare_download error_code=local_access_denied*" -and
      $_.Exception.Message -like "*installation_status=not_started installed_files_changed=false*" -and
      $_.Exception.Message -like "*next_step=review_download_directory_permissions*" -and
      $_.Exception.Message -notlike "*private-directory-error*"
  }
  $state.DenyTempWrite = $false
  if (-not $rejected -or $state.Requests.Count -ne 0) { throw "Temporary-directory denial was not reported before downloading" }
  $state.Mode = "valid"
  $state.Root = Join-Path ([IO.Path]::GetTempPath()) ("pt-host-detection-" + [Guid]::NewGuid().ToString("N"))
  $state.Target = Join-Path $state.Root "target"
  $runtimeDirectory = Join-Path $state.Target ".puretokens-executor"
  New-Item -ItemType Directory -Path $runtimeDirectory -Force | Out-Null
  '{"name":"puretokens-api-executor","version":"0.16.0"}' | Set-Content -LiteralPath (Join-Path $runtimeDirectory "runtime.json") -Encoding UTF8
  $selector = Join-Path $state.Root "selector.ps1"
  @'
param([string]$Command, [string]$HostId, [string]$Target)
if ($Command -ne "locate") { throw "Unexpected selector command" }
$state = $global:ptHostDetectionFixture
if ($state.HostId -cne $HostId) { throw "Selector received the wrong host" }
if ($Target -and $Target -cne $state.Target) { throw "Explicit target was changed" }
Write-Output $state.Target
'@ | Set-Content -LiteralPath $selector -Encoding UTF8
  @{
    schemaVersion = 2; version = "0.17.0"; sourceCommit = "a" * 40
    installers = @{ powershell = @{ filename = "puretokens-skill-install.ps1"; sha256 = (Get-FileHash -LiteralPath $selector -Algorithm SHA256).Hash.ToLowerInvariant() } }
  } | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $state.Root "release.json") -Encoding UTF8
  foreach ($case in @(
    @{ Launcher = "codex.exe"; ExpectedHost = "codex"; Options = @{} },
    @{ Launcher = "opencode.exe"; ExpectedHost = "opencode"; Options = @{} },
    @{ Launcher = "node.exe"; ExpectedHost = "opencode"; Options = @{ HostId = "opencode" } },
    @{ Launcher = "node.exe"; ExpectedHost = ""; Options = @{ Target = $state.Target } }
  )) {
    $state.Launcher = $case.Launcher
    $state.HostId = $case.ExpectedHost
    $state.Queries.Clear()
    $state.Requests.Clear()
    $options = $case.Options
    $result = & $fetch check-update @options
    if (($result -join "`n") -notlike "*installed=0.16.0 available=0.17.0*") { throw "Fetch did not use the verified selector's target" }
    if ($case.Options.Count -gt 0 -and $state.Queries.Count -ne 0) { throw "Explicit host/target triggered auto-detection" }
    if ($case.Options.Count -eq 0 -and $state.Queries.Count -ne 2) { throw "Fetch did not detect its native launcher" }
    if ($state.Requests.Count -ne 2) { throw "Check-update downloaded more than metadata and selector" }
  }
  Write-Output "Windows host detection: direct native launchers, selector handoff, explicit choices, unknown-client boundaries and bounded queries passed."
} finally {
  if ($global:ptHostDetectionFixture.Root) { Remove-Item -LiteralPath $global:ptHostDetectionFixture.Root -Recurse -Force }
  Remove-Variable -Name ptHostDetectionFixture -Scope Global
}
