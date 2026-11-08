#!/bin/sh
# Native download/update entry point. Installation writes belong to sync only.
set -eu
umask 077

stage=arguments
failure_reported=false
download_root=
official_asset_route=false
fail() {
  failure_reported=true
  case "$*" in
    stage=*) printf '%s\n' "Pure Tokens Skill download: $*" >&2 ;;
    *) printf '%s\n' "Pure Tokens Skill download: stage=$stage error_code=validation_failed installation_status=not_completed installed_files_changed=false next_step=review_installation_failure; $*" >&2 ;;
  esac
  exit 1
}
finish_download() {
  result=$?
  trap - EXIT HUP INT TERM
  if [ "$result" -ne 0 ] && [ "$failure_reported" = false ]; then
    state=not_completed
    changed=false
    if [ "$stage" = sync ]; then state=unconfirmed; changed=unknown; fi
    printf '%s\n' "Pure Tokens Skill download: stage=$stage error_code=local_operation_failed installation_status=$state installed_files_changed=$changed next_step=review_local_access; a local operation could not complete. Check available disk space and current-session command/directory access; the exact cause is unconfirmed. No automatic retry." >&2
  fi
  if [ -n "$download_root" ] && ! rm -rf -- "$download_root" 2>/dev/null; then
    printf '%s\n' "cleanup_status: pending; download temporary files were retained. Preserve the preceding install result; do not bypass host cleanup restrictions."
  fi
  exit "$result"
}
trap finish_download EXIT
trap 'exit 130' INT
trap 'exit 129' HUP
trap 'exit 143' TERM
detect_current_host() {
  # Inspect only the direct launcher chain, never arguments, configuration,
  # directory presence or inherited session markers. An unknown launcher is
  # a boundary: a nested client must not borrow an outer client's identity.
  process_id=$$
  depth=0
  while [ "$depth" -lt 8 ]; do
    depth=$((depth + 1))
    process_row=$(LC_ALL=C ps -p "$process_id" -o ppid= -o comm= 2>/dev/null) || break
    read -r parent_id executable <<EOF
$process_row
EOF
    case "$parent_id" in ''|*[!0-9]*) break ;; esac
    case "${executable##*/}" in
      codex) [ -z "${PASEO_HOME:-}" ] || break; printf '%s\n' codex; return 0 ;;
      mimo) printf '%s\n' mimo-coder; return 0 ;;
      droid) printf '%s\n' factory-droid; return 0 ;;
      opencode) printf '%s\n' opencode; return 0 ;;
      grok) printf '%s\n' grok-build; return 0 ;;
      qoder|qodercli) printf '%s\n' qoder; return 0 ;;
      sh|bash|zsh|dash|ksh|fish|pwsh) ;;
      *) break ;;
    esac
    [ "$parent_id" -gt 1 ] && [ "$parent_id" != "$process_id" ] || break
    process_id=$parent_id
  done
  fail "stage=detect_host error_code=current_host_unconfirmed installation_status=not_started installed_files_changed=false next_step=confirm_current_client; confirm this client's host ID and run once with --host <id>. Supported IDs: claude-code, codex, workbuddy, gemini-cli, grok-build, opencode, trae, claude-desktop, dsh-desktop, deepseek-harness, minimax-code, zcode, kimi-code, qoder, pi, hermes, evox, vscode, octop, oh-my-pi, minimax-code-cli, qwen-code, codebuddy-code, copilot-cli, factory-droid, mimo-coder, paseo. No network request or installation write was made."
}
command_name=${1:-}
case "$command_name" in detect-host|check-update|install|update|repair) ;; *) fail "use detect-host, check-update, install, update or explicitly requested repair; --host and --target are optional when the current launcher can be identified" ;; esac
shift
host=
target=
release_metadata=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --host) [ "$#" -ge 2 ] || fail "--host requires a value"; host=$2; shift 2 ;;
    --target) [ "$#" -ge 2 ] || fail "--target requires a value"; target=$2; shift 2 ;;
    --release-metadata) [ "$#" -ge 2 ] || fail "--release-metadata requires a value"; release_metadata=$2; shift 2 ;;
    *) fail "unsupported option" ;;
  esac
