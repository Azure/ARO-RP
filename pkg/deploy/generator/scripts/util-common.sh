#!/bin/bash
# Internal Functions and Constants

# empty_str - constant; used by functions for optional nameref string arguements
# shellcheck disable=SC2034
declare -r empty_str=""

# role_gateway is used to determine which VMSS is being bootstrapped
# this should be referenced by scripts sourcing this file
declare -r role_gateway="gateway"
# role_rp is used to determine which VMSS is being bootstrapped
# this should be referenced by scripts sourcing this file
declare -r role_rp="rp"

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

# write_file
# Args
# 1) filename - string
# 2) file_contents - string
# 3) clobber - boolean; optional - defaults to false
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

# retry Adding retry logic to yum commands in order to avoid stalling out on resource locks
# args:
# 1) cmd_retry - nameref, array; Command and arguement(s) to retry
# 2) wait_time - nameref, integer; Time to wait before retrying command
# 3) retries - integer, optional; Ammount of times to retry command, defaults to 5
retry() {
    local -n cmd_retry="$1"
    local -n wait_time="$2"
    local -ri retries="${3:-5}"

    for attempt in {1..5}; do
        log "attempt #${attempt} - ${FUNCNAME[2]}"
        # shellcheck disable=SC2068
        ${cmd_retry[@]} &

        wait $! && break
        if [ "${attempt}" -le "$retries" ]; then
            sleep "$wait_time"
        else
            abort "attempt #${attempt} - Failed to update packages"
        fi
    done
}

# verify_role
# args:
# 1) test_role - nameref; role being verified
# 2) certs - boolean, optional; defaults to false. Set to true to add devproxy to allowed roles
verify_role() {
    local -n test_role="$1"
    local -r certs="${2:-false}"

    allowed_roles_glob="($role_rp|$role_gateway)"
    if $certs; then
        # remove trailing ")" and append additional role
        allowed_roles_glob="${allowed_roles_glob%\)*}|devproxy)"
    fi

    if [[ "$test_role" =~ $allowed_roles_glob ]]; then
        log "Verified role \"$test_role\""
    else
        abort "failed to verify role, role \"${test_role}\" not in \"${allowed_roles_glob}\""
    fi
}

# get_keyvault_suffix
# args:
# 1) rl - nameref, string; role to get short role for
# 2) kv_suffix - nameref, string; short role will be assigned to this nameref
# 3) sec_prefix - nameref, string; keyvault certificate prefix will be assigned to this nameref
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

# reboot_vm restores all selinux file contexts, then schedules a reboot for one hour later
# Reboots should scheduled after all VM extensions have had time to complete
# Reference: https://learn.microsoft.com/en-us/azure/virtual-machines/extensions/custom-script-linux#tips
reboot_vm() {
    log "starting"

    (shutdown -r now &)
}
