#!/bin/bash

set -o errexit \
    -o nounset

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

main() {
    local -r gateway_logdir='/var/log/aro-gateway'
    parse_run_options "$@" gateway_logdir


    configure_sshd
    configure_and_install_dnf_pkgs_repos
    configure_disk_partitions
    configure_logrotate gateway_logdir
    configure_selinux

    mkdir -p /var/log/journal
    mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store

    configure_firewalld_rules
    pull_container_images
    configure_system_services gateway_logdir
    reboot_vm
}

usage() {
    local -n options="$1"
    log "$(basename "$0") [$options]
        -d Configure Disk Partitions
        -p Configure rpm repositories, import required rpm keys, update & install packages with dnf
        -l Configure logrotate.conf
        -s Make selinux modifications required for ARO RP
        -r Configure sshd - Allow password authenticaiton
        -f Configure firewalld default zone rules
        -u Configure systemd unit files for ARO RP
        -i Pull container images

        Note: steps will be executed in the order that flags are provided
    "
}

# parse_run_options takes all arguements passed to main and parses them
# allowing individual step(s) to be ran, rather than all steps
#
# This is useful for local testing, or possibly modifying the bootstrap execution via environment variables in the deployment pipeline
parse_run_options() {
    # shellcheck disable=SC2206
    local -a options=(${1:-})
    if [ "${#options[@]}" -eq 0 ]; then
        log "Running all steps"
        return 0
    fi

    local OPTIND
    local -r allowed_options="dplsrfui"
    while getopts ${allowed_options} options; do
        case "${options}" in
            d)
                log "Running step configure_disk_partitions"
                configure_disk_partitions
                ;;
            p)
                log "Running step configure_and_install_dnf_pkgs_repos"
                configure_and_install_dnf_pkgs_repos
                ;;
            l)
                log "Running configure_logrotate"
                configure_logrotate "$2"
                ;;
            s)
                log "Running configure_selinux"
                configure_selinux
                ;;
            r)
                log "Running configure_sshd"
                configure_sshd
                ;;
            f)
                log "Running configure_firewalld_rules"
                configure_firewalld_rules
                ;;
            u)
                log "Running pull_container_images & configure_system_services"
                configure_system_services "$2"
                ;;
            i)
                log "Running pull_container_images"
                pull_container_images 
                ;;
            *)
                usage allowed_options
                abort "Unkown option provided"
                ;;
        esac
    done
    
    exit 0
}

# We need to configure PasswordAuthentication to yes in order for the VMSS Access JIT to work
configure_sshd() {
    log "starting"
    log "setting ssh password authentication"
    sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config

    systemctl reload sshd.service
    systemctl is-active --quiet sshd || abort "sshd failed to reload"
}

# configure_and_install_dnf_pkgs_repos
configure_and_install_dnf_pkgs_repos() {
    log "starting"

    # transaction attempt retry time in seconds
    local -ri retry_wait_time=60
    configure_rhui_repo retry_wait_time
    create_azure_rpm_repos retry_wait_time

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs exclude_pkgs retry_wait_time

    local -ra rpm_keys=(
        https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
        https://packages.microsoft.com/keys/microsoft.asc
    )

    rpm_import_keys rpm_keys retry_wait_time

    local -ra repo_rpm_pkgs=(
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    )

    local -ra install_pkgs=(
        clamav
        azsec-clamav
        azsec-monitor
        azure-cli
        azure-mdsd
        azure-security
        podman
        podman-docker
        openssl-perl
        # hack - we are installing python3 on hosts due to an issue with Azure Linux Extensions https://github.com/Azure/azure-linux-extensions/pull/1505
        python3
    )

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time
    dnf_install_pkgs install_pkgs retry_wait_time
}

# configure_rhui_repo
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
    retry cmd "$1"
}

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

    log "Updating all packages"
    retry cmd "$2"
}

