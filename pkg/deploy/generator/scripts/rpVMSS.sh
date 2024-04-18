#!/bin/bash

set -o errexit \
    -o nounset

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

main() {
    # transaction attempt retry time in seconds
    local -ri retry_wait_time=60

    # shellcheck source=common.sh
    source common.sh

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    local -ra rpm_keys=(
        https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
        https://packages.microsoft.com/keys/microsoft.asc
    )

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

    MDMIMAGE="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
    local -ra images=(
        "$MDMIMAGE"
        "$RPIMAGE"
        "$FLUENTBITIMAGE"
    )

    local -ra enable_ports=(
        "443/tcp"
        "444/tcp"
        "445/tcp"
        "2222/tcp"
    )

    local -ra aro_services=(
        "aro-dbtoken"
        "aro-monitor"
        "aro-portal"
        "aro-rp"
        "auoms"
        "azsecd"
        "azsecmond"
        "mdsd"
        "mdm"
        "chronyd"
        "fluentbit"
    )

    local -ra user_options=("$@")
    parse_run_options user_options \
                        retry_wait_time \
                        exclude_pkgs \
                        rpm_keys \
                        repo_rpm_pkgs \
                        install_pkgs \
                        images \
                        enable_ports \
                        aro_services

    configure_sshd
    configure_rpm_repos retry_wait_time

    dnf_update_pkgs exclude_pkgs retry_wait_time

    rpm_import_keys rpm_keys retry_wait_time

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time
    dnf_install_pkgs install_pkgs retry_wait_time
    configure_dnf_cron_job

    configure_disk_partitions
    configure_logrotate
    configure_selinux

    mkdir -p /var/log/journal
    mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store

    configure_firewalld_rules enable_ports

    pull_container_images images
    configure_aro_services
    enable_services

    enable_services aro_services

    reboot_vm
}

usage() {
    local -n options="$1"
    log "$(basename "$0") [$options]
        -d Configure Disk Partitions
        -p Configure rpm repositories, import required rpm keys, update & install packages with dnf
        -l Configure logrotate.conf
        -s Configure selinux
        -r Configure sshd - Allow password authenticaiton
        -f Configure firewalld default zone rules
        -u Configure systemd unit files
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
    local -n run_options="$1"
    local -n retry_time="$2"
    local -n pkgs_to_exclude="$3"
    local -n keys_to_import="$4"
    local -n rpm_pkgs="$5"
    local -n pkgs_to_install="$6"
    local -n images_to_pull="$7"
    local -n ports_to_enable="$8"
    local -n services_to_enable="$9"

    if [ "${#run_options[@]}" -eq 0 ]; then
        log "Running all steps"
        return 0
    fi

    local OPTIND
    local -r allowed_options="dplsrfui"
    while getopts ${allowed_options} options; do
        case "${run_options}" in
            d)
                log "Running step configure_disk_partitions"
                configure_disk_partitions
                ;;
            p)
                log "Running step configure_rpm_repos"
                configure_rpm_repos keys_to_import

                log "Running step dnf_update_pkgs"
                dnf_update_pkgs pkgs_to_exclude retry_time

                log "Running step dnf_install_pkgs rpm_pkgs"
                dnf_install_pkgs rpm_pkgs retry_time

                log "Running step dnf_install_pkgs pkgs"
                dnf_install_pkgs pkgs_to_install retry_time
                
                log "Running step configure_dnf_crond_job"
                configure_dnf_cron_job
                ;;
            l)
                log "Running configure_logrotate"
                configure_logrotate
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
                configure_firewalld_rules ports_to_enable
                ;;
            u)
                log "Running pull_container_images & configure_system_services"
                configure_aro_services
                enable_services services_to_enable
                ;;
            i)
                log "Running pull_container_images"
                pull_container_images images_to_pull
                ;;
            *)
                usage allowed_options
                abort "Unkown option provided"
                ;;
        esac
    done
    
    exit 0
}

