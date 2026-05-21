#!/bin/bash
set -o nounset
set -o pipefail

# This script creates a mock cluster MSI object and (unless SKIP_MIWI_ROLE_ASSIGNMENT=true)
# gives it the federated credential role assignment scoped to the network resource group.

if [[ -z "${AZURE_SUBSCRIPTION_ID:-}" ]]; then
    echo "Error: AZURE_SUBSCRIPTION_ID is not set."
    exit 1
fi

if [[ -z "${CLUSTER_RESOURCEGROUP:-}" ]]; then
    echo "Error: CLUSTER_RESOURCEGROUP is not set."
    exit 1
fi

# So that we can assign the role to the mock MSI, create the RG if it doesn't exist
az group create --name "${CLUSTER_RESOURCEGROUP}" --location "${LOCATION}"

scriptPath=$(realpath "$0")
scriptDir=$(dirname "$scriptPath")

source "$scriptDir/local_dev_env.sh"
create_miwi_env_file
