#!/bin/bash

set -o errexit \
    -o nounset

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

main() {
    # transaction attempt retry time in seconds
    local -ri retry_wait_time=60

    # shellcheck source=commonVMSS.sh
    source commonVMSS.sh

    create_required_dirs
    configure_sshd
    configure_rpm_repos retry_wait_time

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs pkgs_to_exclude retry_wait_time

    local -ra rpm_keys=(
        https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
        https://packages.microsoft.com/keys/microsoft.asc
    )

    rpm_import_keys rpm_keys retry_wait_time

    local -ra repo_rpm_pkgs=(
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    )

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time
    configure_dnf_cron_job

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

    configure_disk_partitions

    local -r gateway_logdir='/var/log/aro-gateway'
    local -r gateway_log_file="# Maximum log directory size is 100G with this configuration
# Setting limit to 100G to allow space for other logging services
# copytruncate is a critical option used to prevent logs from being shipped twice
${gateway_logdir} {
    size 20G
    rotate 5
    create 0600 root root
    copytruncate
    noolddir
    compress
}"

    # Key dictates the filename written in /etc/logrotate.d
    local -rA logrotate_dropins=(
        ["gateway"]="$gateway_log_file"
    )

    configure_logrotate logrotate_dropins
    configure_selinux

    local -ra enable_ports=(
        "80/tcp"
        "8081/tcp"
        "443/tcp"
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

[OUTPUT]
	Name forward
	Match *
	Port 29230"

    configure_service_fluentbit fluentbit_conf_file fluentbit_image
    local -r mdm_role="gateway"
    configure_service_mdm mdm_role mdmimage

    local -r gwy_dbtoken_service_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Gateway
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$RPIMAGE'"
    configure_service_gateway gateway_logdir

    local -r mdsd_config_version="$GATEWAYMDSDCONFIGVERSION"
    configure_service_mdsd mdm_role mdsd_config_version
    local -r mdsd_role="gwy"
    configure_timers_mdm_mdsd mdsd_role

    configure_certs mdm_role

    local -ra gateway_services=(
        "aro-gateway"
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

    enable_services gateway_services

    reboot_vm
}

# configure_service_aro_rp
configure_service_gateway() {
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
  $RPIMAGE \
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
export MONITORING_CONFIG_VERSION=''
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=gateway
export MONITORING_ROLE_INSTANCE=\"$(hostname)\"

export MDSD_MSGPACK_SORT_COLUMNS=1\""

    write_file default_mdsd_filename default_mdsd_file true

}

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
