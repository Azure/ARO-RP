#!/bin/bash
# ARO service setup functions

# enable_services enables the systemd services that are passed in
# args:
# 1) services - array; services to be enabled
enable_services() {
    local -n svcs="$1"
    log "starting"

    systemctl daemon-reload

    log "enabling services ${svcs[*]}"
    # shellcheck disable=SC2068
    for svc in ${svcs[@]}; do
        log "Enabling and starting $svc now"
        systemctl enable \
                  --now \
                  "$svc"
    done
}

# configure_service_aro_gateway
# args:
# 1) image - nameref, string; container image
# 2) role - nameref, string; VMSS role
# 3) conf_file - nameref, string; aro gateway environment file
# 4) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_gateway() {
    local -n image="$1"
    local -n role="$2"
    local -n conf_file="$3"
    local -n ipaddress="$4"
    log "starting"
    log "Configuring aro-gateway service"

    local -r aro_gateway_conf_filename='/etc/sysconfig/aro-gateway'
    local -r add_conf_file="PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'
ROLE='${role,,}'"

    write_file aro_gateway_conf_filename conf_file true
    write_file aro_gateway_conf_filename add_conf_file false

    # shellcheck disable=SC2034
    local -r aro_gateway_service_filename='/etc/systemd/system/aro-gateway.service'

    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script) 
    local -r aro_gateway_service_file='[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-gateway
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_FEATURES \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -p 80:8080 \
  -p 8081:8081 \
  -p 443:8443 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  $ROLE
ExecStop=/usr/bin/podman stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
    '

    write_file aro_gateway_service_filename aro_gateway_service_file true
}

# configure_service_aro_rp
# args:
# 1) image - nameref, string; RP container image
# 2) role - nameref, string; VMSS role
# 3) conf_file - nameref, string; aro rp environment file
# 4) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_rp() {
    local -n image="$1"
    local -n role="$2"
    local -n conf_file="$3"
    local -n ipaddress="$4"
    log "starting"
    log "Configuring aro-rp service"

    local -r aro_rp_conf_filename='/etc/sysconfig/aro-rp'
    local -r add_conf_file="PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'
ROLE='${role,,}'"

    write_file aro_rp_conf_filename conf_file true
    write_file aro_rp_conf_filename add_conf_file false

    # shellcheck disable=SC2034
    local -r aro_rp_service_filename='/etc/systemd/system/aro-rp.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r aro_rp_service_file='[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e ADMIN_API_CLIENT_CERT_COMMON_NAME \
  -e ARM_API_CLIENT_CERT_COMMON_NAME \
  -e AZURE_ARM_CLIENT_ID \
  -e AZURE_FP_CLIENT_ID \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e CLUSTER_MDSD_ACCOUNT \
  -e CLUSTER_MDSD_CONFIG_VERSION \
  -e CLUSTER_MDSD_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e DOMAIN_NAME \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_RESOURCEGROUP \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e MDSD_ENVIRONMENT \
  -e RP_FEATURES \
  -e ARO_INSTALL_VIA_HIVE \
  -e ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC \
  -e ARO_ADOPT_BY_HIVE \
  -e OIDC_AFD_ENDPOINT \
  -e OIDC_STORAGE_ACCOUNT_NAME \
  -e MSI_RP_ENDPOINT \
  -e OTEL_AUDIT_QUEUE_SIZE \
  -e MISE_ADDRESS \
  -m 2g \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -p 443:8443 \
  -v /etc/aro-rp:/etc/aro-rp \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  -v /var/run/mdsd/asa:/var/run/mdsd/asa:z \
  $RPIMAGE \
  $ROLE
ExecStop=/usr/bin/podman stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target'

    write_file aro_rp_service_filename aro_rp_service_file true
}

