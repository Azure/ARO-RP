#!/bin/bash

LOCATION=eastus
SUBSCRIPTION_ID="$(az account show --query id --output tsv)"
CUSTOMER_RG_NAME=bcdr1
MANAGED_RESOURCE_GROUP=bcdr1-managed
CLUSTER_NAME=bcdrhcpmain
CLUSTER_VERSION="4.22"
NP_NAME=bcdr1-nodepool
NP_VERSION="4.22.1"
CUSTOMER_NSG=aro-nsg
CUSTOMER_VNET_NAME=aro-vnet
CUSTOMER_VNET_SUBNET1=aro-subnet1
CUSTOMER_VNET_INTEGRATION_SUBNET_NAME=aro-vnet-int1
KEYVAULT_NAME=bcdr1-kv
KMS_KEY_NAME=etcd-kms-key

kubeconfig=$CLUSTER_NAME.kubeconfig

# create resource group and networking
az group create --name "${CUSTOMER_RG_NAME}" --subscription "${SUBSCRIPTION_ID}" --location "${LOCATION}"
az network nsg create --name "${CUSTOMER_NSG}" --resource-group "${CUSTOMER_RG_NAME}" --location "${LOCATION}"
az network vnet create --name "${CUSTOMER_VNET_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --location "${LOCATION}" --address-prefixes 10.0.0.0/16 --subnet-name "${CUSTOMER_VNET_SUBNET1}" --subnet-prefixes 10.0.0.0/24 --nsg "${CUSTOMER_NSG}"
az network vnet subnet create --name "${CUSTOMER_VNET_INTEGRATION_SUBNET_NAME}" --vnet-name "${CUSTOMER_VNET_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --address-prefixes 10.0.1.0/24 --network-security-group "${CUSTOMER_NSG}" --delegations Microsoft.RedHatOpenShift/hcpOpenShiftClusters

