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
    local use_kusto=false
    if [[ -n "${GATEWAYKUSTOCLUSTERURI:-}" ]]; then
      use_kusto=true
    fi

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

    local -a install_pkgs=(
        azure-cli
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
      if [[ "$use_kusto" != true ]]; then
        install_pkgs+=(azure-mdsd)
      fi

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
    # shellcheck disable=SC2034
    local -r cluster_mdsd_image="${RPIMAGE%%/*}/${CLUSTERMDSDIMAGE#*/}"
    # values are references to variables, they should not be dereferenced here
    local -A aro_images=(
        ["mdm"]="mdmimage"
        ["rp"]="rpimage"
        ["fluentbit"]="fluentbit_image"
        ["otelcollector"]="otel_collector_image"
    )
    if [[ "$use_kusto" != true ]]; then
      aro_images["clustermdsd"]="cluster_mdsd_image"
    fi

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
    local -r cluster_mdsd_conf_file="MDSDIMAGE='$cluster_mdsd_image'
MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
MONITORING_GCS_ACCOUNT='$CLUSTERMDSDACCOUNT'
MONITORING_GCS_REGION='$LOCATION'
MONITORING_GCS_AUTH_ID_TYPE=AuthMSIToken
MONITORING_GCS_AUTH_ID=mi_res_id#\${GATEWAYUSERASSIGNEDIDENTITYRESOURCEID}
MONITORING_GCS_NAMESPACE='$CLUSTERMDSDNAMESPACE'
MONITORING_CONFIG_VERSION='$OTELCLUSTERMDSDCONFIGVERSION'
MONITORING_USE_GENEVA_CONFIG_SERVICE=true
MONITORING_TENANT='$LOCATION'
MONITORING_ROLE=cluster
MONITORING_ROLE_INSTANCE=\"\$(hostname)\"
MONITORING_ENVIRONMENT='$ENVIRONMENT'
ENABLE_GIG_BRIDGE_MODE=1"

    # shellcheck disable=SC2034
    local -r gateway_kusto_cluster_uri="${GATEWAYKUSTOCLUSTERURI:-}"
    # shellcheck disable=SC2034
    local -r gateway_kusto_db_name="${GATEWAYKUSTODBNAME:-oteldb}"
    # shellcheck disable=SC2034
    local -r gateway_kusto_ingestion_type="${GATEWAYKUSTOINGESTIONTYPE:-queued}"
    # shellcheck disable=SC2034
    local -r gateway_kusto_logs_table_name="${GATEWAYKUSTOLOGSTABLENAME:-OTELLogs}"
    # shellcheck disable=SC2034
    local -r gateway_kusto_managed_identity_client_id="${GATEWAYKUSTOMANAGEDIDENTITYCLIENTID:-system}"

    # shellcheck disable=SC2034
    local -r gateway_otel_collector_conf_prefix="extensions:
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

exporters:"

    # shellcheck disable=SC2034
    local -r gateway_otel_collector_conf_suffix="
processors:
  attributes/cluster:
    actions:
      - key: resourceid
        from_context: \"auth.clusterResourceID\"
        action: upsert

  memory_limiter:
    check_interval: 1s
    limit_mib: 512

  batch:
    timeout: 10s
    send_batch_size: 1024

service:
  extensions:
    - health_check
    - gatewayauth
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, attributes/cluster, batch]"

    # shellcheck disable=SC2034
    local gateway_otel_collector_conf
    if [[ -n "$gateway_kusto_cluster_uri" ]]; then
        gateway_otel_collector_conf="${gateway_otel_collector_conf_prefix}
  azuredataexplorer:
    cluster_uri: '$gateway_kusto_cluster_uri'
    managed_identity_id: '$gateway_kusto_managed_identity_client_id'
    db_name: '$gateway_kusto_db_name'
    logs_table_name: '$gateway_kusto_logs_table_name'
    ingestion_type: '$gateway_kusto_ingestion_type'
    sending_queue:
      enabled: true
      num_consumers: 2
      queue_size: 1000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 10m
${gateway_otel_collector_conf_suffix}
      exporters: [azuredataexplorer]"
    else
        gateway_otel_collector_conf="${gateway_otel_collector_conf_prefix}
  otlp/cluster-mdsd:
    endpoint: localhost:2020
    tls:
      insecure: true
${gateway_otel_collector_conf_suffix}
      exporters: [otlp/cluster-mdsd]"
    fi

    # values are references to variables, they should not be dereferenced here
    local -A aro_configs=(
        ["gateway_config"]="aro_gateway_conf_file"
        ["fluentbit"]="fluentbit_conf_file"
        ["gateway_otel_collector"]="gateway_otel_collector_conf"
        ["static_ip_address"]="static_ip_addresses"
    )
    if [[ "$use_kusto" != true ]]; then
      aro_configs["mdsd"]="mdsd_config_version"
      aro_configs["cluster_mdsd"]="cluster_mdsd_conf_file"
    fi

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

    local -a gateway_services=(
        "aro-gateway"
        "azsecd"
        "mdm"
        "chronyd"
        "fluentbit"
        "gateway-otel-collector"
        "download-mdm-credentials.timer"
        "download-gateway-otel-credentials.timer"
        "firewalld"
    )
    if [[ "$use_kusto" != true ]]; then
      gateway_services+=("mdsd" "cluster-mdsd" "download-mdsd-credentials.timer")
    fi

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
