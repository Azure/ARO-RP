#!/bin/bash

set -o errexit \
    -o nounset

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

main() {
    # transaction attempt retry time in seconds
    local -ri retry_wait_time=30
    local -ri pkg_retry_count=60

    # commonVMSS.sh does not exist when deployed to VMSS via VMSS extensions
    # This is because commonVMSS.sh is concatenated with this script
    common_sh="commonVMSS.sh"
    if [ -f "$common_sh" ]; then
        # shellcheck source=commonVMSS.sh
        source "$common_sh"
    fi

    create_required_dirs
    configure_sshd
    configure_rpm_repos retry_wait_time "$pkg_retry_count"

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs exclude_pkgs retry_wait_time "$pkg_retry_count"

    local -ra rpm_keys=(
        https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
        https://packages.microsoft.com/keys/microsoft.asc
    )

    rpm_import_keys rpm_keys retry_wait_time "$pkg_retry_count"

    local -ra repo_rpm_pkgs=(
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    )

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time "$pkg_retry_count"

    local -ra install_pkgs=(
        at
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

    dnf_install_pkgs install_pkgs retry_wait_time "$pkg_retry_count"
    configure_dnf_cron_job
    configure_disk_partitions

    # Key dictates the filename written in /etc/logrotate.d
    # local -rA logrotate_dropins=()
    configure_logrotate
    configure_selinux

    local -ra enable_ports=(
        "443/tcp"
        "444/tcp"
        "445/tcp"
        "2222/tcp"
    )

    configure_firewalld_rules enable_ports

    # shellcheck disable=SC2153
    local -r mdmimage="${RPIMAGE%%/*}/${MDMIMAGE#*/}"
    local -r rpimage="$RPIMAGE"
    local -r fluentbit_image="$FLUENTBITIMAGE"
    local -rA aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
    )
    pull_container_images aro_images true

    # LOGKIND appears to no longer be a variable that is carried over by the deploy pipeline
    # Substituting it with an empty string
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
	Rule ${LOGKIND:-} asyncqos asyncqos true

[FILTER]
	Name modify
	Match asyncqos
	Remove CLIENT_PRINCIPAL_NAME
	Remove FILE
	Remove COMPONENT

[FILTER]
	Name rewrite_tag
	Match journald
	Rule ${LOGKIND:-} ifxaudit ifxaudit false

[OUTPUT]
	Name forward
	Match *
	Port 29230"


    local -r mdsd_config_version="$RPMDSDCONFIGVERSION"
    local -r dbtoken_config_file="DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
    AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
    AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
    KEYVAULT_PREFIX='$KEYVAULTPREFIX'
    MDM_ACCOUNT='$RPMDMACCOUNT'
    MDM_NAMESPACE=DBToken
    RPIMAGE='$rpimage'"

    local -r aro_rp_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
ARM_API_CLIENT_CERT_COMMON_NAME='$ARMAPICLIENTCERTCOMMONNAME'
AZURE_ARM_CLIENT_ID='$ARMCLIENTID'
AZURE_FP_CLIENT_ID='$FPCLIENTID'
AZURE_FP_SERVICE_PRINCIPAL_ID='$FPSERVICEPRINCIPALID'
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
MDM_NAMESPACE='${role_rp^^}'
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
RP_FEATURES='$RPFEATURES'
RPIMAGE='$rpimage'
ARO_INSTALL_VIA_HIVE='$CLUSTERSINSTALLVIAHIVE'
ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC='$CLUSTERDEFAULTINSTALLERPULLSPEC'
ARO_ADOPT_BY_HIVE='$CLUSTERSADOPTBYHIVE'
USE_CHECKACCESS='$USECHECKACCESS'"

    # values are references to variables, they should not be dereferenced here
    local -rA aro_configs=(
        ["rp_config"]="aro_rp_conf_file"
        ["dbtoken"]="dbtoken_config_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["mdsd"]="mdsd_config_version"
    )

    configure_vmss_aro_services role_rp \
                                aro_images \
                                aro_configs

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

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