# configure_rpm_repos
configure_rpm_repos() {
    log "starting"

    configure_rhui_repo "$1"
    create_azure_rpm_repos "$1"
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

# configure_aro_services creates, configures, and enables the following systemd services and timers
# services
#   fluentbit
#   mdm
#   mdsd
#   arp-rp
#   aro-dbtoken
#   aro-monitor
#   aro-portal
configure_aro_services() {
    configure_service_fluentbit
    configure_service_mdm
    configure_timers_mdm_mdsd
    configure_service_aro_rp
    configure_service_aro_dbtoken
    configure_service_aro_monitor
    configure_service_aro_portal
    configure_service_mdsd
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

[FILTER]
	Name rewrite_tag
	Match journald
	Rule $LOGKIND asyncqos asyncqos true

[FILTER]
	Name modify
	Match asyncqos
	Remove CLIENT_PRINCIPAL_NAME
	Remove FILE
	Remove COMPONENT

[FILTER]
	Name rewrite_tag
	Match journald
	Rule $LOGKIND ifxaudit ifxaudit false

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

    write_file fluentbit_service_filename fluentbit_service_file true
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
MDMSOURCEROLE=rp
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

SECRET_NAME=\"rp-\${COMPONENT}\"
NEW_CERT_FILE=\"\$TEMP_DIR/\$COMPONENT.pem\"
for attempt in {1..5}; do
  az keyvault \
    secret \
    download \
    --file \"\$NEW_CERT_FILE\" \
    --id \"https://$KEYVAULTPREFIX-svc.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME\" \
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
configure_service_aro_rp() {
    log "starting"

    local -r aro_rp_conf_filename='/etc/sysconfig/aro-rp'
    local -r aro_rp_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
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
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
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

    write_file aro_rp_conf_filename aro_rp_conf_file true

    local -r aro_rp_service_filename='/etc/systemd/system/aro-rp.service'
    local -r aro_rp_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
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
  -e BILLING_E2E_STORAGE_ACCOUNT_ID \
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
  $RPIMAGE \
  rp
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_rp_service_filename aro_rp_service_file true
}

# configure_service_aro_dbtoken
configure_service_aro_dbtoken() {
    log "starting"

    local -r aro_dbtoken_service_conf_filename='/etc/sysconfig/aro-dbtoken'
    local -r aro_dbtoken_service_conf_file="DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=DBToken
RPIMAGE='$RPIMAGE'"

    write_file aro_dbtoken_service_conf_filename aro_dbtoken_service_conf_file true

    local -r aro_dbtoken_service_filename='/etc/systemd/system/aro-dbtoken.service'
    local -r aro_dbtoken_service_file="[Unit]
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
  $RPIMAGE \
  dbtoken
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_dbtoken_service_filename aro_dbtoken_service_file true
}

# configure_service_aro_monitor
configure_service_aro_monitor() {
    log "starting"
    log "configuring aro-monitor service"

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
RPIMAGE='$RPIMAGE'"

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
  $RPIMAGE \
  monitor
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file aro_monitor_service_filename aro_monitor_service_file true
}

# configure_service_aro_portal
configure_service_aro_portal() {
    log "starting"

    local -r aro_portal_service_conf_filename='/etc/sysconfig/aro-portal'
    local -r aro_portal_service_conf_file="AZURE_PORTAL_ACCESS_GROUP_IDS='$PORTALACCESSGROUPIDS'
AZURE_PORTAL_CLIENT_ID='$PORTALCLIENTID'
AZURE_PORTAL_ELEVATED_GROUP_IDS='$PORTALELEVATEDGROUPIDS'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Portal
PORTAL_HOSTNAME='$LOCATION.admin.$RPPARENTDOMAINNAME'
RPIMAGE='$RPIMAGE'"

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
  $RPIMAGE \
  portal
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target"

    write_file aro_portal_service_filename aro_portal_service_file true
}

# configure_service_mdsd
configure_service_mdsd() {
    log "starting"

    local -r mdsd_service_dir="/etc/systemd/system/mdsd.service.d"
    mkdir -p "$mdsd_service_dir"

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
export MONITORING_CONFIG_VERSION='$RPMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=rp
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

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