done
if [ "$command_name" = detect-host ]; then
  [ -z "$host$target$release_metadata" ] || fail "detect-host takes no host, target or release metadata options"
  stage=detect_host
  detect_current_host
  exit 0
fi
if [ -z "$host" ] && [ -z "$target" ]; then
  stage=detect_host
  host=$(detect_current_host) || { failure_reported=true; exit 1; }
  printf '%s\n' "Pure Tokens Skills current client: $host (detected)."
fi
if [ "$host" = octop ] && [ -z "$target" ]; then
  fail "stage=select_target error_code=workspace_target_required installation_status=not_started installed_files_changed=false next_step=confirm_workspace_target; confirm the current Octop workspace and run once with --host octop --target <absolute-workspace>/.octop/skills. No network request or installation write was made."
fi
command -v curl >/dev/null 2>&1 || fail "curl is unavailable; use a host session with native HTTPS download support"
stage=prepare_download
download_root=$(mktemp -d "${TMPDIR:-/tmp}/puretokens-download.XXXXXX" 2>/dev/null) ||
  fail "stage=$stage error_code=local_io_failure installation_status=not_started installed_files_changed=false next_step=review_download_directory_permissions; cannot create the private download directory. Check available disk space and temporary-directory write permission."

download_once() {
  stage=$3
  # URLs are constructed below from fixed official origins and validated IDs.
  download_budget=45
  [ "$stage" != download_platform_archive ] || download_budget=180
  [ "${5:-}" != 2097152 ] || download_budget=45
  printf '%s\n' "Pure Tokens download progress: stage=$stage status=started" >&2
  download_exit=0
  download_status=$(curl -q --silent --show-error --location --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 15 --max-time "$download_budget" --speed-limit 1 --speed-time 30 \
    --max-redirs 5 --max-filesize "${5:-33554432}" --header "Accept: $4" --user-agent 'puretokens-skill-installer' --output "$2" --write-out '%{http_code}' "$1" 2>/dev/null) || download_exit=$?
  case "$download_status" in [1-5][0-9][0-9]) ;; *) download_status=0 ;; esac
  download_error=
  if [ "$download_exit" -ne 0 ]; then
    case "$download_exit" in
      28) download_error=timeout ;;
      5|6) download_error=dns_failure ;;
      7) download_error=connection_failure ;;
      35|51|58|60|77|80|83|90|91) download_error=tls_failure ;;
      23|26) download_error=local_io_failure ;;
      *) download_error=transport_failure ;;
    esac
  elif [ "$download_status" != 200 ]; then
    download_error=http_error
  fi
}
download_failure() {
    next_step=review_download_failure
    [ "$download_error" != local_io_failure ] || next_step=review_download_directory_permissions
    fail "stage=$1 error_code=$download_error http_status=$download_status installation_status=not_completed installed_files_changed=false next_step=$next_step; the official source download failed. For local I/O failures, check temporary-directory write permission and available disk space. No installed files were changed; no automatic retry."
}
# Self-contained bounded JSON projection; no installed jq, Node or Python.
project_release_asset() {
  LC_ALL=C awk -v expected="$2" -v wanted="$3" '
    function bad() { invalid=1; exit 1 }
    function ws() { while(substr(s,p,1) ~ /[ \t\r\n]/ && p<=length(s)) p++ }
    function string( start,c,e) {
      start=p++; while(p<=length(s)) {
        c=substr(s,p++,1)
        if(c=="\"") return substr(s,start,p-start)
        if(c ~ /[[:cntrl:]]/) bad()
        if(c=="\\") {
          e=substr(s,p++,1)
          if(e=="u") { if(substr(s,p,4) !~ /^[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]$/) bad(); p+=4 }
          else if(e !~ /^["\\\/bfnrt]$/) bad()
        }
      }; bad()
    }
    function value(ctx,idx,depth, c,t,key,id,child,n) {
      if(depth>32) bad(); ws(); c=substr(s,p,1)
      if(c=="{") {
        id=++serial; p++; ws()
        if(substr(s,p,1)=="}") { p++; return "object" }
        while(1) {
          ws(); if(substr(s,p,1)!="\"") bad(); key=string()
          if(index(key,"\\") || seen[id SUBSEP key]++) bad()
          ws(); if(substr(s,p++,1)!=":") bad()
          child=(ctx=="root" && key=="\"assets\"") ? "assets" : "other"
          t=value(child,idx,depth+1)
          if(ctx=="root") root[key]=t
          if(ctx=="asset") asset[idx SUBSEP key]=t
          ws(); c=substr(s,p++,1); if(c=="}") return "object"; if(c!=",") bad()
        }
      }
      if(c=="[") {
        p++; ws(); if(substr(s,p,1)=="]") { p++; return "array" }
        n=0; while(1) {
          n++; if(ctx=="assets" && n>100) bad()
          t=value(ctx=="assets" ? "asset" : "other",n,depth+1)
          if(ctx=="assets") { count=n; if(t!="object") bad() }
          ws(); c=substr(s,p++,1); if(c=="]") return "array"; if(c!=",") bad()
        }
      }
      if(c=="\"") return string()
      if(match(substr(s,p),/^(true|false|null|-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?)/)) {
        t=substr(s,p,RLENGTH); p+=RLENGTH; return t
      }; bad()
    }
    function field(i,k) { return asset[i SUBSEP "\"" k "\""] }
    function unquote(t) { if(t !~ /^"[^"\\]*"$/) bad(); return substr(t,2,length(t)-2) }
    { s=s $0 "\n"; if(length(s)>2097152) bad() }
    END {
      if(invalid) exit 1
      p=1; if(value("root",0,0)!="object") bad(); ws(); if(p<=length(s)) bad()
      tag=unquote(root["\"tag_name\""])
      if(tag !~ /^v[0-9]+\.[0-9]+\.[0-9]+$/ || (expected!="latest" && tag!=expected) || root["\"draft\""]!="false" || root["\"prerelease\""]!="false" || root["\"assets\""]!="array") bad()
      for(i=1;i<=count;i++) if(field(i,"name")=="\"" wanted "\"") { selected=i; matches++ }
      if(matches!=1) bad()
      id=field(selected,"id"); size=field(selected,"size"); digest=unquote(field(selected,"digest"))
      base="https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/"
      public="https://github.com/puretokensx/puretokens-skill/releases/download/" tag "/" wanted
      if(id !~ /^[1-9][0-9]*$/ || length(id)>16 || size !~ /^[1-9][0-9]*$/ || size+0>33554432 || field(selected,"state")!="\"uploaded\"" || digest !~ /^sha256:[0-9a-f]+$/ || length(digest)!=71 || field(selected,"url")!="\"" base id "\"" || field(selected,"browser_download_url")!="\"" public "\"") bad()
      print tag, id, size, substr(digest,8)
    }
  ' "$1"
}
download() {
  if [ "$official_asset_route" = true ]; then
    api_download "$1" "$2" "$3"; return
  fi
  download_once "$1" "$2" "$3" application/vnd.github+json
  [ -n "$download_error" ] || return 0
  # Only an explicit gateway HTTP failure allows one alternate asset GET.
  if [ "$download_exit" -eq 0 ]; then
    case "$download_status:$1" in
      502:https://github.com/puretokensx/puretokens-skill/releases/*|503:https://github.com/puretokensx/puretokens-skill/releases/*|504:https://github.com/puretokensx/puretokens-skill/releases/*)
        api_download "$1" "$2" "$3"; return ;;
    esac
  fi
  download_failure "$3"
}
api_download() {
  asset_name=${1##*/}
  case "$1" in
    https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json) expected_tag=latest; api_metadata_path=latest ;;
    https://github.com/puretokensx/puretokens-skill/releases/download/v*)
      expected_tag=${1#https://github.com/puretokensx/puretokens-skill/releases/download/}; expected_tag=${expected_tag%%/*}
      version_valid "${expected_tag#v}" || fail "invalid fallback release version"
      api_metadata_path=tags/$expected_tag ;;
    *) fail "unsupported official release fallback" ;;
  esac
  printf '%s\n' "Pure Tokens download progress: stage=$3 status=official_asset_fallback" >&2
  if [ ! -f "$download_root/github-release.json" ]; then
    download_once "https://api.github.com/repos/puretokensx/puretokens-skill/releases/$api_metadata_path" "$download_root/github-release.json" "$3" application/vnd.github+json 2097152
    [ -z "$download_error" ] || download_failure "$3"
  fi
  asset_projection=$(project_release_asset "$download_root/github-release.json" "$expected_tag" "$asset_name") || fail "official release asset metadata is invalid or ambiguous"
  read -r api_release_tag api_asset_id api_asset_size api_asset_sha <<EOF
$asset_projection
EOF
  download_once "https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/$api_asset_id" "$2" "$3" application/octet-stream "$api_asset_size"
  [ -z "$download_error" ] || download_failure "$3"
  [ "$(wc -c < "$2" | tr -d ' ')" = "$api_asset_size" ] && [ "$(sha256_file "$2")" = "$api_asset_sha" ] || fail "official release asset size or checksum mismatch"
  official_asset_route=true
}
json_string() { sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$1" 2>/dev/null | sed -n '1p'; }
json_number() { sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\\([0-9][0-9]*\\)[[:space:]]*[,}]*[[:space:]]*$/\\1/p" "$1" 2>/dev/null | sed -n '1p'; }
version_valid() { printf '%s\n' "$1" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; }
sha256_file() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" 2>/dev/null | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" 2>/dev/null | awk '{print $1}'
  else fail "SHA-256 verification is unavailable"; fi
}
run_installer() {
  installer_status=0
  { sh "$@" 2>"$download_root/installer-error"; } 2>/dev/null || installer_status=$?
  [ "$installer_status" -ne 0 ] || return 0
  if grep -q '^Pure Tokens Skill installer: stage=' "$download_root/installer-error" 2>/dev/null; then
    # Forward only reviewed receipt lines, never raw command errors or paths.
    sed -n '/^Pure Tokens Skill installer: stage=/p; /^recovery_status=/p' "$download_root/installer-error" >&2 2>/dev/null
    return "$installer_status"
  fi
  state=not_completed
  changed=false
  if [ "$stage" = sync ]; then state=unconfirmed; changed=unknown; fi
  fail "stage=$stage error_code=installer_unavailable installation_status=$state installed_files_changed=$changed next_step=review_host_execution_permissions; the installer did not return a usable failure receipt. Check current-session execution permissions; the exact cause is unconfirmed. No automatic retry."
}

if [ -n "$release_metadata" ]; then
  stage=verify_bootstrap_metadata
  case "$release_metadata" in /*) ;; *) fail "bootstrap metadata requires an absolute local file" ;; esac
  [ -f "$release_metadata" ] && [ ! -L "$release_metadata" ] || fail "bootstrap metadata must be a regular non-linked file"
  [ "$(wc -c < "$release_metadata" | tr -d ' ')" -le 2097152 ] || fail "bootstrap metadata exceeds the supported size"
  cp "$release_metadata" "$download_root/github-release.json" 2>/dev/null || fail "bootstrap metadata is unreadable"
  bootstrap_projection=$(project_release_asset "$download_root/github-release.json" latest puretokens-skill-fetch.sh) || fail "bootstrap metadata is invalid or ambiguous"
  read -r api_release_tag bootstrap_id bootstrap_size bootstrap_sha <<EOF
$bootstrap_projection
EOF
  [ "$(wc -c < "$0" | tr -d ' ')" = "$bootstrap_size" ] && [ "$(sha256_file "$0")" = "$bootstrap_sha" ] || fail "bootstrap script does not match the selected release"
  official_asset_route=true
fi
download 'https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json' "$download_root/release.json" read_release_manifest
[ "$(json_number "$download_root/release.json" schemaVersion)" = 2 ] || fail "the stable release manifest format is unsupported; no source fallback"
source_commit=$(json_string "$download_root/release.json" sourceCommit)
printf '%s\n' "$source_commit" | grep -Eq '^[0-9a-f]{40}$' || fail "the stable release has no verified source commit"
available_version=$(json_string "$download_root/release.json" version)
version_valid "$available_version" || fail "the stable release version is invalid"
[ -z "${api_release_tag:-}" ] || [ "$api_release_tag" = "v$available_version" ] || fail "the manifest version does not match the selected official release"
release_origin="https://github.com/puretokensx/puretokens-skill/releases/download/v$available_version"
sed -n '/"shell"[[:space:]]*:/,/}/p' "$download_root/release.json" > "$download_root/selector.json"
selector_name=$(json_string "$download_root/selector.json" filename)
selector_checksum=$(json_string "$download_root/selector.json" sha256)
[ "$selector_name" = puretokens-skill-install.sh ] || fail "the stable directory selector is invalid"
printf '%s\n' "$selector_checksum" | grep -Eq '^[0-9a-f]{64}$' || fail "the stable directory selector checksum is invalid"
location_installer="$download_root/puretokens-skill-install.sh"
download "$release_origin/$selector_name" "$location_installer" download_selector
[ "$(sha256_file "$location_installer")" = "$selector_checksum" ] || fail "stable directory selector checksum mismatch"
set -- locate
[ -z "$host" ] || set -- "$@" --host "$host"
[ -z "$target" ] || set -- "$@" --target "$target"
stage=select_target
target=$(run_installer "$location_installer" "$@") || { failure_reported=true; exit 1; }
installed_version=not_installed
if [ -f "$target/.puretokens-executor/runtime.json" ]; then
  if [ "$(json_string "$target/.puretokens-executor/runtime.json" name)" = puretokens-api-executor ]; then
    installed_version=$(json_string "$target/.puretokens-executor/runtime.json" version)
    version_valid "$installed_version" || installed_version=unverified
  fi
fi
if [ "$installed_version" = "$available_version" ]; then update_state=current; else update_state=version_differs; fi
printf '%s\n' "Pure Tokens Skills update check: installed=$installed_version available=$available_version status=$update_state source_commit=$source_commit"
[ "$command_name" != check-update ] || exit 0
if version_valid "$installed_version" && awk -v current="$installed_version" -v available="$available_version" 'BEGIN {split(current,c,"."); split(available,a,"."); for(i=1;i<=3;i++){if(c[i]+0>a[i]+0)exit 0;if(c[i]+0<a[i]+0)exit 1}exit 1}'; then
  fail "the installed version is newer than the stable release; automatic downgrade was stopped"
fi

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) platform=darwin-arm64 ;;
  Darwin-x86_64) platform=darwin-amd64 ;;
  Linux-x86_64) platform=linux-amd64 ;;
  Linux-aarch64|Linux-arm64) platform=linux-arm64 ;;
  *) fail "this operating system and CPU has no platform executor" ;;
esac
sed -n "/\"$platform\"[[:space:]]*:/,/}/p" "$download_root/release.json" > "$download_root/platform.json"
filename=$(json_string "$download_root/platform.json" filename)
checksum=$(json_string "$download_root/platform.json" sha256)
executor_checksum=$(json_string "$download_root/platform.json" executorSha256)
[ "$filename" = "puretokens-skill-$available_version-$platform.zip" ] || fail "published platform metadata is invalid"
printf '%s\n' "$checksum" | grep -Eq '^[0-9a-f]{64}$' || fail "published platform checksum is invalid"
printf '%s\n' "$executor_checksum" | grep -Eq '^[0-9a-f]{64}$' || fail "published executor checksum is invalid"
if [ "$installed_version" = "$available_version" ] && [ "$command_name" != repair ]; then
  installed_executor="$target/.puretokens-executor/puretokens-api"
  [ -f "$installed_executor" ] && [ ! -L "$installed_executor" ] || fail "installed executor is missing or unsupported; existing files were preserved"
  [ "$(sha256_file "$installed_executor")" = "$executor_checksum" ] || fail "installed executor checksum mismatch; existing files were preserved"
  [ "$(json_string "$target/.puretokens-executor/runtime.json" platform)" = "$platform" ] || fail "installed platform does not match; existing files were preserved"
  set -- verify-installed --target "$target" --release-manifest "$download_root/release.json"
  [ -z "$host" ] || set -- "$@" --host "$host"
  stage=verify_installed
  run_installer "$location_installer" "$@" || { failure_reported=true; exit 1; }
  printf '%s\n' "Pure Tokens Skills $available_version are already current and verified; no archive downloaded, no files changed, init not run."
  exit 0
fi
command -v unzip >/dev/null 2>&1 || fail "ZIP extraction is unavailable; use a host session with native ZIP support"
download "$release_origin/$filename" "$download_root/source.zip" download_platform_archive
[ "$(sha256_file "$download_root/source.zip")" = "$checksum" ] || fail "published platform archive checksum mismatch"
archive_root=puretokens-skill
stage=verify_archive

# Reject escaping paths, unexpected archive roots and symlinks before extraction.
unzip -Z1 "$download_root/source.zip" > "$download_root/entries" 2>/dev/null || fail "the archive is unreadable"
awk -v root="$archive_root/" '
  index($0,root)!=1 || $0 ~ /\\/ || $0 ~ /(^|\/)\.\.?($|\/)/ {exit 1}
  END {if(NR==0)exit 1}
' "$download_root/entries" || fail "the archive contains an unsafe path"
unzip -Z -l "$download_root/source.zip" > "$download_root/archive-info" 2>/dev/null || fail "the archive cannot be inspected"
if awk '$1 ~ /^l/ {found=1} END {exit !found}' "$download_root/archive-info"; then fail "archive symlinks are unsupported"; fi
stage=extract_archive
mkdir "$download_root/unpacked" 2>/dev/null
unzip -q "$download_root/source.zip" -d "$download_root/unpacked" 2>/dev/null || fail "the archive could not be extracted; check available disk space and temporary-directory write permission"
stage=verify_archive
source_root="$download_root/unpacked/$archive_root"
[ "$(json_string "$source_root/package.json" version)" = "$available_version" ] || fail "the archive version does not match the stable release"
[ "$(sha256_file "$source_root/runtime/puretokens-skill-install.sh")" = "$selector_checksum" ] || fail "archive selector does not match the stable release"
[ "$(sha256_file "$source_root/runtime/executor/bin/puretokens-api-$platform")" = "$executor_checksum" ] || fail "archive executor does not match the stable release"
sync_command=sync
[ "$command_name" != repair ] || sync_command=repair
set -- "$sync_command" --source "$source_root" --target "$target"
[ "$command_name" = repair ] || set -- "$@" --no-recovery
[ -z "$host" ] || set -- "$@" --host "$host"
stage=sync
run_installer "$source_root/runtime/puretokens-skill-install.sh" "$@" || { failure_reported=true; exit 1; }
