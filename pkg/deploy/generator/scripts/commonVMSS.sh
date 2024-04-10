#!/bin/bash
# This file is intended to be sourced by bootstrapping scripts for commonly used functions

### Internal Functions and Constants ###

# empty_str - constant; used by functions for optional nameref string arguements
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

# configure_selinux
# args:
# 1) relabel - boolean, optional; defaults to false
#                                 Relabel filesystem context
configure_selinux() {
    local -r relabel="${1:-false}"
    log "starting"

    already_defined_ignore_error="File context for /var/log/journal(/.*)? already defined"
    semanage fcontext -a -t var_log_t "/var/log/journal(/.*)?" || log "$already_defined_ignore_error"
    chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

    if $relabel; then
        restorecon -RF /var/log/* || log "$already_defined_ignore_error"
    fi
}

### Shared Functions ###

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

# We need to configure PasswordAuthentication to yes in order for the VMSS Access JIT to work
configure_sshd() {
    log "starting"
    local -r sshd_config="/etc/ssh/sshd_config"

    log "Editing $sshd_config to allow password authentication"
    sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' "$sshd_config"

    systemctl reload sshd.service || abort "sshd failed to reload"
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

# enable_services enables all services required for aro rp
# args:
# 1) services - array; services to be enabled
enable_services() {
    local -n services="$1"
    log "starting"

    systemctl daemon-reload

    log "enabling services ${services[*]}"
    # shellcheck disable=SC2068
    for service in ${services[@]}; do
        log "Enabling and starting $service now"
        systemctl enable \
                  --now \
                  "$service"
    done
}

# reboot_vm restores all selinux file contexts, then schedules a reboot for one hour later
# Reboots should scheduled after all VM extensions have had time to complete
# Reference: https://learn.microsoft.com/en-us/azure/virtual-machines/extensions/custom-script-linux#tips
reboot_vm() {
    log "starting"

    configure_selinux "true"
    
    hour="$(date -d "1 hour" +%H:%M)"
    shutdown -r "$hour" "Post deployment reboot is happening now"
}

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

# configure_service_mdm
# args:
# 1) role - nameref, string; can be "gateway" or "rp"
# 2) image - nameref, string; mdm container image to run
configure_service_mdm() {
    local -n role="$1"
    local -n image="$2"
    log "starting"
    log "Configuring mdm service"

    verify_role role

    local -r sysconfig_mdm_filename="/etc/sysconfig/mdm"
    local -r sysconfig_mdm_file="MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$image'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE='$role'
MDMSOURCEROLEINSTANCE=\"$(hostname)\""

    write_file sysconfig_mdm_filename sysconfig_mdm_file true

    mkdir -p /var/etw
    local -r mdm_service_filename="/etc/systemd/system/mdm.service"
    local -r mdm_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -m 2g \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $image \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $LOCATION \
  -SourceRole $role \
  -SourceRoleInstance $HOSTNAME
ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file mdm_service_filename mdm_service_file true
}

# configure_timers_mdm_mdsd
# args:
# 1) role - string; can be "gateway" or "rp"
configure_timers_mdm_mdsd() {
    local -n role="$1"
    log "starting"

    verify_role role

    local keyvault_suffix secret_prefix
    get_keyvault_suffix role keyvault_suffix secret_prefix

    for var in "mdsd" "mdm"; do
        local download_creds_service_filename="/etc/systemd/system/download-$var-credentials.service"
        local download_creds_service_file="[Unit]
Description=Periodic $var credentials refresh

[Service]
Type=oneshot
ExecStart=/usr/local/bin/download-credentials.sh $var"

        write_file download_creds_service_filename download_creds_service_file true

        local download_creds_timer_filename="/etc/systemd/system/download-$var-credentials.timer"
        local download_creds_timer_file="[Unit]
Description=Periodic $var credentials refresh
After=network-online.target
Wants=network-online.target

[Timer]
OnBootSec=0min
OnCalendar=0/12:00:00
AccuracySec=5s

[Install]
WantedBy=timers.target"

        write_file download_creds_timer_filename download_creds_timer_file true
    done

    local -r download_creds_script_filename="/usr/local/bin/download-credentials.sh"
    local -r download_creds_script_file="#!/bin/bash
set -eu

COMPONENT=\$1
echo \"Download \$COMPONENT credentials\"

TEMP_DIR=\"\$(mktemp -d)\"
export AZURE_CONFIG_DIR=\"\$(mktemp -d)\"

echo \"Logging into Azure...\"
RETRIES=3
while [[ \$RETRIES -gt 0 ]]; do
    if az login -i --allow-no-subscriptions
    then
        echo \"az login successful\"
        break
    else
        echo \"az login failed. Retrying...\"
        let RETRIES-=1
        sleep 5
    fi
done

trap \"cleanup\" EXIT

cleanup() {
  az logout
  [[ \$TEMP_DIR =~ /tmp/.+ ]] && rm -rf \$TEMP_DIR
  [[ \$AZURE_CONFIG_DIR =~ /tmp/.+ ]] && rm -rf \$AZURE_CONFIG_DIR
}

if [[ \$COMPONENT = \"mdm\" ]]; then
  CURRENT_CERT_FILE=\"/etc/mdm.pem\"
elif [[ \$COMPONENT = \"mdsd\" ]]; then
  CURRENT_CERT_FILE=\"/var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem\"
else
  echo Invalid usage && exit 1
fi

SECRET_NAME=\"$secret_prefix-\${COMPONENT}\"
NEW_CERT_FILE=\"\$TEMP_DIR/\$COMPONENT.pem\"
for attempt in {1..5}; do
  az keyvault \
    secret \
    download \
    --file \"\$NEW_CERT_FILE\" \
    --id \"https://$KEYVAULTPREFIX-$keyvault_suffix.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME\" \
    && break
  if [[ \$attempt -lt 5 ]]; then sleep 10; else exit 1; fi
done

if [ -f \$NEW_CERT_FILE ]; then
  if [[ \$COMPONENT = \"mdsd\" ]]; then
    chown syslog:syslog \$NEW_CERT_FILE
  else
    sed -i -ne '1,/END CERTIFICATE/ p' \$NEW_CERT_FILE
  fi

  new_cert_sn=\"\$(openssl x509 -in \"\$NEW_CERT_FILE\" -noout -serial | awk -F= '{print \$2}')\"
  current_cert_sn=\"\$(openssl x509 -in \"\$CURRENT_CERT_FILE\" -noout -serial | awk -F= '{print \$2}')\"
  if [[ ! -z \$new_cert_sn ]] && [[ \$new_cert_sn != \"\$current_cert_sn\" ]]; then
    echo updating certificate for \$COMPONENT
    chmod 0600 \$NEW_CERT_FILE
    mv \$NEW_CERT_FILE \$CURRENT_CERT_FILE
  fi
else
  echo Failed to refresh certificate for \$COMPONENT && exit 1
fi"

    write_file download_creds_script_filename download_creds_script_file true

    chmod u+x /usr/local/bin/download-credentials.sh

    $download_creds_script_filename mdsd
    $download_creds_script_filename mdm

    local -r watch_mdm_creds_service_filename="/etc/systemd/system/watch-mdm-credentials.service"
    local -r watch_mdm_creds_service_file="[Unit]
Description=Watch for changes in mdm.pem and restarts the mdm service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart mdm.service

[Install]
WantedBy=multi-user.target"

    write_file watch_mdm_creds_service_filename watch_mdm_creds_service_file true

    local -r watch_mdm_creds_path_filename='/etc/systemd/system/watch-mdm-credentials.path'
    local -r watch_mdm_creds_path_file='[Path]
PathModified=/etc/mdm.pem

[Install]
WantedBy=multi-user.target'

    write_file watch_mdm_creds_path_filename watch_mdm_creds_path_file true

    local -r watch_mdm_creds='watch-mdm-credentials.path'
    systemctl enable --now "$watch_mdm_creds" || abort "failed to enable and start $watch_mdm_creds"
}

# configure_service_fluentbit
# args:
# 1) conf_file - string; fluenbit configuration file
# 2) image - string; fluentbit container image to run
configure_service_fluentbit() {
    local -n conf_file="$1"
    local -n image="$2"
    log "starting"
    log "Configuring fluentbit service"

    mkdir -p /etc/fluentbit/
    mkdir -p /var/lib/fluent

    local -r conf_filename='/etc/fluentbit/fluentbit.conf'
    write_file conf_filename conf_file true

    local -r sysconfig_filename='/etc/sysconfig/fluentbit'
    local -r sysconfig_file="FLUENTBITIMAGE=$image"

    write_file sysconfig_filename sysconfig_file true

    local -r service_filename='/etc/systemd/system/fluentbit.service'
    local -r service_file="[Unit]
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/fluentbit
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --security-opt label=disable \
  --entrypoint /opt/td-agent-bit/bin/td-agent-bit \
  --net=host \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -v /etc/fluentbit/fluentbit.conf:/etc/fluentbit/fluentbit.conf \
  -v /var/lib/fluent:/var/lib/fluent:z \
  -v /var/log/journal:/var/log/journal:ro \
  -v /etc/machine-id:/etc/machine-id:ro \
  $image \
  -c /etc/fluentbit/fluentbit.conf

ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=5
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file service_filename service_file true
}

# configure_service_mdsd
# args:
# 1) monitoring_role - nameref, string; can be "gateway" or "rp"
# 2) monitor_config_version - nameref, string; mdsd config version
configure_service_mdsd() {
    local -n role="$1"
    local -n monitor_config_version="$2"
    log "starting"
    log "configuring mdsd service"

    verify_role role

    local -r mdsd_service_dir="/etc/systemd/system/mdsd.service.d"
    mkdir -p "$mdsd_service_dir"

    local -r mdsd_override_conf_filename="$mdsd_service_dir/override.conf"
    local -r mdsd_certificate_san="$(openssl x509 -in /var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem -noout -subject | sed -e 's/.*CN = //')"
    local -r mdsd_override_conf_file="[Unit]
After=network-online.target"

    write_file mdsd_override_conf_filename mdsd_override_conf_file true

    local -r default_mdsd_filename="/etc/default/mdsd"
    local -r default_mdsd_file="MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS=\"-A -d -r \$MDSD_ROLE_PREFIX\"

export MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT='$RPMDSDACCOUNT'
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault
export MONITORING_GCS_AUTH_ID='$mdsd_certificate_san'
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$monitor_config_version'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE='$role'
export MONITORING_ROLE_INSTANCE=\"$(hostname)\"

export MDSD_MSGPACK_SORT_COLUMNS=1\""

    write_file default_mdsd_filename default_mdsd_file true
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

### Gateway VMSS services

# configure_service_gateway
# args:
# 1) log_dir - nameref, string; directory to mount for logging directory of container
# 2) image - nameref, string; container image
# 3) role - nameref, string; VMSS role
# 4) conf_file - nameref, string; aro gateway environment file
configure_service_aro_gateway() {
    local -n log_dir="$1"
    local -n image="$2"
    local -n role="$3"
    local -n conf_file="$4"
    log "starting"
    log "Configuring aro-gateway service"

    local -r aro_gateway_conf_filename='/etc/sysconfig/aro-gateway'

    write_file aro_gateway_conf_filename conf_file true

    local -r aro_gateway_service_filename='/etc/systemd/system/aro-gateway.service'

    local -r aro_gateway_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=${aro_gateway_conf_filename}
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStartPre=/usr/bin/mkdir -p ${log_dir}
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e DBTOKEN_URL \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_FEATURES \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 80:8080 \
  -p 8081:8081 \
  -p 443:8443 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  -v ${log_dir}:/ctr.log:z \
  $image \
  ${role,,}
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
    "

    write_file aro_gateway_service_filename aro_gateway_service_file true
}

### RP VMSS Services

# configure_service_aro_rp
# args:
# 1) image - nameref, string; RP container image
# 2) role - nameref, string; VMSS role
# 3) conf_file - nameref, string; aro rp environment file
configure_service_aro_rp() {
    local -n image="$1"
    local -n role="$2"
    local -n conf_file="$3"
    log "starting"
    log "Configuring aro-rp service"

    local -r aro_rp_conf_filename='/etc/sysconfig/aro-rp'

    write_file aro_rp_conf_filename conf_file true

    local -r aro_rp_service_filename='/etc/systemd/system/aro-rp.service'
    local -r aro_rp_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=${aro_rp_conf_filename}
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e ADMIN_API_CLIENT_CERT_COMMON_NAME \
  -e ARM_API_CLIENT_CERT_COMMON_NAME \
  -e AZURE_ARM_CLIENT_ID \
  -e AZURE_FP_CLIENT_ID \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e CLUSTER_MDSD_ACCOUNT \
  -e CLUSTER_MDSD_CONFIG_VERSION \
  -e CLUSTER_MDSD_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e DOMAIN_NAME \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_RESOURCEGROUP \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e MDSD_ENVIRONMENT \
  -e RP_FEATURES \
  -e ARO_INSTALL_VIA_HIVE \
  -e ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC \
  -e ARO_ADOPT_BY_HIVE \
  -e USE_CHECKACCESS \
  -m 2g \
  -p 443:8443 \
  -v /etc/aro-rp:/etc/aro-rp \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $image \
  ${role,,}
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_rp_service_filename aro_rp_service_file true
}

# configure_service_aro_monitor
# args:
# 1) image - nameref, string; RP container image
configure_service_aro_monitor() {
    local -n image="$1"
    log "starting"
    log "Configuring aro-monitor service"

    # DOMAIN_NAME, CLUSTER_MDSD_ACCOUNT, CLUSTER_MDSD_CONFIG_VERSION, GATEWAY_DOMAINS, GATEWAY_RESOURCEGROUP, MDSD_ENVIRONMENT CLUSTER_MDSD_NAMESPACE
    # are not used, but can't easily be refactored out. Should be revisited in the future.
    local -r aro_monitor_service_conf_filename='/etc/sysconfig/aro-monitor'
    local -r aro_monitor_service_conf_file="AZURE_FP_CLIENT_ID='$FPCLIENTID'
DOMAIN_NAME='$LOCATION.$CLUSTERPARENTDOMAINNAME'
CLUSTER_MDSD_ACCOUNT='$CLUSTERMDSDACCOUNT'
CLUSTER_MDSD_CONFIG_VERSION='$CLUSTERMDSDCONFIGVERSION'
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_RESOURCEGROUP='$GATEWAYRESOURCEGROUPNAME'
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
CLUSTER_MDSD_NAMESPACE='$CLUSTERMDSDNAMESPACE'
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMACCOUNT'
CLUSTER_MDM_NAMESPACE=BBM
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=BBM
RPIMAGE='$image'"

    write_file aro_monitor_service_conf_filename aro_monitor_service_conf_file true

    local -r aro_monitor_service_filename='/etc/systemd/system/aro-monitor.service'
    local -r aro_monitor_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e AZURE_FP_CLIENT_ID \
  -e DOMAIN_NAME \
  -e CLUSTER_MDSD_ACCOUNT \
  -e CLUSTER_MDSD_CONFIG_VERSION \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_RESOURCEGROUP \
  -e MDSD_ENVIRONMENT \
  -e CLUSTER_MDSD_NAMESPACE \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2.5g \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $image \
  monitor
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_monitor_service_filename aro_monitor_service_file true
}

# configure_service_aro_portal
# args:
# 1) image - nameref, string; RP container image
configure_service_aro_portal() {
    local -n image="$1"
    log "starting"
    log "Configuring aro portal service"

    local -r aro_portal_service_conf_filename='/etc/sysconfig/aro-portal'
    local -r aro_portal_service_conf_file="AZURE_PORTAL_ACCESS_GROUP_IDS='$PORTALACCESSGROUPIDS'
AZURE_PORTAL_CLIENT_ID='$PORTALCLIENTID'
AZURE_PORTAL_ELEVATED_GROUP_IDS='$PORTALELEVATEDGROUPIDS'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Portal
PORTAL_HOSTNAME='$LOCATION.admin.$RPPARENTDOMAINNAME'
RPIMAGE='$image'"

    write_file aro_portal_service_conf_filename aro_portal_service_conf_file true

    local -r aro_portal_service_filename='/etc/systemd/system/aro-portal.service'
    local -r aro_portal_service_file="[Unit]
After=network-online.target
Wants=network-online.target
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-portal
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e AZURE_PORTAL_ACCESS_GROUP_IDS \
  -e AZURE_PORTAL_CLIENT_ID \
  -e AZURE_PORTAL_ELEVATED_GROUP_IDS \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e PORTAL_HOSTNAME \
  -m 2g \
  -p 444:8444 \
  -p 2222:2222 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $image \
  portal
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target"

    write_file aro_portal_service_filename aro_portal_service_file true
}

# configure_service_dbtoken
# args:
# 1) image - nameref, string; RP container image
# 2) conf_file - nameref, string; dbtoken configuration file
configure_service_dbtoken() {
    local -n image="$1"
    local -n conf_file="$2"
    log "starting"
    log "Configuring dbtoken service"

    local -r conf_filename='/etc/sysconfig/aro-dbtoken'

    write_file conf_filename conf_file true

    local -r service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-dbtoken
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e AZURE_GATEWAY_SERVICE_PRINCIPAL_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 445:8445 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $image \
  dbtoken
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    local -r service_filename='/etc/systemd/system/aro-dbtoken.service'
    write_file service_filename service_file true
}

# configure_vmss_aro_service
# args:
# 1) r - nameref, string; role of VMSS
# 2) images - nameref, associative array; ARO container images
# 3) configs - nameref, associative array; configuration files and versions. The values should be a reference to variables, not dereferenced.
#                                          This is because the value is used when creating nameref variables by helper functions.
configure_vmss_aro_services() {
    local -n r="$1"
    local -n images="$2"
    local -n configs="$3"
    log "starting"
    verify_role "$1"

    if [ "$r" == "$role_gateway" ]; then
        configure_service_aro_gateway "${configs["log_dir"]}" "${images["rp"]}" "$1" "${configs["gateway_config"]}"
    elif [ "$r" == "$role_rp" ]; then
        configure_service_dbtoken "${images["rp"]}" "${configs["dbtoken"]}"
        configure_service_aro_rp "${images["rp"]}" "$1" "${configs["rp_config"]}"
        configure_service_aro_monitor "${images["rp"]}"
        configure_service_aro_portal "${images["rp"]}"
    fi

    configure_service_fluentbit "${configs["fluentbit"]}" "${images["fluentbit"]}"
    configure_service_mdm "$1" "${images["mdm"]}"
    configure_service_mdsd "$1" "${configs["mdsd"]}"
    configure_certs "$1"
    configure_timers_mdm_mdsd "$1"
}
