# puretokens-locate-v1
[CmdletBinding()]
param(
  [Parameter(Position = 0, Mandatory = $true)]
  [ValidateSet("check", "verify-installed", "init", "sync", "repair", "locate")]
  [string]$Command,
  [Parameter(Mandatory = $false)]
  [string]$Target,
  [Parameter(Mandatory = $false)]
  [ValidateSet("claude-code", "codex", "workbuddy", "gemini-cli", "grok-build", "opencode", "trae", "claude-desktop", "dsh-desktop", "deepseek-harness", "minimax-code", "zcode", "kimi-code", "qoder", "pi", "hermes", "evox", "vscode", "octop")]
  [Alias("Host")]
  [string]$HostId,
  [Parameter(Mandatory = $false)]
  [string]$Source,
  [Parameter(Mandatory = $false)]
  [string]$ReleaseManifest,
  [switch]$NoRecovery
)

$ErrorActionPreference = "Stop"
$currentSkills = @("puretokens-balance", "puretokens-connection", "puretokens-models", "puretokens-image", "puretokens-video", "puretokens-audio", "puretokens-evaluate", "puretokens-update")
$script:installStage = "arguments"
$script:installationStatus = "not_started"
$script:installedFilesChanged = "false"

function Fail([string]$Message) {
  $exception = New-Object System.InvalidOperationException("Pure Tokens Skill installer: stage=$script:installStage error_code=validation_failed installation_status=$script:installationStatus installed_files_changed=$script:installedFilesChanged next_step=review_installation_failure; $Message")
  $exception.Data["puretokens_receipt"] = $true
  throw $exception
}

# Only the verified native executor uses this launcher. No shell, window,
# elevation, credential argument or fallback launch is involved.
function Invoke-NativeExecutor([string]$Executable, [string[]]$ArgumentList) {
  $quoted = foreach ($argument in $ArgumentList) {
    if ($argument.Contains('"') -or $argument.Contains("`n") -or $argument.Contains("`r")) { Fail "invalid native executor argument" }
    # Windows command-line quoting: trailing backslashes must be doubled.
    '"' + [regex]::Replace($argument, '(\\+)$', '$1$1') + '"'
  }
  $start = New-Object System.Diagnostics.ProcessStartInfo
  $start.FileName = $Executable
  $start.Arguments = $quoted -join ' '
  $start.UseShellExecute = $false
  $start.CreateNoWindow = $true
  $start.RedirectStandardOutput = $true
  $start.RedirectStandardError = $true
  $start.RedirectStandardInput = $true
  $start.StandardOutputEncoding = New-Object System.Text.UTF8Encoding($false)
  $start.StandardErrorEncoding = New-Object System.Text.UTF8Encoding($false)
  $process = New-Object System.Diagnostics.Process
  $process.StartInfo = $start
  try {
    if (-not $process.Start()) { Fail "native executor could not start" }
    $process.StandardInput.Close()
    # Drain both pipes concurrently; errors cannot block behind a full pipe.
    $stdout = $process.StandardOutput.ReadToEndAsync()
    $stderr = $process.StandardError.ReadToEndAsync()
    $timeoutMs = if ($ArgumentList[0] -eq "init") { 25000 } else { 30000 }
    if (-not $process.WaitForExit($timeoutMs)) {
      try { $process.Kill(); $null = $process.WaitForExit(5000) } catch { }
      Fail "native executor timed out; stop this installation attempt and report the stage; do not retry or modify the installer"
    }
    if (-not [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]]@($stdout, $stderr), 5000)) { Fail "native executor output was not available; stop without a fallback launcher" }
    $text = $stdout.GetAwaiter().GetResult()
    $null = $stderr.GetAwaiter().GetResult()
    return [PSCustomObject]@{ ExitCode = $process.ExitCode; Output = $text }
  } catch { Fail "native executor launch, timeout or output capture failed; stop this attempt and report the stage; do not probe, patch or retry through another launcher" }
  finally { $process.Dispose() }
}

# Cleanup is best effort only after a transaction is committed or restored.
# Never bypass a host deletion guard or retry a denied deletion in finally.
function Remove-CompletedStage([string]$Path) {
  $verified = Invoke-NativeExecutor $script:ownershipGuard @("install-transaction-verify", "--directory", $Path, "--target-directory", $targetRoot, "--source-directory", $sourceRoot)
  if ($verified.ExitCode -ne 0) { Fail "transaction cleanup failed verification; all remaining files were preserved" }
  try { Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop }
  catch { Write-Output "cleanup_status: pending; completed transaction files were retained. Host cleanup permission is required; do not delete unknown directories or reinstall solely for this warning." }
}

function Test-InstallationEntry([string]$Path) {
  return $null -ne (Get-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue)
}

function Test-ManagedSkill([string]$Directory, [string]$Name) {
  $manifest = Join-Path $Directory "skill.json"
  if (-not (Test-Path -LiteralPath (Join-Path $Directory "SKILL.md") -PathType Leaf) -or -not (Test-Path -LiteralPath $manifest -PathType Leaf)) { return $false }
  try { return ((Get-Content -LiteralPath $manifest -Raw -Encoding UTF8 | ConvertFrom-Json).name -eq $Name) } catch { return $false }
}


