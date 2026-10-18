$ErrorActionPreference = 'Stop'
$repository = Split-Path -Parent $PSScriptRoot
$root = Join-Path ([IO.Path]::GetTempPath()) ('pt-installer-test-' + [Guid]::NewGuid().ToString('N'))
$target = Join-Path $root 'skills'
$installer = Join-Path $repository 'runtime/puretokens-skill-install.ps1'
$savedEnvironment = @{}
foreach ($name in @('USERPROFILE', 'HOME', 'CODEX_HOME', 'APPDATA', 'LOCALAPPDATA', 'HERMES_HOME', 'EVOX_AGENT_DIR', 'EVOX_CODING_AGENT_DIR', 'OCTOP_HOME', 'DSH_HOME', 'CLAUDE_CONFIG_DIR', 'ZCODE_DATA_BASE_DIR', 'KIMI_CODE_HOME', 'QODER_CONFIG_DIR', 'QODER_CLI_HOME', 'QODER_CONFIG_DIR_NAME', 'PI_CODING_AGENT_DIR', 'XDG_CONFIG_HOME', 'OPENCODE_CONFIG_DIR', 'MINIMAX_DATA_DIR', 'MAVIS_DATA_DIR', 'MAVIS_PROFILE', 'MINIMAX_PROFILE', 'AGENTARCHON_PROFILE', 'AGENTARCHON_DATA_DIR', '__MAVIS_RUNTIME_PROFILE', '__MAVIS_RUNTIME_DATA_DIR')) { $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
try {
  New-Item -ItemType Directory $root | Out-Null
  $env:USERPROFILE = Join-Path $root 'home'
  $env:HOME = $env:USERPROFILE
  $env:CODEX_HOME = Join-Path $env:USERPROFILE '.codex'
  $env:LOCALAPPDATA = Join-Path $env:USERPROFILE 'local'
  foreach ($variable in @('MINIMAX_DATA_DIR', 'MAVIS_DATA_DIR', 'MAVIS_PROFILE', 'MINIMAX_PROFILE', 'AGENTARCHON_PROFILE', 'AGENTARCHON_DATA_DIR', '__MAVIS_RUNTIME_PROFILE', '__MAVIS_RUNTIME_DATA_DIR')) { [Environment]::SetEnvironmentVariable($variable, '', 'Process') }
  $env:HERMES_HOME = ''
  $env:EVOX_AGENT_DIR = ''
  $env:EVOX_CODING_AGENT_DIR = ''
  $env:OCTOP_HOME = Join-Path $env:USERPROFILE '.octop'
  New-Item -ItemType Directory $env:USERPROFILE | Out-Null
  $engines = @('powershell.exe', 'pwsh')
  $x86PowerShell = Join-Path $env:SystemRoot 'SysWOW64\WindowsPowerShell\v1.0\powershell.exe'
  if (Test-Path -LiteralPath $x86PowerShell) { $engines += 'powershell-x86' }
  foreach ($engine in $engines) {
    $command = if ($engine -eq 'powershell-x86') { [PSCustomObject]@{ Source = $x86PowerShell } } else { Get-Command $engine -ErrorAction Stop }
    if ($engine -eq 'powershell-x86') {
      $actualProcess = & $command.Source -NoProfile -Command '[Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()'
      if ($LASTEXITCODE -ne 0 -or $actualProcess -ne 'X86') { throw "SysWOW64 fixture did not execute a 32-bit PowerShell process" }
    }
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repository 'test/support/installer-local-failures.ps1')
    if ($LASTEXITCODE -ne 0) { throw "$engine local failure regression" }
    foreach ($newHost in @('kimi-code', 'qoder', 'pi', 'hermes', 'evox', 'deepseek-harness', 'minimax-code')) {
      $variable = switch ($newHost) { 'kimi-code' { 'KIMI_CODE_HOME' }; 'qoder' { 'QODER_CONFIG_DIR' }; 'pi' { 'PI_CODING_AGENT_DIR' }; 'hermes' { 'HERMES_HOME' }; 'evox' { 'EVOX_AGENT_DIR' }; 'deepseek-harness' { 'DSH_HOME' }; 'minimax-code' { 'MINIMAX_DATA_DIR' } }
      $hostRoot = Join-Path $root ("$newHost $engine " + [char]0x7528 + [char]0x6237 + " spaces")
      [Environment]::SetEnvironmentVariable($variable, $hostRoot, 'Process')
      $location = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId $newHost
      if ($LASTEXITCODE -ne 0 -or $location -ne (Join-Path $hostRoot 'skills')) { throw "$engine new host directory mismatch" }
      for ($attempt=0; $attempt -lt 2; $attempt++) {
        & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -HostId $newHost
        if ($LASTEXITCODE -ne 0) { throw "$engine new host install/update failed" }
      }
      [Environment]::SetEnvironmentVariable($variable, 'relative', 'Process')
      & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId $newHost *> $null
      if ($LASTEXITCODE -eq 0) { throw "$engine accepted relative new host directory" }
      [Environment]::SetEnvironmentVariable($variable, '', 'Process')
    }
    $location = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId hermes
    if ($LASTEXITCODE -ne 0 -or $location -ne (Join-Path $env:LOCALAPPDATA 'hermes\skills')) { throw "$engine Hermes default mismatch" }
    $location = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId vscode
    if ($LASTEXITCODE -ne 0 -or $location -ne (Join-Path $env:USERPROFILE '.copilot\skills')) { throw "$engine VS Code default mismatch" }
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -HostId vscode
    if ($LASTEXITCODE -ne 0) { throw "$engine VS Code installation failed" }
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId octop *> $null
    if ($LASTEXITCODE -eq 0) { throw "$engine guessed an Octop global directory" }
    $octopTarget = Join-Path $root "$engine-workspace\.octop\skills"
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -HostId octop -Target $octopTarget
    if ($LASTEXITCODE -ne 0) { throw "$engine Octop workspace installation failed" }
    $piLocation = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId pi
    if ($LASTEXITCODE -ne 0 -or $piLocation -ne (Join-Path $env:USERPROFILE '.pi\agent\skills')) { throw "$engine Pi default directory mismatch" }
    foreach ($invalid in @('relative', 'C:relative', '\rooted', (Join-Path $root 'invalid\..\unexpected'))) {
      $env:PI_CODING_AGENT_DIR = $invalid
      & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -HostId pi *> $null
      if ($LASTEXITCODE -eq 0) { throw "$engine accepted an invalid Pi directory" }
    }
    if ((Test-Path -LiteralPath (Join-Path $root 'invalid')) -or (Test-Path -LiteralPath (Join-Path $root 'unexpected'))) { throw "$engine wrote through an invalid Pi root" }
    $env:PI_CODING_AGENT_DIR = ''
    $env:XDG_CONFIG_HOME = Join-Path $root 'xdg config'
    $env:OPENCODE_CONFIG_DIR = ''
    $location = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId opencode
    if ($LASTEXITCODE -ne 0 -or $location -ne (Join-Path $env:XDG_CONFIG_HOME 'opencode\skills')) { throw "$engine OpenCode XDG directory mismatch" }
    $env:OPENCODE_CONFIG_DIR = Join-Path $root 'opencode custom'
    $location = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId opencode
    if ($LASTEXITCODE -ne 0 -or $location -ne (Join-Path $env:OPENCODE_CONFIG_DIR 'skills')) { throw "$engine OpenCode custom directory mismatch" }
    foreach ($invalid in @('relative', 'C:relative', '\rooted', (Join-Path $root 'invalid\..\unexpected'))) {
      $env:OPENCODE_CONFIG_DIR = $invalid
      & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId opencode *> $null
      if ($LASTEXITCODE -eq 0) { throw "$engine accepted an invalid OpenCode directory" }
    }
    $env:XDG_CONFIG_HOME = ''
    $env:OPENCODE_CONFIG_DIR = ''
    $env:APPDATA = Join-Path $root "Roaming with spaces"
    $env:DSH_HOME = ""
    $env:CLAUDE_CONFIG_DIR = ""
    $dshLocation = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId dsh-desktop
    if ($LASTEXITCODE -ne 0 -or $dshLocation -ne (Join-Path $env:APPDATA "dsh-desktop\harness\skills")) { throw "$engine DSH default directory mismatch" }
    $claudeLocation = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId claude-desktop
    if ($LASTEXITCODE -ne 0 -or $claudeLocation -ne (Join-Path $env:USERPROFILE ".claude\skills")) { throw "$engine Claude Desktop directory mismatch" }
    $env:DSH_HOME = Join-Path $root "Harness custom"
    $overrideLocation = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId dsh-desktop
    if ($LASTEXITCODE -ne 0 -or $overrideLocation -ne (Join-Path $env:DSH_HOME "skills")) { throw "$engine DSH override mismatch" }
    $env:DSH_HOME = "relative"
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId dsh-desktop *> $null
    if ($LASTEXITCODE -eq 0) { throw "$engine accepted a relative Harness directory" }
    $env:DSH_HOME = ""
    $env:ZCODE_DATA_BASE_DIR = Join-Path $root "ZCode data"
    $zcodeLocation = & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId zcode
    if ($LASTEXITCODE -ne 0 -or $zcodeLocation -ne (Join-Path $env:ZCODE_DATA_BASE_DIR ".zcode\skills")) { throw "$engine ZCode override mismatch" }
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -HostId zcode
    if ($LASTEXITCODE -ne 0) { throw "$engine ZCode installation failed" }
    $env:ZCODE_DATA_BASE_DIR = "relative"
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer locate -HostId zcode *> $null
    if ($LASTEXITCODE -eq 0) { throw "$engine accepted a relative ZCode directory" }
    $env:ZCODE_DATA_BASE_DIR = ""
    $engineTarget = Join-Path $root $engine
    # Exercise the exact public -File entrance in both Windows PowerShell 5.1
    # and PowerShell 7. Bypass applies to this child process only.
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -Target $engineTarget
    if ($LASTEXITCODE -ne 0) { throw "$engine advertised installer entry failed" }
    $installedRuntime = Get-Content -LiteralPath (Join-Path $engineTarget '.puretokens-executor/runtime.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    $expectedPlatform = if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -eq 'Arm64') { 'windows-arm64' } else { 'windows-amd64' }
    if ($installedRuntime.platform -ne $expectedPlatform) { throw "$engine selected a process architecture instead of the operating system architecture" }
    # Simulate a host deletion guard without bypassing it: successful sync and
    # init must remain observable, the stage is retained, and the lock released.
    $guardRunner = Join-Path $root "guard-$engine.ps1"
    @'
param($Installer, $Target)
$ErrorActionPreference = 'Stop'
function Remove-Item {
  [CmdletBinding(SupportsShouldProcess=$true)]
  param([string]$LiteralPath, [switch]$Recurse, [switch]$Force)
  if ($LiteralPath -like '*.puretokens-skill-stage-*') { throw 'synthetic host cleanup denied' }
  Microsoft.PowerShell.Management\Remove-Item @PSBoundParameters
}
& $Installer sync -Target $Target
'@ | Set-Content -LiteralPath $guardRunner -Encoding ASCII
    $guardTarget = Join-Path $root "guard-target-$engine"
    $guardOutput = @(& $command.Source -NoProfile -ExecutionPolicy Bypass -File $guardRunner -Installer $installer -Target $guardTarget)
    if ($LASTEXITCODE -ne 0) { throw "$engine cleanup denial failed the completed sync" }
    $guardText = $guardOutput -join "`n"
    if ($guardText -notmatch 'synchronized with the native API executor' -or $guardText -notmatch 'cleanup_status: pending' -or $guardText -notmatch 'connection check was deferred') { throw "$engine lost sync, cleanup or init status" }
    foreach ($name in @('puretokens-balance','puretokens-connection','puretokens-models','puretokens-image','puretokens-video','puretokens-audio','puretokens-evaluate','puretokens-embedding','puretokens-rerank','puretokens-update','.puretokens-executor')) {
      if (-not (Test-Path (Join-Path (Join-Path $guardTarget $name) '.puretokens-managed.json'))) { throw "$engine missing installed inventory" }
    }
    if (@(Get-ChildItem $guardTarget -Directory -Force -Filter '.puretokens-skill-stage-*').Count -ne 1) { throw "$engine did not retain denied cleanup stage" }
    $released = [IO.File]::Open((Join-Path $guardTarget '.puretokens-install.lock'), 'OpenOrCreate', 'ReadWrite', 'None')
    $released.Dispose()
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -Target $guardTarget
    if ($LASTEXITCODE -ne 0 -or @(Get-ChildItem $guardTarget -Directory -Force -Filter '.puretokens-skill-stage-*').Count -ne 0) { throw "$engine could not recover retained completed stage" }
    $launchProbe = Join-Path $root "launcher-$engine.ps1"
    @'
param($Installer, $Target)
$ErrorActionPreference = 'Stop'
$tokens = $null; $parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($Installer, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw 'installer parse failed' }
$function = $ast.Find({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Invoke-NativeExecutor' }, $true)
Invoke-Expression $function.Extent.Text
function Fail([string]$Message) { throw $Message }
$binary = Join-Path $Target '.puretokens-executor/puretokens-api.exe'
$ok = Invoke-NativeExecutor $binary @('--version')
if ($ok.ExitCode -ne 0 -or $ok.Output.Trim() -notmatch '^\d+\.\d+\.\d+$') { throw 'launcher lost stdout or exit code' }
$bad = Invoke-NativeExecutor $binary @('unsupported-command')
if ($bad.ExitCode -eq 0) { throw 'launcher swallowed failure' }
$folder = (Join-Path $Target 'puretokens-image') + [IO.Path]::DirectorySeparatorChar
$inventory = Invoke-NativeExecutor $binary @('install-inventory', '--directory', $folder, '--name', 'puretokens-image')
if ($inventory.ExitCode -ne 0 -or ($inventory.Output | ConvertFrom-Json).name -ne 'puretokens-image') { throw 'launcher corrupted quoted path' }
'@ | Set-Content -LiteralPath $launchProbe -Encoding UTF8
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $launchProbe -Installer $installer -Target $engineTarget
    if ($LASTEXITCODE -ne 0) { throw "$engine native launcher regression" }
    $capture = Join-Path $root "capture-$engine.ps1"
    $receipt = Join-Path $root "receipt-$engine.txt"
    @'
param($Installer, $Target, $Receipt)
& $Installer init -Target $Target | Set-Content -LiteralPath $Receipt -Encoding UTF8
'@ | Set-Content -LiteralPath $capture -Encoding ASCII
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $capture -Installer $installer -Target $engineTarget -Receipt $receipt
    $summary = Get-Content -LiteralPath $receipt -Raw -Encoding UTF8
    if (-not $summary.Contains('Example (not executed):') -or $summary.Length -gt 2200 -or $summary.Contains('## ')) { throw "$engine did not return a compact installation summary" }
    # The documented entry must preserve modifications under either engine.
    $personalFile = Join-Path $engineTarget "puretokens-image/personal.txt"
    "keep my work" | Set-Content -LiteralPath $personalFile -Encoding UTF8
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -Target $engineTarget *> $null
    if ($LASTEXITCODE -eq 0 -or -not (Test-Path -LiteralPath $personalFile)) { throw "$engine overwrote personal files" }
    Remove-Item -LiteralPath $personalFile
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File $installer sync -Target $engineTarget
    if ($LASTEXITCODE -ne 0) { throw "$engine could not update an unchanged managed installation" }
    & $command.Source -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repository 'runtime/puretokens-skill-fetch.ps1') -?
    if ($LASTEXITCODE -ne 0) { throw "$engine download entry could not be parsed" }
  }
  & $installer sync -Target $target
  $binary = Join-Path $target '.puretokens-executor/puretokens-api.exe'
  $expected = (Get-Content (Join-Path $repository 'package.json') -Raw | ConvertFrom-Json).version
  if ((& $binary --version) -ne $expected) { throw 'Executor version mismatch' }
  & $installer sync -Target $target
  $lock = [IO.File]::Open((Join-Path $target '.puretokens-install.lock'), 'OpenOrCreate', 'ReadWrite', 'None')
  $rejected = $false
  try { & $installer sync -Target $target } catch { $rejected = $true } finally { $lock.Dispose() }
  if (-not $rejected) { throw 'Concurrent installation was not rejected' }
  $image = Join-Path $target 'puretokens-image'
  $stage = Join-Path $target '.puretokens-skill-stage-fixture'
  New-Item -ItemType Directory (Join-Path $stage 'backup') -Force | Out-Null
  New-Item -ItemType File (Join-Path $stage 'transaction-v1') | Out-Null
  '[{"name":"puretokens-image","action":"replace"}]' | Set-Content (Join-Path $stage 'plan.json')
  Move-Item $image (Join-Path $stage 'backup/puretokens-image')
  & $installer sync -Target $target
  if (Test-Path $stage) { throw 'Interrupted transaction not recovered' }
  if (-not (Test-Path (Join-Path $image 'SKILL.md'))) { throw 'Skill missing after recovery' }
  Remove-Item -LiteralPath (Join-Path $image 'SKILL.md')
  $inspection = & $binary install-inspect --directory $target | ConvertFrom-Json
  if (-not @($inspection.findings | Where-Object { $_.category -eq 'missing' -and $_.path -eq 'puretokens-image/SKILL.md' }).Count) { throw 'Missing managed file was not identified' }
  & $installer repair -Target $target
  if (-not (Test-Path -LiteralPath (Join-Path $image 'SKILL.md'))) { throw 'Explicit repair did not restore the missing managed file' }
  # A kill after planning but before moving the original must remain repairable.
  $orphan = Join-Path $target '.puretokens-skill-stage-orphan'
  'preserve unknown transaction' | Set-Content -LiteralPath $orphan
  $originalImage = [IO.File]::ReadAllBytes((Join-Path $image 'SKILL.md'))
  $rejected = $false
  try { & $installer repair -Target $target } catch { $rejected = $true }
  if (-not $rejected -or -not (Test-Path -LiteralPath $orphan) -or
      [Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $image 'SKILL.md'))) -cne [Convert]::ToBase64String($originalImage)) {
    throw 'Repair accepted an unknown non-directory staging entry or changed existing files'
  }
  Remove-Item -LiteralPath $orphan
  $stage = Join-Path $target '.puretokens-skill-stage-repair-fixture'
  New-Item -ItemType Directory (Join-Path $stage 'backup') -Force | Out-Null
  Copy-Item -LiteralPath $image -Destination (Join-Path $stage 'puretokens-image') -Recurse
  New-Item -ItemType File (Join-Path $stage 'transaction-v1') | Out-Null
  New-Item -ItemType File (Join-Path $stage 'repair-v1') | Out-Null
  '[{"name":"puretokens-image","action":"replace"}]' | Set-Content (Join-Path $stage 'plan.json') -Encoding UTF8
  Remove-Item -LiteralPath (Join-Path $image 'SKILL.md')
  & $installer repair -Target $target
  if ((Test-Path -LiteralPath $stage) -or -not (Test-Path -LiteralPath (Join-Path $image 'SKILL.md'))) { throw 'Interrupted missing-file repair did not complete' }
  'preserve my edit' | Set-Content -LiteralPath (Join-Path $image 'SKILL.md')
  $rejected = $false
  try { & $installer repair -Target $target } catch { $rejected = $true }
  if (-not $rejected -or -not (Get-Content -LiteralPath (Join-Path $image 'SKILL.md') -Raw).Contains('preserve my edit')) { throw 'Repair overwrote local customization' }
  $unmanaged = Join-Path $root "unmanaged/puretokens-image"
  New-Item -ItemType Directory -Path $unmanaged -Force | Out-Null
  "# Independent Skill" | Set-Content -LiteralPath (Join-Path $unmanaged "SKILL.md")
  '{"name":"puretokens-image"}' | Set-Content -LiteralPath (Join-Path $unmanaged "skill.json")
  $rejected = $false
  try { & $installer sync -Target (Split-Path -Parent $unmanaged) } catch { $rejected = $true }
  if (-not $rejected -or -not (Get-Content -LiteralPath (Join-Path $unmanaged "SKILL.md") -Raw).Contains("Independent")) { throw "unmanaged same-name Skill was replaced" }
} finally {
  foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
  if (Test-Path $root) { Remove-Item $root -Recurse -Force }
}
