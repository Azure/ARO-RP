#!/bin/bash
set -e
# shared-miwi-cluster.sh is used to provide a repeatable production miwi cluster create script


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

check_env_set SHARED_MIWI_CLUSTER_LOCATION
check_env_set SHARED_MIWI_CLUSTER_NAME
check_env_set SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME


if [ "$#" -ne 1 ]; then
	usage

elif [[ $1 == create ]]; then
    echo "creating resource group and network"
    az group create \
    --name $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --location $SHARED_MIWI_CLUSTER_LOCATION \
    --tags persist:true  # This tag stops the RG being cleaned up

    az network vnet create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-vnet \
    --address-prefixes 10.0.0.0/22

    az network vnet subnet create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --vnet-name aro-vnet \
    --name master \
    --address-prefixes 10.0.0.0/23 \
    --service-endpoints Microsoft.ContainerRegistry

    az network vnet subnet create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --vnet-name aro-vnet \
    --name worker \
    --address-prefixes 10.0.2.0/23 \
    --service-endpoints Microsoft.ContainerRegistry
    echo "resource group and vnet/subnets created..."

    echo "creating managed identities..."
    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name cloud-controller-manager

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name ingress

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name machine-api

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name disk-csi-driver

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name cloud-network-config

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name image-registry

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name file-csi-driver

    az identity create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-operator
    echo "managed identities created..."

    echo "creating role assignments..."
    SUBSCRIPTION_ID=$(az account show \
    --query 'id' -o tsv)

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/aro-operator"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cloud-controller-manager"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/ingress"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/machine-api"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/disk-csi-driver"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cloud-network-config"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/image-registry"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-cluster \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/ef318e2a-8334-4a05-9e4a-295a196c6a6e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.ManagedIdentity/userAssignedIdentities/file-csi-driver"

    echo "assigning vnet-level permissions for operators that require it, and subnets-level permission for operators that require it..."

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name cloud-controller-manager \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/master"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name cloud-controller-manager \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name ingress \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/master"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name ingress \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name machine-api \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/master"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name machine-api \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name cloud-network-config \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/be7a6435-15ae-4171-8f30-4a343eff9e8f" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name file-csi-driver \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0d7aedc0-15fd-4a67-a412-efad370c947e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/master"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name file-csi-driver \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/0d7aedc0-15fd-4a67-a412-efad370c947e" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-operator \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/master"

    az role assignment create \
    --assignee-object-id "$(az identity show \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name aro-operator \
    --query principalId -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"

    az role assignment create \
    --assignee-object-id "$(az ad sp list \
    --display-name "Azure Red Hat OpenShift RP" \
    --query '[0].id' -o tsv)" \
    --assignee-principal-type ServicePrincipal \
    --role "/subscriptions/$SUBSCRIPTION_ID/providers/Microsoft.Authorization/roleDefinitions/4d97b98b-1d4f-4787-a291-c67834d212e7" \
    --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME/providers/Microsoft.Network/virtualNetworks/aro-vnet"
    echo "role assignments created..."

    echo "creating the cluster..."
    az aro create \
    --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
    --name $SHARED_MIWI_CLUSTER_NAME \
    --vnet aro-vnet \
    --master-subnet master \
    --worker-subnet worker \
    --version 4.15.35 \
    --enable-managed-identity \
    --assign-cluster-identity aro-cluster \
    --assign-platform-workload-identity file-csi-driver file-csi-driver \
    --assign-platform-workload-identity cloud-controller-manager cloud-controller-manager \
    --assign-platform-workload-identity ingress ingress \
    --assign-platform-workload-identity image-registry image-registry \
    --assign-platform-workload-identity machine-api machine-api \
    --assign-platform-workload-identity cloud-network-config cloud-network-config \
    --assign-platform-workload-identity aro-operator aro-operator \
    --assign-platform-workload-identity disk-csi-driver disk-csi-driver
    echo "shared MIWI cluster created."

    CLUSTER_RESOURCE_GROUP_ID=$(az aro show \
      --name $SHARED_MIWI_CLUSTER_NAME \
      --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME \
      | jq -r .clusterProfile.resourceGroupId)

    echo "Adding tag to cluster..."
    # This tag stops the managed RG being cleaned up
    az tag create \
      --resource-id $CLUSTER_RESOURCE_GROUP_ID \
      --tags persist=true

elif [[ $1 == "delete" ]]; then
  echo "Deleting shared MIWI cluster..."
  az aro delete --name $SHARED_MIWI_CLUSTER_NAME -g $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME --yes
  echo "Deleting Vnet..."
  az network vnet delete --name aro-vnet -g $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME
  echo "Deleting resource group..."
  az group delete --resource-group $SHARED_MIWI_CLUSTER_RESOURCE_GROUP_NAME --yes
  echo "Shared MIWI cluster deleted."

else
  usage
fi