# configure_service_aro_monitor
# args:
# 1) image - nameref, string; RP container image
# 2) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_monitor() {
    local -n image="$1"
    local -n ipaddress="$2"
    log "starting"
    log "Configuring aro-monitor service"

    # DOMAIN_NAME, CLUSTER_MDSD_ACCOUNT, CLUSTER_MDSD_CONFIG_VERSION, GATEWAY_DOMAINS, GATEWAY_RESOURCEGROUP, MDSD_ENVIRONMENT CLUSTER_MDSD_NAMESPACE
    # are not used, but can't easily be refactored out. Should be revisited in the future.
    # shellcheck disable=SC2034
    local -r aro_monitor_service_conf_filename='/etc/sysconfig/aro-monitor'
    # shellcheck disable=SC2034
    local -r aro_monitor_service_conf_file="AZURE_FP_CLIENT_ID='$FPCLIENTID'
DOMAIN_NAME='$LOCATION.$CLUSTERPARENTDOMAINNAME'
CLUSTER_MDSD_ACCOUNT='$CLUSTERMDSDACCOUNT'
CLUSTER_MDSD_CONFIG_VERSION='$CLUSTERMDSDCONFIGVERSION'
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_RESOURCEGROUP='$GATEWAYRESOURCEGROUPNAME'
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
CLUSTER_MDSD_NAMESPACE='$CLUSTERMDSDNAMESPACE'
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMACCOUNT'
CLUSTER_MDM_NAMESPACE=BBM
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=BBM
RPIMAGE='$image'
PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'"

    write_file aro_monitor_service_conf_filename aro_monitor_service_conf_file true

    # shellcheck disable=SC2034
    local -r aro_monitor_service_filename='/etc/systemd/system/aro-monitor.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r aro_monitor_service_file='[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -e AZURE_FP_CLIENT_ID \
  -e DOMAIN_NAME \
  -e CLUSTER_MDSD_ACCOUNT \
  -e CLUSTER_MDSD_CONFIG_VERSION \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_RESOURCEGROUP \
  -e MDSD_ENVIRONMENT \
  -e CLUSTER_MDSD_NAMESPACE \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2.5g \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  monitor
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target'

    write_file aro_monitor_service_filename aro_monitor_service_file true
}

# configure_service_aro_portal
# args:
# 1) image - nameref, string; RP container image
# 2) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_portal() {
    local -n image="$1"
    local -n ipaddress="$2"
    log "starting"
    log "Configuring aro portal service"

    # shellcheck disable=SC2034
    local -r aro_portal_service_conf_filename='/etc/sysconfig/aro-portal'
    # shellcheck disable=SC2034
    local -r aro_portal_service_conf_file="AZURE_PORTAL_ACCESS_GROUP_IDS='$PORTALACCESSGROUPIDS'
AZURE_PORTAL_CLIENT_ID='$PORTALCLIENTID'
AZURE_PORTAL_ELEVATED_GROUP_IDS='$PORTALELEVATEDGROUPIDS'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Portal
PORTAL_HOSTNAME='$LOCATION.admin.$RPPARENTDOMAINNAME'
OTEL_AUDIT_QUEUE_SIZE='$OTELAUDITQUEUESIZE'
RPIMAGE='$image'
PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'"

    write_file aro_portal_service_conf_filename aro_portal_service_conf_file true

    # shellcheck disable=SC2034
    local -r aro_portal_service_filename='/etc/systemd/system/aro-portal.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r aro_portal_service_file='[Unit]
After=network-online.target
Wants=network-online.target
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-portal
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -e AZURE_PORTAL_ACCESS_GROUP_IDS \
  -e AZURE_PORTAL_CLIENT_ID \
  -e AZURE_PORTAL_ELEVATED_GROUP_IDS \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e PORTAL_HOSTNAME \
  -e OTEL_AUDIT_QUEUE_SIZE \
  -m 2g \
  -p 444:8444 \
  -p 2222:2222 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  -v /var/run/mdsd/asa:/var/run/mdsd/asa:z \
  $RPIMAGE \
  portal
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target'

    write_file aro_portal_service_filename aro_portal_service_file true
}

