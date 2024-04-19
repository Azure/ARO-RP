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

    configure_sshd
    configure_rpm_repos retry_wait_time

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

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time

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

    dnf_install_pkgs install_pkgs retry_wait_time
    configure_dnf_cron_job
    configure_disk_partitions

    # Key dictates the filename written in /etc/logrotate.d
    # Present for future dropin files, also is required for configure_logrotate
    local -rA logrotate_dropins=()
    configure_logrotate logrotate_dropins
    configure_selinux
    create_required_dirs

    local -ra enable_ports=(
        "443/tcp"
        "444/tcp"
        "445/tcp"
        "2222/tcp"
    )

    configure_firewalld_rules enable_ports

    local -r mdmimage="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
    local -r rpimage="$RPIMAGE"
    local -r fluentbit_image="$FLUENTBITIMAGE"
    local -ra images=(
        "$mdmimage"
        "$rpimage"
        "$fluentbit_image"
    )
	local -r registry_config_file=""
    pull_container_images images registry_config_file

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

    configure_service_fluentbit fluentbit_conf_file fluentbit_image
    local -r monitor_role="rp"
    configure_service_mdm monitor_role mdmimage
    configure_timers_mdm_mdsd monitor_role
    configure_service_aro_rp

    configure_service_dbtoken rpimage
    configure_service_aro_monitor rpimage
    configure_service_aro_portal rpimage
    local -r mdsd_rp_version="$RPMDSDCONFIGVERSION"
    configure_service_mdsd monitor_role mdsd_rp_version

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
        "download-mdsd-credentials.timer"
        "download-mdm-credentials.timer"
    )

    enable_services aro_services

    reboot_vm
}

# configure_rpm_repos
configure_rpm_repos() {
    log "starting"

    configure_rhui_repo "$1"
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

# configure_service_aro_monitor
configure_service_aro_monitor() {
    local -n image="$1"
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
configure_service_aro_portal() {
    local -n image="$1"
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

# configure_service_aro_dbtoken
configure_service_dbtoken() {
    local -n image="$1"
    log "starting"

    local -r conf_file="DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=DBToken
RPIMAGE='$image'"

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


export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
