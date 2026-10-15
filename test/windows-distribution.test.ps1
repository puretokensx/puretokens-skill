param([switch]$Child)
$ErrorActionPreference = "Stop"
if (-not $Child) {
  $engines = @("powershell.exe", "pwsh")
  $x86PowerShell = Join-Path $env:SystemRoot 'SysWOW64\WindowsPowerShell\v1.0\powershell.exe'
  if (Test-Path -LiteralPath $x86PowerShell) { $engines += $x86PowerShell }
  foreach ($engine in $engines) {
    if ($engine -eq $x86PowerShell) {
      $actualProcess = & $engine -NoProfile -Command '[Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()'
      if ($LASTEXITCODE -ne 0 -or $actualProcess -ne 'X86') { throw "SysWOW64 download fixture did not execute a 32-bit process" }
    }
    & $engine -NoProfile -ExecutionPolicy Bypass -File $PSCommandPath -Child
    if ($LASTEXITCODE -ne 0) { throw "$engine distribution fixtures failed" }
  }
  return
}
$repository = Split-Path -Parent $PSScriptRoot
& (Join-Path $PSScriptRoot 'support/windows-download-diagnostics.ps1')
& (Join-Path $PSScriptRoot 'support/windows-host-detection.ps1')
$root = Join-Path ([IO.Path]::GetTempPath()) ("pt-download-test-" + [Guid]::NewGuid().ToString("N"))
$savedEnvironment = @{}
foreach ($name in @("USERPROFILE", "HOME", "CODEX_HOME", "TMP", "TEMP")) { $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, "Process") }
try {
  New-Item -ItemType Directory -Path $root | Out-Null
  $env:USERPROFILE = Join-Path $root "home"
  $env:HOME = $env:USERPROFILE
  $env:CODEX_HOME = Join-Path $env:USERPROFILE ".codex"
  $env:TMP = Join-Path $root "tmp"
  $env:TEMP = $env:TMP
  New-Item -ItemType Directory -Path $env:USERPROFILE, $env:TMP | Out-Null
  $global:fixtureCommit = "a" * 40
  $global:fixtureVersion = (Get-Content -LiteralPath (Join-Path $repository "package.json") -Raw -Encoding UTF8 | ConvertFrom-Json).version
  $platform = if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -eq "Arm64") { "windows-arm64" } else { "windows-amd64" }
  $source = Join-Path $root "puretokens-skill"
  New-Item -ItemType Directory -Path (Join-Path $source "runtime/executor/bin") -Force | Out-Null
  foreach ($file in @("README.md", "package.json")) { Copy-Item -LiteralPath (Join-Path $repository $file) -Destination $source }
  Copy-Item -LiteralPath (Join-Path $repository "skills") -Destination $source -Recurse
  foreach ($file in @("puretokens-skill-install.ps1", "puretokens-skill-fetch.ps1")) {
    Copy-Item -LiteralPath (Join-Path (Join-Path $repository "runtime") $file) -Destination (Join-Path $source "runtime")
  }
  Copy-Item -LiteralPath (Join-Path $repository "runtime/executor/manifest.json") -Destination (Join-Path $source "runtime/executor")
  Copy-Item -LiteralPath (Join-Path $repository "runtime/executor/bin/puretokens-api-$platform.exe") -Destination (Join-Path $source "runtime/executor/bin")
  $global:fixtureSource = $source
  $global:fixtureRoot = $root
  Add-Type -AssemblyName System.IO.Compression
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  # Match the Unix-built release archives even on the .NET Framework ZIP API.
  $zip = [IO.Compression.ZipFile]::Open((Join-Path $root "platform.zip"), [IO.Compression.ZipArchiveMode]::Create)
  try {
    foreach ($file in @(Get-ChildItem -LiteralPath $source -Recurse -Force -File)) {
      $entryName = "puretokens-skill/" + $file.FullName.Substring($source.Length + 1).Replace('\', '/')
      $null = [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, $file.FullName, $entryName, [IO.Compression.CompressionLevel]::Fastest)
    }
  } finally { $zip.Dispose() }
  $release = @{
    schemaVersion = 2
    version = $global:fixtureVersion
    sourceCommit = $global:fixtureCommit
    installers = @{ powershell = @{ filename = "puretokens-skill-install.ps1"; sha256 = (Get-FileHash -LiteralPath (Join-Path $source "runtime/puretokens-skill-install.ps1") -Algorithm SHA256).Hash.ToLowerInvariant() } }
    files = @{ $platform = @{
      filename = "puretokens-skill-$global:fixtureVersion-$platform.zip"
      sha256 = (Get-FileHash -LiteralPath (Join-Path $root "platform.zip") -Algorithm SHA256).Hash.ToLowerInvariant()
      executorSha256 = (Get-FileHash -LiteralPath (Join-Path $source "runtime/executor/bin/puretokens-api-$platform.exe") -Algorithm SHA256).Hash.ToLowerInvariant()
    } }
  }
  $release | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root "release.json") -Encoding UTF8
  $global:fixtureFailureMatch = $null
  $global:fixtureFailureKind = $null
  function global:Invoke-WebRequest {
    param($Uri, $OutFile, $TimeoutSec, $Headers, $UserAgent, [switch]$UseBasicParsing, [switch]$PassThru)
    $global:fixtureRequests.Add([string]$Uri)
    if ($global:fixtureFailureMatch -and $Uri -like $global:fixtureFailureMatch) {
      if ($global:fixtureFailureKind -eq 'network_access_denied') { throw [System.Net.Sockets.SocketException]::new(10013) }
      throw [System.Security.Authentication.AuthenticationException]::new('private-fixture', [System.ComponentModel.Win32Exception]::new(-2146893042, 'private-fixture'))
    }
    switch -Regex ($Uri) {
      "/releases/download/v$global:fixtureVersion/puretokens-skill-install.ps1$" { Copy-Item -LiteralPath (Join-Path $global:fixtureSource "runtime/puretokens-skill-install.ps1") -Destination $OutFile }
      '/releases/latest/download/release-manifest.json$' {
        if ($global:fixtureMode -eq "missing") { return [PSCustomObject]@{ StatusCode = 404 } }
        Copy-Item -LiteralPath (Join-Path $global:fixtureRoot "release.json") -Destination $OutFile
      }
      '/puretokens-skill-[0-9.]+-windows-(amd64|arm64).zip$' {
        if ($global:fixtureConcurrentSource) {
          & (Join-Path $global:fixtureConcurrentSource "runtime/puretokens-skill-install.ps1") sync -Target $global:fixtureConcurrentTarget -Source $global:fixtureConcurrentSource | Out-Null
        }
        Copy-Item -LiteralPath (Join-Path $global:fixtureRoot "platform.zip") -Destination $OutFile
      }
      default { throw "Unexpected fixture request: $Uri" }
    }
    return [PSCustomObject]@{ StatusCode = 200 }
  }
  $fetch = Join-Path $source "runtime/puretokens-skill-fetch.ps1"
  foreach ($mode in @("platform")) {
    $global:fixtureMode = $mode
    $global:fixtureRequests = New-Object 'System.Collections.Generic.List[string]'
    $target = Join-Path $root "target-$mode"
    & $fetch check-update -Target $target
    if ($global:fixtureRequests.Count -ne 2 -or (Test-Path -LiteralPath $target)) { throw "check-update changed local installation or fetched more than metadata" }
    $global:fixtureRequests.Clear()
    & $fetch install -Target $target
    if (-not (Test-Path -LiteralPath (Join-Path $target ".puretokens-executor/puretokens-api.exe"))) { throw "download did not install the platform executor" }
    $installedRuntime = Get-Content -LiteralPath (Join-Path $target ".puretokens-executor/runtime.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($installedRuntime.platform -ne $platform) { throw "download selected the wrong operating system architecture" }
    $sourceRequests = @($global:fixtureRequests | Where-Object { $_ -like "https://codeload.github.com/*" })
    if ($sourceRequests.Count -ne 0 -or $global:fixtureRequests.Count -ne 3) { throw "incorrect stable distribution selection" }
    if (@(Get-ChildItem -LiteralPath $env:TMP -Force).Count -ne 0) { throw "private download staging was not cleaned" }
    function Get-FixtureSnapshot([string]$Directory) {
      return @(Get-ChildItem -LiteralPath $Directory -Recurse -Force -File | Sort-Object FullName | ForEach-Object {
        "$($_.FullName)|$($_.LastWriteTimeUtc.Ticks)|$((Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash)"
      }) -join "`n"
    }
    $before = Get-FixtureSnapshot $target
    foreach ($failure in @(
      @{ Match = '*release-manifest.json'; Stage = 'read_release_manifest'; Count = 1 },
      @{ Match = '*puretokens-skill-install.ps1'; Stage = 'download_selector'; Count = 2 }
    )) {
      foreach ($category in @('network_access_denied', 'tls_security_context_unavailable')) {
        $global:fixtureFailureMatch = $failure.Match
        $global:fixtureFailureKind = $category
        $global:fixtureRequests.Clear()
        $rejected = $false
        try { & $fetch update -Target $target }
        catch {
          $message = $_.Exception.Message
          $rejected = $message.Contains("stage=$($failure.Stage) error_code=$category http_status=0") -and
            $message.Contains('installed_files_changed=false next_step=review_host_permissions') -and
            $message -notmatch 'private-fixture'
        }
        if (-not $rejected -or $global:fixtureRequests.Count -ne $failure.Count -or (Get-FixtureSnapshot $target) -cne $before) { throw 'Denied download retried or changed existing installation' }
        if (@(Get-ChildItem -LiteralPath $env:TMP -Force).Count -ne 0) { throw 'Denied download retained staging files' }
      }
    }
    $global:fixtureFailureMatch = '*-windows-*.zip'
    $global:fixtureFailureKind = 'network_access_denied'
    $global:fixtureRequests.Clear()
    $newTarget = Join-Path $root 'explicit-continuation'
    $rejected = $false
    try { & $fetch install -Target $newTarget }
    catch { $rejected = $_.Exception.Message -like '*stage=download_platform_archive error_code=network_access_denied*' }
    if (-not $rejected -or $global:fixtureRequests.Count -ne 3 -or (Test-Path -LiteralPath $newTarget)) { throw 'Denied archive download created an installation or retried' }
    $global:fixtureFailureMatch = $null
    $global:fixtureRequests.Clear()
    # Explicit new invocation after the simulated blocker changes; still verifies all release inputs.
    & $fetch install -Target $newTarget
    if ($global:fixtureRequests.Count -ne 3 -or -not (Test-Path -LiteralPath (Join-Path $newTarget '.puretokens-executor/puretokens-api.exe'))) { throw 'Explicit installation continuation failed' }
    foreach ($command in @("install", "update")) {
      $global:fixtureRequests.Clear()
      $output = & $fetch $command -Target $target -HostId codex
      if (($output -join "`n") -notlike "*already current and verified*" -or ($output -join "`n") -like "*Pure Tokens Skill init:*") { throw "same-version shortcut did not skip init" }
      if ($global:fixtureRequests.Count -ne 2 -or (Get-FixtureSnapshot $target) -cne $before) { throw "same-version shortcut downloaded an archive or changed files" }
    }
    $skillFile = Join-Path $target "puretokens-image/SKILL.md"
    $original = [IO.File]::ReadAllBytes($skillFile)
    "user changes" | Set-Content -LiteralPath $skillFile -Encoding UTF8
    $global:fixtureRequests.Clear()
    $rejected = $false
    try { & $fetch update -Target $target } catch { $rejected = $true }
    if (-not $rejected -or $global:fixtureRequests.Count -ne 2 -or (Get-Content -LiteralPath $skillFile -Raw) -notlike "*user changes*") { throw "modified same-version files were not preserved" }
    Remove-Item -LiteralPath $skillFile
    $rejected = $false
    try { & $fetch update -Target $target } catch { $rejected = $true }
    if (-not $rejected -or (Test-Path -LiteralPath $skillFile)) { throw "missing same-version file was automatically repaired" }
    [IO.File]::WriteAllBytes($skillFile, $original)
  }
  $bootstrap = Join-Path $root "legacy-bootstrap"
  New-Item -ItemType Directory -Path $bootstrap | Out-Null
  Copy-Item -LiteralPath $fetch -Destination $bootstrap
  "# puretokens-locate-v1`nthrow 'old selector used'" | Set-Content -LiteralPath (Join-Path $bootstrap "puretokens-skill-install.ps1") -Encoding ASCII
  $global:fixtureRequests.Clear()
  & (Join-Path $bootstrap "puretokens-skill-fetch.ps1") check-update -Target (Join-Path $root "legacy-target")
  if (@($global:fixtureRequests | Where-Object { $_ -like "*/releases/download/*/puretokens-skill-install.ps1" }).Count -ne 1) { throw "legacy sibling selector was not replaced by the pinned selector" }

  $global:fixtureRequests.Clear()
  & (Join-Path $bootstrap "puretokens-skill-fetch.ps1") install -HostId zcode -Target (Join-Path $root "new-host-target")
  if (-not (Test-Path -LiteralPath (Join-Path $root "new-host-target/puretokens-image/SKILL.md"))) { throw "marked old selector blocked ZCode" }

  # Old installations retain their managed files until the newly downloaded
  # official bootstrap has verified the complete current release.
  $oldSource = Join-Path $root "old-repository-source"
  Copy-Item -LiteralPath $source -Destination $oldSource -Recurse
  @{ version = "0.16.0" } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $oldSource "package.json") -Encoding UTF8
  $oldFetch = Join-Path $oldSource "runtime/puretokens-skill-fetch.ps1"
  (Get-Content -LiteralPath $oldFetch -Raw -Encoding UTF8).Replace(
    'github.com/puretokensx/puretokens-skill', 'github.com/PureTokens/puretokens-skill'
  ) | Set-Content -LiteralPath $oldFetch -Encoding UTF8
  $oldTarget = Join-Path $root "old-repository-target"
  & (Join-Path $oldSource "runtime/puretokens-skill-install.ps1") sync -Target $oldTarget -Source $oldSource
  'keep personal files' | Set-Content -LiteralPath (Join-Path $oldTarget 'unmanaged.txt') -Encoding UTF8
  $before = Get-FixtureSnapshot $oldTarget
  $validChecksum = $release.files[$platform].sha256
  $release.files[$platform].sha256 = '0' * 64
  $release | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root "release.json") -Encoding UTF8
  $global:fixtureRequests.Clear()
  $rejected = $false
  try { & (Join-Path $bootstrap "puretokens-skill-fetch.ps1") update -Target $oldTarget }
  catch { $rejected = $_.Exception.Message -like '*checksum mismatch*' }
  if (-not $rejected -or (Get-FixtureSnapshot $oldTarget) -cne $before -or $global:fixtureRequests.Count -ne 3) {
    throw 'Failed new-bootstrap verification changed an old installation'
  }
  $release.files[$platform].sha256 = $validChecksum
  $release | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root "release.json") -Encoding UTF8
  $global:fixtureRequests.Clear()
  & (Join-Path $bootstrap "puretokens-skill-fetch.ps1") update -Target $oldTarget
  $upgraded = Get-Content -LiteralPath (Join-Path $oldTarget '.puretokens-executor/runtime.json') -Raw -Encoding UTF8 | ConvertFrom-Json
  if ($upgraded.version -ne $global:fixtureVersion -or
      (Get-Content -LiteralPath (Join-Path $oldTarget 'unmanaged.txt') -Raw) -notlike '*keep personal files*' -or
      (Get-Content -LiteralPath (Join-Path $oldTarget '.puretokens-executor/puretokens-skill-fetch.ps1') -Raw) -like '*github.com/PureTokens/*' -or
      $global:fixtureRequests.Count -ne 3 -or
      @($global:fixtureRequests | Where-Object { $_ -notlike 'https://github.com/puretokensx/puretokens-skill/releases/*' }).Count -ne 0) {
    throw 'The current official bootstrap did not safely upgrade the original installation'
  }

  $newer = Join-Path $root "newer-source"
  Copy-Item -LiteralPath $source -Destination $newer -Recurse
  @{ version = "99.0.0" } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $newer "package.json") -Encoding UTF8
  $global:fixtureConcurrentSource = $newer
  $global:fixtureConcurrentTarget = Join-Path $root "target-concurrent"
  $global:fixtureMode = "platform"
  $rejected = $false
  try { & $fetch update -Target $global:fixtureConcurrentTarget } catch { $rejected = $_.Exception.Message -like "*downgrade was stopped under the update lock*" }
  if (-not $rejected) { throw "an older download overwrote the newer installation" }
  $currentRuntime = Get-Content -LiteralPath (Join-Path $global:fixtureConcurrentTarget ".puretokens-executor/runtime.json") -Raw -Encoding UTF8 | ConvertFrom-Json
  if ($currentRuntime.version -ne "99.0.0") { throw "newer version was not preserved" }
  $global:fixtureConcurrentSource = $null

  $global:fixtureConcurrentTarget = Join-Path $root "target-platform"
  $global:fixtureNewerSource = $newer
  # Interleave after fetch's checksum read but before selector verification.
  function global:Get-FileHash {
    param($LiteralPath, $Algorithm)
    $result = Microsoft.PowerShell.Utility\Get-FileHash -LiteralPath $LiteralPath -Algorithm $Algorithm
    if ($LiteralPath -eq (Join-Path $global:fixtureConcurrentTarget ".puretokens-executor/puretokens-api.exe") -and $global:fixtureNewerSource) {
      $next = $global:fixtureNewerSource
      $global:fixtureNewerSource = $null
      & (Join-Path $next "runtime/puretokens-skill-install.ps1") sync -Target $global:fixtureConcurrentTarget -Source $next | Out-Null
    }
    return $result
  }
  $global:fixtureRequests.Clear()
  $rejected = $false
  try { & $fetch update -Target $global:fixtureConcurrentTarget }
  catch { $rejected = $_.Exception.Message -like "*installed release changed during verification*" }
  finally { Remove-Item function:global:Get-FileHash }
  if (-not $rejected -or $global:fixtureRequests.Count -ne 2) { throw "same-version verification accepted a newer concurrent installation" }
  $currentRuntime = Get-Content -LiteralPath (Join-Path $global:fixtureConcurrentTarget ".puretokens-executor/runtime.json") -Raw -Encoding UTF8 | ConvertFrom-Json
  if ($currentRuntime.version -ne "99.0.0") { throw "same-version verification did not preserve the newer installation" }

  $global:fixtureMode = "missing"
  $rejected = $false
  $global:fixtureRequests.Clear()
  try { & $fetch install -Target (Join-Path $root "missing-release") } catch { $rejected = $_.Exception.Message -like "*read_release_manifest*" }
  if (-not $rejected -or $global:fixtureRequests.Count -ne 1 -or (Test-Path -LiteralPath (Join-Path $root "missing-release"))) { throw "missing stable release did not stop without fallback" }
  $global:fixtureMode = "platform"
  $release.files[$platform].sha256 = "0" * 64
  $release | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root "release.json") -Encoding UTF8
  $rejected = $false
  try { & $fetch install -Target (Join-Path $root "bad-checksum") } catch { $rejected = $_.Exception.Message -like "*checksum mismatch*" }
  if (-not $rejected -or (Test-Path -LiteralPath (Join-Path $root "bad-checksum"))) { throw "checksum mismatch was not rejected before mutation" }

  # A matching checksum must not make a backslash member acceptable.
  $zip = [IO.Compression.ZipFile]::Open((Join-Path $root "platform.zip"), [IO.Compression.ZipArchiveMode]::Update)
  try { $null = $zip.CreateEntry('puretokens-skill/unsafe\member.txt') } finally { $zip.Dispose() }
  $release.files[$platform].sha256 = (Get-FileHash -LiteralPath (Join-Path $root "platform.zip") -Algorithm SHA256).Hash.ToLowerInvariant()
  $release | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $root "release.json") -Encoding UTF8
  $rejected = $false
  try { & $fetch install -Target (Join-Path $root "unsafe-archive") } catch { $rejected = $_.Exception.Message -like "*unsafe path or symlink*" }
  if (-not $rejected -or (Test-Path -LiteralPath (Join-Path $root "unsafe-archive"))) { throw "backslash archive member was not rejected before mutation" }
} finally {
  Remove-Item function:global:Invoke-WebRequest -ErrorAction SilentlyContinue
  foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], "Process") }
  if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
