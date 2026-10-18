#!/bin/sh
# puretokens-locate-v1

set -eu

current_skills="puretokens-balance puretokens-connection puretokens-models puretokens-image puretokens-video puretokens-audio puretokens-evaluate puretokens-embedding puretokens-rerank puretokens-update"
stage=arguments
installation_status=not_started
installed_files_changed=false
failure_reported=false

report_unexpected_failure() {
  if [ "$1" -ne 0 ] && [ "$failure_reported" = false ]; then
    printf '%s\n' "Pure Tokens Skill installer: stage=$stage error_code=local_operation_failed installation_status=$installation_status installed_files_changed=$installed_files_changed next_step=review_local_access; a local operation could not complete. Check current-session command/directory access and available disk space; the exact cause is unconfirmed. No automatic retry." >&2
  fi
}
trap 'report_unexpected_failure $?' EXIT

usage() {
  printf '%s\n' "Usage: puretokens-skill-install.sh <check|verify-installed|init|sync|repair|locate> (--host <claude-code|codex|workbuddy|gemini-cli|grok-build|opencode|trae|claude-desktop|dsh-desktop|deepseek-harness|minimax-code|zcode|kimi-code|qoder|pi|hermes|evox|vscode|octop> | --target <absolute-skill-directory>) [--source <absolute-official-source-directory>]"
}

fail() {
  failure_reported=true
  printf '%s\n' "Pure Tokens Skill installer: stage=$stage error_code=validation_failed installation_status=$installation_status installed_files_changed=$installed_files_changed next_step=review_installation_failure; $*" >&2
  exit 1
}

local_io() {
  if "$@" 2>/dev/null; then return 0; fi
  failure_reported=true
  printf '%s\n' "Pure Tokens Skill installer: stage=$stage error_code=local_io_failure installation_status=$installation_status installed_files_changed=$installed_files_changed next_step=review_install_directory_access; the local file operation failed. Check current-session directory permissions, file locks and available disk space; the exact cause is unconfirmed. No automatic retry." >&2
  exit 1
}

managed_skill() {
  directory=$1
  name=$2
  [ -f "$directory/SKILL.md" ] && [ -f "$directory/skill.json" ] &&
    grep -Fq "\"name\": \"$name\"" "$directory/skill.json"
}

managed_executor() {
  directory=$1
  [ -f "$directory/runtime.json" ] && [ -f "$directory/puretokens-api" ] &&
    grep -Eq '"name"[[:space:]]*:[[:space:]]*"puretokens-api-executor"' "$directory/runtime.json"
}

executor_platform() {
  case "$(uname -s 2>/dev/null)-$(uname -m 2>/dev/null)" in
    Darwin-arm64) printf '%s\n' "darwin-arm64" ;;
    Darwin-x86_64) printf '%s\n' "darwin-amd64" ;;
    Linux-x86_64) printf '%s\n' "linux-amd64" ;;
    Linux-aarch64|Linux-arm64) printf '%s\n' "linux-arm64" ;;
    *) fail "no Pure Tokens executor is available for this operating system and CPU" ;;
  esac
}

sha256_file() {
  file=$1
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  else
    fail "cannot verify the Pure Tokens executor because SHA-256 is unavailable"
  fi
}

executor_artifact() {
  source_root=$1
  platform=$(executor_platform)
  manifest="$source_root/runtime/executor/manifest.json"
  expected_path=$(sed -n "/\"$platform\"[[:space:]]*:/,/}/ s/.*\"path\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$manifest" | sed -n '1p')
  expected_sha=$(sed -n "/\"$platform\"[[:space:]]*:/,/}/ s/.*\"sha256\"[[:space:]]*:[[:space:]]*\"\\([0-9a-f][0-9a-f]*\\)\".*/\\1/p" "$manifest" | sed -n '1p')
  [ -n "$expected_path" ] && [ -n "$expected_sha" ] || fail "official source is missing the executor artifact for $platform"
  artifact="$source_root/runtime/executor/$expected_path"
  [ -f "$artifact" ] || fail "official source is missing the executor binary for $platform"
  [ "$(sha256_file "$artifact")" = "$expected_sha" ] || fail "official source executor checksum mismatch for $platform"
  printf '%s\n' "$artifact"
}

source_release_version() {
  version=$(sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([0-9][0-9.]*\)".*$/\1/p' "$1/package.json" | sed -n '1p')
  case "$version" in
    *.*.*) printf '%s\n' "$version" ;;
    *) return 1 ;;
  esac
}

