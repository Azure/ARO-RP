#!/bin/bash

set -o errexit \
    -o pipefail \
    -o nounset

main() {
    # transaction attempt retry time in seconds
    # shellcheck disable=SC2034
    local -ri retry_wait_time=30
    # shellcheck disable=SC2068
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

    # shellcheck disable=SC2034 disable=SC2153
    local -r mdmimage="${RPIMAGE%%/*}/${MDMIMAGE#*/}"
    local -r rpimage="$RPIMAGE"
    # shellcheck disable=SC2034
    local -r fluentbit_image="$FLUENTBITIMAGE"
    # values are references to variables, they should not be dereferenced here
    # shellcheck disable=SC2034
    local -rA aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
    )

    pull_container_images aro_images

    # shellcheck disable=SC2034
    local -ra enable_ports=(
        # RP gateway
        "80/tcp"
        "8081/tcp"
        "443/tcp"
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

[OUTPUT]
	Name forward
	Match *
	Port 29230"

    # shellcheck disable=SC2034
    local -r aro_gateway_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE='${role_gateway^}'
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$rpimage'"

    # shellcheck disable=SC2034
    local -r mdsd_config_version="$GATEWAYMDSDCONFIGVERSION"

    # values are references to variables, they should not be dereferenced here
    # shellcheck disable=SC2034
    local -rA aro_configs=(
        ["gateway_config"]="aro_gateway_conf_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["mdsd"]="mdsd_config_version"
        ["static_ip_address"]="static_ip_addresses"
    )

    # shellcheck disable=SC2034
    # use default podman network with range 10.88.0.0/16
    local -rA static_ip_addresses=(
        ["gateway"]="10.88.0.2"
        ["fluentbit"]="10.88.0.7"
        ["mdm"]="10.88.0.8"
    )

    configure_vmss_aro_services role_gateway \
                                aro_images \
                                aro_configs

    # shellcheck disable=SC2034
    local -ra gateway_services=(
        "aro-gateway"
        "azsecd"
        "mdsd"
        "mdm"
        "chronyd"
        "fluentbit"
        "download-mdsd-credentials.timer"
        "download-mdm-credentials.timer"
        "firewalld"
    )

    enable_services gateway_services

    reboot_vm
}

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

# util.sh does not exist when deployed to VMSS via VMSS extensions
# Provides shellcheck definitions
util="util.sh"
if [ -f "$util" ]; then
    # shellcheck source=util.sh
    source "$util"
fi

main "$@"
