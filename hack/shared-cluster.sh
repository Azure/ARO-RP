#!/bin/bash
set -e
# shared-cluster.sh is used to provide a repeatable production cluster create script


usage() {
	echo -e "usage: ${0} <create|delete>"
	exit 1
}

check_env_set(){
  if [[ -z "${!1}" ]]; then
    echo "$1 is a required ENV but is unset."
    exit 1
  fi
}

check_env_set SHARED_CLUSTER_LOCATION
check_env_set SHARED_CLUSTER_NAME
check_env_set SHARED_CLUSTER_RESOURCE_GROUP_NAME
check_env_set SHARED_CLUSTER_CLUSTER_RESOURCE_GROUP_NAME


if [ "$#" -ne 1 ]; then
	usage

elif [[ $1 == create ]]; then
    echo "creating resource group and network"
    az group create \
    --name $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
    --location $SHARED_CLUSTER_LOCATION \
    --tags persist:true  # This tag stops the RG being cleaned up

    az network vnet create \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      --name aro-vnet \
      --address-prefixes 10.0.0.0/22

    az network vnet subnet create \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      --vnet-name aro-vnet \
      --name master-subnet \
      --address-prefixes 10.0.0.0/23 \
      --service-endpoints Microsoft.ContainerRegistry

    az network vnet subnet create \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      --vnet-name aro-vnet \
      --name worker-subnet \
      --address-prefixes 10.0.2.0/23 \
      --service-endpoints Microsoft.ContainerRegistry

    az network vnet subnet update \
      --name master-subnet \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      --vnet-name aro-vnet \
      --disable-private-link-service-network-policies true
    echo "resource group and vnet/subnets created..."
    
    echo "creating cluster..."
    az aro create \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      --cluster-resource-group $SHARED_CLUSTER_CLUSTER_RESOURCE_GROUP_NAME \
      --name $SHARED_CLUSTER_NAME \
      --vnet aro-vnet \
      --master-subnet master-subnet \
      --worker-subnet worker-subnet

    CLUSTER_RESOURCE_GROUP_ID=$(az aro show \
      --name $SHARED_CLUSTER_NAME \
      --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME \
      | jq .clusterProfile.resourceGroupId)

    echo "Adding tag to cluster..."
    # This tag stops the RG being cleaned up
    az tag create \
      --resource-id $CLUSTER_RESOURCE_GROUP_ID \
      --tags persist=true

elif [[ $1 == "delete" ]]; then
  echo "Deleting cluster..."
  az aro delete --name $SHARED_CLUSTER_NAME -g $SHARED_CLUSTER_RESOURCE_GROUP_NAME --yes
  echo "Deleting Vnet..."
  az network vnet delete --name aro-vnet -g $SHARED_CLUSTER_RESOURCE_GROUP_NAME
  echo "Deleting group..."
  az group delete --resource-group $SHARED_CLUSTER_RESOURCE_GROUP_NAME --yes

else
  usage
fi
