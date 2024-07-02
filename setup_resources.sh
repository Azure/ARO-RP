#!/bin/bash

set -e

# Function to validate RP running
validate_rp_running() {
    echo "########## Checking ARO RP Status ##########"
    ELAPSED=0
    while true; do
        sleep 5
        http_code=$(curl -k -s -o /dev/null -w '%{http_code}' https://localhost:8443/healthz/ready || true)
        case $http_code in
        "200")
            echo "########## ✅ ARO RP Running ##########"
            break
            ;;
        *)
            echo "Attempt $ELAPSED - local RP is NOT up. Code : $http_code, waiting"
            sleep 2
            # after 40 secs return exit 1 to not block ci
            ELAPSED=$((ELAPSED + 1))
            if [ $ELAPSED -eq 20 ]; then
                exit 1
            fi
            ;;
        esac
    done
}

# Ensure all env vars are set (CLUSTER_LOCATION, CLUSTER_RESOURCEGROUP, CLUSTER_NAME)
ALL_SET="true"
if [ -z ${AZURE_SUBSCRIPTION_ID} ]; then ALL_SET="false" && echo "AZURE_SUBSCRIPTION_ID is unset"; else echo "AZURE_SUBSCRIPTION_ID is set to '$AZURE_SUBSCRIPTION_ID'"; fi
if [ -z ${LOCATION} ]; then ALL_SET="false" && echo "LOCATION is unset"; else echo "LOCATION is set to '$LOCATION'"; fi
if [ -z ${CLUSTER_RESOURCEGROUP} ]; then ALL_SET="false" && echo "CLUSTER_RESOURCEGROUP is unset"; else echo "CLUSTER_RESOURCEGROUP is set to '$CLUSTER_RESOURCEGROUP'"; fi
if [ -z ${CLUSTER_NAME} ]; then ALL_SET="false" && echo "CLUSTER_NAME is unset"; else echo "CLUSTER_NAME is set to '$CLUSTER_NAME'"; fi
if [ -z ${CLUSTER_VNET} ]; then CLUSTER_VNET="aro-vnet2"; echo "CLUSTER_VNET is ${CLUSTER_VNET}"; fi
if [ -z ${CLUSTER_MASTER_SUBNET} ]; then CLUSTER_MASTER_SUBNET="master-subnet"; echo "CLUSTER_MASTER_SUBNET is ${CLUSTER_MASTER_SUBNET}"; fi
if [ -z ${CLUSTER_WORKER_SUBNET} ]; then CLUSTER_WORKER_SUBNET="worker-subnet"; echo "CLUSTER_WORKER_SUBNET is ${CLUSTER_WORKER_SUBNET}"; fi
if [ -z ${OPENSHIFT_VERSION} ]; then ALL_SET="false" && echo "OPENSHIFT_VERSION is unset"; else echo "OPENSHIFT_VERSION is set to '4.13.23'"; fi

if [[ "${ALL_SET}" != "true" ]]; then exit 1; fi

# Check Azure CLI version
echo "Checking Azure CLI version..."
az_version=$(az --version | grep 'azure-cli' | awk '{print $2}')
required_version="2.30.0"
if  [ "$(printf '%s\n' "$required_version" "$az_version" | sort -V | head -n1)" = "$required_version" ]; then
    echo "Azure CLI version is compatible"
else
    echo "Azure CLI version must be $required_version or later. Please upgrade."
    exit 1
fi

# Set the subscription
echo "Setting the subscription..."
az account set --subscription $AZURE_SUBSCRIPTION_ID

# Register the subscription directly
echo "Registering the subscription directly..."
curl -k -X PUT \
  -H 'Content-Type: application/json' \
  -d '{
    "state": "Registered",
    "properties": {
        "tenantId": "'"$AZURE_TENANT_ID"'",
        "registeredFeatures": [
            {
                "name": "Microsoft.RedHatOpenShift/RedHatEngineering",
                "state": "Registered"
            }
        ]
    }
}' "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"

# Function to add supported OpenShift version
add_openshift_version() {
  local version=$1
  local openshift_pullspec=$2
  local installer_pullspec=$3

  echo "Adding OpenShift version $version..."
  curl -k -X PUT "https://localhost:8443/admin/versions" --header "Content-Type: application/json" -d '{
    "properties": {
      "version": "'"$version"'",
      "enabled": true,
      "openShiftPullspec": "'"$openshift_pullspec"'",
      "installerPullspec": "'"$installer_pullspec"'"
    }
  }'
}