# configure_service_aro_mise
# args:
# 1) image - nameref, string; MISE container image
# 2) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_mise() {
    local -n image="$1"
    local -n ipaddress="$2"
    log "starting"
    log "Configuring aro-mise service"

    LOGININSTANCE="https://login.microsoftonline.com"
    if [[ $AZURECLOUDNAME == "$us_gov_cloud" ]]; then
        LOGININSTANCE="https://login.microsoftonline.us"
    fi
    # shellcheck disable=SC2034
    local -r aro_mise_service_conf_filename='/etc/sysconfig/aro-mise'
    # shellcheck disable=SC2034
    local -r aro_mise_service_conf_file="FPCLIENTID='$FPCLIENTID'
FPTENANTID='$FPTENANTID'
MISEIMAGE='$image'
MISEVALIDAUDIENCES='$MISEVALIDAUDIENCES'
MISEVALIDAPPIDS='$MISEVALIDAPPIDS'
LOGININSTANCE='$LOGININSTANCE'
PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'"

    write_file aro_mise_service_conf_filename aro_mise_service_conf_file true

    mkdir -p /app/mise
    # shellcheck disable=SC2034
    local -r aro_mise_appsettings_filename='/app/mise/appsettings.json'
    # shellcheck disable=SC2034
    local -r aro_mise_appsettings_file="{
    \"Version\": \"1\",
    \"HeartbeatIntervalMs\": 5000,
    \"AzureAd\": {
        \"Instance\": \"$LOGININSTANCE\",
        \"ClientId\": \"$FPCLIENTID\",
        \"TenantId\": \"$FPTENANTID\",
        \"Audience\": \"api://$FPCLIENTID\",
        \"ShowPII\": false,
        \"InboundPolicies\": [
            {
                \"Label\": \"arorp-arm-inbound-policy\",
                \"Authority\": \"$LOGININSTANCE/$FPTENANTID/\"
,
                \"AuthenticationSchemes\": [
                    \"PoP\"
                ],
                \"ValidAudiences\": $MISEVALIDAUDIENCES,
                \"SignedHttpRequestValidationPolicy\": {
                    \"ValidateTs\": true,
                    \"ValidateM\": true,
                    \"ValidateU\": true,
                    \"ValidateP\": true
                },
                \"ValidApplicationIds\": $MISEVALIDAPPIDS
            }
        ],
        \"Logging\": {
            \"LogLevel\": \"Information\"
        },
        \"Modules\": {
            \"TrV2\": {
                \"ModuleType\": \"TrV2Module\",
                \"Enabled\": true
            }
        }
    },
    \"AllowedHosts\": \"*\",
    \"Kestrel\": {
        \"Endpoints\": {
            \"Http\": {
                \"Url\": \"http://$ipaddress:5000\"
            }
        }
    },
    \"Logging\": {
        \"LogLevel\": {
            \"Default\": \"Information\",
            \"Microsoft\": \"Information\",
            \"Microsoft.Hosting.Lifetime\": \"Information\"
        }
    }
}"

    write_file aro_mise_appsettings_filename aro_mise_appsettings_file true

    # shellcheck disable=SC2034
    local -r aro_mise_service_filename='/etc/systemd/system/aro-mise.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r aro_mise_service_file='[Unit]
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0
[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/aro-mise
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  -p 5000:5000 \
  -v /app/mise/appsettings.json:/app/appsettings.json:z \
  --hostname %H \
  --name %N \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  --rm \
  $MISEIMAGE
ExecStop=/usr/bin/podman stop %N
Restart=always
RestartSec=3
StartLimitInterval=0
[Install]
WantedBy=multi-user.target'

    write_file aro_mise_service_filename aro_mise_service_file true
}
# configure_service_aro_otel_collector
# args:
# 1) image - nameref, string; OTEL container image
# 2) static_ip_address - nameref, array; static ips of all services
# 3) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_aro_otel_collector() {
    local -n image="$1"
    local -n static_ip_address="$2"
    local -n ipaddress="$3"
    log "starting"
    log "Configuring aro-otel-collector service"

    # shellcheck disable=SC2034
    local -r aro_otel_collector_service_conf_filename='/etc/sysconfig/aro-otel-collector'
    # shellcheck disable=SC2034
    local -r aro_otel_collector_service_conf_file="GOMEMLIMIT=1000MiB
