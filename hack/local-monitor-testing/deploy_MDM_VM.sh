#!/bin/bash -e
set +x

BASE=$( git rev-parse --show-toplevel)

HOSTNAME=$( hostname )
NAME="mdm"
MDMIMAGE=linuxgeneva-microsoft.azurecr.io/genevamdm:master_20211120.1
MDMFRONTENDURL=https://int2.int.microsoftmetrics.com/
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

VMName="$USER-mdm-link"

CLOUDUSER="cloud-user"



if [ "$(az vm show -g $RESOURCEGROUP --name $VMName)" = "" ];
then
  echo "Creating VM $VMName in RG $RESOURCEGROUP"   
  az vm create -g $RESOURCEGROUP -n $VMName --image RedHat:RHEL:7-LVM:latest --ssh-key-values @~/.ssh/id_rsa.pub --admin-username $CLOUDUSER
else
  echo "VM already exists, skipping..."
fi


PUBLICIP=$( az vm list-ip-addresses --name $VMName -g $RESOURCEGROUP | jq -r '.[0].virtualMachine.network.publicIpAddresses[0].ipAddress' )

echo "Found IP $PUBLICIP"

scp $BASE/secrets/rp-metrics-int.pem $CLOUDUSER@$PUBLICIP:mdm.pem
scp $BASE/hack/local-monitor-testing/configureRemote.sh $CLOUDUSER@$PUBLICIP:

ssh $CLOUDUSER@$PUBLICIP "sudo cp mdm.pem /etc/mdm.pem"
ssh $CLOUDUSER@$PUBLICIP "sudo ./configureRemote.sh"


ssh $CLOUDUSER@$PUBLICIP "sudo docker pull $MDMIMAGE"

cat <<EOF > $BASE/dockerStartCommand.sh
docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname $HOSTNAME \
  --name $NAME \
  -d \
  --restart=always \
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
EOF


#disable SELINUX (don't shoot me)
ssh $CLOUDUSER@$PUBLICIP "sudo setenforce 0"
ssh $CLOUDUSER@$PUBLICIP "sudo getenforce"

#make it permanent
ssh $CLOUDUSER@$PUBLICIP "sudo sed -i 's/SELINUX=enforcing/SELINUX=permissive/g' /etc/selinux/config"


ssh $CLOUDUSER@$PUBLICIP "sudo firewall-cmd --zone=public --add-port=12345/tcp --permanent"
ssh $CLOUDUSER@$PUBLICIP "sudo firewall-cmd --reload"


scp $BASE/dockerStartCommand.sh $CLOUDUSER@$PUBLICIP:
ssh $CLOUDUSER@$PUBLICIP "chmod +x dockerStartCommand.sh"
ssh $CLOUDUSER@$PUBLICIP "sudo ./dockerStartCommand.sh &"
