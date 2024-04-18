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

    local -ra enable_ports=(
        "443/tcp"
    )

    local -ra proxy_images=("$PROXYIMAGE")
	local -r registry_config_file="{
    \"auths\": {
        \"${proxy_images[0]%%/*}\": {
            \"auth\": \"$PROXYIMAGEAUTH\"
        }
    }"

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    local -ra install_pkgs=(
        podman
        podman-docker
    )

    local -ra proxy_services=(
		proxy
    )

    local -ra user_options=("$@")
    parse_run_options user_options \
                        exclude_pkgs \
                        install_pkgs \
                        enable_ports \
                        proxy_services \
                        proxy_images

    dnf_update_pkgs pkgs_to_exclude retry_wait_time
    dnf_install_pkgs install_pkgs retry_wait_time
    configure_dnf_cron_job

    configure_firewalld_rules enable_ports
    pull_container_images proxy_image registry_config_file
    configure_devproxy_services proxy_images
    enable_services proxy_services
    reboot_vm
}

usage() {
    local -n options="$1"
    log "$(basename "$0") [$options]
        -p Configure rpm repositories, import required rpm keys, update & install packages with dnf
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
    local -n pkgs_to_exclude="$2"
    local -n pkgs_to_install="$3"
    local -n ports_to_enable="$4"
    local -n services_to_enable="$5"
    local -n images_to_pull="$6"

    if [ "${#run_options[@]}" -eq 0 ]; then
        log "Running all steps"
        return 0
    fi

    local OPTIND
	local -r allowed_options="pfui"
    while getopts ${allowed_options} options; do
        case "${run_options}" in
            p)
                log "Running step dnf_update_pkgs"
                dnf_update_pkgs pkgs_to_exclude retry_wait_time

                log "Running step dnf_install_pkgs"
                dnf_install_pkgs pkgs_to_install retry_wait_time

                log "Running step configure_dnf_cron_job"
                configure_dnf_cron_job
                ;;
            f)
                log "Running configure_firewalld_rules"
                configure_firewalld_rules ports_to_enable
                ;;
            u)
                log "Running step configure_devproxy_services"
                configure_devproxy_services proxy_images
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

# configure_system_services creates, configures, and enables the following systemd services and timers
# services
#	proxy
configure_devproxy_services() {
	configure_service_proxy "$1"
}

# enable_aro_services enables all services required for aro rp
enable_aro_services() {
    log "starting"

    log "enabling aro services ${aro_services[*]}"
    # shellcheck disable=SC2068
    for service in ${aro_services[@]}; do
        log "Enabling $service now"
        systemctl enable "$service.service"
    done
}

# configure_certs
configure_certs() {
    log "starting"

	base64 -d <<<"$PROXYCERT" >/etc/proxy/proxy.crt
	base64 -d <<<"$PROXYKEY" >/etc/proxy/proxy.key
	base64 -d <<<"$PROXYCLIENTCERT" >/etc/proxy/proxy-client.crt
	chown -R 1000:1000 /etc/proxy
	chmod 0600 /etc/proxy/proxy.key
}

configure_service_proxy() {
	local -r sysconfig_proxy_filename='/etc/sysconfig/proxy'
	local -r sysconfig_proxy_file="PROXY_IMAGE='$1'"

	write_file sysconfig_proxy_filename sysconfig_proxy_file true

	local -r proxy_service_filename='/etc/systemd/system/proxy.service'
	local -r proxy_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $1
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

    write_file proxy_service_filename proxy_service_file true

	local -r cron_weekly_pull_image_filename='/etc/cron.weekly/pull-image'
	local -r cron_weekly_pull_image_file="#!/bin/bash
docker pull $1
systemctl restart proxy.service"
	
	write_file cron_weekly_pull_image_filename cron_weekly_pull_image_file true
	chmod +x "$cron_weekly_pull_image_filename"

	local -r cron_daily_restart_proxy_filename='/etc/cron.daily/restart-proxy'
	local -r cron_daily_restart_proxy_file="#!/bin/bash
systemctl restart proxy.service"
	
	write_file cron_daily_restart_proxy_filename cron_daily_restart_proxy_file
	chmod +x "$cron_daily_restart_proxy_filename"
}

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

main "$@"
