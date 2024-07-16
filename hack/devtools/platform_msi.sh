#!/bin/bash

# This script creates a platform Identities to use for local development
# The script reads the env var PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS and creates platform identities for each operator

get_platform_workloadIdentity_role_sets() {
    local platformWorkloadIdentityRoles
   
    # Parse the JSON data using jq
    platformWorkloadIdentityRoles=$(echo "${PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS}" | jq -c '.[].platformWorkloadIdentityRoles[]')

    echo "${platformWorkloadIdentityRoles}"
}

assign_role_to_platform_identity() {
    local principalId=$1
    local roleId=$2
    local scope="/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}"

    echo "INFO: Assigning roles to platform identity: ${principalId}"
    local result=$(az role assignment create --assignee "${principalId}" --role "${roleId}"  --scope "${scope}" --output json)

    echo "Role assignment result: $result"
    echo ""
}

create_platform_identity_and_assign_role() {
    local operatorName="${1}"
    local roleDefinitionId="${2}"

    echo "INFO: Creating platform identity for operator: ${operatorName}"
    local result=$(az identity create --name "${operatorName}" --resource-group "${RESOURCEGROUP}" --subscription "${AZURE_SUBSCRIPTION_ID}" --output json)

    # Extract the client ID, principal Id, resource ID and name from the result
    clientID=$(echo $result | jq -r .clientId)
    principalId=$(echo $result | jq -r .principalId)
    resourceId=$(echo $result | jq -r .id)
    name=$(echo $result | jq -r .name)

    echo "Client ID: $clientID"
    echo "Principal ID: $principalId"
    echo "Resource ID: $resourceId"
    echo "Name: $name"
    echo ""

    if [[ "${operatorName}" == "MachineApiOperator" || "${operatorName}" == "NetworkOperator" \
        || "${operatorName}" == "AzureFilesStorageOperator" || "${operatorName}" == "ServiceOperator" ]]; then

        assign_role_to_platform_identity "${principalId}" "${roleDefinitionId}"
    fi
}

setup_platform_identity() {
    local platformWorkloadIdentityRoles=$(get_platform_workloadIdentity_role_sets)

    echo "INFO: Creating platform identities under RG ($RESOURCEGROUP) and Sub Id ($AZURE_SUBSCRIPTION_ID)"
    echo ""

    # Loop through each element under platformWorkloadIdentityRoles
    while read -r role; do
        operatorName=$(echo "$role" | jq -r '.operatorName')
        roleDefinitionId=$(echo "$role" | jq -r '.roleDefinitionId' | awk -F'/' '{print $NF}')

        create_platform_identity_and_assign_role "${operatorName}" "${roleDefinitionId}"

    done <<< "$platformWorkloadIdentityRoles"
}

main() {

    if [[ -z "${PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS}" ]]; then
        echo "ERROR: Env Variable PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS is not set."
        exit 1
    fi

    setup_platform_identity
}

main