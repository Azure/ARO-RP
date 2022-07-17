#!/bin/bash

# shared-cluster.sh is used to provide a repeatable production cluster create script

LOCATION="westcentralus"
CLUSTER="shared-cluster"
RESOURCEGROUP="shared-cluster"
CLUSTERRESOURCEGROUP="aro-shared-cluster"

usage() {
	echo -e "usage: ${0} <create|delete>"
	exit 1
}

if [ "$#" -ne 1 ]; then
	usage

elif [[ $1 == create ]]; then
    echo "creating resource group and network"
    az group create --name $RESOURCEGROUP --location $LOCATION

    az network vnet create \
      --resource-group $RESOURCEGROUP \
      --name aro-vnet \
      --address-prefixes 10.0.0.0/22

    az network vnet subnet create \
      --resource-group $RESOURCEGROUP \
      --vnet-name aro-vnet \
      --name master-subnet \
      --address-prefixes 10.0.0.0/23 \
      --service-endpoints Microsoft.ContainerRegistry

    az network vnet subnet create \
      --resource-group $RESOURCEGROUP \
      --vnet-name aro-vnet \
      --name worker-subnet \
      --address-prefixes 10.0.2.0/23 \
      --service-endpoints Microsoft.ContainerRegistry

    az network vnet subnet update \
      --name master-subnet \
      --resource-group $RESOURCEGROUP \
      --vnet-name aro-vnet \
      --disable-private-link-service-network-policies true
    echo "resource group and vnet/subnets created..."
    
    echo "creating cluster..."
    az aro create \
      --resource-group $RESOURCEGROUP \
      --cluster-resource-group $CLUSTERRESOURCEGROUP \
      --name $CLUSTER \
      --vnet aro-vnet \
      --master-subnet master-subnet \
      --worker-subnet worker-subnet

elif [[ $1 == "delete" ]]; then
  az aro delete --name $CLUSTER -g $RESOURCEGROUP --yes
  az network vnet delete --name aro-vnet -g $RESOURCEGROUP
  az group delete --resource-group $RESOURCEGROUP --yes

else
  usage
fi
