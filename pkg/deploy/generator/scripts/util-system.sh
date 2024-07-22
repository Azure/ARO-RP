#!/bin/bash
# This file is intended to be sourced by bootstrapping scripts for commonly used functions

# configure_sshd
# We need to configure PasswordAuthentication to yes in order for the VMSS Access JIT to work
configure_sshd() {
    log "starting"
    local -r sshd_config="/etc/ssh/sshd_config"

    log "Editing $sshd_config to allow password authentication"
    sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' "$sshd_config"

    systemctl reload sshd.service || abort "sshd failed to reload"
}

# configure_firewalld_rules
# args:
# 1) ports - nameref, string array; ports to be enabled.
#       Ports must be postfixed with /tcp or /udp
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

    log "Enabling ports ${ports[*]} on default firewalld zone"
    # shellcheck disable=SC2068
    for port in ${ports[@]}; do
        log "Enabling port $port now"
        firewall-cmd "--add-port=$port"
    done

    log "Writing runtime config to permanent config"
    firewall-cmd --runtime-to-permanent
}

# configure_logrotate clobbers /etc/logrotate.conf
# args:
# 1) dropin_files - nameref, associative array, optional; logrotate files to write to /etc/logrotate.d
#       Key name dictates filenames written to /etc/logrotate.d.
configure_logrotate() {
    local -n dropin_files="${1:-empty_str}"
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

    if [ -n "${dropin_files[*]}" ]; then
        local -r logrotate_d="/etc/logrotate.d"
        log "Writing logrotate files to $logrotate_d"
        for dropin_name in "${!dropin_files[@]}"; do
            local -r dropin_filename="$logrotate_d/$dropin_name"
            local -r dropin_file="${dropin_files["$dropin_name"]}"
            write_file dropin_filename dropin_file true
        done
    fi
}

# pull_container_images
# args:
# 1) pull_images - nameref, string array
# 2) az_login - boolean; login with az login and az acr login
# 3) registry_conf - nameref, string, optional; path to docker/podman configuration file
pull_container_images() {
    local -n pull_images="$1"
    local -r az_login="${2}"
    local -n registry_conf="${3:-empty_str}"
    log "starting"

    local -ri retry_time=30
    # The managed identity that the VM runs as only has a single roleassignment.
    # This role assignment is ACRPull which is not necessarily present in the
    # subscription we're deploying into.  If the identity does not have any
    # role assignments scoped on the subscription we're deploying into, it will
    # not show on az login -i, which is why the below line is commented.
    # az account set -s "$SUBSCRIPTIONID"
    if $az_login; then
        cmd=(
            az
            login
            -i
            --allow-no-subscriptions
        )

        log "Running az login with retries"
        retry cmd retry_time
    fi

    # Suppress emulation output for podman instead of docker for az acr compatability
    mkdir -p /etc/containers/
    mkdir -p /root/.docker
    touch /etc/containers/nodocker

    # This name is used in the case that az acr login searches for this in it's environment
    export REGISTRY_AUTH_FILE="/root/.docker/config.json"
    
    if [ -n "${registry_conf}" ]; then
        write_file REGISTRY_AUTH_FILE registry_conf true
    fi

    log "logging into prod acr"
    if $az_login; then
        cmd=(
            az
            acr
            login
            --name
            "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"
        )

        log "Running az login with retries"
        retry cmd retry_time
    fi

    # shellcheck disable=SC2068
    for i in ${pull_images[@]}; do
        local -n image="$i"
        cmd=(
            podman
            pull
            "$image"
        )

        log "Pulling image $image with retries now"
        retry cmd retry_time
    done

    if $az_login; then
        cmd=(
            az
            logout
        )

        log "Running az logout with retries"
        retry cmd retry_time
    fi
}

