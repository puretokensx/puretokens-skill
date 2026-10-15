# Run with powershell.exe -NoProfile -ExecutionPolicy Bypass -File <this-file>.
# This process-scoped option does not change machine-wide execution policy.
[CmdletBinding()]
param(
  [Parameter(Position = 0, Mandatory = $true)]
  [ValidateSet("detect-host", "check-update", "install", "update", "repair")]
  [string]$Command,
  [Alias("Host")]
  [ValidateSet("claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop")]
  [string]$HostId,
  [string]$Target
)
$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
function Fail([string]$Message) { throw "Pure Tokens Skill download: $Message" }
function Read-Json([string]$File) { return (Get-Content -LiteralPath $File -Raw -Encoding UTF8 | ConvertFrom-Json) }
function Get-CurrentHost {
  # Query only the direct launcher chain's names and parent IDs. Never read
  # command lines, credentials, configuration or the complete process list.
  $candidatePid = $PID
  for ($depth = 0; $depth -lt 8; $depth++) {
    try {
      $process = Get-CimInstance -ClassName Win32_Process -Filter "ProcessId=$candidatePid" -Property Name, ParentProcessId -ErrorAction Stop
    } catch { break }
    if ($null -eq $process -or @($process).Count -ne 1 -or
        $process.Name -isnot [string] -or [string]::IsNullOrWhiteSpace($process.Name) -or
        [string]$process.ParentProcessId -notmatch '^\d+$') { break }
    switch ($process.Name.ToLowerInvariant()) {
      "codex.exe" { return "codex" }
      "opencode.exe" { return "opencode" }
      { $_ -in @("powershell.exe", "pwsh.exe", "cmd.exe", "bash.exe", "sh.exe") } {}
      default { $candidatePid = 0 }
    }
    if ($candidatePid -eq 0 -or $process.ParentProcessId -le 1 -or $process.ParentProcessId -eq $candidatePid) { break }
    $candidatePid = $process.ParentProcessId
  }
  Fail "stage=detect_host error_code=current_host_unconfirmed installation_status=not_started installed_files_changed=false next_step=confirm_current_client; confirm this client's host ID and run once with -Host <id>. Supported IDs: claude-code, codex, workbuddy, gemini-cli, grok-build, opencode, trae, claude-desktop, dsh-desktop, deepseek-harness, minimax-code, zcode, kimi-code, qoder, pi, hermes, evox, vscode, octop. No network request or installation write was made."
}
if ($Command -eq "detect-host") {
  if ($HostId -or $Target) { Fail "detect-host takes no host or target options" }
  Get-CurrentHost
  return
}
if (-not $HostId -and -not $Target) {
  $HostId = Get-CurrentHost
  Write-Output "Pure Tokens Skills current client: $HostId (detected)."
}
if ($HostId -eq "octop" -and -not $Target) {
  Fail "stage=select_target error_code=workspace_target_required installation_status=not_started installed_files_changed=false next_step=confirm_workspace_target; confirm the current Octop workspace and run once with -Host octop -Target <absolute-workspace>/.octop/skills. No network request or installation write was made."
}
function Get-OfficialFile([string]$Url, [string]$Destination, [string]$Stage) {
  $status = 0
  $category = "transport_failure"
  $completed = $false
  $networkAccessDenied = $false
  $securityContextUnavailable = $false
  try {
    $result = Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination -PassThru -TimeoutSec 180 -Headers @{ Accept = "application/vnd.github+json" } -UserAgent "puretokens-skill-installer"
    $status = [int]$result.StatusCode
    $category = "http_error"
    $completed = $true
  } catch {
    # Classify types/statuses only; exception messages can contain private data.
    $exception = $_.Exception
    for ($depth = 0; $null -ne $exception -and $depth -lt 8; $depth++) {
      if ($exception.PSObject.Properties['Response'] -and $null -ne $exception.Response) {
        $status = [int]$exception.Response.StatusCode
      }
      if ($exception -is [System.Net.WebException]) {
        switch ($exception.Status.ToString()) {
          'Timeout' { $category = 'timeout' }
          'NameResolutionFailure' { $category = 'dns_failure' }
          'ProxyNameResolutionFailure' { $category = 'dns_failure' }
          'ConnectFailure' { $category = 'connection_failure' }
          'TrustFailure' { $category = 'tls_failure' }
          'SecureChannelFailure' { $category = 'tls_failure' }
        }
      }
      if ($exception -is [System.OperationCanceledException] -or $exception -is [System.TimeoutException]) { $category = 'timeout' }
      if ($exception -is [System.Security.Authentication.AuthenticationException]) { $category = 'tls_failure' }
      if ($exception -is [System.Net.Sockets.SocketException]) {
        if ($exception.SocketErrorCode -eq [System.Net.Sockets.SocketError]::AccessDenied) { $networkAccessDenied = $true }
        $category = if ($exception.SocketErrorCode.ToString() -in @('HostNotFound','TryAgain','NoData')) { 'dns_failure' } elseif ($exception.SocketErrorCode.ToString() -eq 'TimedOut') { 'timeout' } else { 'connection_failure' }
      }
      # SEC_E_NO_CREDENTIALS is a Windows security-context failure, not an API-key rejection.
      # Match a typed native code only; never classify or display private exception text.
      if ($exception -is [System.ComponentModel.Win32Exception] -and $exception.NativeErrorCode -eq -2146893042) { $securityContextUnavailable = $true }
      if ($exception -is [System.UnauthorizedAccessException]) { $category = 'local_io_failure' }
      $exception = $exception.InnerException
    }
  }
  if ($status -eq 200 -and $completed) { return 200 }
  if ($networkAccessDenied) { $category = 'network_access_denied' }
  if ($securityContextUnavailable) { $category = 'tls_security_context_unavailable' }
  if ($status -ge 400 -and $status -le 599) { $category = 'http_error' } elseif ($status -lt 100 -or $status -gt 599) { $status = 0 }
  $nextStep = 'review_download_failure'
  $guidance = 'Review the failure stage before explicitly continuing.'
  switch ($category) {
    'network_access_denied' {
      $nextStep = 'review_host_permissions'
      $guidance = 'The socket denied access. Check current-session command and network permissions in the client. The code alone does not identify the blocking policy. Full Access is not a default installation requirement.'
    }
    'tls_security_context_unavailable' {
      $nextStep = 'review_host_permissions'
      $guidance = 'Windows HTTPS security context is unavailable. Check current-session execution and network permissions first; a network grant alone does not verify the full execution context. This does not establish a missing API key, damaged certificate or client defect. Full Access is not a default installation requirement.'
    }
    'local_io_failure' {
      $nextStep = 'review_download_directory_permissions'
      $guidance = 'Check permission to write the temporary download directory in the current client session.'
    }
  }
  Fail "stage=$Stage error_code=$category http_status=$status installation_status=not_completed installed_files_changed=false next_step=$nextStep; $guidance Installed files were not changed. After resolving the issue and explicitly choosing to continue, run the same official fetch command once; no automatic retry."
}
$locationOptions = @{}
if ($HostId) { $locationOptions.HostId = $HostId }
if ($Target) { $locationOptions.Target = $Target }
$downloadRoot = Join-Path ([IO.Path]::GetTempPath()) ("puretokens-download-" + [Guid]::NewGuid().ToString("N"))
try {
  New-Item -ItemType Directory -Path $downloadRoot | Out-Null
  $releaseFile = Join-Path $downloadRoot "release.json"
  $null = Get-OfficialFile "https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json" $releaseFile read_release_manifest
  try { $release = Read-Json $releaseFile } catch { Fail "the stable release manifest is unreadable" }
  if ($release.schemaVersion -ne 2) { Fail "the stable release manifest format is unsupported; no source fallback" }
  $sourceCommit = $release.sourceCommit
  if ($sourceCommit -cnotmatch '^[0-9a-f]{40}$') { Fail "the stable release has no verified source commit" }
  $availableVersion = $release.version
  if ($availableVersion -cnotmatch '^\d+\.\d+\.\d+$') { Fail "the stable release version is invalid" }
  $releaseOrigin = "https://github.com/puretokensx/puretokens-skill/releases/download/v$availableVersion"
  $selector = $release.installers.powershell
  if ($null -eq $selector -or $selector.filename -cne "puretokens-skill-install.ps1" -or $selector.sha256 -cnotmatch '^[0-9a-f]{64}$') { Fail "the stable directory selector is invalid" }
  $locationInstaller = Join-Path $downloadRoot "puretokens-skill-install.ps1"
  $null = Get-OfficialFile "$releaseOrigin/$($selector.filename)" $locationInstaller download_selector
  if ((Get-FileHash -LiteralPath $locationInstaller -Algorithm SHA256).Hash.ToLowerInvariant() -cne $selector.sha256) { Fail "stable directory selector checksum mismatch" }
  $Target = & $locationInstaller locate @locationOptions
  $installedVersion = "not_installed"
  $installedFile = Join-Path $Target ".puretokens-executor/runtime.json"
  if (Test-Path -LiteralPath $installedFile -PathType Leaf) {
    try {
      $installed = Read-Json $installedFile
      if ($installed.name -eq "puretokens-api-executor") {
        $installedVersion = if ($installed.version -match '^\d+\.\d+\.\d+$') { $installed.version } else { "unverified" }
      }
    } catch { $installedVersion = "unverified" }
  }
  $state = if ($installedVersion -eq $availableVersion) { "current" } else { "version_differs" }
  Write-Output "Pure Tokens Skills update check: installed=$installedVersion available=$availableVersion status=$state source_commit=$sourceCommit"
  if ($Command -eq "check-update") { return }
  if ($installedVersion -match '^\d+\.\d+\.\d+$' -and [version]$installedVersion -gt [version]$availableVersion) { Fail "the installed version is newer than the stable release; automatic downgrade was stopped" }
  $architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  $platform = switch ($architecture) { "x64" { "windows-amd64" }; "arm64" { "windows-arm64" }; default { Fail "this operating system and CPU has no platform executor" } }
  $archive = Join-Path $downloadRoot "source.zip"
  $artifact = $release.files.PSObject.Properties[$platform].Value
  if ($null -eq $artifact -or $artifact.filename -cne "puretokens-skill-$availableVersion-$platform.zip" -or $artifact.sha256 -cnotmatch '^[0-9a-f]{64}$' -or $artifact.executorSha256 -cnotmatch '^[0-9a-f]{64}$') { Fail "published platform metadata is invalid" }
  if ($installedVersion -eq $availableVersion -and $Command -ne "repair") {
    $executor = Join-Path $Target ".puretokens-executor/puretokens-api.exe"
    if (-not (Test-Path -LiteralPath $executor -PathType Leaf) -or ((Get-Item -LiteralPath $executor).Attributes -band [IO.FileAttributes]::ReparsePoint)) { Fail "installed executor is missing or unsupported; existing files were preserved" }
    if ((Get-FileHash -LiteralPath $executor -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.executorSha256) { Fail "installed executor checksum mismatch; existing files were preserved" }
    if ($installed.platform -cne $platform) { Fail "installed platform does not match; existing files were preserved" }
    $verifyOptions = @{ Target = $Target; ReleaseManifest = $releaseFile }
    if ($HostId) { $verifyOptions.HostId = $HostId }
    & $locationInstaller verify-installed @verifyOptions
    Write-Output "Pure Tokens Skills $availableVersion are already current and verified; no archive downloaded, no files changed, init not run."
    return
  }
  $null = Get-OfficialFile "$releaseOrigin/$($artifact.filename)" $archive download_platform_archive
  if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.sha256) { Fail "published platform archive checksum mismatch" }
  $archiveRoot = "puretokens-skill"
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  $zip = [IO.Compression.ZipFile]::OpenRead($archive)
  try {
    if ($zip.Entries.Count -eq 0) { Fail "the archive is empty" }
    foreach ($entry in $zip.Entries) {
      if (-not $entry.FullName.StartsWith("$archiveRoot/", [StringComparison]::Ordinal) -or $entry.FullName -match '\\|(^|/)\.\.?(/|$)' -or (($entry.ExternalAttributes -shr 16) -band 0xF000) -eq 0xA000) { Fail "the archive contains an unsafe path or symlink" }
    }
  } finally { $zip.Dispose() }
  $unpacked = Join-Path $downloadRoot "unpacked"
  [IO.Compression.ZipFile]::ExtractToDirectory($archive, $unpacked)
  $sourceRoot = Join-Path $unpacked $archiveRoot
  if ((Read-Json (Join-Path $sourceRoot "package.json")).version -ne $availableVersion) { Fail "the archive version does not match the stable release" }
  if ((Get-FileHash -LiteralPath (Join-Path $sourceRoot "runtime/puretokens-skill-install.ps1") -Algorithm SHA256).Hash.ToLowerInvariant() -cne $selector.sha256) { Fail "archive selector does not match the stable release" }
  if ((Get-FileHash -LiteralPath (Join-Path $sourceRoot "runtime/executor/bin/puretokens-api-$platform.exe") -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.executorSha256) { Fail "archive executor does not match the stable release" }
  $syncOptions = @{ Source = $sourceRoot; Target = $Target }
  if ($Command -ne "repair") { $syncOptions.NoRecovery = $true }
  if ($HostId) { $syncOptions.HostId = $HostId }
  $syncCommand = if ($Command -eq "repair") { "repair" } else { "sync" }
  & (Join-Path $sourceRoot "runtime/puretokens-skill-install.ps1") $syncCommand @syncOptions
} finally {
  if (Test-Path -LiteralPath $downloadRoot) {
    try { Remove-Item -LiteralPath $downloadRoot -Recurse -Force -ErrorAction Stop }
    catch { Write-Output "cleanup_status: pending; download temporary files were retained. Preserve the preceding install result; do not bypass host cleanup restrictions." }
  }
}