validate_client_root() {
  case "$1" in /*) ;; *) fail "host directory must be absolute" ;; esac
  case "$1/" in //*|*\\*|*/../*) fail "host directory must be local without parent traversal" ;; esac
  case "$1" in *'
'*) fail "host directory contains control characters" ;; esac
  if printf '%s' "$1" | LC_ALL=C grep -q '[[:cntrl:]]'; then
    fail "host directory contains control characters"
  fi
}

target_for_host() {
  host=$1
  [ -n "${HOME:-}" ] || fail "cannot resolve a host Skill directory because HOME is unavailable"
  case "$host" in
    claude-code|claude-desktop) printf '%s\n' "${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills" ;;
    kimi-code) printf '%s\n' "${KIMI_CODE_HOME:-$HOME/.kimi-code}/skills" ;;
    pi)
      pi_root=${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}
      case "$pi_root" in /*) ;; *) fail "Pi agent directory must be absolute" ;; esac
      case "$pi_root/" in */../*) fail "Pi agent directory must not contain parent traversal" ;; esac
      printf '%s\n' "$pi_root/skills"
      ;;
    hermes|evox)
      if [ "$host" = hermes ]; then
        client_root=${HERMES_HOME:-$HOME/.hermes}
      else
        if [ -n "${EVOX_AGENT_DIR:-}" ] && [ -n "${EVOX_CODING_AGENT_DIR:-}" ] && [ "$EVOX_AGENT_DIR" != "$EVOX_CODING_AGENT_DIR" ]; then
          fail "EvoX directory overrides disagree"
        fi
        client_root=${EVOX_AGENT_DIR:-${EVOX_CODING_AGENT_DIR:-$HOME/.evox/agent}}
      fi
      validate_client_root "$client_root"
      printf '%s\n' "$client_root/skills"
      ;;
    vscode) printf '%s\n' "$HOME/.copilot/skills" ;;
    octop) fail "Octop requires --target with the current local workspace's absolute .octop/skills directory; no global directory is assumed" ;;
    qoder)
      if [ -n "${QODER_CONFIG_DIR:-}" ]; then
        printf '%s\n' "$QODER_CONFIG_DIR/skills"
      else
        qoder_name=${QODER_CONFIG_DIR_NAME:-.qoder}
        case "$qoder_name" in .|..|*/*|*\\*) fail "invalid Qoder directory name" ;; esac
        printf '%s\n' "${QODER_CLI_HOME:-$HOME}/$qoder_name/skills"
      fi
      ;;
    zcode) printf '%s\n' "${ZCODE_DATA_BASE_DIR:-$HOME}/.zcode/skills" ;;
    minimax-code)
      [ "$(uname -s)" = Darwin ] || fail "MiniMax Code requires local macOS or Windows Desktop"
      [ -z "${MAVIS_PROFILE:-}${MINIMAX_PROFILE:-}${AGENTARCHON_PROFILE:-}${AGENTARCHON_DATA_DIR:-}${__MAVIS_RUNTIME_PROFILE:-}${__MAVIS_RUNTIME_DATA_DIR:-}" ] || fail "MiniMax CLI and runtime profile overrides are unsupported"
      client_root=${MINIMAX_DATA_DIR:-${MAVIS_DATA_DIR:-}}
      if [ -z "$client_root" ]; then
        for app in "MiniMax Code" "MiniMax" "MiniMax Agent"; do
          for store in minimax-agent-cn-config.json minimax-agent-config.json; do
            prefs="$HOME/Library/Application Support/$app/$store"
            if [ -e "$prefs" ] || [ -L "$prefs" ] || [ -L "$(dirname "$prefs")" ]; then
              fail "MiniMax Desktop preferences require its runtime data directory; run inside the local client with MINIMAX_DATA_DIR or pass the confirmed absolute --target; no directory was guessed"
            fi
          done
        done
        if [ ! -e "$HOME/.minimax/config.yaml" ] && { [ -e "$HOME/.mavis" ] || [ -L "$HOME/.mavis" ]; }; then
          fail "MiniMax legacy migration is pending; let the client migrate before installing"
        fi
        client_root="$HOME/.minimax"
      fi
      validate_client_root "$client_root"
      printf '%s\n' "$client_root/skills"
      ;;
    deepseek-harness)
      client_root=${DSH_HOME:-$HOME/.dsh}
      validate_client_root "$client_root"
      printf '%s\n' "$client_root/skills"
      ;;
    dsh-desktop)
      if [ -n "${DSH_HOME:-}" ]; then
        printf '%s\n' "$DSH_HOME/skills"
      elif [ "$(uname -s)" = Darwin ]; then
        printf '%s\n' "$HOME/Library/Application Support/dsh-desktop/harness/skills"
      else
        fail "DSH Desktop requires its local macOS or Windows environment"
      fi
      ;;
    codex) printf '%s\n' "$HOME/.agents/skills" ;;
    workbuddy) printf '%s\n' "${WORKBUDDY_CONFIG_DIR:-${CODEBUDDY_CONFIG_DIR:-$HOME/.workbuddy}}/skills" ;;
    gemini-cli)
      # Gemini resolves each .agents Skill before the same .gemini Skill.
      for skill in $current_skills; do
        if [ -e "$HOME/.agents/skills/$skill" ] || [ -L "$HOME/.agents/skills/$skill" ]; then
          printf '%s\n' "$HOME/.agents/skills"
          return
        fi
      done
      printf '%s\n' "$HOME/.gemini/skills"
      ;;
    grok-build) printf '%s\n' "$HOME/.grok/skills" ;;
    opencode)
      opencode_config=${XDG_CONFIG_HOME:-$HOME/.config}
      case "$opencode_config" in /*) ;; *) fail "OpenCode XDG config directory must be absolute" ;; esac
      case "$opencode_config/" in */../*) fail "OpenCode XDG config directory must not contain parent traversal" ;; esac
      opencode_root=${OPENCODE_CONFIG_DIR:-$opencode_config/opencode}
      case "$opencode_root" in /*) ;; *) fail "OpenCode config directory must be absolute" ;; esac
      case "$opencode_root/" in */../*) fail "OpenCode config directory must not contain parent traversal" ;; esac
      printf '%s\n' "$opencode_root/skills"
      ;;
    trae) printf '%s\n' "$HOME/.trae/skills" ;;
    *) fail "unsupported host: $host" ;;
  esac
}

check_gemini_duplicates() {
  [ "$host" = gemini-cli ] && [ -n "${HOME:-}" ] || return 0
  shared_root="$HOME/.agents/skills"
  if [ -d "$shared_root" ]; then shared_root=$(cd "$shared_root" && pwd -P); fi
  for skill in $current_skills; do
    if [ "$target_root" != "$shared_root" ] && { [ -e "$shared_root/$skill" ] || [ -L "$shared_root/$skill" ]; }; then
      fail "Gemini would load $skill from its higher-priority .agents/skills directory. Run sync --host gemini-cli without a custom target; existing directories were left untouched"
    fi
    if [ "$target_root" = "$shared_root" ] && managed_skill "$HOME/.gemini/skills/$skill" "$skill"; then
      printf '%s\n' "Managed duplicate detected: Gemini will use the updated shared $skill; its lower-priority .gemini copy was left untouched."
    fi
  done
}

validate_source() {
  source_root=$1
  [ -f "$source_root/README.md" ] || fail "official source is missing README.md"
  [ -f "$source_root/package.json" ] || fail "official source is missing package.json"
  source_release_version "$source_root" >/dev/null || fail "official source has an invalid Skill version"
  [ -f "$source_root/runtime/puretokens-skill-install.sh" ] || fail "official source is missing the macOS/Linux installer"
  [ -f "$source_root/runtime/puretokens-skill-fetch.sh" ] || fail "official source is missing the download wrapper"
  [ -f "$source_root/runtime/executor/manifest.json" ] || fail "official source is missing the executor manifest"
  executor_artifact "$source_root" >/dev/null
  for name in $current_skills; do
    managed_skill "$source_root/skills/$name" "$name" || fail "official source has an invalid Skill: $name"
  done
}

validate_target() {
  target_root=$1
  [ -d "$target_root" ] || fail "target Skill directory does not exist or is inaccessible"
  for name in $current_skills; do
    managed_skill "$target_root/$name" "$name" || fail "target is missing the managed Skill: $name"
  done
  managed_executor "$target_root/.puretokens-executor" || fail "target is missing the managed native executor"
}

verify_no_transaction() {
  for entry in "$target_root/.puretokens-install-lock" "$target_root"/.puretokens-skill-stage.*; do
    [ ! -e "$entry" ] && [ ! -L "$entry" ] || fail "installation transaction requires review; existing files were preserved"
  done
}

verify_release_identity() {
  [ -n "$release_manifest" ] || return 0
  expected_version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$release_manifest" | sed -n '1p')
  expected_platform=$(executor_platform)
  expected_checksum=$(sed -n "/\"$expected_platform\"[[:space:]]*:/,/}/ s/.*\"executorSha256\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$release_manifest" | sed -n '1p')
  printf '%s\n' "$expected_version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || fail "selected release version is invalid"
  printf '%s\n' "$expected_checksum" | grep -Eq '^[0-9a-f]{64}$' || fail "selected release executor checksum is invalid"
  identity="$target_root/.puretokens-executor/runtime.json"
  actual_version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$identity" | sed -n '1p')
  actual_platform=$(sed -n 's/.*"platform"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$identity" | sed -n '1p')
  [ "$actual_version" = "$expected_version" ] && [ "$actual_platform" = "$expected_platform" ] ||
    fail "installed release changed during verification; existing files were preserved"
  [ -f "$executor" ] && [ ! -L "$executor" ] && [ "$(sha256_file "$executor")" = "$expected_checksum" ] ||
    fail "installed executor changed during verification; existing files were preserved"
}

verify_installed() {
  target_root=$(cd "$1" 2>/dev/null && pwd -P) || fail "could not resolve --target"
  validate_target "$target_root"
  verify_no_transaction
  check_gemini_duplicates
  executor="$target_root/.puretokens-executor/puretokens-api"
  verify_release_identity
  for name in $current_skills .puretokens-executor; do
    "$executor" install-verify --directory "$target_root/$name" --name "$name" ||
      fail "installed files are missing or modified; existing files were preserved"
  done
  # A completed concurrent sync must not turn this into a stale success receipt.
  verify_release_identity
  verify_no_transaction
  printf '%s\n' "Pure Tokens installed inventories verified; no files changed and init was not run."
}

# generated:install-activation
installation_summary() {
  case "$1" in
    zcode)
      printf '%s\n' 'Next: Refresh and enable the installed Skills in ZCode, then start a new conversation. Loading has not been verified in this session.' 'Available instructions: images, video, audio/music, Jev evaluation, embeddings, reranking, models and balance. Connection and task results remain separate checks.' 'Example (not executed): Generate an image of a lake at sunset.' 'Example (not executed): Generate a 5-second video of a paper boat on a river.' 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
      ;;
    trae)
      printf '%s\n' 'Next: This is a historical installation directory; Skill discovery in current TraeWork is not confirmed.' 'API tasks: unsupported in this host; no credential adapter is available. Installing instructions does not enable generation.' 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
      ;;
    *)
      printf '%s\n' 'Next: Follow this client'"'"'s Skill loading controls, then start a new conversation. Loading has not been verified in this session.' 'Available instructions: images, video, audio/music, Jev evaluation, embeddings, reranking, models and balance. Connection and task results remain separate checks.' 'Example (not executed): Generate an image of a lake at sunset.' 'Example (not executed): Generate a 5-second video of a paper boat on a river.' 'Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.'
      ;;
  esac
}
# /generated:install-activation

init_json_string() {
  field=$1
  document=$2
  printf '%s' "$document" | sed -n "s/.*\"$field\":\"\([^\"]*\)\".*/\1/p" | sed -n '1p'
}

