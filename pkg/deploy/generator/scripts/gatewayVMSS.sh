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
        azure-cli
        azure-mdsd
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
    # shellcheck disable=SC2034
    local -r otel_collector_image="$GATEWAYOTELCOLLECTORIMAGE"
    # values are references to variables, they should not be dereferenced here
    # shellcheck disable=SC2034
    local -rA aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
        ["otelcollector"]="otel_collector_image"
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
        # OTel collector
        "4317/tcp"
        "13133/tcp"
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
	Add Environment \${ENVIRONMENT}

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
RPIMAGE='$rpimage'
ENVIRONMENT='$ENVIRONMENT'"

    # shellcheck disable=SC2034
    local -r mdsd_config_version="$GATEWAYMDSDCONFIGVERSION"

    # shellcheck disable=SC2034
    local -r azuremonitor_tenant_conf_file="### Geneva Linux Agent tenant settings file

TENANT_NAME=AROClusterLogs
MDSD_VAR=/var/opt/microsoft/azuremonitoragent
MDSD_CONFIG_DIR=/etc/opt/microsoft/azuremonitoragent/\${TENANT_NAME}
MDSD_RUN_DIR=/var/run/azuremonitoragent/\${TENANT_NAME}
MDSD_ROLE_PREFIX=\${MDSD_RUN_DIR}/default
MDSD_LOG=\${MDSD_VAR}/log/\${TENANT_NAME}
MDSD_SPOOL_DIRECTORY=\${MDSD_VAR}/spool/\${TENANT_NAME}

MDSD_OPTIONS=\"-A -c /etc/opt/microsoft/azuremonitoragent/mdsd.xml -C -R -r \${MDSD_ROLE_PREFIX} -S \${MDSD_SPOOL_DIRECTORY}/eh -e \${MDSD_LOG}/\${TENANT_NAME}.err -w \${MDSD_LOG}/\${TENANT_NAME}.warn -o \${MDSD_LOG}/\${TENANT_NAME}.info -q \${MDSD_LOG}/\${TENANT_NAME}.qos\"

MONITORING_TENANT=$LOCATION
MONITORING_ROLE=cluster
MONITORING_ROLE_INSTANCE=$(hostname)

MONITORING_GCS_ENVIRONMENT=$MDSDENVIRONMENT
MONITORING_GCS_ACCOUNT=$CLUSTERMDSDACCOUNT
MONITORING_GCS_NAMESPACE=$CLUSTERMDSDNAMESPACE
MONITORING_GCS_REGION=$LOCATION
MONITORING_CONFIG_VERSION=$OTELCLUSTERMDSDCONFIGVERSION
MONITORING_USE_GENEVA_CONFIG_SERVICE=true
MONITORING_GCS_AUTH_ID_TYPE=AuthMSIToken
MONITORING_GCS_AUTH_ID=mi_res_id#${GATEWAYUSERASSIGNEDIDENTITYRESOURCEID}
MONITORING_ENVIRONMENT=$ENVIRONMENT

ENABLE_GIG_BRIDGE_MODE=1"

    # shellcheck disable=SC2034
    local -r gateway_otel_collector_conf="extensions:
  health_check:
    endpoint: 0.0.0.0:13133
  gatewayauth:
    tls:
      cert_file: /etc/otel-collector/tls/tls-cert.pem
      key_file: /etc/otel-collector/tls/tls-key.pem

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        middlewares:
          - id: gatewayauth
        auth:
          authenticator: gatewayauth

exporters:
  otlp/cluster-mdsd:
    endpoint: host.containers.internal:2020
    tls:
      insecure: true
    # Gateway Otel Collector runs alongside mission-critical workloads:
    # allow only a very small retry budget, then fail fast.
    retry_on_failure:
      enabled: true
      initial_interval: 1s
      max_interval: 1s
      max_elapsed_time: 2s
    sending_queue:
      enabled: true
      queue_size: 128
      num_consumers: 2

processors:
  attributes/cluster:
    actions:
      - key: environment
        value: \"${ENVIRONMENT,,}\"
        action: upsert
      - key: region
        value: \"${LOCATION,,}\"
        action: upsert
      - key: subscription_id
        from_context: \"auth.clusterSubscriptionID\"
        action: upsert
      - key: resource_group
        from_context: \"auth.clusterResourceGroup\"
        action: upsert
      - key: resource_name
        from_context: \"auth.clusterResourceName\"
        action: upsert
      - key: resource_id
        from_context: \"auth.clusterResourceID\"
        action: upsert

  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 64

  batch:
    timeout: 30s
    send_batch_size: 4096
    send_batch_max_size: 8192

service:
  extensions:
    - health_check
    - gatewayauth
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, attributes/cluster, batch]
      exporters: [otlp/cluster-mdsd]"

    # values are references to variables, they should not be dereferenced here
    # shellcheck disable=SC2034
    local -rA aro_configs=(
        ["gateway_config"]="aro_gateway_conf_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["mdsd"]="mdsd_config_version"
        ["gateway_otel_collector"]="gateway_otel_collector_conf"
        ["azuremonitor_tenant"]="azuremonitor_tenant_conf_file"
        ["static_ip_address"]="static_ip_addresses"
    )

    # shellcheck disable=SC2034
    # use default podman network with range 10.88.0.0/16
    local -rA static_ip_addresses=(
        ["gateway"]="10.88.0.2"
        ["otelcollector"]="10.88.0.9"
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
        "gateway-otel-collector"
        "download-mdsd-credentials.timer"
        "download-mdm-credentials.timer"
        "download-gateway-otel-credentials.timer"
        "firewalld"
    )

    enable_services gateway_services

    reboot_vm
}

# export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"
export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

# util="util.sh"
#
# util.sh does not exist when deployed to VMSS via VMSS extensions
# Provides shellcheck definitions
util="util.sh"
if [ -f "$util" ]; then
    # shellcheck source=util.sh
    source "$util"
fi

main "$@"