function Test-ManagedExecutor([string]$Directory) {
  $manifest = Join-Path $Directory "runtime.json"
  return (Test-Path -LiteralPath (Join-Path $Directory "puretokens-api.exe") -PathType Leaf) -and (Test-Path -LiteralPath $manifest -PathType Leaf) -and ((Get-Content -LiteralPath $manifest -Raw -Encoding UTF8 | ConvertFrom-Json).name -eq "puretokens-api-executor")
}

function Assert-OwnedInstallation([string]$Directory, [string]$Name, [string]$SourceDirectory = "") {
  $verificationCommand = if ($Command -eq "repair") { "install-repair-verify" } else { "install-verify" }
  $options = @($verificationCommand, "--directory", $Directory, "--name", $Name)
  if ($SourceDirectory) { $options += @("--source-directory", $SourceDirectory) }
  $result = Invoke-NativeExecutor $script:ownershipGuard $options
  if ($result.ExitCode -ne 0) { Fail "installation ownership or file changes require review: $Name; existing files were preserved" }
}

function Get-ExecutorPlatform() {
  $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  switch ($architecture) {
    "x64" { return "windows-amd64" }
    "arm64" { return "windows-arm64" }
    default { Fail "no Pure Tokens executor is available for this operating system and CPU" }
  }
}

function Get-ExecutorArtifact([string]$SourceRoot) {
  $manifestPath = Join-Path $SourceRoot "runtime\executor\manifest.json"
  if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) { Fail "official source is missing the executor manifest" }
  try { $manifest = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json } catch { Fail "official source executor manifest is invalid" }
  $platform = Get-ExecutorPlatform
  $artifact = $manifest.artifacts.PSObject.Properties[$platform].Value
  if ($null -eq $artifact -or [string]::IsNullOrWhiteSpace($artifact.path) -or [string]::IsNullOrWhiteSpace($artifact.sha256)) { Fail "official source is missing the executor artifact for $platform" }
  $binary = Join-Path (Join-Path $SourceRoot "runtime\executor") $artifact.path
  if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) { Fail "official source is missing the executor binary for $platform" }
  $actualHash = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actualHash -ne $artifact.sha256.ToLowerInvariant()) { Fail "official source executor checksum mismatch for $platform" }
  return $binary
}

function Get-SourceVersion([string]$SourceRoot) {
  try {
    $version = (Get-Content -LiteralPath (Join-Path $SourceRoot "package.json") -Raw -Encoding UTF8 | ConvertFrom-Json).version
    if ($version -notmatch '^\d+\.\d+\.\d+$') { Fail "official source has an invalid Skill version" }
    return $version
  } catch { Fail "official source has an invalid Skill version" }
}

function Test-OfficialSource([string]$SourceRoot) {
  if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot "README.md") -PathType Leaf)) { Fail "official source is missing README.md" }
  if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot "package.json") -PathType Leaf)) { Fail "official source is missing package.json" }
  $null = Get-SourceVersion $SourceRoot
  if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot "runtime\puretokens-skill-install.ps1") -PathType Leaf)) { Fail "official source is missing the Windows installer" }
  if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot "runtime\puretokens-skill-fetch.ps1") -PathType Leaf)) { Fail "official source is missing the download wrapper" }
  $null = Get-ExecutorArtifact $SourceRoot
  foreach ($name in $currentSkills) {
    if (-not (Test-ManagedSkill (Join-Path (Join-Path $SourceRoot "skills") $name) $name)) { Fail "official source has an invalid Skill: $name" }
  }
}

function Test-InstalledTarget([string]$TargetRoot) {
  if (-not (Test-Path -LiteralPath $TargetRoot -PathType Container)) { Fail "target Skill directory does not exist or is inaccessible" }
  foreach ($name in $currentSkills) {
    if (-not (Test-ManagedSkill (Join-Path $TargetRoot $name) $name)) { Fail "target is missing the managed Skill: $name" }
  }
  if (-not (Test-ManagedExecutor (Join-Path $TargetRoot ".puretokens-executor"))) { Fail "target is missing the managed native executor" }
}

function Confirm-ReleaseIdentity([string]$TargetRoot) {
  if ([string]::IsNullOrWhiteSpace($ReleaseManifest)) { return }
  try {
    $release = Get-Content -LiteralPath $ReleaseManifest -Raw -Encoding UTF8 | ConvertFrom-Json
    $installed = Get-Content -LiteralPath (Join-Path $TargetRoot ".puretokens-executor/runtime.json") -Raw -Encoding UTF8 | ConvertFrom-Json
  } catch { Fail "release identity is unreadable; existing files were preserved" }
  $platform = Get-ExecutorPlatform
  $artifact = $release.files.PSObject.Properties[$platform].Value
  if ($release.version -cnotmatch '^\d+\.\d+\.\d+$' -or $null -eq $artifact -or $artifact.executorSha256 -cnotmatch '^[0-9a-f]{64}$') { Fail "selected release identity is invalid" }
  if ($installed.version -cne $release.version -or $installed.platform -cne $platform) { Fail "installed release changed during verification; existing files were preserved" }
  $executor = Join-Path $TargetRoot ".puretokens-executor/puretokens-api.exe"
  if (-not (Test-Path -LiteralPath $executor -PathType Leaf) -or ((Get-Item -LiteralPath $executor).Attributes -band [IO.FileAttributes]::ReparsePoint)) { Fail "installed executor is missing or unsupported; existing files were preserved" }
  if ((Get-FileHash -LiteralPath $executor -Algorithm SHA256).Hash.ToLowerInvariant() -cne $artifact.executorSha256) { Fail "installed executor changed during verification; existing files were preserved" }
}

