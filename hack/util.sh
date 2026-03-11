# shellcheck shell=bash

# declare -r __hack_util_sourced="true"
declare -r __hack_util_sourced="true"

[ "${DEBUG:-false}" == true ] && set -x

# log()
# log is a wrapper for echo that includes the function name
#
# args:
#   * 1) msg - string
#   * 2) stack_level - int; optional, defaults to calling function
log() {
    local -r msg="${1:-"log message is empty"}"
    local -r stack_level="${2:-1}"

    echo "${FUNCNAME[${stack_level}]}: ${msg}"
}

# abort is a wrapper for log followed by exit code 1
#
# args:
#   * 1) message - string; error message passed to log
abort() {
    local -ri origin_stacklevel=2

    log "${1}" "$origin_stacklevel"
    log "Exiting"
    exit 1
}

# cleanup() {
# Deletes all files and directories provided.
#
# args:
#   * @) tmp - array; files and/or directories to be deleted
cleanup() {
    # shellcheck disable=SC2068
    for tmp in $@; do
        if [[ -d "$tmp" || -f "$tmp" ]]; then
            log "Deleting $tmp"
            rm -fr "$tmp"
        fi
    done
}
