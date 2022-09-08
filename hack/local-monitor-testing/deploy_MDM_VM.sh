#!/bin/bash -e
set +x

BASE=$( git rev-parse --show-toplevel)
CLOUDUSER="cloud-user"
MDM_CONTAINER_NAME="mdm"

echo "Using:"

echo "RESOURCEGROUP             = $RESOURCEGROUP"
echo "HOSTNAME                  = $HOSTNAME"
echo "LOCATION                  = $LOCATION"
echo "MDM_CONTAINER_NAME        = $MDM_CONTAINER_NAME"
echo "MDM_FRONTEND_URL          = $MDM_FRONTEND_URL"
echo "MDM_IMAGE                 = $MDM_IMAGE"
echo "MDM_SOURCE_ENV            = $MDM_SOURCE_ENVIRONMENT"
echo "MDM_SOURCE_ROLE           = $MDM_SOURCE_ROLE"
echo "MDM_SOURCE_ROLE_INSTANCE  = $MDM_SOURCE_ROLE_INSTANCE"
echo "MDM_VM_NAME               = $MDM_VM_NAME"
echo "(Check MDM_IMAGE against MdmImage function in pkg/util/version/const.go for latest version)"

MDM_VM_VNET=""
if [ "$(az vm show -g $RESOURCEGROUP --name $MDM_VM_NAME)" = "" ];
then
  echo "Creating VM $MDM_VM_NAME in RG $RESOURCEGROUP"
  if [ "$MDM_VM_PRIVATE" != "" ] || [ "$MDM_VM_PRIVATE" == "null" ];
  then
    MDM_VM_VNET="--vnet-name dev-vnet --subnet ToolingSubnet"
  fi   
  az vm create -g $RESOURCEGROUP -n $MDM_VM_NAME --image RedHat:RHEL:7-LVM:latest --ssh-key-values @./secrets/mdm_id_rsa.pub --admin-username $CLOUDUSER $MDM_VM_VNET
else
  echo "VM already exists, skipping..."
fi

VM_IP_PROPERTY=".[0].virtualMachine.network.publicIpAddresses[0].ipAddress"
if [ "$MDM_VM_PRIVATE" != "" ] || [ "$MDM_VM_PRIVATE" == "null" ];
then
  VM_IP_PROPERTY=".[0].virtualMachine.network.privateIpAddresses[0]"
fi
MDM_VM_IP=$( az vm list-ip-addresses --name $MDM_VM_NAME -g $RESOURCEGROUP -o json | jq -r $VM_IP_PROPERTY )
echo "Found IP $MDM_VM_IP"

scp -i ./secrets/mdm_id_rsa $BASE/secrets/rp-metrics-int.pem $CLOUDUSER@$MDM_VM_IP:mdm.pem
scp -i ./secrets/mdm_id_rsa $BASE/hack/local-monitor-testing/configureRemote.sh $CLOUDUSER@$MDM_VM_IP:

ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo cp mdm.pem /etc/mdm.pem"
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo ./configureRemote.sh"

ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo docker pull $MDM_IMAGE"

cat <<EOF > $BASE/dockerStartCommand.sh
docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname $HOSTNAME \
  --name $MDM_CONTAINER_NAME \
  -d \
  --restart=always \
  -m 2g \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $MDM_IMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDM_FRONTEND_URL \
  -Logger Console \
  -LogLevel Debug \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDM_SOURCE_ENVIRONMENT \
  -SourceRole $MDM_SOURCE_ROLE \
  -SourceRoleInstance $MDM_SOURCE_ROLE_INSTANCE
EOF

#disable SELINUX (don't shoot me)
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo setenforce 0"
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo getenforce"

#make it permanent
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo sed -i 's/SELINUX=enforcing/SELINUX=permissive/g' /etc/selinux/config"

ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo firewall-cmd --zone=public --add-port=12345/tcp --permanent"
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo firewall-cmd --reload"

scp -i ./secrets/mdm_id_rsa $BASE/dockerStartCommand.sh $CLOUDUSER@$MDM_VM_IP:
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "chmod +x dockerStartCommand.sh"
ssh -i ./secrets/mdm_id_rsa $CLOUDUSER@$MDM_VM_IP "sudo ./dockerStartCommand.sh &"