OTELIMAGE='$image'
PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'"

    write_file aro_otel_collector_service_conf_filename aro_otel_collector_service_conf_file true

    mkdir -p /app/otel
    # shellcheck disable=SC2034
    local -r aro_otel_collector_appconfig_filename='/app/otel/config.yaml'
    # shellcheck disable=SC2034
    local -r aro_otel_collector_appconfig_file="receivers:
  httpcheck:
    targets:
    # MISE Endpoints
      - endpoint: http://${static_ip_address["mise"]}:5000/healthz
        method: GET
      - endpoint: http://${static_ip_address["mise"]}:5000/readyz
        method: GET
    # OTELs own Endpoints
      - endpoint: http://$ipaddress:13133/healthz
        method: GET
      - endpoint: http://$ipaddress:13133/readyz
        method: GET
    collection_interval: 20s
processors:
  batch:
  attributes/insert:
    actions:
      - key: \"location\"
        action: insert
        value: \"$LOCATION\"
      - key: \"host\"
        action: insert
        value: \"$(hostname)\"
extensions:
  health_check:
    endpoint: $ipaddress:13133
exporters:
  otlp:
    endpoint: ${static_ip_address["mdm"]}:4317
    tls:
      insecure: true
service:
  extensions: [health_check]
  pipelines:
    metrics:
      receivers: [httpcheck]
      processors: [batch, attributes/insert]
      exporters: [otlp]"

    write_file aro_otel_collector_appconfig_filename aro_otel_collector_appconfig_file true

    # shellcheck disable=SC2034
    local -r aro_otel_collector_service_filename='/etc/systemd/system/aro-otel-collector.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r aro_otel_collector_service_file='[Unit]
After=mdm.service
Wants=mdm.service
StartLimitIntervalSec=0
[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/aro-otel-collector
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --hostname %H \
  --name %N \
  --rm \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -m 2g \
  -v /app/otel/config.yaml:/etc/otelcol-contrib/config.yaml:z \
  $OTELIMAGE
ExecStop=/usr/bin/podman stop %N
Restart=always
RestartSec=3
StartLimitInterval=0
[Install]
WantedBy=multi-user.target'

    write_file aro_otel_collector_service_filename aro_otel_collector_service_file true
}

# configure_service_mdsd
# args:
# 1) monitoring_role - nameref, string; can be "gateway" or "rp"
# 2) monitor_config_version - nameref, string; mdsd config version
configure_service_mdsd() {
    local -n role="$1"
    local -n monitor_config_version="$2"
    log "starting"
    log "configuring mdsd service"

    verify_role role

    local -r mdsd_service_dir="/etc/systemd/system/mdsd.service.d"
    mkdir -p "$mdsd_service_dir"

    # shellcheck disable=SC2034
    local -r mdsd_override_conf_filename="$mdsd_service_dir/override.conf"
    local -r mdsd_certificate_san="$(openssl x509 -in /var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem -noout -subject | sed -e 's/.*CN = //')"
    # shellcheck disable=SC2034
    local -r mdsd_override_conf_file="[Unit]
After=network-online.target"

    write_file mdsd_override_conf_filename mdsd_override_conf_file true

    # shellcheck disable=SC2034
    local -r default_mdsd_filename="/etc/default/mdsd"
    # shellcheck disable=SC2034
    local -r default_mdsd_file="MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS=\"-A -d -r \$MDSD_ROLE_PREFIX\"

export MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT='$RPMDSDACCOUNT'
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault
export MONITORING_GCS_AUTH_ID='$mdsd_certificate_san'
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$monitor_config_version'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE='$role'
export MONITORING_ROLE_INSTANCE=\"$(hostname)\"

export MDSD_MSGPACK_SORT_COLUMNS=\"1\""

    write_file default_mdsd_filename default_mdsd_file true
}

