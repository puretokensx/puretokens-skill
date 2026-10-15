# Offline fixture: execute the production PowerShell transaction body with
# isolated directories and injected local failures. Never call init or a network.
$ErrorActionPreference = "Stop"
$repository = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$installer = Join-Path $repository "runtime/puretokens-skill-install.ps1"
$sourceText = Get-Content -LiteralPath $installer -Raw -Encoding UTF8
$bodyStart = $sourceText.IndexOf('$ErrorActionPreference = "Stop"')
$mainStart = $sourceText.IndexOf('$stageRoot = $null')
if ($bodyStart -lt 0 -or $mainStart -le $bodyStart) { throw "Fixture cannot locate the installer body" }
$architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
$architecture = if ($architecture -eq "x64") { "amd64" } else { $architecture }
$system = if ($env:OS -eq "Windows_NT") { "windows" } elseif ([Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::OSX)) { "darwin" } else { "linux" }
$manifest = Get-Content -LiteralPath (Join-Path $repository "runtime/executor/manifest.json") -Raw | ConvertFrom-Json
$fixtureGuard = Join-Path (Join-Path $repository "runtime/executor") $manifest.artifacts.PSObject.Properties["$system-$architecture"].Value.path
$root = Join-Path ([IO.Path]::GetTempPath()) ("pt-local-failures-" + [Guid]::NewGuid().ToString("N"))
$script:fixtureState = @{ Mode = ""; Injected = $false; InitCalls = 0; InitResult = $null }

