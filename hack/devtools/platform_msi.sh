#!/bin/bash

# This script creates a platform Identities to use for local development
# The script reads the env var PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS and creates platform identities for each operator
# The script require following env vars to be set:
#       - PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS: JSON data containing the role sets for each operator
#       - AZURE_SUBSCRIPTION_ID: Azure subscription ID
#       - RESOURCEGROUP: Azure resource group name
#       - CLUSTER: Cluster name
#       - CLUSTER_MSI_APP_ID: Cluster MSI App ID

get_platform_workloadIdentity_role_sets() {
    local platformWorkloadIdentityRoles
   
    # Parse the JSON data using jq
    platformWorkloadIdentityRoles=$(echo "${PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS}" | jq -c '.[].platformWorkloadIdentityRoles[]')

    echo "${platformWorkloadIdentityRoles}"
}

assign_role_to_identity() {
    local principalId=$1
    local roleId=$2
    local scope="/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}"
    local result=$(az role assignment list --assignee "${principalId}" --role "${roleId}" --scope "${scope}" 2>/dev/null | wc -l)

    if [[ $result -gt 1 ]]; then
        echo "INFO: Role already assigned to identity: ${principalId}"
        echo ""
        return
    fi

    echo "INFO: Assigning roles to identity: ${principalId}"
    result=$(az role assignment create --assignee-object-id "${principalId}" --assignee-principal-type "ServicePrincipal" --role "${roleId}"  --scope "${scope}" --output json)

    echo "Role assignment result: ${result}"
    echo ""
}

create_platform_identity_and_assign_role() {
    local operatorName="${1}"
    local roleDefinitionId="${2}"
    local identityName="${CLUSTER}-${operatorName}"
    local result=$(az identity show --name "${identityName}" --resource-group "${RESOURCEGROUP}" --subscription "${AZURE_SUBSCRIPTION_ID}" --output json 2>/dev/null)

    if [[ ! -z ${result} ]]; then
        echo "INFO: Platform identity ${identityName} already exists for operator: ${operatorName}"
        echo ""
    else
        echo "INFO: Creating platform identity for operator: ${operatorName}"
        result=$(az identity create --name "${identityName}" --resource-group "${RESOURCEGROUP}" --subscription "${AZURE_SUBSCRIPTION_ID}" --output json)
    fi

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

        assign_role_to_identity "${principalId}" "${roleDefinitionId}"
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

cluster_msi_role_assignment() {
    local FEDERATED_CREDENTIAL_ROLE_ID="ef318e2a-8334-4a05-9e4a-295a196c6a6e"

    if [[ -z ${CLUSTER_MSI_APP_ID} ]]; then
        echo "ERROR: Env Variable CLUSTER_MSI_APP_ID is not set."
        exit 1
    fi

    local principalId=$(az ad sp show --id "${CLUSTER_MSI_APP_ID}" --query '{objectId: id}' | jq -r .objectId)
    assign_role_to_identity "${principalId}" "${FEDERATED_CREDENTIAL_ROLE_ID}"
}

main() {

    if [[ -z "${PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS}" ]]; then
        echo "ERROR: Env Variable PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS is not set."
        exit 1
    fi

    # setup_platform_identity
    cluster_msi_role_assignment
}

main