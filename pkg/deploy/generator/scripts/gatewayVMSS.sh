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

    # log directory to be mounted to running container
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

    # shellcheck disable=SC2153
    local -r mdmimage="${RPIMAGE%%/*}/${MDMIMAGE#*/}"
    local -r rpimage="$RPIMAGE"
    local -r fluentbit_image="$FLUENTBITIMAGE"
    # values are references to variables, they should not be dereferenced here
    local -rA aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
    )
    pull_container_images aro_images true

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

    local -r aro_gateway_conf_file="ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE='${role_gateway^}'
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$rpimage'"

    local -r mdsd_config_version="$GATEWAYMDSDCONFIGVERSION"
    # values are references to variables, they should not be dereferenced here
    local -rA aro_configs=(
        ["gateway_config"]="aro_gateway_conf_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["mdsd"]="mdsd_config_version"
        ["log_dir"]="gateway_logdir"
    )

    configure_vmss_aro_services role_gateway \
                                aro_images \
                                aro_configs

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

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