function Confirm-InstalledInventories([string]$TargetRoot) {
  $TargetRoot = (Resolve-Path -LiteralPath $TargetRoot).Path
  $readLock = $null
  try {
    $lockFile = Join-Path $TargetRoot ".puretokens-install.lock"
    if (Test-InstallationEntry $lockFile) {
      try { $readLock = [IO.File]::Open($lockFile, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::None) }
      catch { Fail "installation transaction requires review; existing files were preserved" }
    }
    Test-InstalledTarget $TargetRoot
    if (@(Get-ChildItem -LiteralPath $TargetRoot -Force -Filter ".puretokens-skill-stage-*").Count -gt 0) { Fail "installation transaction requires review; existing files were preserved" }
    Test-GeminiDuplicates $TargetRoot
    Confirm-ReleaseIdentity $TargetRoot
    $script:ownershipGuard = Join-Path $TargetRoot ".puretokens-executor/puretokens-api.exe"
    foreach ($name in ($currentSkills + @(".puretokens-executor"))) {
      Assert-OwnedInstallation (Join-Path $TargetRoot $name) $name
    }
    Confirm-ReleaseIdentity $TargetRoot
  } finally { if ($null -ne $readLock) { $readLock.Dispose() } }
  Write-Output "Pure Tokens installed inventories verified; no files changed and init was not run."
}
# generated:install-activation
function Show-InstallationSummary([string]$RequestedHost) {
  switch ($RequestedHost) {
    'zcode' {
      Write-Output 'Next: Refresh and enable the installed Skills in ZCode, then start a new conversation. Loading has not been verified in this session.'
      Write-Output 'Available instructions: images, video, audio/music, Jev evaluation, models and balance. Connection and task results remain separate checks.'
      Write-Output 'Example (not executed): Generate an image of a lake at sunset.'
      Write-Output 'Example (not executed): Generate a 5-second video of a paper boat on a river.'
      Write-Output 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
    }
    'trae' {
      Write-Output 'Next: This is a historical installation directory; Skill discovery in current TraeWork is not confirmed.'
      Write-Output 'API tasks: unsupported in this host; no credential adapter is available. Installing instructions does not enable generation.'
      Write-Output 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
    }
    default {
      Write-Output 'Next: Follow this client''s Skill loading controls, then start a new conversation. Loading has not been verified in this session.'
      Write-Output 'Available instructions: images, video, audio/music, Jev evaluation, models and balance. Connection and task results remain separate checks.'
      Write-Output 'Example (not executed): Generate an image of a lake at sunset.'
      Write-Output 'Example (not executed): Generate a 5-second video of a paper boat on a river.'
      Write-Output 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
    }
  }
}
# /generated:install-activation

function Invoke-Init([string]$TargetRoot, [string]$RequestedHost) {
  try {
  Test-InstalledTarget $TargetRoot
  if ([string]::IsNullOrWhiteSpace($RequestedHost)) {
    Write-Output "Pure Tokens Skill init: host ID was not supplied, so the connection check was deferred."
    Write-Output "init_status=deferred next_step=confirm_current_client"
  } else {
    Write-Output "Installation host: $RequestedHost"
    $executor = Join-Path (Join-Path $TargetRoot ".puretokens-executor") "puretokens-api.exe"
    $result = Invoke-NativeExecutor $executor @("init", "--host", $RequestedHost)
    $initOutput = @($result.Output)
    $json = $null
    if ($initOutput.Count -gt 0) {
      try { $json = ($initOutput -join "`n") | ConvertFrom-Json } catch { $json = $null }
    }
    if ($result.ExitCode -eq 0 -and $null -ne $json -and $json.configuration_status -eq "verified" -and
        $json.api_identity_confirmed -is [bool] -and $json.api_identity_confirmed -eq $true -and
        $json.credential_verified -is [bool] -and $json.credential_verified -eq $true) {
      Write-Output "Pure Tokens Skill init: fixed API identity verified for the current host."
      Write-Output "init_status=verified"
    } elseif ($null -ne $json) {
      $state = if ([string]::IsNullOrWhiteSpace($json.configuration_status)) { "unverified" } else { $json.configuration_status }
      $message = if ([string]::IsNullOrWhiteSpace($json.message)) { "The fixed API identity is not verified in this host session." } else { $json.message }
      if ($state -eq "verified") {
        $state = "unverified"
        $message = "The verification receipt did not confirm both API identity and credential authentication."
      }
      Write-Output "Pure Tokens Skill init: $message [$state]"
      Write-Output "init_status=unverified"
      if ($null -ne $json.http_status -and [int]$json.http_status -gt 0) {
        Write-Output "HTTP status: $($json.http_status)"
      }
      if (-not [string]::IsNullOrWhiteSpace($json.next_action)) {
        Write-Output "Next: $($json.next_action)"
      }
    } else {
      Write-Output "Pure Tokens Skill init: configuration could not be checked in this host session."
      Write-Output "init_status=unverified stage=init error_code=executor_output_unavailable next_step=review_host_execution_permissions"
    }
  }
  } catch {
    # Installation is already committed. Never throw an init-only failure into
    # transaction recovery or relabel successfully synchronized files as failed.
    Write-Output "Pure Tokens Skill init: configuration could not be checked in this host session."
    Write-Output "init_status=unverified stage=init error_code=executor_unavailable next_step=review_host_execution_permissions; check whether this session permits the installed native executor and file checks. The exact cause is unconfirmed; Full Access is not a default requirement."
  }
  Write-Output ""
  Show-InstallationSummary $RequestedHost
}

