# Run with powershell.exe -NoProfile -ExecutionPolicy Bypass -File <this-file>.
# This process-scoped option does not change machine-wide execution policy.
[CmdletBinding()]
param(
  [Parameter(Position = 0, Mandatory = $true)]
  [ValidateSet("detect-host", "check-update", "install", "update", "repair")]
  [string]$Command,
  [Alias("Host")]
  [ValidateSet("claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop", "oh-my-pi", "minimax-code-cli", "qwen-code", "codebuddy-code", "copilot-cli", "factory-droid", "mimo-coder", "paseo")]
  [string]$HostId,
  [string]$Target,
  [string]$ReleaseMetadata
)
$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$script:installStage = "arguments"
function Fail([string]$Message) {
  if (-not $Message.StartsWith("stage=")) {
    $Message = "stage=$script:installStage error_code=validation_failed installation_status=not_completed installed_files_changed=false next_step=review_installation_failure; $Message"
  }
  $exception = New-Object System.InvalidOperationException("Pure Tokens Skill download: $Message")
  $exception.Data["puretokens_receipt"] = $true
  throw $exception
}
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
      "codex.exe" { if ($env:PASEO_HOME) { $candidatePid = 0 } else { return "codex" } }
      "mimo.exe" { return "mimo-coder" }
      "droid.exe" { return "factory-droid" }
      "opencode.exe" { return "opencode" }
      "grok.exe" { return "grok-build" }
      "qodercli.exe" { return "qoder" }
      "qoder.exe" { return "qoder" }
      { $_ -in @("powershell.exe", "pwsh.exe", "cmd.exe", "bash.exe", "sh.exe") } {}
      default { $candidatePid = 0 }
    }
    if ($candidatePid -eq 0 -or $process.ParentProcessId -le 1 -or $process.ParentProcessId -eq $candidatePid) { break }
    $candidatePid = $process.ParentProcessId
  }
  Fail "stage=detect_host error_code=current_host_unconfirmed installation_status=not_started installed_files_changed=false next_step=confirm_current_client; confirm this client's host ID and run once with -Host <id>. Supported IDs: claude-code, codex, workbuddy, gemini-cli, grok-build, opencode, trae, claude-desktop, dsh-desktop, deepseek-harness, minimax-code, zcode, kimi-code, qoder, pi, hermes, evox, vscode, octop, oh-my-pi, minimax-code-cli, qwen-code, codebuddy-code, copilot-cli, factory-droid, mimo-coder, paseo. No network request or installation write was made."
}
function Get-OfficialAsset([string]$Url, [string]$Destination, [string]$Stage) {
  $prefix = 'https://github.com/puretokensx/puretokens-skill/releases/'
  $suffix = $Url.Substring($prefix.Length)
  if ($suffix -ceq 'latest/download/release-manifest.json') {
    $expectedTag = 'latest'; $metadataPath = 'latest'; $name = 'release-manifest.json'
  } elseif ($suffix -cmatch '^download/(v[0-9]+[.][0-9]+[.][0-9]+)/([^/]+)$') {
    $expectedTag = $Matches[1]; $metadataPath = "tags/$expectedTag"; $name = $Matches[2]
  } else { Fail 'unsupported official release fallback' }
  Write-Information "Pure Tokens download progress: stage=$Stage status=official_asset_fallback" -InformationAction Continue
  if ($null -eq $script:officialReleaseMetadata) {
    $metadataFile = Join-Path $downloadRoot 'github-release.json'
    $null = Get-OfficialFile "https://api.github.com/repos/puretokensx/puretokens-skill/releases/$metadataPath" $metadataFile $Stage 'application/vnd.github+json' -NoFallback
    if ((Get-Item -LiteralPath $metadataFile).Length -gt 2097152) { Fail 'official release metadata exceeds the supported size' }
    try { $script:officialReleaseMetadata = Read-Json $metadataFile } catch { Fail 'official release metadata is unreadable' }
  }
  $asset = Get-ReleaseAsset $script:officialReleaseMetadata $expectedTag $name
  $null = Get-OfficialFile $asset.url $Destination $Stage 'application/octet-stream' -NoFallback
  if ((Get-Item -LiteralPath $Destination).Length -ne [long]$asset.size -or
      (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash.ToLowerInvariant() -cne $asset.digest.Substring(7)) { Fail 'official release asset size or checksum mismatch' }
  $script:officialReleaseTag = $script:officialReleaseMetadata.tag_name
  $script:officialAssetRoute = $true
  return 200
}
function Get-ReleaseAsset($Metadata, [string]$ExpectedTag, [string]$Name) {
  $prefix = 'https://github.com/puretokensx/puretokens-skill/releases/'
  $tag = $metadata.tag_name
  if ($tag -isnot [string] -or $tag -cnotmatch '^v[0-9]+[.][0-9]+[.][0-9]+$' -or
      ($expectedTag -cne 'latest' -and $tag -cne $expectedTag) -or
      $metadata.draft -isnot [bool] -or $metadata.draft -or $metadata.prerelease -isnot [bool] -or $metadata.prerelease -or
      $metadata.assets -isnot [array] -or $metadata.assets.Count -gt 100) { Fail 'official release asset metadata is invalid or ambiguous' }
  $assets = @($metadata.assets | Where-Object { $_.name -ceq $name })
  if ($assets.Count -ne 1) { Fail 'official release asset metadata is invalid or ambiguous' }
  $asset = $assets[0]
  $id = [string]$asset.id
  $size = [string]$asset.size
  if ($id -cnotmatch '^[1-9][0-9]{0,15}$' -or $size -cnotmatch '^[1-9][0-9]{0,8}$' -or [long]$size -gt 33554432 -or
      $asset.state -cne 'uploaded' -or $asset.digest -cnotmatch '^sha256:[0-9a-f]{64}$' -or
      $asset.url -cne "https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/$id" -or
      $asset.browser_download_url -cne ($prefix + "download/$tag/$name")) { Fail 'official release asset metadata is invalid or ambiguous' }
  return $asset
}
function Test-AbsoluteLocalMetadataPath([string]$Path, [bool]$Windows = ([IO.Path]::DirectorySeparatorChar -eq '\')) {
  # IsPathRooted also accepts drive-relative paths and remote shares on Windows.
  if ($Windows) { return $Path -match '^[A-Za-z]:[\\/]' }
  return $Path.StartsWith('/', [StringComparison]::Ordinal) -and -not $Path.StartsWith('//', [StringComparison]::Ordinal)
}
function Initialize-BootstrapMetadata([string]$MetadataFile, [string]$FetchFile) {
  $script:installStage = 'verify_bootstrap_metadata'
  if (-not (Test-AbsoluteLocalMetadataPath $MetadataFile)) { Fail 'bootstrap metadata requires an absolute local file' }
  try { $file = Get-Item -LiteralPath $MetadataFile -ErrorAction Stop } catch { Fail 'bootstrap metadata is unreadable' }
  if ($file -isnot [IO.FileInfo] -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -or $file.Length -gt 2097152) { Fail 'bootstrap metadata must be a bounded regular non-linked file' }
  try { $metadata = Read-Json $MetadataFile } catch { Fail 'bootstrap metadata is unreadable' }
  $asset = Get-ReleaseAsset $metadata latest puretokens-skill-fetch.ps1
  if ((Get-Item -LiteralPath $FetchFile).Length -ne [long]$asset.size -or
      (Get-FileHash -LiteralPath $FetchFile -Algorithm SHA256).Hash.ToLowerInvariant() -cne $asset.digest.Substring(7)) { Fail 'bootstrap script does not match the selected release' }
  $script:officialReleaseMetadata = $metadata
  $script:officialReleaseTag = $metadata.tag_name
  $script:officialAssetRoute = $true
}
function Get-OfficialFile([string]$Url, [string]$Destination, [string]$Stage, [string]$Accept = 'application/vnd.github+json', [switch]$NoFallback) {
  $script:installStage = $Stage
  if (-not $NoFallback -and $script:officialAssetRoute -and
      $Url.StartsWith('https://github.com/puretokensx/puretokens-skill/releases/', [StringComparison]::Ordinal)) {
    return Get-OfficialAsset $Url $Destination $Stage
  }
  $status = 0
  $category = "transport_failure"
  $completed = $false
  $networkAccessDenied = $false
  $securityContextUnavailable = $false
  $timeoutSeconds = if ($Stage -eq 'download_platform_archive') { 180 } else { 45 }
  if ($NoFallback -and $Accept -eq 'application/vnd.github+json') { $timeoutSeconds = 45 }
  Write-Information "Pure Tokens download progress: stage=$Stage status=started" -InformationAction Continue
  try {
    $ProgressPreference = 'SilentlyContinue'
    $result = Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination -PassThru -TimeoutSec $timeoutSeconds -MaximumRedirection 5 -Headers @{ Accept = $Accept } -UserAgent "puretokens-skill-installer"
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
  if (-not $NoFallback -and -not $networkAccessDenied -and -not $securityContextUnavailable -and
      $category -in @('http_error','transport_failure') -and $status -in @(502,503,504) -and
      $Url.StartsWith('https://github.com/puretokensx/puretokens-skill/releases/', [StringComparison]::Ordinal)) {
    return Get-OfficialAsset $Url $Destination $Stage
  }
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
$downloadRoot = $null
$script:officialReleaseMetadata = $null
$script:officialReleaseTag = $null
$script:officialAssetRoute = $false
$delegated = $false
try {
  $script:installStage = "detect_host"
  # Keep selection in this script's scope so confirmed values reach sync/init.
  if ($Command -eq "detect-host") {
    if ($HostId -or $Target -or $ReleaseMetadata) { Fail "detect-host takes no host, target or release metadata options" }
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
  $locationOptions = @{}
  if ($HostId) { $locationOptions.HostId = $HostId }
  if ($Target) { $locationOptions.Target = $Target }
  $script:installStage = "prepare_download"
  $downloadRoot = Join-Path ([IO.Path]::GetTempPath()) ("puretokens-download-" + [Guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $downloadRoot | Out-Null
  if ($ReleaseMetadata) { Initialize-BootstrapMetadata $ReleaseMetadata $PSCommandPath }
  $releaseFile = Join-Path $downloadRoot "release.json"
  $script:installStage = "read_release_manifest"
  $null = Get-OfficialFile "https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json" $releaseFile read_release_manifest
  try { $release = Read-Json $releaseFile } catch { Fail "the stable release manifest is unreadable" }
  if ($release.schemaVersion -ne 2) { Fail "the stable release manifest format is unsupported; no source fallback" }
  $sourceCommit = $release.sourceCommit
  if ($sourceCommit -cnotmatch '^[0-9a-f]{40}$') { Fail "the stable release has no verified source commit" }
  $availableVersion = $release.version
  if ($availableVersion -cnotmatch '^\d+\.\d+\.\d+$') { Fail "the stable release version is invalid" }
  if ($script:officialReleaseTag -and $script:officialReleaseTag -cne "v$availableVersion") { Fail 'the manifest version does not match the selected official release' }
  $releaseOrigin = "https://github.com/puretokensx/puretokens-skill/releases/download/v$availableVersion"
  $selector = $release.installers.powershell
  if ($null -eq $selector -or $selector.filename -cne "puretokens-skill-install.ps1" -or $selector.sha256 -cnotmatch '^[0-9a-f]{64}$') { Fail "the stable directory selector is invalid" }
  $locationInstaller = Join-Path $downloadRoot "puretokens-skill-install.ps1"
  $script:installStage = "download_selector"
  $null = Get-OfficialFile "$releaseOrigin/$($selector.filename)" $locationInstaller download_selector
  if ((Get-FileHash -LiteralPath $locationInstaller -Algorithm SHA256).Hash.ToLowerInvariant() -cne $selector.sha256) { Fail "stable directory selector checksum mismatch" }
  $script:installStage = "select_target"
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
    $script:installStage = "verify_installed"
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
  $script:installStage = "download_platform_archive"
  $null = Get-OfficialFile "$releaseOrigin/$($artifact.filename)" $archive download_platform_archive
  if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.sha256) { Fail "published platform archive checksum mismatch" }
  $script:installStage = "verify_archive"
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
  $script:installStage = "extract_archive"
  [IO.Compression.ZipFile]::ExtractToDirectory($archive, $unpacked)
  $script:installStage = "verify_archive"
  $sourceRoot = Join-Path $unpacked $archiveRoot
  if ((Read-Json (Join-Path $sourceRoot "package.json")).version -ne $availableVersion) { Fail "the archive version does not match the stable release" }
  if ((Get-FileHash -LiteralPath (Join-Path $sourceRoot "runtime/puretokens-skill-install.ps1") -Algorithm SHA256).Hash.ToLowerInvariant() -cne $selector.sha256) { Fail "archive selector does not match the stable release" }
  if ((Get-FileHash -LiteralPath (Join-Path $sourceRoot "runtime/executor/bin/puretokens-api-$platform.exe") -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.executorSha256) { Fail "archive executor does not match the stable release" }
  $syncOptions = @{ Source = $sourceRoot; Target = $Target }
  if ($Command -ne "repair") { $syncOptions.NoRecovery = $true }
  if ($HostId) { $syncOptions.HostId = $HostId }
  $syncCommand = if ($Command -eq "repair") { "repair" } else { "sync" }
  $script:installStage = "sync"
  $delegated = $true
  & (Join-Path $sourceRoot "runtime/puretokens-skill-install.ps1") $syncCommand @syncOptions
} catch {
  if ($_.Exception.Data["puretokens_receipt"]) { throw }
  $category = "local_operation_failed"
  $next = "review_local_access"
  if ($_.Exception -is [System.UnauthorizedAccessException] -or $_.CategoryInfo.Category -eq "PermissionDenied") {
    $category = "local_access_denied"
    $next = "review_host_permissions"
  } elseif ($_.Exception -is [System.IO.IOException]) {
    $category = "local_io_failure"
    $next = "review_download_directory_permissions"
  }
  $state = if ($delegated) { "unconfirmed" } else { "not_completed" }
  if ($script:installStage -eq "prepare_download") {
    $state = "not_started"
    $next = "review_download_directory_permissions"
  }
  $changed = if ($delegated) { "unknown" } else { "false" }
  Fail "stage=$script:installStage error_code=$category installation_status=$state installed_files_changed=$changed next_step=$next; a local operation could not complete. Check current-session command/directory access, file locks and available disk space; the exact cause is unconfirmed. No automatic retry."
} finally {
  if ($downloadRoot) {
    try {
      if (Test-Path -LiteralPath $downloadRoot) { Remove-Item -LiteralPath $downloadRoot -Recurse -Force -ErrorAction Stop }
    } catch { Write-Output "cleanup_status: pending; download temporary files were retained. Preserve the preceding install result; do not bypass host cleanup restrictions." }
  }
}