init_json_number() {
  field=$1
  document=$2
  printf '%s' "$document" | sed -n "s/.*\"$field\":\([0-9][0-9]*\).*/\1/p" | sed -n '1p'
}

init_target() {
  target_root=$1
  host_id=$2
  if ! (validate_target "$target_root") >/dev/null 2>&1; then
    printf '%s\n' "Pure Tokens Skill init: installed files could not be checked in this host session."
    printf '%s\n' "init_status=unverified stage=init error_code=installed_files_unreadable next_step=review_install_directory_access"
  elif [ -n "$host_id" ]; then
    printf '%s\n' "Installation host: $host_id"
    executor="$target_root/.puretokens-executor/puretokens-api"
    init_output=
    init_status=0
    init_output=$("$executor" init --host "$host_id" 2>/dev/null) || init_status=$?
    if [ "$init_status" -eq 0 ] &&
      printf '%s' "$init_output" | grep -Fq '"configuration_status":"verified"' &&
      printf '%s' "$init_output" | grep -Fq '"api_identity_confirmed":true' &&
      printf '%s' "$init_output" | grep -Fq '"credential_verified":true'; then
      printf '%s\n' "Pure Tokens Skill init: fixed API identity verified for the current host."
      printf '%s\n' "init_status=verified"
    elif [ "$init_status" -ne 0 ] && [ -z "$init_output" ]; then
      printf '%s\n' "Pure Tokens Skill init: configuration could not be checked in this host session."
      printf '%s\n' "init_status=unverified stage=init error_code=executor_unavailable next_step=review_host_execution_permissions; check whether this session permits the installed native executor to run. The exact cause is unconfirmed; Full Access is not a default requirement."
    else
      init_state=$(init_json_string "configuration_status" "$init_output")
      init_message=$(init_json_string "message" "$init_output")
      init_next=$(init_json_string "next_action" "$init_output")
      init_http_status=$(init_json_number "http_status" "$init_output")
      [ -n "$init_state" ] || init_state="unverified"
      if [ "$init_state" = verified ]; then
        init_state=unverified
        init_message="The verification receipt did not confirm both API identity and credential authentication."
      fi
      [ -n "$init_message" ] || init_message="The fixed API identity is not verified in this host session."
      printf '%s\n' "Pure Tokens Skill init: $init_message [$init_state]"
      printf '%s\n' "init_status=unverified"
      if [ -n "$init_http_status" ]; then
        printf '%s\n' "HTTP status: $init_http_status"
      fi
      if [ -n "$init_next" ]; then
        printf '%s\n' "Next: $init_next"
      fi
    fi
  else
    printf '%s\n' "Pure Tokens Skill init: host ID was not supplied, so the connection check was deferred."
    printf '%s\n' "init_status=deferred next_step=confirm_current_client"
  fi
  printf '%s\n' ""
  installation_summary "$host_id"
}

