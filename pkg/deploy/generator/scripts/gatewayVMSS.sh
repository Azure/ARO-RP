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
  yum -y install clamav azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security podman-docker openssl-perl python3 && break
  # hack - we are installing python3 on hosts due to an issue with Azure Linux Extensions https://github.com/Azure/azure-linux-extensions/pull/1505
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

echo "applying firewall rules"
# https://access.redhat.com/security/cve/cve-2020-13401
cat >/etc/sysctl.d/02-disable-accept-ra.conf <<'EOF'
net.ipv6.conf.all.accept_ra=0
EOF

cat >/etc/sysctl.d/01-disable-core.conf <<'EOF'
kernel.core_pattern = |/bin/true
EOF
sysctl --system

firewall-cmd --add-port=80/tcp --permanent
firewall-cmd --add-port=443/tcp --permanent

echo "logging into prod acr"
export AZURE_CLOUD_NAME=$AZURECLOUDNAME
az login -i --allow-no-subscriptions

# The managed identity that the VM runs as only has a single roleassignment.
# This role assignment is ACRPull which is not necessarily present in the
# subscription we're deploying into.  If the identity does not have any
# role assignments scoped on the subscription we're deploying into, it will
# not show on az login -i, which is why the below line is commented.
# az account set -s "$SUBSCRIPTIONID"

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

echo "configuring mdm service"
cat >/etc/sysconfig/mdm <<EOF
MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=gateway
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

echo "configuring aro-gateway service"
cat >/etc/sysconfig/aro-gateway <<EOF
ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT="$RPMDMACCOUNT"
MDM_NAMESPACE=Gateway
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-gateway.service <<'EOF'
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-gateway
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e DBTOKEN_URL \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_FEATURES \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 80:8080 \
  -p 443:8443 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  gateway
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store

echo "configuring mdsd and mdm services"
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

SECRET_NAME="gwy-\${COMPONENT}"
NEW_CERT_FILE="\$TEMP_DIR/\$COMPONENT.pem"
for attempt in {1..5}; do
  az keyvault secret download --file \$NEW_CERT_FILE --id "https://$KEYVAULTPREFIX-gwy.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME" && break
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
export MONITORING_CONFIG_VERSION='$GATEWAYMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=gateway
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
for service in aro-gateway auoms azsecd azsecmond mdsd mdm chronyd fluentbit; do
  systemctl enable $service.service
done

for scan in baseline clamav software; do
  /usr/local/bin/azsecd config -s $scan -d P1D
done

echo "rebooting"
restorecon -RF /var/log/*
(sleep 30; reboot) &
