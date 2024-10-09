#!/bin/bash

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

# log is a wrapper for echo that includes the function name
# Args
# 1) msg - string
# 2) stack_level - int; optional, defaults to calling function
log() {
    local -r msg="${1:-"log message is empty"}"
    local -r stack_level="${2:-1}"
    echo "${FUNCNAME[${stack_level}]}: ${msg}"
}

# abort is a wrapper for log that exits with an error code
abort() {
    local -ri origin_stacklevel=2
    log "${1}" "$origin_stacklevel"
    log "Exiting"
    exit 1
}

# abort_directory is a wrapper of abort that aborts when the go.mod and .git directory are missing from the current directory
abort_directory() {
    if [ ! -f go.mod ] || [ ! -d ".git" ]; then
        abort "this script must by run from the repo's root directory"
    fi
}