# get networking IDs
NSG_ID=$(az network nsg show --name "${CUSTOMER_NSG}" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
SUBNET_ID=$(az network vnet subnet show --name "${CUSTOMER_VNET_SUBNET1}" --vnet-name "${CUSTOMER_VNET_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
VNET_ID=$(az network vnet show --name "${CUSTOMER_VNET_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
VNET_INTEGRATION_SUBNET_ID=$(az network vnet subnet show --name "${CUSTOMER_VNET_INTEGRATION_SUBNET_NAME}" --vnet-name "${CUSTOMER_VNET_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)

# Control plane operator identities
az identity create --name "${CLUSTER_NAME}-cp-cluster-api-azure" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-control-plane" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-cloud-controller-manager" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-ingress" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-disk-csi-driver" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-image-registry" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-cp-cloud-network-config" --resource-group "${CUSTOMER_RG_NAME}"
# Data plane operator identities
az identity create --name "${CLUSTER_NAME}-dp-disk-csi-driver" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-dp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}"
az identity create --name "${CLUSTER_NAME}-dp-image-registry" --resource-group "${CUSTOMER_RG_NAME}"
# Service managed identity
az identity create --name "${CLUSTER_NAME}-service-managed-identity" --resource-group "${CUSTOMER_RG_NAME}"

# get control plane identity resource IDs and principal IDs
CLUSTER_API_AZURE_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-cluster-api-azure" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
CLUSTER_API_AZURE_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-cluster-api-azure" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
CONTROL_PLANE_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-control-plane" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
CONTROL_PLANE_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-control-plane" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
CLOUD_CONTROLLER_MANAGER_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-cloud-controller-manager" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
CLOUD_CONTROLLER_MANAGER_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-cloud-controller-manager" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
INGRESS_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-ingress" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
INGRESS_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-ingress" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
DISK_CSI_DRIVER_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-disk-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
FILE_CSI_DRIVER_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
FILE_CSI_DRIVER_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
IMAGE_REGISTRY_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-image-registry" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
IMAGE_REGISTRY_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-image-registry" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
CLOUD_NETWORK_CONFIG_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-cloud-network-config" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
CLOUD_NETWORK_CONFIG_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-cloud-network-config" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
# get data plane identity resource IDs and principal IDs
DP_DISK_CSI_DRIVER_MI_ID=$(az identity show --name "${CLUSTER_NAME}-dp-disk-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
DP_FILE_CSI_DRIVER_MI_ID=$(az identity show --name "${CLUSTER_NAME}-dp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
DP_FILE_CSI_DRIVER_MI_PID=$(az identity show --name "${CLUSTER_NAME}-dp-file-csi-driver" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
DP_IMAGE_REGISTRY_MI_ID=$(az identity show --name "${CLUSTER_NAME}-dp-image-registry" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
DP_IMAGE_REGISTRY_MI_PID=$(az identity show --name "${CLUSTER_NAME}-dp-image-registry" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)
# get service managed identity resource ID and principal ID
SERVICE_MI_ID=$(az identity show --name "${CLUSTER_NAME}-service-managed-identity" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
SERVICE_MI_PID=$(az identity show --name "${CLUSTER_NAME}-service-managed-identity" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)

# Set the role definition GUIDs.
READER_ROLE="acdd72a7-3385-48ef-bd42-f606fba81ae7"
HCP_CLUSTER_API_PROVIDER_ROLE="88366f10-ed47-4cc0-9fab-c8a06148393e"
HCP_CONTROL_PLANE_OPERATOR_ROLE="fc0c873f-45e9-4d0d-a7d1-585aab30c6ed"
CLOUD_CONTROLLER_MANAGER_ROLE="a1f96423-95ce-4224-ab27-4e3dc72facd4"
INGRESS_OPERATOR_ROLE="0336e1d3-7a87-462b-b6db-342b63f7802c"
FILE_STORAGE_OPERATOR_ROLE="0d7aedc0-15fd-4a67-a412-efad370c947e"
NETWORK_OPERATOR_ROLE="be7a6435-15ae-4171-8f30-4a343eff9e8f"
FEDERATED_CREDENTIAL_ROLE="ef318e2a-8334-4a05-9e4a-295a196c6a6e"
HCP_SERVICE_MI_ROLE="c0ff367d-66d8-445e-917c-583feb0ef0d4"
IMAGE_REGISTRY_OPERATOR_ROLE="8b32b316-c2f5-4ddf-b05b-83dacd2d08b5"

# Cluster API Azure → ARO HCP Cluster API Provider on worker subnet
az role assignment create --assignee-object-id "${CLUSTER_API_AZURE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_CLUSTER_API_PROVIDER_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${CLUSTER_API_AZURE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_CLUSTER_API_PROVIDER_ROLE}" --scope "${VNET_ID}"
# Control Plane → ARO HCP Control Plane Operator on VNet and NSG
az role assignment create --assignee-object-id "${CONTROL_PLANE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_CONTROL_PLANE_OPERATOR_ROLE}" --scope "${VNET_ID}"
az role assignment create --assignee-object-id "${CONTROL_PLANE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_CONTROL_PLANE_OPERATOR_ROLE}" --scope "${NSG_ID}"
# Cloud Controller Manager → ARO Cloud Controller Manager on worker subnet and NSG
az role assignment create --assignee-object-id "${CLOUD_CONTROLLER_MANAGER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${CLOUD_CONTROLLER_MANAGER_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${CLOUD_CONTROLLER_MANAGER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${CLOUD_CONTROLLER_MANAGER_ROLE}" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${CLOUD_CONTROLLER_MANAGER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${CLOUD_CONTROLLER_MANAGER_ROLE}" --scope "${VNET_ID}"
# Ingress → ARO Cluster Ingress Operator on worker subnet
az role assignment create --assignee-object-id "${INGRESS_MI_PID}" --assignee-principal-type ServicePrincipal --role "${INGRESS_OPERATOR_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${INGRESS_MI_PID}" --assignee-principal-type ServicePrincipal --role "${INGRESS_OPERATOR_ROLE}" --scope "${VNET_ID}"
# File CSI Driver → ARO File Storage Operator on worker subnet and NSG
az role assignment create --assignee-object-id "${FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${VNET_ID}"
# Cloud Network Config → ARO Network Operator on worker subnet and VNet
az role assignment create --assignee-object-id "${CLOUD_NETWORK_CONFIG_MI_PID}" --assignee-principal-type ServicePrincipal --role "${NETWORK_OPERATOR_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${CLOUD_NETWORK_CONFIG_MI_PID}" --assignee-principal-type ServicePrincipal --role "${NETWORK_OPERATOR_ROLE}" --scope "${VNET_ID}"
# Image Registry → ARO Image Registry Operator on VNet
az role assignment create --assignee-object-id "${IMAGE_REGISTRY_MI_PID}" --assignee-principal-type ServicePrincipal --role "${IMAGE_REGISTRY_OPERATOR_ROLE}" --scope "${VNET_ID}"

# Service MI → ARO HCP Service Managed Identity on VNet and NSG
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_SERVICE_MI_ROLE}" --scope "${VNET_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${HCP_SERVICE_MI_ROLE}" --scope "${NSG_ID}"

# Service MI → Reader on each control plane identity
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${CLUSTER_API_AZURE_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${CONTROL_PLANE_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${CLOUD_CONTROLLER_MANAGER_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${INGRESS_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${DISK_CSI_DRIVER_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${FILE_CSI_DRIVER_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${IMAGE_REGISTRY_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${CLOUD_NETWORK_CONFIG_MI_ID}"

# Service MI → ARO Federated Credential on each data plane identity
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FEDERATED_CREDENTIAL_ROLE}" --scope "${DP_DISK_CSI_DRIVER_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FEDERATED_CREDENTIAL_ROLE}" --scope "${DP_FILE_CSI_DRIVER_MI_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FEDERATED_CREDENTIAL_ROLE}" --scope "${DP_IMAGE_REGISTRY_MI_ID}"

# Data Plane File CSI Driver → ARO File Storage Operator on worker subnet and NSG
az role assignment create --assignee-object-id "${DP_FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${DP_FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${DP_FILE_CSI_DRIVER_MI_PID}" --assignee-principal-type ServicePrincipal --role "${FILE_STORAGE_OPERATOR_ROLE}" --scope "${VNET_ID}"

# Data Plane Image Registry → ARO Image Registry Operator on VNet
az role assignment create --assignee-object-id "${DP_IMAGE_REGISTRY_MI_PID}" --assignee-principal-type ServicePrincipal --role "${IMAGE_REGISTRY_OPERATOR_ROLE}" --scope "${VNET_ID}"

# set up etcd encryption
az identity create --name "${CLUSTER_NAME}-cp-kms" --resource-group "${CUSTOMER_RG_NAME}"

# get the kms managed identity IDs
KMS_MI_ID=$(az identity show --name "${CLUSTER_NAME}-cp-kms" --resource-group "${CUSTOMER_RG_NAME}" --query id --output tsv)
KMS_MI_PID=$(az identity show --name "${CLUSTER_NAME}-cp-kms" --resource-group "${CUSTOMER_RG_NAME}" --query principalId --output tsv)

az role assignment create --assignee-object-id "${SERVICE_MI_PID}" --assignee-principal-type ServicePrincipal --role "${READER_ROLE}" --scope "${KMS_MI_ID}"

# create the keyvault and key
az keyvault create --name "${KEYVAULT_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --location "${LOCATION}" --enable-rbac-authorization true
sleep 60
az role assignment create --role "Key Vault Crypto Officer" --assignee "$(az account show --query user.name -o tsv)" --scope "$(az keyvault show --name "${KEYVAULT_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --query id -o tsv)"
sleep 60
az keyvault key create --vault-name "${KEYVAULT_NAME}" --name "${KMS_KEY_NAME}" --kty RSA --size 2048
KMS_KEY_VERSION=$(az keyvault key show --vault-name "${KEYVAULT_NAME}" --name "${KMS_KEY_NAME}" --query "key.kid" --output tsv | rev | cut -d'/' -f1 | rev)
KEY_VAULT_CRYPTO_USER_ROLE="12338af0-0e69-4776-bea7-57ae8d297424"
KEYVAULT_ID=$(az keyvault show --name "${KEYVAULT_NAME}" --query id --output tsv)

az role assignment create --assignee-object-id "${KMS_MI_PID}" --assignee-principal-type ServicePrincipal --role "${KEY_VAULT_CRYPTO_USER_ROLE}" --scope "${KEYVAULT_ID}"

# create the cluster
az aro hcp cluster create --name "${CLUSTER_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --location "${LOCATION}" --version "${CLUSTER_VERSION}" --channel-group stable \
  --subnet-id "${SUBNET_ID}" --vnet-integration-subnet-id "${VNET_INTEGRATION_SUBNET_ID}" --nsg "${NSG_ID}" --managed-resource-group-name "${MANAGED_RESOURCE_GROUP}" \
  --key-management-mode CustomerManaged --etcd-encryption-type KMS --kms-vault-name "${KEYVAULT_NAME}" --vault-visibility Public \
  --kms-active-key "{name:${KMS_KEY_NAME},version:${KMS_KEY_VERSION}}" --api-visibility Public --ingress-visibility Public \
  --user-assigned-identities "{${SERVICE_MI_ID}:{},${CLUSTER_API_AZURE_MI_ID}:{},${CONTROL_PLANE_MI_ID}:{},${CLOUD_CONTROLLER_MANAGER_MI_ID}:{},${INGRESS_MI_ID}:{},${DISK_CSI_DRIVER_MI_ID}:{},${FILE_CSI_DRIVER_MI_ID}:{},${IMAGE_REGISTRY_MI_ID}:{},${CLOUD_NETWORK_CONFIG_MI_ID}:{},${KMS_MI_ID}:{}}" --operators-authentication "{user-assigned-identities:{control-plane-operators:{cluster-api-azure:${CLUSTER_API_AZURE_MI_ID},control-plane:${CONTROL_PLANE_MI_ID},cloud-controller-manager:${CLOUD_CONTROLLER_MANAGER_MI_ID},ingress:${INGRESS_MI_ID},disk-csi-driver:${DISK_CSI_DRIVER_MI_ID},file-csi-driver:${FILE_CSI_DRIVER_MI_ID},image-registry:${IMAGE_REGISTRY_MI_ID},cloud-network-config:${CLOUD_NETWORK_CONFIG_MI_ID},kms:${KMS_MI_ID}},data-plane-operators:{disk-csi-driver:${DP_DISK_CSI_DRIVER_MI_ID},file-csi-driver:${DP_FILE_CSI_DRIVER_MI_ID},image-registry:${DP_IMAGE_REGISTRY_MI_ID}},service-managed-identity:${SERVICE_MI_ID}}}"

# create the nodepool
az aro hcp cluster nodepool create --cluster-name "${CLUSTER_NAME}" --name "${NP_NAME}" --resource-group "${CUSTOMER_RG_NAME}" --replicas 2 --vm-size Standard_D8s_v3 --version "${NP_VERSION}" --channel-group stable

# wait for cluster to complete
az aro hcp cluster request-credential --admin -g $CUSTOMER_RG_NAME -n $CLUSTER_NAME -f $kubeconfig
export KUBECONFIG=$kubeconfig

# set up a namespace for a static web server to identify which cluster this is
oc create namespace staticweb
oc config set contexts.${CLUSTER_NAME}.namespace staticweb

# deploy nginx in it
oc create deployment nginx --image=nginxinc/nginx-unprivileged:latest -r 2 --port 8080
pod=$(oc get pod -l app=nginx -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | head -1)
oc wait --for=condition=Ready pod/$pod

# create a volume to hold a basic index.html
oc set volume deploy/nginx --add --name=staticweb --type=pvc --claim-size=1M --claim-class=azurefile-csi --claim-mode=ReadOnlyMany --mount-path=/usr/share/nginx/html/
pod=$(oc get pod --sort-by .metadata.creationTimestamp -l app=nginx -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | tail -1)
oc wait --for=condition=Ready pod/$pod
echo "welcome to cluster1" > index.html
oc cp ./index.html $pod:/usr/share/nginx/html/ 
rm ./index.html

# expose the nginx service and stash the route's hostname for use in AFD
oc create service clusterip nginx --tcp=8080:8080
oc expose service nginx
oc get route nginx -o jsonpath='{.status.ingress[0].host}{"\n"}' > cluster1_route.txt