restore_transaction() (
  recovery_root=$1
  recovery_stage=$2
  "$ownership_guard" install-transaction-verify --directory "$recovery_stage" --target-directory "$recovery_root" --source-directory "$source_root" || return 1
  while read -r action entry; do
    case "$entry" in puretokens-balance|puretokens-connection|puretokens-models|puretokens-image|puretokens-video|puretokens-audio|puretokens-evaluate|puretokens-embedding|puretokens-rerank|puretokens-update|.puretokens-executor) ;; *) return 1 ;; esac
    case "$entry" in */*|*..*) return 1 ;; esac
    if [ "$action" = replace ] && [ -e "$recovery_stage/backup/$entry" ]; then
      if [ -e "$recovery_root/$entry" ] || [ -L "$recovery_root/$entry" ]; then
        "$ownership_guard" install-verify --directory "$recovery_root/$entry" --name "$entry" --source-directory "$source_root/skills/$entry" || return 1
        rm -rf -- "$recovery_root/$entry" || return 1
      fi
      mv "$recovery_stage/backup/$entry" "$recovery_root/$entry" || return 1
    elif [ "$action" = create ] && [ ! -e "$recovery_stage/$entry" ]; then
      if [ -e "$recovery_root/$entry" ] || [ -L "$recovery_root/$entry" ]; then
        "$ownership_guard" install-verify --directory "$recovery_root/$entry" --name "$entry" --source-directory "$source_root/skills/$entry" || return 1
        rm -rf -- "$recovery_root/$entry" || return 1
      fi
    fi
  done < "$recovery_stage/plan"
)

remove_completed_stage() {
  "$ownership_guard" install-transaction-verify --directory "$1" --target-directory "$target_root" --source-directory "$source_root" || {
    printf '%s\n' "Transaction cleanup failed verification; all remaining files were preserved." >&2
    return 1
  }
  if ! rm -rf -- "$1" 2>/dev/null; then
    printf '%s\n' "cleanup_status: pending; completed transaction files were retained. Host cleanup permission is required; do not delete unknown directories or reinstall solely for this warning."
  fi
}

finish_transaction() {
  code=$?
  trap - EXIT HUP INT TERM
  if [ -n "${stage_root:-}" ] && [ -d "$stage_root" ]; then
    if [ -f "$stage_root/committed" ] || restore_transaction "$target_root" "$stage_root" >/dev/null 2>&1; then
      if [ "$installation_status" != completed ] && [ "$installed_files_changed" != false ]; then
        installed_files_changed=false
        printf '%s\n' "recovery_status=restored installation_status=not_completed installed_files_changed=false; the previous managed files were restored."
      fi
      remove_completed_stage "$stage_root" || code=1
    else
      printf '%s\n' "recovery_status=incomplete installation_status=$installation_status installed_files_changed=$installed_files_changed next_step=inspect_installation; retained the managed recovery stage. Installation state needs explicit local diagnosis; do not reinstall or retry automatically." >&2
      code=1
    fi
  fi
  release_update_lock
  report_unexpected_failure "$code"
  exit "$code"
}

read_lock_owner() {
  lock_record=$(cat "$1" 2>/dev/null) || return 1
  case "$lock_record" in
    *" "*) lock_owner=${lock_record%% *}; lock_token=${lock_record#* } ;;
    *) lock_owner=$lock_record; lock_token="legacy-$lock_owner" ;;
  esac
  case "$lock_owner" in ''|*[!0-9]*) return 1 ;; esac
  case "$lock_token" in ''|*[!A-Za-z0-9._-]*) return 1 ;; esac
}

owns_update_lock() (
  lock_slot="$lock_root/pid"
  lock_seen=" "
  while read_lock_owner "$lock_slot"; do
    case "$lock_seen" in *" $lock_token "*) return 1 ;; esac
    lock_seen="$lock_seen$lock_token "
    if [ "$lock_owner" = "$$" ] && [ "$lock_token" = "${lock_candidate##*/}" ]; then return 0; fi
    lock_slot="$lock_root/next.$lock_token"
  done
  return 1
)