# configure_disk_partitions
configure_disk_partitions() {
    log "starting"
    log "extending partition table"

    # Linux block devices are inconsistently named
    # it's difficult to tie the lvm pv to the physical disk using /dev/disk files, which is why lvs is used here
    local -r physical_disk="$(lvs -o devices -a | head -n2 | tail -n1 | cut -d ' ' -f 3 | cut -d \( -f 1 | tr -d '[:digit:]')"
    growpart "$physical_disk" 2

    log "extending filesystems"
    log "extending root lvm"
    lvextend -l +20%FREE /dev/rootvg/rootlv
    log "growing root filesystem"
    xfs_growfs /

    log "extending var lvm"
    lvextend -l +100%FREE /dev/rootvg/varlv
    log "growing var filesystem"
    xfs_growfs /var
}

# configure_certs
# args:
# 1) role - string; can be "devproxy" or "rp"
configure_certs() {
    local -n role="$1"
    log "starting"
    log "Configuring certificates for $role"

    verify_role role true

    if [ "$role" == "devproxy" ]; then
        local -r proxy_certs_basedir="/etc/proxy"
        mkdir -p "$proxy_certs_basedir"
        base64 -d <<<"$PROXYCERT" > "$proxy_certs_basedir/proxy.crt"
        base64 -d <<<"$PROXYKEY" > "$proxy_certs_basedir/proxy.key"
        base64 -d <<<"$PROXYCLIENTCERT" > "$proxy_certs_basedir/proxy-client.crt"
        chown -R 1000:1000 /etc/proxy
        chmod 0600 "$proxy_certs_basedir/proxy.key"
        return 0
    fi

    if [ "$role" == "rp" ]; then
        local -r rp_certs_basedir="/etc/aro-rp"
        mkdir -p "$rp_certs_basedir"
        base64 -d <<<"$ADMINAPICABUNDLE" > "$rp_certs_basedir/admin-ca-bundle.pem"
        if [[ -n "$ARMAPICABUNDLE" ]]; then
        base64 -d <<<"$ARMAPICABUNDLE" > "$rp_certs_basedir/arm-ca-bundle.pem"
        fi
        chown -R 1000:1000 "$rp_certs_basedir"
    fi

    # setting MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault seems to have caused mdsd not
    # to honour SSL_CERT_FILE any more, heaven only knows why.
    local -r ssl_certs_basedir="/usr/lib/ssl/certs"
    mkdir -p "$ssl_certs_basedir"
    csplit -f "$ssl_certs_basedir/cert-" -b %03d.pem /etc/pki/tls/certs/ca-bundle.crt /^$/1 "{*}" 1>/dev/null
    c_rehash "$ssl_certs_basedir"

    # we leave clientId blank as long as only 1 managed identity assigned to vmss
    # if we have more than 1, we will need to populate with clientId used for off-node scanning
    local -r nodescan_agent_filename="/etc/default/vsa-nodescan-agent.config"
    local -r nodescan_agent_file="{
    \"Nice\": 19,
    \"Timeout\": 10800,
    \"ClientId\": \"\",
    \"TenantId\": $AZURESECPACKVSATENANTID,
    \"QualysStoreBaseUrl\": $AZURESECPACKQUALYSURL,
    \"ProcessTimeout\": 300,
    \"CommandDelay\": 0
  }"

    write_file nodescan_agent_filename nodescan_agent_file true
}

# run_azsecd_config_scan
run_azsecd_config_scan() {
    log "starting"

    local -ar configs=(
        "baseline"
        "clamav"
        "software"
    )

    log "Scanning configuration files with azsecd ${configs[*]}"
    # shellcheck disable=SC2068
    for scan in ${configs[@]}; do
        log "Scanning config file $scan now"
        /usr/local/bin/azsecd config -s "$scan" -d P1D
    done
}

# create_required_dirs
create_required_dirs() {
    create_dirs=(
        /var/log/journal
        /var/lib/waagent/Microsoft.Azure.KeyVault.Store
        # Does not exist on devProxyVMSS
        /var/opt/microsoft/linuxmonagent
    )

    # shellcheck disable=SC2068
    for d in ${create_dirs[@]}; do
        log "Creating directory $d"
        mkdir -p "$d" || abort "failed to create directory $d"
    done
}
