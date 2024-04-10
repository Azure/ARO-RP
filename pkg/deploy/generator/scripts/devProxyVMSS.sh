#!/bin/bash

set -o errexit \
    -o nounset

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

main() {
    parse_run_options "$@"

    configure_and_install_dnf_pkgs_repos

    configure_firewalld_rules
    pull_container_images
    configure_system_services
    reboot_vm
}

usage() {
    local -n options="$1"
    log "$(basename "$0") [$options]
        -p Configure rpm repositories, import required rpm keys, update & install packages with dnf
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
	local -r allowed_options="pfui"
    while getopts ${allowed_options} options; do
        case "${options}" in
            p)
                log "Running step configure_and_install_dnf_pkgs_repos"
                configure_and_install_dnf_pkgs_repos
                ;;
            f)
                log "Running configure_firewalld_rules"
                configure_firewalld_rules
                ;;
            u)
                log "Running pull_container_images & configure_system_services"
                configure_system_services
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

# configure_and_install_dnf_pkgs_repos
configure_and_install_dnf_pkgs_repos() {
    log "starting"

    # transaction attempt retry time in seconds
    local -ri retry_wait_time=60
    configure_rhui_repo retry_wait_time

    local -ar exclude_pkgs=(
        "-x WALinuxAgent"
        "-x WALinuxAgent-udev"
    )

    dnf_update_pkgs exclude_pkgs retry_wait_time

    local -ra repo_rpm_pkgs=(
        https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    )

    local -ra install_pkgs=(
        podman
        podman-docker
    )

    dnf_install_pkgs repo_rpm_pkgs retry_wait_time
    dnf_install_pkgs install_pkgs retry_wait_time

	local -r cron_weekly_dnf_update_filename='/etc/cron.weekly/dnfupdate'
	local -r cron_weekly_dnf_update_file="#!/bin/bash
dnf update -y"

	write_file cron_weekly_dnf_update_filename cron_weekly_dnf_update_file true
	chmod +x "$cron_weekly_dnf_update_filename"
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

# configure_logrotate clobbers /etc/logrotate.conf
configure_logrotate() {
    log "starting"

    local -r logrotate_conf_filename='/etc/logrotate.conf'
    local -r logrotate_conf_file='# see "man logrotate" for details
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
}'

    write_file logrotate_conf_filename logrotate_conf_file true
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
	local -r registry_config_file="{
	"auths": {
		\"${PROXYIMAGE%%/*}\": {
			\"auth\": \"$PROXYIMAGEAUTH\"
		}
	}"

	write_file REGISTRY_AUTH_FILE registry_config_file true

    log "logging into prod acr"

    az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

    docker pull "$PROXYIMAGE"

    az logout
}

# configure_system_services creates, configures, and enables the following systemd services and timers
# services
#	proxy
configure_system_services() {
	configure_service_proxy
}

# enable_aro_services enables all services required for aro rp
enable_aro_services() {
    log "starting"

    local -ra aro_services=(
		proxy
    )
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
	local -r sysconfig_proxy_file="PROXY_IMAGE='$PROXYIMAGE'"

	write_file sysconfig_proxy_filename sysconfig_proxy_file true

	local -r proxy_service_filename='/etc/systemd/system/proxy.service'
	local -r proxy_service_file="[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $PROXY_IMAGE
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target"

	local -r cron_weekly_pull_image_filename='/etc/cron.weekly/pull-image'
	local -r cron_weekly_pull_image_file="#!/bin/bash
docker pull $PROXYIMAGE
systemctl restart proxy.service"
	
	write_file cron_weekly_pull_image_filename cron_weekly_pull_image_file true
	chmod +x "$cron_weekly_pull_image_filename"

	local -r cron_daily_restart_proxy_filename='/etc/cron.daily/restart-proxy'
	local -r cron_daily_restart_proxy_file="#!/bin/bash
systemctl restart proxy.service"
	
	write_file cron_daily_restart_proxy_filename cron_daily_restart_proxy_file
	chmod +x "$cron_daily_restart_proxy_filename"
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