release_update_lock() {
  if owns_update_lock && ! rm -rf -- "$lock_root" 2>/dev/null; then
    printf '%s\n' "cleanup_status: pending; the installation lock was retained. Resolve host file access before another installation; do not retry automatically."
  fi
}

claim_update_lock() (
  ln "$lock_candidate" "$lock_slot" 2>/dev/null && return 0
  [ ! -e "$lock_slot" ] && [ ! -L "$lock_slot" ] || return 1
  # Filesystems without hard links still support exclusive creation. Never
  # replace a competing record; an interrupted partial write stops recovery.
  set -C
  cat "$lock_candidate" > "$lock_slot"
)

acquire_update_lock() {
  lock_root="$target_root/.puretokens-install-lock"
  if [ "$no_recovery" = true ]; then verify_no_transaction; fi
  mkdir "$lock_root" 2>/dev/null || [ -d "$lock_root" ] || fail "cannot create the update lock"
  for lock_entry in "$lock_root"/* "$lock_root"/.[!.]* "$lock_root"/..?*; do
    [ -e "$lock_entry" ] || [ -L "$lock_entry" ] || continue
    case "${lock_entry##*/}" in
      pid|owner.*|next.*) [ -f "$lock_entry" ] && [ ! -L "$lock_entry" ] || fail "unrecognized update lock contents; left untouched" ;;
      *) fail "unrecognized update lock contents; left untouched" ;;
    esac
  done
  lock_candidate=$(mktemp "$lock_root/owner.$$.XXXXXX" 2>/dev/null) || fail "cannot prepare the update lock; check directory write permission and available disk space"
  { printf '%s %s\n' "$$" "${lock_candidate##*/}" > "$lock_candidate"; } 2>/dev/null || fail "cannot write the update lock; check directory write permission and available disk space"
  lock_slot="$lock_root/pid"
  lock_seen=" "
  # Publish complete, immutable owner records with an atomic hard link. A dead
  # owner has exactly one successor, so simultaneous recovery cannot replace a
  # live lock. An interrupted takeover is simply another dead owner to follow.
  while ! claim_update_lock 2>/dev/null; do
    if ! read_lock_owner "$lock_slot"; then
      rm -f -- "$lock_candidate"
      fail "another installation is starting or the update lock needs inspection"
    fi
    case "$lock_seen" in *" $lock_token "*) rm -f -- "$lock_candidate"; fail "invalid update lock history; left untouched" ;; esac
    lock_seen="$lock_seen$lock_token "
    if kill -0 "$lock_owner" 2>/dev/null; then
      rm -f -- "$lock_candidate"
      fail "another installation is in progress; wait for it to finish"
    fi
    lock_slot="$lock_root/next.$lock_token"
  done
  # A previous owner may have finished and removed its directory while this
  # process was reading it. Only a claim reachable from the current root owns it.
  if ! owns_update_lock; then
    rm -f -- "$lock_candidate"
    fail "another installation acquired the update lock"
  fi
  stage_root=
  trap finish_transaction EXIT
  trap 'exit 130' INT
  trap 'exit 129' HUP
  trap 'exit 143' TERM
  if [ "$command_name" = repair ]; then
    for entry in $current_skills .puretokens-executor; do
      if [ -e "$target_root/$entry" ] || [ -L "$target_root/$entry" ]; then
        "$ownership_guard" install-repair-verify --directory "$target_root/$entry" --name "$entry" || fail "repair stopped: local additions, changes or unknown ownership require review; existing files were preserved"
      fi
    done
  fi
  for previous_stage in "$target_root"/.puretokens-skill-stage.*; do
    [ -e "$previous_stage" ] || [ -L "$previous_stage" ] || continue
    [ -d "$previous_stage" ] && [ ! -L "$previous_stage" ] || fail "an unrecognized staging entry needs inspection; left untouched"
    [ -f "$previous_stage/transaction-v1" ] || fail "an unrecognized staging directory needs inspection; left untouched"
    "$ownership_guard" install-transaction-verify --directory "$previous_stage" --target-directory "$target_root" --source-directory "$source_root" || fail "previous update recovery failed verification; all transaction files were preserved"
  done
  for previous_stage in "$target_root"/.puretokens-skill-stage.*; do
    [ -d "$previous_stage" ] || continue
    installed_files_changed=unknown
    if [ -f "$previous_stage/committed" ] || restore_transaction "$target_root" "$previous_stage" >/dev/null 2>&1; then
      rm -rf -- "$previous_stage" 2>/dev/null || fail "previous transaction cleanup is pending; resolve host file access before continuing"
    else
      fail "previous update recovery failed; retained its backup"
    fi
  done
}

