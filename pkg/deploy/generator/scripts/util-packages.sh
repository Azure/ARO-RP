#!/bin/bash
# Repository and package management related functions

# configure_rpm_repos
# New repositories should be added in their own functions, and called here
# args:
# 1) wait_time - nameref, integer; Time to wait before retrying command
# 2) retries - integer, optional; Amount of times to retry command, defaults to 5
configure_rpm_repos() {
    log "starting"

    configure_rhui_repo "$1" "${2:-}"
    create_azure_rpm_repos
}

# dnf_install_pkgs
# args:
# 1) pkgs - nameref, string array; Packages to be installed
# 2) wait_time - nameref, integer; Time to wait before retrying command
# 3) retries - integer, optional; Amount of times to retry command, defaults to 5
dnf_install_pkgs() {
    local -n pkgs="$1"
    log "starting"

    local -a cmd=(
        dnf
        -y
        install
    )
    
    # Reference: https://www.shellcheck.net/wiki/SC2206
    # append pkgs array to cmd
    mapfile -O $(( ${#cmd[@]} + 1 )) -d ' ' cmd <<< "${pkgs[@]}"
    local -r cmd

    log "Attempting to install packages: ${pkgs[*]}"
    retry cmd "$2" "${3:-}"
}


# dnf_update_pkgs
# args:
# 1) excludes - nameref, string array, optional; Packages to exclude from updating
#       Each index must be prefixed with -x 
# 2) wait_time - nameref, integer; Time to wait before retrying command
# 3) retries - integer, optional; Ammount of times to retry command, defaults to 5
dnf_update_pkgs() {
    local -n excludes="${1:-empty_str}"
    log "starting"

    local -a cmd=(
        dnf
        -y
        # Replaced with excludes
        ""
        update
        --allowerasing
    )

    if [ -n "${excludes}" ]; then
        # Reference https://www.shellcheck.net/wiki/SC2206
        mapfile -O 2 cmd <<< "${excludes[@]}"
    else
        # Remove empty string if we aren't replacing them, probably doesn't matter, but why not be safe
        unset "cmd[2]"
    fi
    local -r cmd

    log "Updating all packages excluding \"${excludes[*]:-}\""
    retry cmd "$2" "${3:-}"
}


# create_azure_rpm_repos creates /etc/yum.repos.d/azure.repo repository file
create_azure_rpm_repos() {
    log "starting"

    local -r azure_repo_filename='/etc/yum.repos.d/azure.repo'
    local -r azure_repo_file='[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes

[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no'

    write_file azure_repo_filename azure_repo_file true
}

# configure_rhui_repo enables all rhui-microsoft-azure* repos
# args:
# 1) wait_time - nameref, integer; Time to wait before retrying command
# 2) retries - integer, optional; Amount of times to retry command, defaults to 5
configure_rhui_repo() {
    log "starting"

    local -ra cmd=(
        dnf
        update
        -y
        --disablerepo='*'
        --enablerepo='rhui-microsoft-azure*'
    )

    log "running RHUI package updates"
    retry cmd "$1" "${2:-}"
}

# configure_dnf_cron_job
# create cron job to auto update rpm packages
configure_dnf_cron_job() {
    log "starting"
    local -r cron_weekly_dnf_update_filename='/etc/cron.weekly/dnfupdate'
    local -r cron_weekly_dnf_update_file="#!/bin/bash
dnf update -y"

    write_file cron_weekly_dnf_update_filename cron_weekly_dnf_update_file true
    chmod u+x "$cron_weekly_dnf_update_filename"
}

# rpm_import_keys
# args:
# 1) keys - nameref, string array; rpm keys to be imported
# 2) wait_time - nameref, integer; Time to wait before retrying command
rpm_import_keys() {
    local -n keys="$1"
    log "starting"

    # shellcheck disable=SC2068
    for key in ${keys[@]}; do
        if [ ${#keys[@]} -eq 0 ]; then
            break
        fi
            local -a cmd=(
                rpm
                --import
                -v
                "$key"
            )

            log "Importing rpm repository key $key"
            retry cmd "$2" "${3:-}" && unset key
    done
}