# configure_service_fluentbit
# args:
# 1) conf_file - string; fluenbit configuration file
# 2) image - string; fluentbit container image to run
# 3) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_fluentbit() {
    # shellcheck disable=SC2034
    local -n conf_file="$1"
    local -n image="$2"
    log "starting"
    log "Configuring fluentbit service"

    mkdir -p /etc/fluentbit/
    mkdir -p /var/lib/fluent

    # shellcheck disable=SC2034
    local -r conf_filename='/etc/fluentbit/fluentbit.conf'
    write_file conf_filename conf_file true

    # shellcheck disable=SC2034
    local -r sysconfig_filename='/etc/sysconfig/fluentbit'
    # shellcheck disable=SC2034
    local -r sysconfig_file="FLUENTBITIMAGE=$image"

    write_file sysconfig_filename sysconfig_file true

    # shellcheck disable=SC2034
    local -r service_filename='/etc/systemd/system/fluentbit.service'
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r service_file='[Unit]
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/fluentbit
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --security-opt label=disable \
  --entrypoint /opt/td-agent-bit/bin/td-agent-bit \
  --net=host \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -v /etc/fluentbit/fluentbit.conf:/etc/fluentbit/fluentbit.conf \
  -v /var/lib/fluent:/var/lib/fluent:z \
  -v /var/log/journal:/var/log/journal:ro \
  -v /etc/machine-id:/etc/machine-id:ro \
  $FLUENTBITIMAGE \
  -c /etc/fluentbit/fluentbit.conf

ExecStop=/usr/bin/podman stop %N
Restart=always
RestartSec=5
StartLimitInterval=0

[Install]
WantedBy=multi-user.target'

    write_file service_filename service_file true
}