sync_target() {
  source_root=$1
  target_root=$2
  release_version=$(source_release_version "$source_root") || fail "official source has an invalid Skill version"
  installation_status=not_completed
  stage=prepare_target
  [ -d "$target_root" ] || local_io mkdir -p "$target_root"
  target_root=$(cd "$target_root" 2>/dev/null && pwd -P) || fail "target directory could not be accessed; check current-session directory permissions"
  check_gemini_duplicates
  ownership_guard=$(executor_artifact "$source_root")
  stage=acquire_lock
  acquire_update_lock
  stage=verify_installed
  verify_owned() {
    verification_command=install-verify
    [ "$command_name" != repair ] || verification_command=install-repair-verify
    set -- "$verification_command" --directory "$1" --name "$2" --source-directory "${3:-$source_root/skills/$2}"
    "$ownership_guard" "$@" ||
      fail "installation ownership or file changes require review: $5; existing files were preserved"
  }
  # Downloads happen outside this lock. Re-read the installed version after
  # acquiring it so an older download cannot overwrite a newer completed sync.
  installed_manifest="$target_root/.puretokens-executor/runtime.json"
  if managed_executor "$target_root/.puretokens-executor"; then
    installed_version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([0-9][0-9.]*\)".*/\1/p' "$installed_manifest" | sed -n '1p')
    printf '%s\n' "$installed_version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || fail "installed executor version is invalid; left untouched"
    if awk -v current="$installed_version" -v available="$release_version" 'BEGIN {split(current,c,"."); split(available,a,"."); for(i=1;i<=3;i++){if(c[i]+0>a[i]+0)exit 0;if(c[i]+0<a[i]+0)exit 1}exit 1}'; then
      fail "a newer executor version is already installed; downgrade was stopped under the update lock"
    fi
  fi

  for name in $current_skills; do
    destination="$target_root/$name"
    if [ -e "$destination" ] || [ -L "$destination" ]; then verify_owned "$destination" "$name" "$source_root/skills/$name"; fi
  done
  executor_destination="$target_root/.puretokens-executor"
  if [ -e "$executor_destination" ] || [ -L "$executor_destination" ]; then verify_owned "$executor_destination" .puretokens-executor; fi

  stage=stage_files
  stage_root=$(mktemp -d "$target_root/.puretokens-skill-stage.XXXXXX" 2>/dev/null) || fail "could not create a private update staging directory; check directory write permission and available disk space"
  local_io touch "$stage_root/transaction-v1"
  [ "$command_name" != repair ] || local_io touch "$stage_root/repair-v1"
  local_io mkdir -p "$stage_root/backup"
  for name in $current_skills; do local_io cp -R "$source_root/skills/$name" "$stage_root/$name"; done
  executor_source=$(executor_artifact "$source_root")
  local_io mkdir -p "$stage_root/.puretokens-executor"
  local_io cp "$executor_source" "$stage_root/.puretokens-executor/puretokens-api"
  local_io cp "$source_root/runtime/puretokens-skill-install.sh" "$source_root/runtime/puretokens-skill-fetch.sh" "$stage_root/.puretokens-executor/"
  local_io chmod 700 "$stage_root/.puretokens-executor/puretokens-api"
  { printf '{\n  "schemaVersion": 1,\n  "name": "puretokens-api-executor",\n  "version": "%s",\n  "platform": "%s"\n}\n' "$release_version" "$(executor_platform)" > "$stage_root/.puretokens-executor/runtime.json"; } 2>/dev/null || fail "cannot write the staged executor metadata; check directory permissions and available disk space"
  for entry in $current_skills .puretokens-executor; do
    { "$ownership_guard" install-inventory --directory "$stage_root/$entry" --name "$entry" > "$stage_root/$entry/.puretokens-managed.json"; } 2>/dev/null ||
      fail "could not record managed files for $entry"
  done


  { : > "$stage_root/plan"; } 2>/dev/null || fail "cannot create the transaction plan; check directory write permission and available disk space"
  for entry in $current_skills .puretokens-executor; do
    if [ -e "$target_root/$entry" ]; then action=replace; else action=create; fi
    { printf '%s %s\n' "$action" "$entry" >> "$stage_root/plan"; } 2>/dev/null || fail "cannot write the transaction plan; check directory write permission and available disk space"
  done
  stage=replace_files
  while read -r action entry; do
    if [ "$action" = replace ]; then
      verify_owned "$target_root/$entry" "$entry" "$source_root/skills/$entry"
      installed_files_changed=unknown
      local_io mv "$target_root/$entry" "$stage_root/backup/$entry"
    elif [ -e "$target_root/$entry" ] || [ -L "$target_root/$entry" ]; then
      fail "a destination appeared during installation; existing files were preserved"
    fi
    installed_files_changed=unknown
    local_io mv "$stage_root/$entry" "$target_root/$entry"
  done < "$stage_root/plan"
  stage=verify_synced
  for entry in $current_skills .puretokens-executor; do
    "$ownership_guard" install-verify --directory "$target_root/$entry" --name "$entry" || fail "installed managed files could not be verified"
  done
  local_io touch "$stage_root/committed"
  installation_status=completed
  installed_files_changed=true
  printf '%s\n' "Pure Tokens Skills $release_version synchronized with the native API executor at $target_root"
  printf '%s\n' "installation_status=completed installed_files_changed=true"
  remove_completed_stage "$stage_root"
  stage_root=
  release_update_lock
  trap - EXIT HUP INT TERM
  # Optional anonymous completion receipt. No credentials, paths or host data;
  # failure must not alter the committed install or connection check.
  if [ "${PTP_OPERATIONS_RECEIPTS:-}" = "1" ]; then
    "$target_root/.puretokens-executor/puretokens-api" operations-receipt installed >/dev/null 2>&1 || :
  fi
  stage=init
  init_target "$target_root" "$host"
}

