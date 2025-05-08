


#/bin/bash

failedChecks=0
while read var; do
  [ -z "${!var}" ] && { echo "required $var is empty or not set."; let failedChecks=failedChecks+1; }
done << EOF
LOCATION
RESOURCEGROUP
USER
EOF

if [ $failedChecks -gt 0 ]; then
  exit 1
fi

BASE=$( git rev-parse --show-toplevel)

SOCKETPATH="$BASE/cmd/aro"

HOSTNAME=$( hostname )
NAME="mdm"
MDMIMAGE=linuxgeneva-microsoft.azurecr.io/genevamdm:master_20211120.1
MDMFRONTENDURL=https://global.ppe.microsoftmetrics.com/
MDMSOURCEENVIRONMENT=$LOCATION
MDMSOURCEROLE=rp
MDMSOURCEROLEINSTANCE=$HOSTNAME


echo "Using:"

echo "Resourcegroup = $RESOURCEGROUP"
echo "User          = $USER"
echo "HOSTNAME      = $HOSTNAME"
echo "Containername = $NAME"
echo "Location      = $LOCATION"
echo "MDM image     = $MDMIMAGE"
echo "  (version hardcoded. Check against pkg/util/version/const.go if things don't work)"
echo "Geneva API URL= $MDMFRONTENDURL"
echo "MDMSOURCEENV  = $MDMSOURCEENVIRONMENT"
echo "MDMSOURCEROLE  = $MDMSOURCEROLE"
echo "MDMSOURCEROLEINSTANCE  = $MDMSOURCEROLEINSTANCE"


podman run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname $HOSTNAME \
  --name $NAME \
  -d \
  --restart=always \
  -m 2g \
  -v $BASE/secrets/rp-metrics-int.pem:/etc/mdm.pem \
  -v $SOCKETPATH:/var/etw:z \
  $MDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Debug \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