# configure_timers_mdm_mdsd
# args:
# 1) role - string; can be "gateway" or "rp"
configure_timers_mdm_mdsd() {
    local -n role="$1"
    log "starting"

    verify_role role

    local keyvault_suffix secret_prefix
    get_keyvault_suffix role keyvault_suffix secret_prefix

    for var in "mdsd" "mdm"; do
        # shellcheck disable=SC2034
        local download_creds_service_filename="/etc/systemd/system/download-$var-credentials.service"
        # shellcheck disable=SC2034
        local download_creds_service_file="[Unit]
Description=Periodic $var credentials refresh

[Service]
Type=oneshot
ExecStart=/usr/local/bin/download-credentials.sh $var"

        write_file download_creds_service_filename download_creds_service_file true

        # shellcheck disable=SC2034
        local download_creds_timer_filename="/etc/systemd/system/download-$var-credentials.timer"
        # shellcheck disable=SC2034
        local download_creds_timer_file="[Unit]
Description=Periodic $var credentials refresh
After=network-online.target
Wants=network-online.target

[Timer]
OnBootSec=0min
OnCalendar=0/12:00:00
AccuracySec=5s

[Install]
WantedBy=timers.target"

        write_file download_creds_timer_filename download_creds_timer_file true
    done

    local -r download_creds_script_filename="/usr/local/bin/download-credentials.sh"
    # shellcheck disable=SC2034
    local -r download_creds_script_file="#!/bin/bash
set -eu

COMPONENT=\$1
echo \"Download \$COMPONENT credentials\"

TEMP_DIR=\"\$(mktemp -d)\"
export AZURE_CONFIG_DIR=\"\$(mktemp -d)\"

echo \"Logging into Azure...\"
RETRIES=3
while [[ \$RETRIES -gt 0 ]]; do
    if az login -i --allow-no-subscriptions
    then
        echo \"az login successful\"
        break
    else
        echo \"az login failed. Retrying...\"
        let RETRIES-=1
        sleep 5
    fi
done

trap \"cleanup\" EXIT

cleanup() {
  az logout
  [[ \$TEMP_DIR =~ /tmp/.+ ]] && rm -rf \$TEMP_DIR
  [[ \$AZURE_CONFIG_DIR =~ /tmp/.+ ]] && rm -rf \$AZURE_CONFIG_DIR
}

if [[ \$COMPONENT = \"mdm\" ]]; then
  CURRENT_CERT_FILE=\"/etc/mdm.pem\"
elif [[ \$COMPONENT = \"mdsd\" ]]; then
  CURRENT_CERT_FILE=\"/var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem\"
else
  echo Invalid usage && exit 1
fi

SECRET_NAME=\"$secret_prefix-\${COMPONENT}\"
NEW_CERT_FILE=\"\$TEMP_DIR/\$COMPONENT.pem\"
for attempt in {1..5}; do
  az keyvault \
    secret \
    download \
    --file \"\$NEW_CERT_FILE\" \
    --id \"https://$KEYVAULTPREFIX-$keyvault_suffix.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME\" \
    && break
  if [[ \$attempt -lt 5 ]]; then sleep 10; else exit 1; fi
done

if [ -f \$NEW_CERT_FILE ]; then
  if [[ \$COMPONENT = \"mdsd\" ]]; then
    chown syslog:syslog \$NEW_CERT_FILE
  else
    sed -i -ne '1,/END CERTIFICATE/ p' \$NEW_CERT_FILE
  fi

  new_cert_sn=\"\$(openssl x509 -in \"\$NEW_CERT_FILE\" -noout -serial | awk -F= '{print \$2}')\"
  current_cert_sn=\"\$(openssl x509 -in \"\$CURRENT_CERT_FILE\" -noout -serial | awk -F= '{print \$2}')\"
  if [[ ! -z \$new_cert_sn ]] && [[ \$new_cert_sn != \"\$current_cert_sn\" ]]; then
    echo updating certificate for \$COMPONENT
    chmod 0600 \$NEW_CERT_FILE
    mv \$NEW_CERT_FILE \$CURRENT_CERT_FILE
  fi
else
  echo Failed to refresh certificate for \$COMPONENT && exit 1
fi"

    write_file download_creds_script_filename download_creds_script_file true

    chmod u+x /usr/local/bin/download-credentials.sh

    $download_creds_script_filename mdsd &
    wait "$!"


    $download_creds_script_filename mdm &
    wait "$!"

    # shellcheck disable=SC2034
    local -r watch_mdm_creds_service_filename="/etc/systemd/system/watch-mdm-credentials.service"
    # shellcheck disable=SC2034
    local -r watch_mdm_creds_service_file="[Unit]
Description=Watch for changes in mdm.pem and restarts the mdm service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart mdm.service

[Install]
WantedBy=multi-user.target"

    write_file watch_mdm_creds_service_filename watch_mdm_creds_service_file true

    # shellcheck disable=SC2034
    local -r watch_mdm_creds_path_filename='/usr/lib/systemd/system/watch-mdm-credentials.path'
    # shellcheck disable=SC2034
    local -r watch_mdm_creds_path_file='[Path]
PathModified=/etc/mdm.pem

[Install]
WantedBy=multi-user.target'

    write_file watch_mdm_creds_path_filename watch_mdm_creds_path_file true

    local -r watch_mdm_creds='watch-mdm-credentials.path'
    systemctl enable --now "$watch_mdm_creds" || abort "failed to enable and start $watch_mdm_creds"
}

# configure_service_mdm
# args:
# 1) role - nameref, string; can be "gateway" or "rp"
# 2) image - nameref, string; mdm container image to run
# 3) ipaddress - nameref, string; static ip of podman network to be attached
configure_service_mdm() {
    local -n role="$1"
    local -n image="$2"
    local -n ipaddress="$3"
    log "starting"
    log "Configuring mdm service"

    verify_role role

    # shellcheck disable=SC2034
    local -r sysconfig_mdm_filename="/etc/sysconfig/mdm"
    # shellcheck disable=SC2034
    local -r sysconfig_mdm_file="MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$image'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE='$role'
MDMSOURCEROLEINSTANCE=\"$(hostname)\"
MDM_INPUT=statsd_local,otlp_grpc
MDM_NAMESPACE='OTEL'
MDM_ACCOUNT='AzureRedHatOpenShiftRP'
PODMAN_NETWORK='podman'
IPADDRESS='$ipaddress'"

    write_file sysconfig_mdm_filename sysconfig_mdm_file true

    mkdir -p /var/etw
    # shellcheck disable=SC2034
    local -r mdm_service_filename="/etc/systemd/system/mdm.service"
    # shellcheck disable=SC2034
    # below variable is in single quotes 
    # as it is to be expanded at systemd start time (by systemd, not this script)
    local -r mdm_service_file='[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/podman rm -f %N
ExecStart=/usr/bin/podman run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  --network=$PODMAN_NETWORK \
  --ip $IPADDRESS \
  -m 2g \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $MDMIMAGE \
  -Input $MDM_INPUT \
  -MetricNamespace $MDM_NAMESPACE \
  -MonitoringAccount $MDM_ACCOUNT \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/podman stop %N
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target'

    write_file mdm_service_filename mdm_service_file true
}

# configure_vmss_aro_service
# args:
# 1) r - nameref, string; role of VMSS
# 2) images - nameref, associative array; ARO container images
# 3) configs - nameref, associative array; configuration files and versions. The values should be a reference to variables, not dereferenced.
#                                          This is because the value is used when creating nameref variables by helper functions.
configure_vmss_aro_services() {
    local -n r="$1"
    local -n images="$2"
    local -n configs="$3"
    log "starting"
    verify_role "$1"

    if [ "$r" == "$role_gateway" ]; then
        configure_service_aro_gateway "${images["rp"]}" "$1" "${configs["gateway_config"]}" "${configs["static_ip_address"]}["gateway"]"
        configure_certs_gateway
    elif [ "$r" == "$role_rp" ]; then
        configure_service_aro_rp "${images["rp"]}" "$1" "${configs["rp_config"]}" "${configs["static_ip_address"]}["rp"]"
        configure_service_aro_monitor "${images["rp"]}" "${configs["static_ip_address"]}["monitor"]"
        configure_service_aro_portal "${images["rp"]}" "${configs["static_ip_address"]}["portal"]"
        configure_service_aro_mise "${images["mise"]}" "${configs["static_ip_address"]}["mise"]"
        configure_service_aro_otel_collector "${images["otel"]}" "${configs["static_ip_address"]}" "${configs["static_ip_address"]}["otel_collector"]"
        configure_certs_rp
    fi

    configure_service_fluentbit "${configs["fluentbit"]}" "${images["fluentbit"]}"
    configure_timers_mdm_mdsd "$1"
    configure_service_mdm "$1" "${images["mdm"]}" "${configs["static_ip_address"]}["mdm"]"
    configure_service_mdsd "$1" "${configs["mdsd"]}"
    run_azsecd_config_scan
}

# util-common.sh does not exist when deployed to VMSS via VMSS extensions
# Provides shellcheck definitions
util_common="util-common.sh"
if [ -f "$util_common" ]; then
    # shellcheck source=util-common.sh
    source "$util_common"
fi

# util-system.sh does not exist when deployed to VMSS via VMSS extensions
# Provides shellcheck definitions
util_system="util-system.sh"
if [ -f "$util_system" ]; then
    # shellcheck source=util-system.sh
    source "$util_system"
fi
