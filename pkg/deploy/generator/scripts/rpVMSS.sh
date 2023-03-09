echo "setting ssh password authentication"
# We need to manually set PasswordAuthentication to true in order for the VMSS Access JIT to work
sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config
systemctl reload sshd.service

echo "running RHUI fix"
yum update -y --disablerepo='*' --enablerepo='rhui-microsoft-azure*'

echo "running yum update"
yum -y -x WALinuxAgent -x WALinuxAgent-udev update

echo "extending filesystems"
lvextend -l +50%FREE /dev/rootvg/rootlv
xfs_growfs /

lvextend -l +100%FREE /dev/rootvg/varlv
xfs_growfs /var

echo "importing rpm repositories"
rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
rpm --import https://packages.microsoft.com/keys/microsoft.asc

for attempt in {1..5}; do
  yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

echo "configuring logrotate"
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

echo "configuring yum repository and running yum update"
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

semanage fcontext -a -t var_log_t "/var/log/journal(/.*)?"
mkdir -p /var/log/journal

for attempt in {1..5}; do
yum -y install clamav azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security podman podman-docker openssl-perl python3 && break
  # hack - we are installing python3 on hosts due to an issue with Azure Linux Extensions https://github.com/Azure/azure-linux-extensions/pull/1505
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

# https://access.redhat.com/security/cve/cve-2020-13401
echo "applying firewall rules"
cat >/etc/sysctl.d/02-disable-accept-ra.conf <<'EOF'
net.ipv6.conf.all.accept_ra=0
EOF

cat >/etc/sysctl.d/01-disable-core.conf <<'EOF'
kernel.core_pattern = |/bin/true
EOF
sysctl --system

firewall-cmd --add-port=443/tcp --permanent
firewall-cmd --add-port=444/tcp --permanent
firewall-cmd --add-port=445/tcp --permanent
firewall-cmd --add-port=2222/tcp --permanent

export AZURE_CLOUD_NAME=$AZURECLOUDNAME

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

echo "configuring fluentbit service"
mkdir -p /etc/fluentbit/
mkdir -p /var/lib/fluent

cat >/etc/fluentbit/fluentbit.conf <<'EOF'
[INPUT]
	Name systemd
	Tag journald
	Systemd_Filter _COMM=aro

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

echo "FLUENTBITIMAGE=$FLUENTBITIMAGE" >/etc/sysconfig/fluentbit

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

mkdir /etc/aro-rp
base64 -d <<<"$ADMINAPICABUNDLE" >/etc/aro-rp/admin-ca-bundle.pem
if [[ -n "$ARMAPICABUNDLE" ]]; then
  base64 -d <<<"$ARMAPICABUNDLE" >/etc/aro-rp/arm-ca-bundle.pem
fi
chown -R 1000:1000 /etc/aro-rp

echo "configuring mdm service"
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

echo "configuring aro-rp service"
cat >/etc/sysconfig/aro-rp <<EOF
ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
ARM_API_CLIENT_CERT_COMMON_NAME='$ARMAPICLIENTCERTCOMMONNAME'
AZURE_ARM_CLIENT_ID='$ARMCLIENTID'
AZURE_FP_CLIENT_ID='$FPCLIENTID'
AZURE_FP_SERVICE_PRINCIPAL_ID='$FPSERVICEPRINCIPALID'
BILLING_E2E_STORAGE_ACCOUNT_ID='$BILLINGE2ESTORAGEACCOUNTID'
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
EOF

cat >/etc/systemd/system/aro-rp.service <<'EOF'
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

echo "configuring aro-dbtoken service"
cat >/etc/sysconfig/aro-dbtoken <<EOF
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=DBToken
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-dbtoken.service <<'EOF'
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

echo "configuring aro-monitor service"
cat >/etc/sysconfig/aro-monitor <<EOF
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMACCOUNT'
CLUSTER_MDM_NAMESPACE=BBM
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=BBM
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-monitor.service <<'EOF'
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

echo "configuring aro-portal service"
cat >/etc/sysconfig/aro-portal <<EOF
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

cat >/etc/systemd/system/aro-portal.service <<'EOF'
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

echo "configuring mdsd and mdm services"
chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store

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
  if ! diff $NEW_CERT_FILE $CURRENT_CERT_FILE >/dev/null 2>&1; then
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

systemctl enable watch-mdm-credentials.path
systemctl start watch-mdm-credentials.path

mkdir /etc/systemd/system/mdsd.service.d
cat >/etc/systemd/system/mdsd.service.d/override.conf <<'EOF'
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

# we start a cron job to run every hour to ensure the said directory is accessible
# by the correct user as it gets created by root and may cause a race condition
# where root owns the dir instead of syslog
# TODO: https://msazure.visualstudio.com/AzureRedHatOpenShift/_workitems/edit/12591207
cat >/etc/cron.d/mdsd-chown-workaround <<EOF
SHELL=/bin/bash
PATH=/bin
0 * * * * root chown syslog:syslog /var/opt/microsoft/linuxmonagent/eh/EventNotice/arorplogs*
EOF

echo "enabling aro services"
for service in aro-dbtoken aro-monitor aro-portal aro-rp auoms azsecd azsecmond mdsd mdm chronyd fluentbit; do
  systemctl enable $service.service
done

for scan in baseline clamav software; do
  /usr/local/bin/azsecd config -s $scan -d P1D
done

echo "rebooting"
restorecon -RF /var/log/*
(sleep 30; reboot) &