# dnf_install_pkgs
dnf_install_pkgs() {
    local -n pkgs="$1"
    log "starting"

    local -ra cmd=(
        dnf
        -y
        install
        ${pkgs[@]}
    )

    log "Attempting to install packages: ${pkgs[*]}"
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

# configure_disk_partitions
configure_disk_partitions() {
    log "starting"
    log "extending partition table"

    # Linux block devices are inconsistently named
    # it's difficult to tie the lvm pv to the physical disk using /dev/disk files, which is why lvs is used here
    physical_disk="$(lvs -o devices -a | head -n2 | tail -n1 | cut -d ' ' -f 3 | cut -d \( -f 1 | tr -d '[:digit:]')"
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

# configure_logrotate clobbers /etc/logrotate.conf
configure_logrotate() {
    local -n log_dir="$1"
    log "starting"

    local -r logrotate_conf_filename='/etc/logrotate.conf'
    local -r logrotate_conf_file="# see 'man logrotate' for details
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
}

# Maximum log directory size is 100G with this configuration
# Setting limit to 100G to allow space for other logging services
# copytruncate is a critical option used to prevent logs from being shipped twice
${log_dir} {
    size 20G
    rotate 5
    create 0600 root root
    copytruncate
    noolddir
    compress
}"

    write_file logrotate_conf_filename logrotate_conf_file true
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

    local -ra enable_ports=(
        "80/tcp"
        "8081/tcp"
        "443/tcp"
    )

    log "Enabling ports ${enable_ports[*]} on default firewalld zone"
    # shellcheck disable=SC2068
    for port in ${enable_ports[@]}; do
        log "Enabling port $port now"
        firewall-cmd "--add-port=$port"
    done

    firewall-cmd --runtime-to-permanent
}

# pull_container_images
pull_container_images() {
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

    MDMIMAGE="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
    docker pull "$MDMIMAGE"
    docker pull "$RPIMAGE"
    docker pull "$FLUENTBITIMAGE"

    az logout
}

# configure_system_services creates, configures, and enables the following systemd services and timers
# services
#   fluentbit
#   mdm
#   mdsd
#   arp-rp
#   aro-dbtoken
#   aro-monitor
#   aro-portal
configure_system_services() {
    configure_service_fluentbit
    configure_service_mdm
    configure_timers_mdm_mdsd
    configure_service_aro_gateway "$1"
    configure_service_mdsd
}

# enable_aro_services enables all services required for aro rp
enable_aro_services() {
    log "starting"

    local -ra aro_services=(
      aro-gateway
      auoms
      azsecd
      azsecmond
      mdsd
      mdm
      chronyd
      fluentbit
    )
    log "enabling gateway services ${aro_services[*]}"
    # shellcheck disable=SC2068
    for service in ${aro_services[@]}; do
        log "Enabling $service now"
        systemctl enable "$service.service"
    done
}

# configure_service_fluentbit
configure_service_fluentbit() {
    log "starting"
    log "configuring fluentbit service"

    mkdir -p /etc/fluentbit/
    mkdir -p /var/lib/fluent

    local -r fluentbit_conf_filename='/etc/fluentbit/fluentbit.conf'
    local -r fluentbit_conf_file="[INPUT]
Name systemd
Tag journald
Systemd_Filter _COMM=aro
DB /var/lib/fluent/journaldb

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP

[OUTPUT]
	Name forward
	Match *
	Port 29230"

    write_file fluentbit_conf_filename fluentbit_conf_file true

    local -r sysconfig_fluentbit_filename='/etc/sysconfig/fluentbit'
    local -r sysconfig_fluentbit_file="FLUENTBITIMAGE=$FLUENTBITIMAGE"

    write_file sysconfig_fluentbit_filename sysconfig_fluentbit_file true

    local -r fluentbit_service_filename='/etc/systemd/system/fluentbit.service'

    local -r fluentbit_service_file="[Unit]
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
  $FLUENTBITIMAGE \
  -c /etc/fluentbit/fluentbit.conf

ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=5
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file fluentbit_conf_filename fluentbit_conf_file true
}

# configure_certs
configure_certs() {
    log "starting"

    mkdir /etc/aro-rp
    base64 -d <<<"$ADMINAPICABUNDLE" >/etc/aro-rp/admin-ca-bundle.pem
    if [[ -n "$ARMAPICABUNDLE" ]]; then
    base64 -d <<<"$ARMAPICABUNDLE" >/etc/aro-rp/arm-ca-bundle.pem
    fi
    chown -R 1000:1000 /etc/aro-rp

    # setting MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault seems to have caused mdsd not
    # to honour SSL_CERT_FILE any more, heaven only knows why.
    mkdir -p /usr/lib/ssl/certs
    csplit -f /usr/lib/ssl/certs/cert- -b %03d.pem /etc/pki/tls/certs/ca-bundle.crt /^$/1 "{*}" >/dev/null
    c_rehash /usr/lib/ssl/certs

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
configure_service_mdm() {
    log "starting"
    log "configuring mdm service"

    local -r sysconfig_mdm_filename="/etc/sysconfig/mdm"
    local -r sysconfig_mdm_file="MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=gateway
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
  $MDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file mdm_service_filename mdm_service_file true
}

# configure_timers_mdm_mdsd
configure_timers_mdm_mdsd() {
    log "starting"

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
elif [[ \$COMPONENT\ = \"mdsd\" ]]; then
  CURRENT_CERT_FILE=\"/var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem\"
else
  echo Invalid usage && exit 1
fi

SECRET_NAME=\"gwy-\${COMPONENT}\"
NEW_CERT_FILE=\"\$TEMP_DIR/\$COMPONENT.pem\"
for attempt in {1..5}; do
  az keyvault \
    secret \
    download \
    --file \"\$NEW_CERT_FILE\" \
    --id \"https://$KEYVAULTPREFIX-gwy.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME\" \
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

    systemctl enable download-mdsd-credentials.timer
    systemctl enable download-mdm-credentials.timer

    /usr/local/bin/download-credentials.sh mdsd
    /usr/local/bin/download-credentials.sh mdm

    local -r MDSDCERTIFICATESAN="$(openssl x509 -in /var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem -noout -subject | sed -e 's/.*CN = //')"
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
    systemctl enable "$watch_mdm_creds" || abort "failed to enable $watch_mdm_creds"
    systemctl start "$watch_mdm_creds" || abort "failed to start $watch_mdm_creds"
}

# configure_service_aro_rp
configure_service_aro_gateway() {
    local -n log_dir="$1"
    log "starting"

    local -r aro_gateway_conf_filename='/etc/sysconfig/aro-gateway'
    local -r aro_gateway_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
ARM_API_CLIENT_CERT_COMMON_NAME='$ARMAPICLIENTCERTCOMMONNAME'
AZURE_ARM_CLIENT_ID='$ARMCLIENTID'
AZURE_FP_CLIENT_ID='$FPCLIENTID'
AZURE_FP_SERVICE_PRINCIPAL_ID='$FPSERVICEPRINCIPALID'
BILLING_E2E_STORAGE_ACCOUNT_ID='$BILLINGE2ESTORAGEACCOUNTID'
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMACCOUNT'
CLUSTER_MDM_NAMESPACE=RP
CLUSTER_MDSD_ACCOUNT='$CLUSTERMDSDACCOUNT'
CLUSTER_MDSD_CONFIG_VERSION='$CLUSTERMDSDCONFIGVERSION'
CLUSTER_MDSD_NAMESPACE='$CLUSTERMDSDNAMESPACE'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
DOMAIN_NAME='$LOCATION.$CLUSTERPARENTDOMAINNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Gateway
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
GATEWAY_RESOURCEGROUP='$GATEWAYRESOURCEGROUPNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=RP
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
RP_FEATURES='$RPFEATURES'
RPIMAGE='$RPIMAGE'
ARO_INSTALL_VIA_HIVE='$CLUSTERSINSTALLVIAHIVE'
ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC='$CLUSTERDEFAULTINSTALLERPULLSPEC'
ARO_ADOPT_BY_HIVE='$CLUSTERSADOPTBYHIVE'
USE_CHECKACCESS='$USECHECKACCESS'"

    write_file aro_gateway_conf_filename aro_gateway_conf_file true

    local -r aro_gateway_service_filename='/etc/systemd/system/aro-gateway.service'

    local -r aro_gateway_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-gateway
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
  \$RPIMAGE \
  gateway
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

# configure_service_aro_dbtoken
configure_service_aro_dbtoken() {
    log "starting"

    local -r aro_dbtoken_service_conf_filename='/etc/sysconfig/aro-dbtoken'
    local -r aro_dbtoken_service_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Gateway
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$RPIMAGE'"

    write_file aro_dbtoken_service_conf_filename aro_dbtoken_service_conf_file true

    local -r aro_dbtoken_service_filename='/etc/systemd/system/aro-dbtoken.service'
    local -r aro_dbtoken_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-gateway
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStartPre=/usr/bin/mkdir -p ${gateway_logdir}
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
  -v ${gateway_logdir}:/ctr.log:z \
  \$RPIMAGE \
  gateway
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_dbtoken_service_filename aro_dbtoken_service_file true
}

# configure_service_mdsd
configure_service_mdsd() {
    log "starting"

    local -r mdsd_service_dir="/etc/systemd/system/mdsd.service.d"
    mkdir "$mdsd_service_dir"

    local -r mdsd_override_conf_filename="$mdsd_service_dir/override.conf"
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
export MONITORING_GCS_AUTH_ID='$MDSDCERTIFICATESAN'
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$GATEWAYMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=gateway
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

    log "Scanning configuration files ${configs[*]}"
    # shellcheck disable=SC2068
    for scan in ${configs[@]}; do
        log "Scanning config file $scan now"
        /usr/local/bin/azsecd config -s "$scan" -d P1D
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

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