# Add the required OpenShift version
add_openshift_version "4.13.23" "quay.io/openshift-release-dev/ocp-release@sha256:<sha256>" "<name>.azurecr.io/installer:release-4.x"

# Delete the existing cluster if it exists
echo "Deleting the existing cluster if it exists..."
az aro delete --resource-group $CLUSTER_RESOURCEGROUP --name $CLUSTER_NAME --yes --no-wait || true

# Wait for the cluster deletion to complete
echo "Waiting for the cluster to be deleted..."
while az aro show --name $CLUSTER_NAME --resource-group $CLUSTER_RESOURCEGROUP &> /dev/null; do
  echo "Cluster is still being deleted...waiting 30 seconds."
  sleep 30
done

# Create resource group
echo "Creating resource group $CLUSTER_RESOURCEGROUP in $LOCATION..."
az group create --name $CLUSTER_RESOURCEGROUP --location $LOCATION

# Create virtual network
echo "Creating virtual network $CLUSTER_VNET in $CLUSTER_RESOURCEGROUP..."
az network vnet create --resource-group $CLUSTER_RESOURCEGROUP --name $CLUSTER_VNET --address-prefixes 10.0.0.0/22

# Delete any existing subnets and associated resources
echo "Deleting any existing master subnet resources..."
az network vnet subnet delete --resource-group $CLUSTER_RESOURCEGROUP --vnet-name $CLUSTER_VNET --name $CLUSTER_MASTER_SUBNET || true

echo "Deleting any existing worker subnet resources..."
az network vnet subnet delete --resource-group $CLUSTER_RESOURCEGROUP --vnet-name $CLUSTER_VNET --name $CLUSTER_WORKER_SUBNET || true

# Create master subnet
echo "Creating master subnet $CLUSTER_MASTER_SUBNET in $CLUSTER_VNET..."
az network vnet subnet create --resource-group $CLUSTER_RESOURCEGROUP --vnet-name $CLUSTER_VNET --name $CLUSTER_MASTER_SUBNET --address-prefixes 10.0.0.0/23 --service-endpoints Microsoft.ContainerRegistry

# Create worker subnet
echo "Creating worker subnet $CLUSTER_WORKER_SUBNET in $CLUSTER_VNET..."
az network vnet subnet create --resource-group $CLUSTER_RESOURCEGROUP --vnet-name $CLUSTER_VNET --name $CLUSTER_WORKER_SUBNET --address-prefixes 10.0.2.0/23 --service-endpoints Microsoft.ContainerRegistry

# Create cluster
echo "Creating cluster $CLUSTER_NAME in $CLUSTER_RESOURCEGROUP..."
az aro create --resource-group $CLUSTER_RESOURCEGROUP --name $CLUSTER_NAME --vnet $CLUSTER_VNET --master-subnet $CLUSTER_MASTER_SUBNET --worker-subnet $CLUSTER_WORKER_SUBNET --pull-secret "$PULL_SECRET" --location $LOCATION --version $OPENSHIFT_VERSION || {
  echo "Cluster creation failed. Fetching deployment logs..."

  # Fetch the deployment logs for further analysis
  deployment_name=$(az deployment group list --resource-group $CLUSTER_RESOURCEGROUP --query '[0].name' -o tsv)
  if [ -n "$deployment_name" ]; then
    az deployment group show --name $deployment_name --resource-group $CLUSTER_RESOURCEGROUP
  else
    echo "No deployment found for resource group $CLUSTER_RESOURCEGROUP."
  fi

  exit 1
}

# Check for the existence of the cluster
if az aro show --name $CLUSTER_NAME --resource-group $CLUSTER_RESOURCEGROUP &> /dev/null; then
  echo "Cluster creation successful."
else
  echo "Cluster creation failed. Please check the logs for more details."
  exit 1
fi

echo "To list cluster credentials, run:"
echo "    az aro list-credentials --name $CLUSTER_NAME --resource-group $CLUSTER_RESOURCEGROUP"

# Validate RP running
validate_rp_running
