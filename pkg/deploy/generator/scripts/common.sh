#!/bin/bash
# This file is intended to be sourced by bootstrapping scripts for commonly used functions

# retry Adding retry logic to yum commands in order to avoid stalling out on resource locks
retry() {
    local -n cmd_retry="$1"
    local -n wait_time="$2"

    for attempt in {1..5}; do
        log "attempt #${attempt} - ${FUNCNAME[2]}"
        ${cmd_retry[@]} &

        wait $! && break
        if [[ ${attempt} -lt 5 ]]; then
            sleep "$wait_time"
        else
            abort "attempt #${attempt} - Failed to update packages"
        fi
    done
}


# We need to configure PasswordAuthentication to yes in order for the VMSS Access JIT to work
configure_sshd() {
    log "starting"
    local sshd_config="/etc/ssh/sshd_config"

    log "Editing $sshd_config to allow password authentication"
    sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' "$sshd_config"

    systemctl reload sshd.service || abort "sshd failed to reload"
}

# dnf_update_pkgs
dnf_update_pkgs() {
    local -n excludes="$1"
    log "starting"

    local -ra cmd=(
        dnf
        -y
        ${excludes[@]}
        update
        --allowerasing
    )

    log "Updating all packages excluding ${excludes[*]}"
    retry cmd "$2"
}

# rpm_import_keys
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

            log "attempt #$attempt - importing rpm repository key $key"
            retry cmd "$2" && unset key
    done
}

# configure_selinux
configure_selinux() {
    log "starting"

    local -r relabel="${1:-false}"

    already_defined_ignore_error="File context for /var/log/journal(/.*)? already defined"
    semanage fcontext -a -t var_log_t "/var/log/journal(/.*)?" || log "$already_defined_ignore_error"
    chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

    if "$relabel"; then
        restorecon -RF /var/log/* || log "$already_defined_ignore_error"
    fi
}

# configure_firewalld_rules
configure_firewalld_rules() {
    local -n ports="$1"
    log "starting"

    # https://access.redhat.com/security/cve/cve-2020-13401
    local -r prefix="/etc/sysctl.d"
    local -r disable_accept_ra_conf_filename="$prefix/02-disable-accept-ra.conf"
    local -r disable_accept_ra_conf_file="net.ipv6.conf.all.accept_ra=0"

    write_file disable_accept_ra_conf_filename disable_accept_ra_conf_file true

    local -r disable_core_filename="$prefix/01-disable-core.conf"
    local -r disable_core_file="kernel.core_pattern = |/bin/true
    "
    write_file disable_core_filename disable_core_file true

    sysctl --system

    log "Enabling ports ${enable_ports[*]} on default firewalld zone"
    # shellcheck disable=SC2068
    for port in ${ports[@]}; do
        log "Enabling port $port now"
        firewall-cmd "--add-port=$port"
    done

    firewall-cmd --runtime-to-permanent
}

# configure_logrotate clobbers /etc/logrotate.conf
configure_logrotate() {
    log "starting"

    local -r logrotate_conf_filename='/etc/logrotate.conf'
    local -r logrotate_conf_file='# see "man logrotate" for details
# rotate log files weekly
weekly

# keep 2 weeks worth of backlogs
rotate 2

# create new (empty) log files after rotating old ones
create

# use date as a suffix of the rotated file
dateext

# uncomment this if you want your log files compressed
compress

# RPM packages drop log rotation information into this directory
include /etc/logrotate.d

# no packages own wtmp and btmp -- we will rotate them here
/var/log/wtmp {
    monthly
    create 0664 root utmp
        minsize 1M
    rotate 1
}

/var/log/btmp {
    missingok
    monthly
    create 0600 root utmp
    rotate 1
}'

    write_file logrotate_conf_filename logrotate_conf_file true
}

# pull_container_images
pull_container_images() {
    local -n pull_images="$1"
    log "starting"

    # The managed identity that the VM runs as only has a single roleassignment.
    # This role assignment is ACRPull which is not necessarily present in the
    # subscription we're deploying into.  If the identity does not have any
    # role assignments scoped on the subscription we're deploying into, it will
    # not show on az login -i, which is why the below line is commented.
    # az account set -s "$SUBSCRIPTIONID"
    az login -i --allow-no-subscriptions

    # Suppress emulation output for podman instead of docker for az acr compatability
    mkdir -p /etc/containers/
    mkdir -p /root/.docker
    touch /etc/containers/nodocker

	# This name is used in the case that az acr login searches for this in it's environment
    local -r REGISTRY_AUTH_FILE="/root/.docker/config.json"
    
    log "logging into prod acr"
    az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

    # shellcheck disable=SC2068
    for i in ${pull_images[@]}; do
        log "Pulling image $i now"
        podman pull "$i"
    done

    az logout
}

# enable_services enables all services required for aro rp
enable_services() {
    local -n services="$1"
    log "starting"

    log "enabling aro services ${aro_services[*]}"
    # shellcheck disable=SC2068
    for service in ${aro_services[@]}; do
        log "Enabling $service now"
        systemctl enable "$service.service"
    done
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

# reboot_vm restores all selinux file contexts, waits 30 seconds then reboots
reboot_vm() {
    log "starting"

    configure_selinux "true"
    (sleep 30 && log "rebooting vm now"; reboot) &
}

# log is a wrapper for echo that includes the function name
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
