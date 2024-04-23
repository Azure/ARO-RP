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

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs pkgs_to_exclude retry_wait_time

    local -ra install_pkgs=(
        podman
        podman-docker
    )

    dnf_install_pkgs install_pkgs retry_wait_time
    configure_dnf_cron_job

    local -ra enable_ports=(
        "443/tcp"
    )

    configure_firewalld_rules enable_ports

    local -ra proxy_images=("$PROXYIMAGE")
	local -r registry_config_file="{
    \"auths\": {
        \"${proxy_images[0]%%/*}\": {
            \"auth\": \"$PROXYIMAGEAUTH\"
        }
    }
}"

    pull_container_images proxy_image registry_config_file false
    configure_devproxy_services proxy_images

    local -r vmss_role="devproxy"
    configure_certs vmss_role

    local -ra proxy_services=(
        proxy
    )

    enable_services proxy_services
    reboot_vm
}

# configure_system_services creates, configures, and enables the following systemd services and timers
# services
#	proxy
configure_devproxy_services() {
	configure_service_proxy "$1"
}

configure_service_proxy() {
    local -n proxy_image="$1"
	local -r sysconfig_proxy_filename='/etc/sysconfig/proxy'
	local -r sysconfig_proxy_file="PROXY_IMAGE='$proxy_image'"

	write_file sysconfig_proxy_filename sysconfig_proxy_file true

	local -r proxy_service_filename='/etc/systemd/system/proxy.service'
	local -r proxy_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $proxy_image
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file proxy_service_filename proxy_service_file true

	local -r cron_weekly_pull_image_filename='/etc/cron.weekly/pull-image'
	local -r cron_weekly_pull_image_file="#!/bin/bash
docker pull $proxy_image
systemctl restart proxy.service"
	
	write_file cron_weekly_pull_image_filename cron_weekly_pull_image_file true
	chmod +x "$cron_weekly_pull_image_filename"

	local -r cron_daily_restart_proxy_filename='/etc/cron.daily/restart-proxy'
	local -r cron_daily_restart_proxy_file="#!/bin/bash
systemctl restart proxy.service"
	
	write_file cron_daily_restart_proxy_filename cron_daily_restart_proxy_file true
	chmod +x "$cron_daily_restart_proxy_filename"
}

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
