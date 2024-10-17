#!/bin/bash

set -o errexit \
    -o pipefail \
    -o nounset

main() {
    # transaction attempt retry time in seconds
    # shellcheck disable=SC2034
    local -ri retry_wait_time=30
    local -ri pkg_retry_count=60

    create_required_dirs
    configure_sshd
    configure_rpm_repos retry_wait_time \
                        "$pkg_retry_count"

    # shellcheck disable=SC2034
    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs exclude_pkgs \
                    retry_wait_time \
                    "$pkg_retry_count"

    # shellcheck disable=SC2034
    local -ra install_pkgs=(
        clamav
        azsec-clamav
        azure-cli
        azure-mdsd
        azure-security
        podman
        podman-docker
        openssl-perl
        # hack - we are installing python3 on hosts due to an issue with Azure Linux Extensions https://github.com/Azure/azure-linux-extensions/pull/1505
        python3
        # required for podman networking
        firewalld
        # Required to enable fips
        grubby
        dracut-fips
    )

    dnf_install_pkgs install_pkgs \
                      retry_wait_time \
                      "$pkg_retry_count"

    fips_configure

    # shellcheck disable=SC2119
    configure_logrotate

    # shellcheck disable=SC2153 disable=SC2034
    local -r mdmimage="${RPIMAGE%%/*}/${MDMIMAGE#*/}"
    local -r rpimage="$RPIMAGE"
    local -r miseimage="${RPIMAGE%%/*}/${MISEIMAGE#*/}"
    local -r otelimage="$OTELIMAGE"
    # shellcheck disable=SC2034
    local -r fluentbit_image="$FLUENTBITIMAGE"
    # shellcheck disable=SC2034
    local -rA aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
        ["mise"]="miseimage"
        ["otel"]="otelimage"
    )

    pull_container_images aro_images

    # shellcheck disable=SC2034
    local -ra enable_ports=(
        # RP frontend
        "443/tcp"
        # Portal web
        "444/tcp"
        # Portal ssh
        "2222/tcp"
        # JIT ssh
        "22/tcp"
    )

    firewalld_configure enable_ports

    # shellcheck disable=SC2034
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
	Rule \$LOGKIND asyncqos asyncqos true

[FILTER]
	Name modify
	Match asyncqos
	Remove CLIENT_PRINCIPAL_NAME
	Remove FILE
	Remove COMPONENT

[FILTER]
	Name rewrite_tag
	Match journald
	Rule \$LOGKIND ifxaudit ifxaudit false

[OUTPUT]
	Name forward
	Match *
	Port 29230"

    # values are references to variables, they should not be dereferenced here
    # shellcheck disable=SC2034
    local -rA aro_configs=(
        ["rp_config"]="aro_rp_conf_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["mdsd"]="mdsd_config_version"
        ["static_ip_address"]="static_ip_addresses"
    )

    # shellcheck disable=SC2034
    # use default podman network with range 10.88.0.0/16
    local -rA static_ip_addresses=(
        ["rp"]="10.88.0.2"
        ["monitor"]="10.88.0.3"
        ["portal"]="10.88.0.4"
        ["mise"]="10.88.0.5"
        ["otel_collector"]="10.88.0.6"
        ["fluentbit"]="10.88.0.7"
        ["mdm"]="10.88.0.8"
    )

    # shellcheck disable=SC2034
    local -r mdsd_config_version="$RPMDSDCONFIGVERSION"
    # shellcheck disable=SC2034
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
MISE_ADDRESS='http://${static_ip_addresses["mise"]}:5000'
RP_FEATURES='$RPFEATURES'
RPIMAGE='$rpimage'
ARO_INSTALL_VIA_HIVE='$CLUSTERSINSTALLVIAHIVE'
ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC='$CLUSTERDEFAULTINSTALLERPULLSPEC'
ARO_ADOPT_BY_HIVE='$CLUSTERSADOPTBYHIVE'
OIDC_AFD_ENDPOINT='$LOCATION.oic.$RPPARENTDOMAINNAME'
OIDC_STORAGE_ACCOUNT_NAME='$OIDCSTORAGEACCOUNTNAME'
MSI_RP_ENDPOINT='$MSIRPENDPOINT'
"

    configure_vmss_aro_services role_rp \
                                aro_images \
                                aro_configs

    # shellcheck disable=SC2034
    local -ra aro_services=(
        "aro-mise"
        "aro-monitor"
        "aro-otel-collector"
        "aro-portal"
        "aro-rp"
        "azsecd"
        "mdsd"
        "mdm"
        "chronyd"
        "fluentbit"
        "download-mdsd-credentials.timer"
        "download-mdm-credentials.timer"
        "firewalld"
    )

    enable_services aro_services

    reboot_vm
}

# This variable is used by az-cli
# It's assumed that if this variable hasn't been carried over, that others are also not present, so we fail early by returning an error
# This was mostly helpful when testing on a development VM, but is still applicable
export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

# util.sh does not exist when deployed to VMSS via VMSS extensions
# Provides shellcheck definitions
util="util.sh"
if [ -f "$util" ]; then
    # shellcheck source=util.sh
    source "$util"
fi

main "$@"
