#!/bin/bash

set -o errexit \
    -o nounset \

# trap 'catch' ERR

main() {
    configure_sshd
    configure_rhui_repo
    dnf_update_pkgs
    configure_disk_partitions
    configure_logrotate
    create_azure_rpm_repos
    configure_selinux
    mkdir -p /var/log/journal
    mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store
    configure_firewalld_rules
    pull_container_images
}

# We need to configure PasswordAuthentication to yes in order for the VMSS Access JIT to work
configure_sshd() {
    log "setting ssh password authentication"
    sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config

    systemctl reload sshd.service
    systemctl is-active --quiet sshd || abort "sshd failed to reload"
}

configure_rhui_repo() {
    log "running RHUI package updates"
    #Adding retry logic to yum commands in order to avoid stalling out on resource locks
    for attempt in {1..5}; do
        dnf update \
            -y \
            --disablerepo='*' \
            --enablerepo='rhui-microsoft-azure*' \
            && break
        if [[ ${attempt} -lt 5 ]]; then
            sleep 10
        else
            abort "failed to run dnf update"
        fi
    done
}

dnf_update_pkgs() {
    log "running dnf update"
    for attempt in {1..5}; do
        dnf -y \
            -x WALinuxAgent \
            -x WALinuxAgent-udev \
            update --allowerasing \
            && break
        if [[ ${attempt} -lt 5 ]]; then
            sleep 10
        else
            return 1
        fi
    done
}

dnf_install_pkgs() {
    log "importing rpm repositories"
    rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
    rpm --import https://packages.microsoft.com/keys/microsoft.asc

    for attempt in {1..5}; do
        yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm && break
        if [[ ${attempt} -lt 5 ]]; then
            sleep 10
        else
            return 1
        fi
    done

    for attempt in {1..5}; do
        yum -y \
            install \
            clamav \
            azsec-clamav \
            azsec-monitor \
            azure-cli \
            azure-mdsd \
            azure-security \
            podman \
            podman-docker \
            openssl-perl \
            python3 \
            && break
        # hack - we are installing python3 on hosts due to an issue with Azure Linux Extensions https://github.com/Azure/azure-linux-extensions/pull/1505
        if [[ ${attempt} -lt 5 ]]; then
            sleep 10
        else
            abort "failed to install required packages"
        fi
    done
}

configure_disk_partitions() {
    log "extending partition table"
    # Linux block devices are inconsistently named
    # it's difficult to tie the lvm pv to the physical disk using /dev/disk files, which is why lvs is used here
    physicalDisk="$(lvs -o devices -a | head -n2 | tail -n1 | cut -d ' ' -f 3 | cut -d \( -f 1 | tr -d '[:digit:]')"
    growpart "$physicalDisk" 2

    log "extending filesystems"
    log "extending root lvm"
    lvextend -l +20%FREE /dev/rootvg/rootlv
    log "growing root filesystem"
    xfs_growfs /

    log "extending var lvm"
    lvextend -l +100%FREE /dev/rootvg/varlv
    log "growing var filesystem"
    xfs_growfs /var
}

# configure_logrotate clobbers /etc/logrotate.conf
configure_logrotate() {
    log "configuring logrotate"
cat >/etc/logrotate.conf <<'EOF'
# see "man logrotate" for details
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

# no packages own wtmp and btmp -- we'll rotate them here
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
}
EOF
}