function Get-TargetForHost([string]$RequestedHost) {
  if ([string]::IsNullOrWhiteSpace($env:USERPROFILE)) { Fail "cannot resolve a host Skill directory because USERPROFILE is unavailable" }
  switch ($RequestedHost) {
    "claude-code" { if ($env:CLAUDE_CONFIG_DIR) { return (Join-Path $env:CLAUDE_CONFIG_DIR "skills") }; return (Join-Path $env:USERPROFILE ".claude\skills") }
    "claude-desktop" { if ($env:CLAUDE_CONFIG_DIR) { return (Join-Path $env:CLAUDE_CONFIG_DIR "skills") }; return (Join-Path $env:USERPROFILE ".claude\skills") }
    "kimi-code" {
      if ($env:KIMI_CODE_HOME) { return (Join-Path $env:KIMI_CODE_HOME "skills") }
      return (Join-Path $env:USERPROFILE ".kimi-code\skills")
    }
    "pi" {
      $piRoot = if ($env:PI_CODING_AGENT_DIR) { $env:PI_CODING_AGENT_DIR } else { Join-Path $env:USERPROFILE ".pi\agent" }
      if ($piRoot -notmatch '^(?:[A-Za-z]:[\\/]|[\\/]{2}[^\\/]+[\\/][^\\/]+(?:[\\/]|$))') { Fail "Pi agent directory must be absolute" }
      if (($piRoot -split '[\\/]') -contains '..') { Fail "Pi agent directory must not contain parent traversal" }
      return (Join-Path $piRoot "skills")
    }
    "hermes" {
      $root = $env:HERMES_HOME
      if (-not $root) {
        if (-not $env:LOCALAPPDATA) { Fail "Hermes local data directory unavailable" }
        $root = Join-Path $env:LOCALAPPDATA "hermes"
        $legacy = Join-Path $env:USERPROFILE ".hermes"
        if (-not (Test-Path -LiteralPath $root) -and (Test-Path -LiteralPath $legacy -PathType Container)) { $root = $legacy }
      }
      if ($root -notmatch '^[A-Za-z]:[\\/]' -or ($root -split '[\\/]') -contains '..' -or $root -match '[\x00-\x1f\x7f]') { Fail "Hermes directory must be local and absolute without parent traversal" }
      return (Join-Path $root "skills")
    }
    "evox" {
      if ($env:EVOX_AGENT_DIR -and $env:EVOX_CODING_AGENT_DIR -and $env:EVOX_AGENT_DIR -cne $env:EVOX_CODING_AGENT_DIR) { Fail "EvoX directory overrides disagree" }
      $root = if ($env:EVOX_AGENT_DIR) { $env:EVOX_AGENT_DIR } elseif ($env:EVOX_CODING_AGENT_DIR) { $env:EVOX_CODING_AGENT_DIR } else { Join-Path $env:USERPROFILE ".evox\agent" }
      if ($root -notmatch '^[A-Za-z]:[\\/]' -or ($root -split '[\\/]') -contains '..' -or $root -match '[\x00-\x1f\x7f]') { Fail "EvoX directory must be local and absolute without parent traversal" }
      return (Join-Path $root "skills")
    }
    "vscode" { return (Join-Path $env:USERPROFILE ".copilot\skills") }
    "octop" { Fail "Octop requires -Target with the current local workspace's absolute .octop/skills directory; no global directory is assumed" }
    "qoder" {
      if ($env:QODER_CONFIG_DIR) { return (Join-Path $env:QODER_CONFIG_DIR "skills") }
      $qoderName = ".qoder"
      if ($env:QODER_CONFIG_DIR_NAME) { $qoderName = $env:QODER_CONFIG_DIR_NAME }
      if ($qoderName -eq "." -or $qoderName -eq ".." -or $qoderName.Contains("/") -or $qoderName.Contains("\")) { Fail "invalid Qoder directory name" }
      $qoderBase = $env:USERPROFILE
      if ($env:QODER_CLI_HOME) { $qoderBase = $env:QODER_CLI_HOME }
      return (Join-Path (Join-Path $qoderBase $qoderName) "skills")
    }
    "zcode" {
      if ($env:ZCODE_DATA_BASE_DIR) {
        if (-not [System.IO.Path]::IsPathRooted($env:ZCODE_DATA_BASE_DIR)) { Fail "ZCode data directory must be absolute" }
        return (Join-Path $env:ZCODE_DATA_BASE_DIR ".zcode\skills")
      }
      return (Join-Path $env:USERPROFILE ".zcode\skills")
    }
    "minimax-code" {
      foreach ($override in @("MAVIS_PROFILE", "MINIMAX_PROFILE", "AGENTARCHON_PROFILE", "AGENTARCHON_DATA_DIR", "__MAVIS_RUNTIME_PROFILE", "__MAVIS_RUNTIME_DATA_DIR")) {
        if (-not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($override))) { Fail "MiniMax CLI and runtime profile overrides are unsupported" }
      }
      $root = if ($env:MINIMAX_DATA_DIR) { $env:MINIMAX_DATA_DIR } else { $env:MAVIS_DATA_DIR }
      if ([string]::IsNullOrWhiteSpace($root)) {
        if (-not $env:APPDATA -or -not [IO.Path]::IsPathRooted($env:APPDATA)) { Fail "MiniMax Desktop application directory is unavailable" }
        foreach ($app in @("MiniMax Code", "MiniMax", "MiniMax Agent")) {
          $appRoot = Join-Path $env:APPDATA $app
          foreach ($store in @("minimax-agent-cn-config.json", "minimax-agent-config.json")) {
            if ((Test-Path -LiteralPath (Join-Path $appRoot $store)) -or ((Test-Path -LiteralPath $appRoot) -and ((Get-Item -LiteralPath $appRoot).Attributes -band [IO.FileAttributes]::ReparsePoint))) {
              Fail "MiniMax Desktop preferences require its runtime data directory; run inside the local client with MINIMAX_DATA_DIR or pass the confirmed absolute -Target; no directory was guessed"
            }
          }
        }
        $root = Join-Path $env:USERPROFILE ".minimax"
        if (-not (Test-Path -LiteralPath (Join-Path $root "config.yaml")) -and (Test-Path -LiteralPath (Join-Path $env:USERPROFILE ".mavis"))) { Fail "MiniMax legacy migration is pending; let the client migrate before installing" }
      }
      if ($root -notmatch '^[A-Za-z]:[\\/]' -or ($root -split '[\\/]') -contains '..' -or $root -match '[\x00-\x1f\x7f]') { Fail "MiniMax directory must be local and absolute without parent traversal" }
      return (Join-Path $root "skills")
    }
    "deepseek-harness" {
      $root = if ($env:DSH_HOME) { $env:DSH_HOME } else { Join-Path $env:USERPROFILE ".dsh" }
      if ($root -notmatch '^[A-Za-z]:[\\/]' -or ($root -split '[\\/]') -contains '..' -or $root -match '[\x00-\x1f\x7f]') { Fail "DeepSeek Harness directory must be local and absolute without parent traversal" }
      return (Join-Path $root "skills")
    }
    "dsh-desktop" {
      if ($env:DSH_HOME) { return (Join-Path $env:DSH_HOME "skills") }
      if (-not $env:APPDATA -or -not [System.IO.Path]::IsPathRooted($env:APPDATA)) { throw "DSH Desktop application directory is unavailable" }
      return (Join-Path $env:APPDATA "dsh-desktop\harness\skills")
    }
    "codex" { return (Join-Path $env:USERPROFILE ".agents\skills") }
    "workbuddy" { if ($env:WORKBUDDY_CONFIG_DIR) { return (Join-Path $env:WORKBUDDY_CONFIG_DIR "skills") }; if ($env:CODEBUDDY_CONFIG_DIR) { return (Join-Path $env:CODEBUDDY_CONFIG_DIR "skills") }; return (Join-Path $env:USERPROFILE ".workbuddy\skills") }
    "gemini-cli" {
      $shared = Join-Path $env:USERPROFILE ".agents\skills"
      foreach ($skill in $currentSkills) { if (Test-Path -LiteralPath (Join-Path $shared $skill)) { return $shared } }
      return (Join-Path $env:USERPROFILE ".gemini\skills")
    }
    "grok-build" { return (Join-Path $env:USERPROFILE ".grok\skills") }
    "opencode" {
      $xdg = if ($env:XDG_CONFIG_HOME) { $env:XDG_CONFIG_HOME } else { Join-Path $env:USERPROFILE ".config" }
      $config = if ($env:OPENCODE_CONFIG_DIR) { $env:OPENCODE_CONFIG_DIR } else { Join-Path $xdg "opencode" }
      foreach ($directory in @($xdg, $config)) {
        if ($directory -notmatch '^(?:[A-Za-z]:[\\/]|[\\/]{2}[^\\/]+[\\/][^\\/]+)' -or $directory -match '(^|[\\/])\.\.([\\/]|$)' -or $directory -match '[\x00-\x1f\x7f]') {
          Fail "OpenCode config directories must be absolute without parent traversal"
        }
      }
      return (Join-Path $config "skills")
    }
    "trae" { return (Join-Path $env:USERPROFILE ".trae\skills") }
    default { Fail "unsupported host: $RequestedHost" }
  }
}

function Test-GeminiDuplicates([string]$TargetRoot) {
  if ($HostId -ne "gemini-cli") { return }
  $shared = [IO.Path]::GetFullPath((Join-Path $env:USERPROFILE ".agents\skills")).TrimEnd('\')
  $isShared = [string]::Equals($TargetRoot.TrimEnd('\'), $shared, [StringComparison]::OrdinalIgnoreCase)
  foreach ($skill in $currentSkills) {
    if (-not $isShared -and (Test-Path -LiteralPath (Join-Path $shared $skill))) {
      Fail "Gemini would load $skill from its higher-priority .agents/skills directory. Run sync -Host gemini-cli without a custom target; existing directories were left untouched"
    }
    if ($isShared -and (Test-ManagedSkill (Join-Path (Join-Path $env:USERPROFILE ".gemini\skills") $skill) $skill)) {
      Write-Output "Managed duplicate detected: Gemini will use the updated shared $skill; its lower-priority .gemini copy was left untouched."
    }
  }
}

function Restore-Transaction([string]$TargetRoot, [string]$StageRoot) {
  $verified = Invoke-NativeExecutor $script:ownershipGuard @("install-transaction-verify", "--directory", $StageRoot, "--target-directory", $TargetRoot, "--source-directory", $sourceRoot)
  if ($verified.ExitCode -ne 0) { Fail "transaction recovery failed verification; all remaining files were preserved" }
  $planFile = Join-Path $StageRoot "plan.json"
  # Assign first: PowerShell 5.1 emits the JSON array as one pipeline object,
  # while newer versions enumerate it. foreach must visit entries in both.
  $recoveryPlan = Get-Content -LiteralPath $planFile -Raw -Encoding UTF8 | ConvertFrom-Json
  foreach ($entry in $recoveryPlan) {
    if ($entry.name -notin ($currentSkills + @(".puretokens-executor"))) { Fail "unknown recovery entry; retained backup" }
    $destination = Join-Path $TargetRoot $entry.name
    $backup = Join-Path (Join-Path $StageRoot "backup") $entry.name
    if ($entry.action -eq "replace" -and (Test-Path -LiteralPath $backup)) {
      if (Test-InstallationEntry $destination) {
        Assert-OwnedInstallation $destination $entry.name (Join-Path (Join-Path $sourceRoot "skills") $entry.name)
        Remove-Item -LiteralPath $destination -Recurse -Force
      }
      Move-Item -LiteralPath $backup -Destination $destination
    } elseif ($entry.action -eq "create" -and -not (Test-Path -LiteralPath (Join-Path $StageRoot $entry.name))) {
      if (Test-InstallationEntry $destination) {
        Assert-OwnedInstallation $destination $entry.name (Join-Path (Join-Path $sourceRoot "skills") $entry.name)
        Remove-Item -LiteralPath $destination -Recurse -Force
      }
    }
  }
}


$stageRoot = $null
$updateLock = $null
try {
  $script:installStage = "select_target"
  if ([string]::IsNullOrWhiteSpace($Target) -and [string]::IsNullOrWhiteSpace($HostId)) { Fail "-Host or -Target is required" }
  if ($HostId -in @("pi", "hermes", "evox")) { $null = Get-TargetForHost $HostId }
  if ([string]::IsNullOrWhiteSpace($Target)) { $Target = Get-TargetForHost $HostId }
  if ($HostId -eq "octop" -and ($Target -notmatch '^[A-Za-z]:[\\/]' -or ($Target -split '[\\/]') -contains '..' -or $Target -match '[\x00-\x1f\x7f]' -or $Target -notmatch '[\\/]skills$')) {
    Fail "Octop target must be the current local workspace's absolute .octop/skills or skills directory"
  }
  if (-not [System.IO.Path]::IsPathRooted($Target)) { Fail "-Target must be an absolute Skill directory" }
  if ($ReleaseManifest -and ($Command -ne "verify-installed" -or -not (Test-Path -LiteralPath $ReleaseManifest -PathType Leaf))) { Fail "release manifest is only supported for installed verification" }
  if ($Command -eq "locate") { Write-Output $Target; exit 0 }

  if ($Command -eq "verify-installed") {
    $script:installStage = "verify_installed"
    Confirm-InstalledInventories $Target
    exit 0
  }

  if ($Command -eq "init") {
    $script:installStage = "init"
    $targetRoot = (Resolve-Path -LiteralPath $Target).Path
    Invoke-Init $targetRoot $HostId
    exit 0
  }

  $script:installStage = "verify_source"
  if (-not [string]::IsNullOrWhiteSpace($Source)) {
    if (-not [System.IO.Path]::IsPathRooted($Source)) { Fail "-Source must be an absolute official source directory" }
    if (-not (Test-Path -LiteralPath $Source -PathType Container)) { Fail "-Source does not exist" }
    $sourceRoot = (Resolve-Path -LiteralPath $Source).Path
  } else {
    $bundledSourceRoot = Split-Path -Parent $PSScriptRoot
    if ((Test-Path -LiteralPath (Join-Path $bundledSourceRoot "README.md") -PathType Leaf) -and (Test-Path -LiteralPath (Join-Path $bundledSourceRoot "package.json") -PathType Leaf) -and (Test-Path -LiteralPath (Join-Path $bundledSourceRoot "skills") -PathType Container)) {
      $sourceRoot = $bundledSourceRoot
    } else { Fail "run this source-only sync script from a fresh official Pure Tokens Skills main checkout or pass -Source <absolute-checkout-directory>" }
  }
  Test-OfficialSource $sourceRoot
  if ($Command -eq "check") { Write-Output "Official Pure Tokens Skill source passed native static validation."; exit 0 }
  $releaseVersion = Get-SourceVersion $sourceRoot

  $script:installationStatus = "not_completed"
  $script:installStage = "prepare_target"
  New-Item -ItemType Directory -Path $Target -Force | Out-Null
  $targetRoot = (Resolve-Path -LiteralPath $Target).Path
  Test-GeminiDuplicates $targetRoot
  $script:ownershipGuard = Get-ExecutorArtifact $sourceRoot
  if ($NoRecovery -and @(Get-ChildItem -LiteralPath $targetRoot -Force -Filter ".puretokens-skill-stage-*").Count -gt 0) { Fail "installation transaction requires review; existing files were preserved" }
  $script:installStage = "acquire_lock"
  try { $updateLock = [System.IO.File]::Open((Join-Path $targetRoot ".puretokens-install.lock"), [System.IO.FileMode]::OpenOrCreate, [System.IO.FileAccess]::ReadWrite, [System.IO.FileShare]::None) } catch { Fail "another update is in progress or the update lock is not writable; check directory access and other installations" }
  $script:installStage = "recover_transaction"
  if ($Command -eq "repair") {
    foreach ($name in ($currentSkills + @(".puretokens-executor"))) {
      $destination = Join-Path $targetRoot $name
      if (Test-InstallationEntry $destination) { Assert-OwnedInstallation $destination $name }
    }
  }
  foreach ($previous in @(Get-ChildItem -LiteralPath $targetRoot -Force | Where-Object { $_.Name -like ".puretokens-skill-stage-*" })) {
    if (-not $previous.PSIsContainer -or ($previous.Attributes -band [IO.FileAttributes]::ReparsePoint)) { Fail "unknown staging entry; left untouched" }
    if (-not (Test-Path -LiteralPath (Join-Path $previous.FullName "transaction-v1"))) { Fail "unknown staging directory; left untouched" }
    $verified = Invoke-NativeExecutor $script:ownershipGuard @("install-transaction-verify", "--directory", $previous.FullName, "--target-directory", $targetRoot, "--source-directory", $sourceRoot)
    if ($verified.ExitCode -ne 0) { Fail "previous update recovery failed verification; all transaction files were preserved" }
  }
  foreach ($previous in @(Get-ChildItem -LiteralPath $targetRoot -Directory -Force | Where-Object { $_.Name -like ".puretokens-skill-stage-*" })) {
    if (-not (Test-Path -LiteralPath (Join-Path $previous.FullName "committed"))) {
      $script:installedFilesChanged = "unknown"
      Restore-Transaction $targetRoot $previous.FullName
    }
    Remove-CompletedStage $previous.FullName
  }
  # Recheck after obtaining the installation lock: another sync may have
  # completed while this version's archive was still downloading.
  $script:installStage = "verify_installed"
  if (Test-ManagedExecutor (Join-Path $targetRoot ".puretokens-executor")) {
    $installedVersion = (Get-Content -LiteralPath (Join-Path $targetRoot ".puretokens-executor/runtime.json") -Raw -Encoding UTF8 | ConvertFrom-Json).version
    if ($installedVersion -notmatch '^\d+\.\d+\.\d+$') { Fail "installed executor version is invalid; left untouched" }
    if ([version]$installedVersion -gt [version]$releaseVersion) { Fail "a newer executor version is already installed; downgrade was stopped under the update lock" }
  }
  foreach ($name in $currentSkills) {
    $destination = Join-Path $targetRoot $name
    if (Test-InstallationEntry $destination) { Assert-OwnedInstallation $destination $name (Join-Path (Join-Path $sourceRoot "skills") $name) }
  }
  $executorDestination = Join-Path $targetRoot ".puretokens-executor"
  if (Test-InstallationEntry $executorDestination) { Assert-OwnedInstallation $executorDestination ".puretokens-executor" }

  $script:installStage = "stage_files"
  $stageRoot = Join-Path $targetRoot (".puretokens-skill-stage-" + [Guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path (Join-Path $stageRoot "backup") -Force | Out-Null
  New-Item -ItemType File -Path (Join-Path $stageRoot "transaction-v1") | Out-Null
  if ($Command -eq "repair") { New-Item -ItemType File -Path (Join-Path $stageRoot "repair-v1") | Out-Null }
  foreach ($name in $currentSkills) { Copy-Item -LiteralPath (Join-Path (Join-Path $sourceRoot "skills") $name) -Destination $stageRoot -Recurse }
  $executorSource = Get-ExecutorArtifact $sourceRoot
  $stageExecutor = Join-Path $stageRoot ".puretokens-executor"
  New-Item -ItemType Directory -Path $stageExecutor -Force | Out-Null
  Copy-Item -LiteralPath $executorSource -Destination (Join-Path $stageExecutor "puretokens-api.exe")
  foreach ($script in @("puretokens-skill-install.ps1", "puretokens-skill-fetch.ps1")) {
    Copy-Item -LiteralPath (Join-Path (Join-Path $sourceRoot "runtime") $script) -Destination $stageExecutor
  }
  [PSCustomObject]@{ schemaVersion = 1; name = "puretokens-api-executor"; version = $releaseVersion; platform = (Get-ExecutorPlatform) } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $stageExecutor "runtime.json") -Encoding utf8
  foreach ($name in ($currentSkills + @(".puretokens-executor"))) {
    $directory = Join-Path $stageRoot $name
    $result = Invoke-NativeExecutor $script:ownershipGuard @("install-inventory", "--directory", $directory, "--name", $name)
    $inventory = $result.Output
    if ($result.ExitCode -ne 0) { Fail "could not record managed files for $name" }
    $inventory | Set-Content -LiteralPath (Join-Path $directory ".puretokens-managed.json") -Encoding UTF8
  }

  $plan = @()
  foreach ($name in ($currentSkills + @(".puretokens-executor"))) {
    $action = if (Test-InstallationEntry (Join-Path $targetRoot $name)) { "replace" } else { "create" }
    $plan += [PSCustomObject]@{ name = $name; action = $action }
  }
  $plan | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $stageRoot "plan.json") -Encoding utf8
  $script:installStage = "replace_files"
  foreach ($entry in $plan) {
    $destination = Join-Path $targetRoot $entry.name
    if ($entry.action -eq "replace") {
      Assert-OwnedInstallation $destination $entry.name (Join-Path (Join-Path $sourceRoot "skills") $entry.name)
      $script:installedFilesChanged = "unknown"
      Move-Item -LiteralPath $destination -Destination (Join-Path (Join-Path $stageRoot "backup") $entry.name)
    } elseif (Test-InstallationEntry $destination) { Fail "a destination appeared during installation; existing files were preserved" }
    $script:installedFilesChanged = "unknown"
    Move-Item -LiteralPath (Join-Path $stageRoot $entry.name) -Destination $destination
  }
  # Require the inventory itself, even if install-verify could adopt an exact source copy.
  $script:installStage = "verify_synced"
  foreach ($name in ($currentSkills + @(".puretokens-executor"))) {
    $installed = Join-Path $targetRoot $name
    if (-not (Test-Path -LiteralPath (Join-Path $installed ".puretokens-managed.json") -PathType Leaf)) { Fail "managed inventory missing; installation is incomplete" }
    $result = Invoke-NativeExecutor $script:ownershipGuard @("install-verify", "--directory", $installed, "--name", $name)
    if ($result.ExitCode -ne 0) { Fail "installed managed files could not be verified" }
  }
  New-Item -ItemType File -Path (Join-Path $stageRoot "committed") | Out-Null
  $script:installationStatus = "completed"
  $script:installedFilesChanged = "true"
  Write-Output "Pure Tokens Skills $releaseVersion synchronized with the native API executor at $targetRoot"
  Write-Output "installation_status=completed installed_files_changed=true"
  Remove-CompletedStage $stageRoot
  $stageRoot = $null
  $updateLock.Dispose(); $updateLock = $null
  if ($env:PTP_OPERATIONS_RECEIPTS -eq "1") {
    try { $null = Invoke-NativeExecutor (Join-Path (Join-Path $targetRoot ".puretokens-executor") "puretokens-api.exe") @("operations-receipt", "installed") } catch { }
  }
  $script:installStage = "init"
  Invoke-Init $targetRoot $HostId
} catch {
  if ($_.Exception.Data["puretokens_receipt"]) { throw }
  $category = "local_operation_failed"
  $next = "review_local_access"
  if ($_.Exception -is [System.UnauthorizedAccessException] -or $_.CategoryInfo.Category -eq "PermissionDenied") {
    $category = "local_access_denied"
    $next = "review_host_permissions"
  } elseif ($_.Exception -is [System.IO.IOException]) {
    $category = "local_io_failure"
    $next = "review_install_directory_access"
  }
  $exception = New-Object System.InvalidOperationException("Pure Tokens Skill installer: stage=$script:installStage error_code=$category installation_status=$script:installationStatus installed_files_changed=$script:installedFilesChanged next_step=$next; a local operation could not complete. Check current-session directory/command permissions, file locks and available disk space; the exact cause is unconfirmed. No automatic retry.")
  $exception.Data["puretokens_receipt"] = $true
  throw $exception
} finally {
  try {
    if ($null -ne $stageRoot -and (Test-Path -LiteralPath $stageRoot)) {
      if (-not (Test-Path -LiteralPath (Join-Path $stageRoot "committed"))) {
        Restore-Transaction $targetRoot $stageRoot
        $script:installedFilesChanged = "false"
        Write-Output "recovery_status=restored installation_status=not_completed installed_files_changed=false; the previous managed files were restored."
      }
      Remove-CompletedStage $stageRoot
    }
  } catch {
    Write-Output "recovery_status=incomplete installation_status=$script:installationStatus installed_files_changed=$script:installedFilesChanged next_step=inspect_installation; retained the managed transaction. Do not reinstall or retry automatically."
  } finally { if ($null -ne $updateLock) { $updateLock.Dispose() } }
}
