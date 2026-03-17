#!/bin/bash
# Internal Functions and Constants

# declare -r empty_str=""
#
# empty_str - constant; used by functions for optional nameref string arguements
# shellcheck disable=SC2034
declare -r empty_str=""

# declare -r role_gateway="gateway"
#
# this should be referenced by scripts sourcing this file
# role_gateway="gateway"
declare -r role_gateway="gateway"

# declare -r role_rp="rp"
#
# this should be referenced by scripts sourcing this file
# role_rp="rp"
declare -r role_rp="rp"

# declare -r role_devproxy="devproxy"
#
# role_devproxy - constant; Is used to determine which VMSS is being bootstrapped
declare -r role_devproxy="devproxy"

# declare -r us_gov_cloud="AzureUSGovernment"
#
# us_gov_cloud - constant; Is the name of AZURECLOUDNAME for US government cloud
declare -r us_gov_cloud="AzureUSGovernment"

# declare -i XTRACE_SET=1
#
# constant value signifying xtrace shell value is/should be set
declare -ir XTRACE_SET=1

# declare -i XTRACE_UNSET=0
#
# constant value signifying xtrace shell value is/should be unset
declare -ir XTRACE_UNSET=0

# xtrace_is_set()
#
# Check if xtrace shell option is enabled/disabled
#   * Returns XTRACE_SET value if set
#   * Returns XTRACE_UNSET value if unset
xtrace_is_set() {
    if [[ $- =~ "x" ]]; then
        echo XTRACE_SET
    fi
    
    echo XTRACE_UNSET
}

# xtrace_toggle()
#
# sets/unsets xtrace shell option
# args:
#   1) string - nameref
#       * Must be XTRACE_SET or XTRACE_UNSET
xtrace_toggle() {
    if ! [[ $1 =~ ("XTRACE_SET"|"XTRACE_UNSET") ]]; then
        log "\$1 invalid; \$1 must be XTRACE_SET or XTRACE_UNSET. \$1: $1"
        return 1
    fi

    if (( $1 == XTRACE_SET )); then
        set -x 
    elif
        (( $1 == XTRACE_UNSET )); then
        set +x
    fi
}

# log()
#
# Wrapper for echo that includes the function name
# args:
#   1) msg - string
#   2) stack_level - int; optional
#       * defaults to the function at the bottom of the call stack
log() {
    local -r msg="${1:-"log message is empty"}"
    local -r stack_level="${2:-1}"
    echo "${FUNCNAME[${stack_level}]}: ${msg}"
}

# abort()
#
# Wrapper for log that exits with an error code
abort() {
    local -ri origin_stacklevel=2
    log "${1}" "$origin_stacklevel"
    log "Exiting"
    exit 1
}

# write_file()
#
# args:
#   1) filename - string
#   2) file_contents - string
#   3) clobber - boolean; optional
#       * defaults to false
write_file() {
    local -n filename="$1"
    local -n file_contents="$2"
    local -r clobber="${3:-false}"

    if $clobber; then
        log "Overwriting file $filename"
        echo "$file_contents" > "$filename"
    else
        log "Appending to $filename"
        echo "$file_contents" >> "$filename"
    fi
}

# retry()
#
# Add retry logic to commands in order to avoid stalling out on resource locks
# args:
#   1) cmd_retry - nameref, array
#       * Command and argument(s) to retry
#   2) wait_time - nameref, integer
#       * Time to wait before retrying command
#   3) retries - integer, optional
#       * Amount of times to retry command, defaults to 5
retry() {
    local -n cmd_retry="$1"
    local -n wait_time="$2"
    local -ri retries="${3:-5}"

    
    for attempt in $(seq 1 $retries); do
        log "attempt #${attempt} - ${FUNCNAME[2]}"
        # shellcheck disable=SC2068
        ${cmd_retry[@]} &

        wait -f $! && return 0
        sleep "$wait_time"
    done

    abort "${cmd_retry[*]} failed after #$retries attempts"
}

# verify_role()
#
# args:
#   1) test_role - nameref
#       * role being verified
verify_role() {
    local -n test_role="$1"

    allowed_roles_glob="($role_rp|$role_gateway|$role_devproxy)"
    if [[ "$test_role" =~ $allowed_roles_glob ]]; then
        log "Verified role \"$test_role\""
    else
        abort "failed to verify role, role \"${test_role}\" not in \"${allowed_roles_glob}\""
    fi
}

# get_keyvault_suffix()
#
# args:
#   1) rl - nameref, string;
#       * role to get short role for
#   2) kv_suffix - nameref, string;
#       * short role will be assigned to this nameref
#   3) sec_prefix - nameref, string;
#       * keyvault certificate prefix will be assigned to this nameref
get_keyvault_suffix() {
    local -n rl="$1"
    local -n kv_suffix="$2"
    local -n sec_prefix="$3"

    local -r keyvault_suffix_rp="svc"
    local -r keyvault_prefix_gateway="gwy"

    case "$rl" in
        "$role_gateway")
            kv_suffix="$keyvault_prefix_gateway"
            sec_prefix="$keyvault_prefix_gateway"
            ;;
        "$role_rp")
            kv_suffix="$keyvault_suffix_rp"
            sec_prefix="$role_rp"
            ;;
        *)
            abort "unkown role $rl"
            ;;
    esac
}

# reboot_vm()
#
# reboot_vm restores calls shutdown -r in a subshell
#   * Reboots should scheduled after all VM extensions have had time to complete
#   * Reference: https://learn.microsoft.com/en-us/azure/virtual-machines/extensions/custom-script-linux#tips
reboot_vm() {
    log "starting"

    (shutdown -r now &)
}