# create_azure_rpm_repos creates /etc/yum.repos.d/azure.repo repository file
create_azure_rpm_repos() {
    log "configuring yum repository and running yum update"
cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes

[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
EOF
}

configure_selinux() {
    semanage fcontext -a -t var_log_t "/var/log/journal(/.*)?"
    chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent
}

configure_firewalld_rules() {
    # https://access.redhat.com/security/cve/cve-2020-13401
    log "applying firewall rules"
cat >/etc/sysctl.d/02-disable-accept-ra.conf <<'EOF'
net.ipv6.conf.all.accept_ra=0
EOF

cat >/etc/sysctl.d/01-disable-core.conf <<'EOF'
kernel.core_pattern = |/bin/true
EOF
    sysctl --system

    log "adding firewalld ports to default zone"
    firewall-cmd \
        --add-port=443/tcp \
        --permanent
    firewall-cmd \
        --add-port=444/tcp \
        --permanent
    firewall-cmd \
        --add-port=445/tcp \
        --permanent
    firewall-cmd \
        --add-port=2222/tcp \
        --permanent
    
    log "reloading firewalld"
    firewall-cmd reload
}

export AZURE_CLOUD_NAME="${AZURECLOUDNAME:?"Failed to carry over variables"}"

pull_container_images() {
    echo "logging into prod acr"
    az login -i --allow-no-subscriptions

    # Suppress emulation output for podman instead of docker for az acr compatability
    mkdir -p /etc/containers/
    touch /etc/containers/nodocker

    mkdir -p /root/.docker
    REGISTRY_AUTH_FILE=/root/.docker/config.json az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

    MDMIMAGE="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
    docker pull "$MDMIMAGE"
    docker pull "$RPIMAGE"
    docker pull "$FLUENTBITIMAGE"

    az logout
}

# configure_system_services creates, configures, and enables the following systemd services and timers
# services
#   fluentbit
#   mdm
#   mdsd
#   arp-rp
#   aro-dbtoken
#   aro-monitor
#   aro-portal
configure_system_services() {
    configure_service_fluentbit
    configure_service_mdm
    configure_timers_mdm_mdsd
    configure_service_aro_rp
    configure_service_aro_dbtoken
    configure_service_aro_monitor
    configure_service_aro_portal
    configure_service_mdsd
}

# enable_aro_services enables all services required for aro rp
enable_aro_services() {
    local -a aro_services=(
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
    )
    log "enabling aro services ${aro_services[*]}"
    # shellcheck disable=SC2068
    for service in ${aro_services[@]}; do
        log "Enabling $service now"
        systemctl enable "$service.service"
    done
}

# configure_service_fluentbit
configure_service_fluentbit() {
    log "configuring fluentbit service"
    mkdir -p /etc/fluentbit/
    mkdir -p /var/lib/fluent

cat >/etc/fluentbit/fluentbit.conf <<'EOF'
[INPUT]
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
	Rule $LOGKIND asyncqos asyncqos true

[FILTER]
	Name modify
	Match asyncqos
	Remove CLIENT_PRINCIPAL_NAME
	Remove FILE
	Remove COMPONENT

[FILTER]
	Name rewrite_tag
	Match journald
	Rule $LOGKIND ifxaudit ifxaudit false

[OUTPUT]
	Name forward
	Match *
	Port 29230
EOF

    log "FLUENTBITIMAGE=$FLUENTBITIMAGE" >/etc/sysconfig/fluentbit

cat >/etc/systemd/system/fluentbit.service <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/fluentbit
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
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

ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=5
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF
}

# configure_certs
configure_certs() {
    mkdir /etc/aro-rp
    base64 -d <<<"$ADMINAPICABUNDLE" >/etc/aro-rp/admin-ca-bundle.pem
    if [[ -n "$ARMAPICABUNDLE" ]]; then
    base64 -d <<<"$ARMAPICABUNDLE" >/etc/aro-rp/arm-ca-bundle.pem
    fi
    chown -R 1000:1000 /etc/aro-rp

    # setting MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault seems to have caused mdsd not
    # to honour SSL_CERT_FILE any more, heaven only knows why.
    mkdir -p /usr/lib/ssl/certs
    csplit -f /usr/lib/ssl/certs/cert- -b %03d.pem /etc/pki/tls/certs/ca-bundle.crt /^$/1 {*} >/dev/null
    c_rehash /usr/lib/ssl/certs

# we leave clientId blank as long as only 1 managed identity assigned to vmss
# if we have more than 1, we will need to populate with clientId used for off-node scanning
cat >/etc/default/vsa-nodescan-agent.config <<EOF
{
    "Nice": 19,
    "Timeout": 10800,
    "ClientId": "",
    "TenantId": "$AZURESECPACKVSATENANTID",
    "QualysStoreBaseUrl": "$AZURESECPACKQUALYSURL",
    "ProcessTimeout": 300,
    "CommandDelay": 0
  }
EOF
}

# configure_service_mdm
configure_service_mdm() {
    log "configuring mdm service"

cat >/etc/sysconfig/mdm <<EOF
MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=rp
MDMSOURCEROLEINSTANCE='$(hostname)'
EOF

    mkdir /var/etw
cat >/etc/systemd/system/mdm.service <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -m 2g \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $MDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF
}

# configure_timers_mdm_mdsd
configure_timers_mdm_mdsd() {
    for var in "mdsd" "mdm"; do
cat >/etc/systemd/system/download-$var-credentials.service <<EOF
[Unit]
Description=Periodic $var credentials refresh

[Service]
Type=oneshot
ExecStart=/usr/local/bin/download-credentials.sh $var
EOF

cat >/etc/systemd/system/download-$var-credentials.timer <<EOF
[Unit]
Description=Periodic $var credentials refresh
After=network-online.target
Wants=network-online.target

[Timer]
OnBootSec=0min
OnCalendar=0/12:00:00
AccuracySec=5s

[Install]
WantedBy=timers.target
EOF
    done

cat >/usr/local/bin/download-credentials.sh <<EOF
#!/bin/bash
set -eu

COMPONENT="\$1"
echo "Download \$COMPONENT credentials"

TEMP_DIR=\$(mktemp -d)
export AZURE_CONFIG_DIR=\$(mktemp -d)

echo "Logging into Azure..."
RETRIES=3
while [ "\$RETRIES" -gt 0 ]; do
    if az login -i --allow-no-subscriptions
    then
        echo "az login successful"
        break
    else
        echo "az login failed. Retrying..."
        let RETRIES-=1
        sleep 5
    fi
done

trap "cleanup" EXIT

cleanup() {
  az logout
  [[ "\$TEMP_DIR" =~ /tmp/.+ ]] && rm -rf \$TEMP_DIR
  [[ "\$AZURE_CONFIG_DIR" =~ /tmp/.+ ]] && rm -rf \$AZURE_CONFIG_DIR
}

if [ "\$COMPONENT" = "mdm" ]; then
  CURRENT_CERT_FILE="/etc/mdm.pem"
elif [ "\$COMPONENT" = "mdsd" ]; then
  CURRENT_CERT_FILE="/var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem"
else
  echo Invalid usage && exit 1
fi

SECRET_NAME="rp-\${COMPONENT}"
NEW_CERT_FILE="\$TEMP_DIR/\$COMPONENT.pem"
for attempt in {1..5}; do
  az keyvault secret download --file \$NEW_CERT_FILE --id "https://$KEYVAULTPREFIX-svc.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME" && break
  if [[ \$attempt -lt 5 ]]; then sleep 10; else exit 1; fi
done

if [ -f \$NEW_CERT_FILE ]; then
  if [ "\$COMPONENT" = "mdsd" ]; then
    chown syslog:syslog \$NEW_CERT_FILE
  else
    sed -i -ne '1,/END CERTIFICATE/ p' \$NEW_CERT_FILE
  fi

  new_cert_sn="\$(openssl x509 -in "\$NEW_CERT_FILE" -noout -serial | awk -F= '{print \$2}')"
  current_cert_sn="\$(openssl x509 -in "\$CURRENT_CERT_FILE" -noout -serial | awk -F= '{print \$2}')"
  if [[ ! -z \$new_cert_sn ]] && [[ \$new_cert_sn != "\$current_cert_sn" ]]; then
    echo updating certificate for \$COMPONENT
    chmod 0600 \$NEW_CERT_FILE
    mv \$NEW_CERT_FILE \$CURRENT_CERT_FILE
  fi
else
  echo Failed to refresh certificate for \$COMPONENT && exit 1
fi
EOF

    chmod u+x /usr/local/bin/download-credentials.sh

    systemctl enable download-mdsd-credentials.timer
    systemctl enable download-mdm-credentials.timer

    /usr/local/bin/download-credentials.sh mdsd
    /usr/local/bin/download-credentials.sh mdm
    MDSDCERTIFICATESAN=$(openssl x509 -in /var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem -noout -subject | sed -e 's/.*CN = //')

cat >/etc/systemd/system/watch-mdm-credentials.service <<EOF
[Unit]
Description=Watch for changes in mdm.pem and restarts the mdm service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart mdm.service

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/watch-mdm-credentials.path <<EOF
[Path]
PathModified=/etc/mdm.pem

[Install]
WantedBy=multi-user.target
EOF

    local watch_mdm_creds="watch-mdm-credentials.path"
    systemctl enable "$watch_mdm_creds" || abort "failed to enable $watch_mdm_creds"

    systemctl start "$watch_mdm_creds" || abort "failed to start $watch_mdm_creds"
}

# configure_service_aro_rp
configure_service_aro_rp() {
    local arp_rp_config_file="/etc/sysconfig/aro-rp"
    log "Writing $arp_rp_config_file"
cat >"$arp_rp_config_file"<<EOF
ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
ARM_API_CLIENT_CERT_COMMON_NAME='$ARMAPICLIENTCERTCOMMONNAME'
AZURE_ARM_CLIENT_ID='$ARMCLIENTID'
AZURE_FP_CLIENT_ID='$FPCLIENTID'
AZURE_FP_SERVICE_PRINCIPAL_ID='$FPSERVICEPRINCIPALID'
BILLING_E2E_STORAGE_ACCOUNT_ID='$BILLINGE2ESTORAGEACCOUNTID'
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
MDM_NAMESPACE=RP
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
RP_FEATURES='$RPFEATURES'
RPIMAGE='$RPIMAGE'
ARO_INSTALL_VIA_HIVE='$CLUSTERSINSTALLVIAHIVE'
ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC='$CLUSTERDEFAULTINSTALLERPULLSPEC'
ARO_ADOPT_BY_HIVE='$CLUSTERSADOPTBYHIVE'
USE_CHECKACCESS='$USECHECKACCESS'
EOF

    local aro_rp_service_file="/etc/systemd/system/aro-rp.service"
    log "Writing $aro_rp_service_file"
cat >"$aro_rp_service_file" <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e ADMIN_API_CLIENT_CERT_COMMON_NAME \
  -e ARM_API_CLIENT_CERT_COMMON_NAME \
  -e AZURE_ARM_CLIENT_ID \
  -e AZURE_FP_CLIENT_ID \
  -e BILLING_E2E_STORAGE_ACCOUNT_ID \
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
  -e USE_CHECKACCESS \
  -m 2g \
  -p 443:8443 \
  -v /etc/aro-rp:/etc/aro-rp \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  rp
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF
}

# configure_service_aro_dbtoken
configure_service_aro_dbtoken() {
    local aro_dbtoken_service_config_file="/etc/sysconfig/aro-dbtoken"
    log "Writing $aro_dbtoken_service_file"
cat >"$aro_dbtoken_service_file" <<EOF
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=DBToken
RPIMAGE='$RPIMAGE'
EOF

    local aro_dbtoken_service_file="/etc/systemd/system/aro-dbtoken.service"
    log "Writing $aro_dbtoken_service_file"
cat >"$aro_dbtoken_service_config_file" <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-dbtoken
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e AZURE_GATEWAY_SERVICE_PRINCIPAL_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 445:8445 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  dbtoken
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF
}

# configure_service_aro_monitor
configure_service_aro_monitor() {
    local aro_monitor_service_config="/etc/sysconfig/aro-monitor"
    log "configuring aro-monitor service"
# DOMAIN_NAME, CLUSTER_MDSD_ACCOUNT, CLUSTER_MDSD_CONFIG_VERSION, GATEWAY_DOMAINS, GATEWAY_RESOURCEGROUP, MDSD_ENVIRONMENT CLUSTER_MDSD_NAMESPACE
# are not used, but can't easily be refactored out. Should be revisited in the future.
cat >"$aro_monitor_service_config" <<EOF
AZURE_FP_CLIENT_ID='$FPCLIENTID'
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
RPIMAGE='$RPIMAGE'
EOF

    local aro_monitor_service_file="/etc/systemd/system/aro-monitor.service"
    log "Writing $aro_monitor_service_file"
cat >"$aro_monitor_service_file" <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
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
WantedBy=multi-user.target
EOF
}

# configure_service_aro_portal
configure_service_aro_portal() {
    local aro_portal_service_config="/etc/sysconfig/aro-portal"
    log "Writing $aro_portal_service_config"
cat >"$aro_portal_service_config" <<EOF
AZURE_PORTAL_ACCESS_GROUP_IDS='$PORTALACCESSGROUPIDS'
AZURE_PORTAL_CLIENT_ID='$PORTALCLIENTID'
AZURE_PORTAL_ELEVATED_GROUP_IDS='$PORTALELEVATEDGROUPIDS'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Portal
PORTAL_HOSTNAME='$LOCATION.admin.$RPPARENTDOMAINNAME'
RPIMAGE='$RPIMAGE'
EOF

    local aro_portal_service_file="/etc/systemd/system/aro-portal.service"
    log "Writing $aro_portal_service_config"
cat >"$aro_portal_service_file" <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-portal
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e AZURE_PORTAL_ACCESS_GROUP_IDS \
  -e AZURE_PORTAL_CLIENT_ID \
  -e AZURE_PORTAL_ELEVATED_GROUP_IDS \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e PORTAL_HOSTNAME \
  -m 2g \
  -p 444:8444 \
  -p 2222:2222 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  portal
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF
}

# configure_service_mdsd
configure_service_mdsd() {
    local mdsd_service_dir="/etc/systemd/system/mdsd.service.d"
    log "Creating $mdsd_service_dir"
    mkdir "$mdsd_service_dir"
    local mdsd_override_conf="$mdsd_service_dir/override.conf"
    log "Writing $mdsd_override_conf"
cat >"$mdsd_override_conf" <<'EOF'
[Unit]
After=network-online.target
EOF

cat >/etc/default/mdsd <<EOF
MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS="-A -d -r \$MDSD_ROLE_PREFIX"

export MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT='$RPMDSDACCOUNT'
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault
export MONITORING_GCS_AUTH_ID='$MDSDCERTIFICATESAN'
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$RPMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=rp
export MONITORING_ROLE_INSTANCE='$(hostname)'

export MDSD_MSGPACK_SORT_COLUMNS=1
EOF
}

# run_azsecd_config_scan
run_azsecd_config_scan() {
    local -a configs=(
        "baseline"
        "clamav"
        "software"
    )

    log "Scanning configuration files ${configs[*]}"
    # shellcheck disable=SC2068
    for scan in ${configs[@]}; do
        log "Scanning config file $scan now"
        /usr/local/bin/azsecd config -s "$scan" -d P1D
    done
}

# reboot_vm restores all selinux file contexts, waits 30 seconds then reboots
reboot_vm() {
    restorecon -RF /var/log/*
    (sleep 30 && log "rebooting vm now"; reboot) &
}

# log is a wrapper for echo that includes the function name
log() {
    local msg="${1:-"log message is empty"}"
    local stack_level="${2:-1}"
    echo "${FUNCNAME[${stack_level}]}: ${msg}"
}

# abort is a wrapper for log that exits with an error code
abort() {
    log "${1}" "2"
    exit 1
}