command_name=${1:-}
[ -n "$command_name" ] || { usage; exit 2; }
shift
target=
host=
source=
release_manifest=
no_recovery=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --target) [ "$#" -ge 2 ] || fail "--target requires an absolute directory"; target=$2; shift 2 ;;
    --host) [ "$#" -ge 2 ] || fail "--host requires a supported host ID"; host=$2; shift 2 ;;
    --source) [ "$#" -ge 2 ] || fail "--source requires an absolute official source directory"; source=$2; shift 2 ;;
    --release-manifest) [ "$#" -ge 2 ] || fail "--release-manifest requires a file"; release_manifest=$2; shift 2 ;;
    --no-recovery) no_recovery=true; shift ;;
    *) usage; exit 2 ;;
  esac
done

[ -n "$target" ] || [ -n "$host" ] || fail "--host or --target is required"
case "$host" in ''|claude-code|codex|workbuddy|gemini-cli|grok-build|opencode|trae|claude-desktop|dsh-desktop|deepseek-harness|minimax-code|zcode|kimi-code|qoder|pi|hermes|evox|vscode|octop) ;; *) fail "unsupported host" ;; esac
case "$host" in pi|hermes|evox) target_for_host "$host" >/dev/null ;; esac
if [ "$host" = octop ] && [ -n "$target" ]; then
  validate_client_root "$target"
  case "$target" in */.octop/skills|*/skills) ;; *) fail "Octop target must be the current local workspace's .octop/skills or skills directory" ;; esac
