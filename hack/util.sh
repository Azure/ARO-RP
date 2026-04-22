# shellcheck shell=bash

# declare -r __hack_util_sourced="true"
declare -r __hack_util_sourced="true"

[ "${XTRACE:-'false'}" == "true" ] && set -x

## Log level colors

# declare -r COLOR_RED='\e[31m'
declare -r COLOR_RED='\e[31m'
# declare -r COLOR_RED_BG='\033[41m'
declare -r COLOR_RED_BG='\033[41m'
# declare -r COLOR_GREEN='\e[32m'
declare -r COLOR_GREEN='\e[32m'
# declare -r COLOR_YELLOW='\e[33m'
declare -r COLOR_YELLOW='\e[33m'
# declare -r COLOR_BLUE='\e[34m'
# shellcheck disable=SC2034
declare -r COLOR_BLUE='\e[34m'
# declare -r COLOR_CYAN='\e[36m'
declare -r COLOR_CYAN='\e[36m'
# declare -r COLOR_RESET='\e[0m'
#
# Reset color
declare -r COLOR_RESET='\e[0m'

## Log levels

# declare -ri LOG_LEVEL_CRIT=2
declare -ri LOG_LEVEL_CRIT=2
# declare -ri LOG_LEVEL_ERR=3
declare -ri LOG_LEVEL_ERR=3
# declare -ri LOG_LEVEL_WARN=4
declare -ri LOG_LEVEL_WARN=4
# declare -ri LOG_LEVEL_INFO=6
declare -ri LOG_LEVEL_INFO=6
# declare -ri LOG_LEVEL_DEBUG=7
declare -ri LOG_LEVEL_DEBUG=7

# declare -r LOG_LEVEL="${LOG_LEVEL:-$LOG_LEVEL_INFO}"
declare -r LOG_LEVEL="${LOG_LEVEL:-$LOG_LEVEL_INFO}"

# log()
# log is a wrapper for echo that includes the function name
#
# args:
#   * 1) msg - string
#   * 2) stack_level - int; optional, defaults to calling function
log() {
    local -r msg_prefix="${1:-}"
    local -r msg="${2:-}"
    local -r stack_level="${3:-1}"

    echo -e "${msg_prefix} ${FUNCNAME[${stack_level}]}: ${msg}"
}

# error()
error() {
    if (( LOG_LEVEL_ERR <= LOG_LEVEL )); then
        log "${COLOR_RED}ERROR${COLOR_RESET}[$(date -Iminutes)]" "$1" 2
    fi
}

# warn()
warn() {
    if (( LOG_LEVEL_WARN <= LOG_LEVEL )); then
        log "${COLOR_YELLOW}WARN${COLOR_RESET}[$(date -Iminutes)]" "$1" 2
    fi
}

# info()
info() {
    if (( LOG_LEVEL_INFO <= LOG_LEVEL )); then
        log "${COLOR_CYAN}INFO${COLOR_RESET}[$(date -Iminutes)]" "$1" 2
    fi
}

debug() {
    if (( LOG_LEVEL_DEBUG <= LOG_LEVEL )); then
        log "${COLOR_GREEN}INFO${COLOR_RESET}[$(date -Iminutes)]" "$1" 2
    fi
}

# fatal is a wrapper for log followed by exit code 1
#
# args:
#   * 1) message - string; error message passed to log
fatal() {
    if (( LOG_LEVEL_CRIT <= LOG_LEVEL )); then
        prefix="${COLOR_RED_BG}FATAL${COLOR_RESET}[$(date -Iminutes)]"
    fi

    log  "$prefix" "$1" 2
    log "$prefix" "Exiting"
    exit 1
}

# cleanup() {
# Deletes all files and directories provided.
#
# args:
#   * @) tmp - nameref, array; files and/or directories to be deleted
cleanup() {
    for tmp in $@; do
        if [[ -d "$tmp" || -f "$tmp" ]]; then
            info "Deleting $tmp"
            rm -fr "$tmp"
        fi
    done
}

# retry()
#
# retry commands
# args:
#   * 1) cmd_retry_str - string; Command and all it's options/arguments in a single string
#   * 2) wait - int; Optional - sleep time, Defaults to 5m
#   * 3) retries - integer; Optional - Amount of times to retry command, defaults to 5
retry() {
    local cmd_retry_str="$1"
    local wait="${2:-"5m"}"
    local -ri retries="${3:-5}"

    mapfile -t cmd_retry <<< "$cmd_retry_str"

    for attempt in $(seq 1 $retries); do
        info "attempt #${attempt} - ${cmd_retry[*]}"
        # shellcheck disable=SC2068
        ${cmd_retry[@]} &

        wait -f $! && return 0
        sleep "$wait"
    done

    error "${cmd_retry[*]} failed after #$retries attempts"
}
