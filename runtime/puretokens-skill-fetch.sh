#!/bin/sh
# Native download/update entry point. Installation writes belong to sync only.
set -eu
umask 077

fail() { printf '%s\n' "Pure Tokens Skill download: $*" >&2; exit 1; }
command_name=${1:-}
case "$command_name" in check-update|install|update) ;; *) fail "use check-update, install or update with --host or --target" ;; esac
shift
host=
target=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --host) [ "$#" -ge 2 ] || fail "--host requires a value"; host=$2; shift 2 ;;
    --target) [ "$#" -ge 2 ] || fail "--target requires a value"; target=$2; shift 2 ;;
    *) fail "unsupported option" ;;
  esac
done
[ -n "$host" ] || [ -n "$target" ] || fail "--host or --target is required"
command -v curl >/dev/null 2>&1 || fail "curl is unavailable; use a host session with native HTTPS download support"
download_root=$(mktemp -d "${TMPDIR:-/tmp}/puretokens-download.XXXXXX") || fail "cannot create a private download directory"
trap 'rm -rf -- "$download_root"' EXIT
trap 'exit 130' INT
trap 'exit 129' HUP
trap 'exit 143' TERM

download() {
  # URLs are constructed below from fixed official origins and validated IDs.
  download_exit=0
  download_status=$(curl --silent --show-error --location --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 15 --max-time 180 \
    --header 'Accept: application/vnd.github+json' --user-agent 'puretokens-skill-installer' --output "$2" --write-out '%{http_code}' "$1" 2>/dev/null) || download_exit=$?
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
  [ -z "$download_error" ] || fail "stage=$3 error_code=$download_error http_status=$download_status installation_status=not_completed installed_files_changed=false; the official source download failed; no installed files were changed; no automatic retry"
}
json_string() { sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$1" | sed -n '1p'; }
json_number() { sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\\([0-9][0-9]*\\)[[:space:]]*[,}]*[[:space:]]*$/\\1/p" "$1" | sed -n '1p'; }
version_valid() { printf '%s\n' "$1" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; }
sha256_file() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
  else fail "SHA-256 verification is unavailable"; fi
}

download 'https://github.com/puretokensx/puretokens-skill/releases/latest/download/release-manifest.json' "$download_root/release.json" read_release_manifest
[ "$(json_number "$download_root/release.json" schemaVersion)" = 2 ] || fail "the stable release manifest format is unsupported; no source fallback"
source_commit=$(json_string "$download_root/release.json" sourceCommit)
printf '%s\n' "$source_commit" | grep -Eq '^[0-9a-f]{40}$' || fail "the stable release has no verified source commit"
available_version=$(json_string "$download_root/release.json" version)
version_valid "$available_version" || fail "the stable release version is invalid"
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
target=$(sh "$location_installer" "$@") || exit 1
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
if [ "$installed_version" = "$available_version" ]; then
  installed_executor="$target/.puretokens-executor/puretokens-api"
  [ -f "$installed_executor" ] && [ ! -L "$installed_executor" ] || fail "installed executor is missing or unsupported; existing files were preserved"
  [ "$(sha256_file "$installed_executor")" = "$executor_checksum" ] || fail "installed executor checksum mismatch; existing files were preserved"
  [ "$(json_string "$target/.puretokens-executor/runtime.json" platform)" = "$platform" ] || fail "installed platform does not match; existing files were preserved"
  set -- verify-installed --target "$target" --release-manifest "$download_root/release.json"
  [ -z "$host" ] || set -- "$@" --host "$host"
  sh "$location_installer" "$@" || exit 1
  printf '%s\n' "Pure Tokens Skills $available_version are already current and verified; no archive downloaded, no files changed, init not run."
  exit 0
fi
command -v unzip >/dev/null 2>&1 || fail "ZIP extraction is unavailable; use a host session with native ZIP support"
download "$release_origin/$filename" "$download_root/source.zip" download_platform_archive
[ "$(sha256_file "$download_root/source.zip")" = "$checksum" ] || fail "published platform archive checksum mismatch"
archive_root=puretokens-skill

# Reject escaping paths, unexpected archive roots and symlinks before extraction.
unzip -Z1 "$download_root/source.zip" > "$download_root/entries" 2>/dev/null || fail "the archive is unreadable"
awk -v root="$archive_root/" '
  index($0,root)!=1 || $0 ~ /\\/ || $0 ~ /(^|\/)\.\.?($|\/)/ {exit 1}
  END {if(NR==0)exit 1}
' "$download_root/entries" || fail "the archive contains an unsafe path"
unzip -Z -l "$download_root/source.zip" > "$download_root/archive-info" 2>/dev/null || fail "the archive cannot be inspected"
if awk '$1 ~ /^l/ {found=1} END {exit !found}' "$download_root/archive-info"; then fail "archive symlinks are unsupported"; fi
mkdir "$download_root/unpacked"
unzip -q "$download_root/source.zip" -d "$download_root/unpacked" || fail "the archive could not be extracted"
source_root="$download_root/unpacked/$archive_root"
[ "$(json_string "$source_root/package.json" version)" = "$available_version" ] || fail "the archive version does not match the stable release"
[ "$(sha256_file "$source_root/runtime/puretokens-skill-install.sh")" = "$selector_checksum" ] || fail "archive selector does not match the stable release"
[ "$(sha256_file "$source_root/runtime/executor/bin/puretokens-api-$platform")" = "$executor_checksum" ] || fail "archive executor does not match the stable release"
set -- sync --source "$source_root" --target "$target"
[ -z "$host" ] || set -- "$@" --host "$host"
sh "$source_root/runtime/puretokens-skill-install.sh" "$@"