fi
stage=select_target
[ -n "$target" ] || target=$(target_for_host "$host") || { failure_reported=true; exit 1; }
[ "${target#/}" != "$target" ] || fail "--target must be an absolute Skill directory"
[ -z "$source" ] || [ "${source#/}" != "$source" ] || fail "--source must be an absolute official source directory"
case "$command_name" in check|verify-installed|init|sync|repair|locate) ;; *) usage; exit 2 ;; esac
[ -z "$release_manifest" ] || { [ "$command_name" = verify-installed ] && [ -f "$release_manifest" ]; } || fail "release manifest is only supported for installed verification"

if [ "$command_name" = locate ]; then
  printf '%s\n' "$target"
  exit 0
fi

if [ "$command_name" = "verify-installed" ]; then
  stage=verify_installed
  verify_installed "$target"
  exit 0
fi

if [ "$command_name" = "init" ]; then
  stage=init
  [ -n "$target" ] || fail "--target or --host is required for init"
  [ "${target#/}" != "$target" ] || fail "--target must be an absolute Skill directory"
  target_root=$(cd "$target" 2>/dev/null && pwd -P) || fail "could not resolve --target"
  init_target "$target_root" "$host"
  exit 0
fi

stage=verify_source
if [ -n "$source" ]; then
  [ -d "$source" ] || fail "--source does not exist"
  source_root=$(cd "$source" 2>/dev/null && pwd -P) || fail "could not resolve --source"
  validate_source "$source_root"
elif bundled_source_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." 2>/dev/null && pwd -P) && [ -f "$bundled_source_root/README.md" ] && [ -f "$bundled_source_root/package.json" ] && [ -d "$bundled_source_root/skills" ]; then
  source_root=$bundled_source_root
  validate_source "$source_root"
else
  fail "run this source-only sync script from a fresh official Pure Tokens Skills main checkout or pass --source <absolute-checkout-directory>"
fi
if [ "$command_name" = "check" ]; then
  printf '%s\n' "Official Pure Tokens Skill source passed native static validation."
else
  sync_target "$source_root" "$target"
fi
