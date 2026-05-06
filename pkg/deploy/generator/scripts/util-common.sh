#!/bin/bash
# Internal Functions and Constants

# CSE_LOG_FILE - persistent log file for debugging CSE failures
# Logs are written here AND to stdout for Azure diagnostics
declare -r CSE_LOG_FILE="/var/log/azure/aro-vmss-setup.log"

# Initialize log file and directory
setup_logging() {
    mkdir -p "$(dirname "$CSE_LOG_FILE")"
    touch "$CSE_LOG_FILE"
    chmod 644 "$CSE_LOG_FILE"
    echo "=== ARO VMSS Setup Started: $(date -u '+%Y-%m-%d %H:%M:%S UTC') ===" | tee -a "$CSE_LOG_FILE"
    echo "Hostname: $(hostname)" | tee -a "$CSE_LOG_FILE"
    echo "Script: $0" | tee -a "$CSE_LOG_FILE"
    echo "======================================" | tee -a "$CSE_LOG_FILE"
}

# ERR trap handler - logs which command failed before exit
err_handler() {
    local -r exit_code=$?
    local -r line_number=$1
    local -r bash_lineno=$2
    local -r last_command="$3"
    local -r func_name="${FUNCNAME[1]:-main}"

    local err_msg="ERROR: Command failed with exit code $exit_code"
    err_msg="$err_msg\n  Function: $func_name"
    err_msg="$err_msg\n  Line: $line_number (BASH_LINENO: $bash_lineno)"
    err_msg="$err_msg\n  Command: $last_command"
    err_msg="$err_msg\n  Timestamp: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"

    echo -e "$err_msg" | tee -a "$CSE_LOG_FILE" >&2
    echo "=== VMSS Setup FAILED ===" | tee -a "$CSE_LOG_FILE" >&2

    exit "$exit_code"
}

# Set up error trap - must be called after errexit is set
setup_error_trap() {
    # shellcheck disable=SC2016
    trap 'err_handler ${LINENO} ${BASH_LINENO} "$BASH_COMMAND"' ERR
}

# declare -r empty_str=""
#
# empty_str - constant
#   * used by functions for optional nameref string arguments
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
# role_devproxy - constant
#   * Is used to determine which VMSS is being bootstrapped
declare -r role_devproxy="devproxy"

# declare -r us_gov_cloud="AzureUSGovernment"
#
# us_gov_cloud - constant
#   * Is the name of AZURECLOUDNAME for US government cloud
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
# set/unset xtrace shell option
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
# Wrapper for echo that includes timestamp, function name, and writes to persistent log
# args:
#   1) msg - string
#   2) stack_level - int
#       * optional
#       * defaults to the function at the bottom of the call stack
log() {
    local -r msg="${1:-"log message is empty"}"
    local -r stack_level="${2:-1}"
    local -r timestamp="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    local -r func_name="${FUNCNAME[${stack_level}]}"
    local -r log_line="[$timestamp] $func_name: $msg"

    # Write to both stdout and persistent log file
    echo "$log_line" | tee -a "$CSE_LOG_FILE"
}

# abort()
#
# Wrapper for log that exits with an error code
# Logs to both stdout and persistent log file
abort() {
    local -ri origin_stacklevel=2
    local -r timestamp="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"

    echo "[$timestamp] ABORT: ${FUNCNAME[1]}: ${1}" | tee -a "$CSE_LOG_FILE" >&2
    echo "[$timestamp] Stack trace:" | tee -a "$CSE_LOG_FILE" >&2

    # Print stack trace for debugging
    local i=0
    while caller $i 2>/dev/null | tee -a "$CSE_LOG_FILE" >&2; do
        ((i++))
    done

    echo "[$timestamp] === VMSS Setup ABORTED ===" | tee -a "$CSE_LOG_FILE" >&2
    exit 1
}

# write_file()
#
# args:
#   1) filename - string
#   2) file_contents - string
#   3) clobber - boolean
#       * Optional; defaults to false
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
        log "Retry attempt #${attempt}/${retries} for: ${cmd_retry[*]}"
        # shellcheck disable=SC2068
        ${cmd_retry[@]} &

        if wait -f $!; then
            log "Command succeeded on attempt #${attempt}: ${cmd_retry[*]}"
            return 0
        fi

        local -r exit_code=$?
        log "Command failed with exit code ${exit_code} on attempt #${attempt}: ${cmd_retry[*]}"

        if [ "$attempt" -lt "$retries" ]; then
            log "Waiting ${wait_time} seconds before retry..."
            sleep "$wait_time"
        fi
    done

    abort "Command failed after ${retries} attempts: ${cmd_retry[*]}"
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
#   1) rl - nameref, string
#       * role to get short role for
#   2) kv_suffix - nameref, string
#       * short role will be assigned to this nameref
#   3) sec_prefix - nameref, string
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
            abort "unknown role $rl"
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