# Load the real functions and the real transaction/catch/finally body. Replace
# only the platform binary selection and init transport for this offline fixture.
$mocks = @'
function Get-ExecutorArtifact([string]$SourceRoot) { return $fixtureGuard }
function Invoke-NativeExecutor([string]$Executable, [string[]]$ArgumentList) {
  if ($ArgumentList[0] -eq "init") {
    $script:fixtureState.InitCalls++
    if ($script:fixtureState.Mode -eq "init-denied") { throw [UnauthorizedAccessException]::new("private-fixture-launch") }
    return [PSCustomObject]@{ ExitCode = 0; Output = ($script:fixtureState.InitResult | ConvertTo-Json -Compress) }
  }
  $output = & $Executable @ArgumentList 2>$null
  return [PSCustomObject]@{ ExitCode = $LASTEXITCODE; Output = ($output -join "`n") }
}
function New-Item {
  [CmdletBinding()]
  param($ItemType, $Path, [switch]$Force)
  if ($script:fixtureState.Mode -eq "target-denied" -and $Path -eq $Target) { throw [UnauthorizedAccessException]::new("private-fixture-target") }
  Microsoft.PowerShell.Management\New-Item @PSBoundParameters
}
function Copy-Item {
  [CmdletBinding()]
  param($LiteralPath, $Destination, [switch]$Recurse)
  if ($script:fixtureState.Mode -eq "stage-denied" -and $LiteralPath -like "*puretokens-balance") { throw [UnauthorizedAccessException]::new("private-fixture-copy") }
  Microsoft.PowerShell.Management\Copy-Item @PSBoundParameters
}
function Move-Item {
  [CmdletBinding()]
  param($LiteralPath, $Destination)
  if ($script:fixtureState.Mode -eq "replace-denied" -and -not $script:fixtureState.Injected -and $Destination -match '[\\/]backup[\\/]puretokens-connection$') {
    $script:fixtureState.Injected = $true
    throw [IO.IOException]::new("private-fixture-locked")
  }
  Microsoft.PowerShell.Management\Move-Item @PSBoundParameters
}
'@
$body = $sourceText.Substring($bodyStart, $mainStart - $bodyStart) + "`n" + $mocks + "`n" + $sourceText.Substring($mainStart)
$run = [ScriptBlock]::Create('param($Command, $Target, $Source, $HostId, $fixtureGuard)' + "`n" + $body)
function Invoke-Fixture([string]$TargetPath, [string]$Mode, [string]$RequestedHost = "") {
  $script:fixtureState.Mode = $Mode
  $script:fixtureState.Injected = $false
  $script:fixtureState.InitCalls = 0
  $lines = New-Object 'System.Collections.Generic.List[string]'
  $failed = $false
  $message = ""
  try { & $run "sync" $TargetPath $repository $RequestedHost $fixtureGuard | ForEach-Object { $lines.Add([string]$_) } }
  catch { $failed = $true; $message = $_.Exception.Message }
  if ($message -match 'private-fixture' -or ($lines -join "`n") -match 'private-fixture') { throw "Private failure data leaked" }
  return [PSCustomObject]@{ Failed = $failed; Message = $message; Output = ($lines -join "`n"); InitCalls = $script:fixtureState.InitCalls }
}
try {
  $target = Join-Path $root "skills with spaces"
  $denied = Invoke-Fixture $target "target-denied"
  if (-not $denied.Failed -or $denied.Message -notmatch 'stage=prepare_target error_code=local_access_denied' -or $denied.Message -notmatch 'installed_files_changed=false' -or (Test-Path -LiteralPath $target)) { throw "Target denial lost stage/state or wrote files" }
  $baseline = Invoke-Fixture $target "baseline"
  if ($baseline.Failed -or $baseline.Output -notmatch 'installation_status=completed' -or $baseline.InitCalls -ne 0) { throw "Offline baseline sync failed" }
  $original = [Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target "puretokens-balance/SKILL.md")))
  $replace = Invoke-Fixture $target "replace-denied"
  if (-not $replace.Failed -or $replace.Message -notmatch 'stage=replace_files error_code=local_io_failure' -or $replace.Message -notmatch 'installed_files_changed=unknown' -or $replace.Output -notmatch 'recovery_status=restored' -or $replace.InitCalls -ne 0) { throw "Replacement failure lost recovery/state evidence" }
  if ([Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target "puretokens-balance/SKILL.md"))) -cne $original) { throw "Original managed file was not restored" }
  if (@(Get-ChildItem -LiteralPath $target -Force -Filter ".puretokens-skill-stage-*").Count -ne 0) { throw "Restored transaction was not cleaned up" }
  $stage = Invoke-Fixture $target "stage-denied"
  if (-not $stage.Failed -or $stage.Message -notmatch 'stage=stage_files error_code=local_access_denied' -or $stage.Output -notmatch 'recovery_status=incomplete' -or $stage.InitCalls -ne 0) { throw "Incomplete staging did not preserve the transaction" }
  if ([Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target "puretokens-balance/SKILL.md"))) -cne $original) { throw "Staging failure modified existing files" }
  $initTarget = Join-Path $root "init with spaces"
  $init = Invoke-Fixture $initTarget "init-denied" "codex"
  if ($init.Failed -or $init.Output -notmatch 'installation_status=completed' -or $init.Output -notmatch 'init_status=unverified stage=init error_code=executor_unavailable' -or $init.InitCalls -ne 1) { throw "Init failure invalidated committed installation or retried" }
  foreach ($receipt in @(
    @{ configuration_status = "verified" },
    @{ configuration_status = "verified"; api_identity_confirmed = $true; credential_verified = $false },
    @{ configuration_status = "verified"; api_identity_confirmed = "true"; credential_verified = "true" },
    @{ configuration_status = "verified"; api_identity_confirmed = $true; credential_verified = $true }
  )) {
    $script:fixtureState.InitResult = $receipt
    $check = Invoke-Fixture $initTarget "init-result" "codex"
    $verified = $receipt.api_identity_confirmed -is [bool] -and $receipt.api_identity_confirmed -eq $true -and $receipt.credential_verified -is [bool] -and $receipt.credential_verified -eq $true
    $expected = if ($verified) { "verified" } else { "unverified" }
    if ($check.Failed -or $check.Output -notmatch "(?m)^init_status=$expected$" -or $check.InitCalls -ne 1) { throw "Init accepted incomplete evidence or repeated a request" }
  }
  Write-Output "PowerShell local failures: target access, staging, replacement recovery, init denial and verification evidence passed (offline)."
} finally {
  if (Test-Path -LiteralPath $root) { Microsoft.PowerShell.Management\Remove-Item -LiteralPath $root -Recurse -Force }
}
