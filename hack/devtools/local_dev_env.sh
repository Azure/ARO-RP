#!/bin/bash

# Local development environment script.
# Execute this script from the root folder of the repo (ARO-RP).
# This script is aimed to provide an automatic and easy way to prepare 
# the environment and execute the ARO RP locally.
# The steps here are the ones defined in docs/deploy-development-rp.md
# We recommend to use this script after you understand the steps of the process, not before.

PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS='[
    {
        "openShiftVersion": "4.14",
        "platformWorkloadIdentityRoles": [
            {
                "operatorName": "CloudControllerManager",
                "roleDefinitionName": "Azure Red Hat OpenShift Cloud Controller Manager Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
                "serviceAccounts": ["openshift-cloud-controller-manager:cloud-controller-manager"]
            },
            {
                "operatorName": "ClusterIngressOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Cluster Ingress Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
                "serviceAccounts": ["openshift-ingress-operator:ingress-operator"]
            },
            {
                "operatorName": "MachineApiOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Machine API Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637",
                "serviceAccounts": ["openshift-machine-api:machine-api-operator"]
            },
            {
                "operatorName": "StorageOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Storage Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/5b7237c5-45e1-49d6-bc18-a1f62f400748",
                "serviceAccounts": [
                    "openshift-cluster-csi-drivers:azure-disk-csi-driver-operator",
                    "openshift-cluster-csi-drivers:azure-disk-csi-driver-controller-sa"
                ]
            },
            {
                "operatorName": "NetworkOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Network Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/be7a6435-15ae-4171-8f30-4a343eff9e8f",
                "serviceAccounts": ["openshift-cloud-network-config-controller:cloud-network-config-controller"]
            },
            {
                "operatorName": "ImageRegistryOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Image Registry Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/8b32b316-c2f5-4ddf-b05b-83dacd2d08b5",
                "serviceAccounts": [
                    "openshift-image-registry:cluster-image-registry-operator",
                    "openshift-image-registry:registry"
                ]
            },
            {
                "operatorName": "AzureFilesStorageOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Azure Files Storage Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/0d7aedc0-15fd-4a67-a412-efad370c947e",
                "serviceAccounts": [
                    "openshift-cluster-csi-drivers:azure-file-csi-driver-operator",
                    "openshift-cluster-csi-drivers:azure-file-csi-driver-controller-sa",
                    "openshift-cluster-csi-drivers:azure-file-csi-driver-node-sa"
                ]
            },
            {
                "operatorName": "ServiceOperator",
                "roleDefinitionName": "Azure Red Hat OpenShift Service Operator Role",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2",
                "serviceAccounts": ["openshift-azure-operator:aro-operator-master"]
            }
        ]
    }
]'

build_development_az_aro_extension() {
    echo "INFO: Building development az aro extension..."
    make az
}

verify_aro_extension() {
    echo "INFO: Verifying aro extension..."
    grep -q 'dev_sources' ~/.azure/config || cat >>~/.azure/config <<EOF
[extension]
dev_sources = $PWD/python
EOF
}

set_storage_account() {
    echo "INFO: Setting storage account..."
    export SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
}

ask_to_create_default_env_config() {
    local answer
    read -r -p "Do you want to create a default env file? (existing one will be overwritten, if any) (y / n) " answer

    if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
        create_env_file
    elif [[ "$answer" == "n" || "$answer" == "N" ]]; then
        echo "INFO: Skipping creation of default env file..."
    else
        echo "INFO: Unknown option, skipping step..."
    fi
}

# We use a service principal and certificate as the mock MSI object
create_mock_msi() {
    appName="mock-msi-$(openssl rand -base64 9 | tr -dc 'a-zA-Z0-9' | head -c 6)"
    az ad sp create-for-rbac --name "$appName" --create-cert --output json
}

get_mock_msi_clientID() {
    echo "$1" | jq -r .appId
}

get_mock_msi_tenantID() {
    echo "$1" | jq -r .tenant
}

get_mock_msi_cert() {
    certFilePath=$(echo "$1" | jq -r '.fileWithCertAndPrivateKey')
    base64EncodedCert=$(base64 -w 0 "$certFilePath")
    rm "$certFilePath"
    echo "$base64EncodedCert"
}

create_env_file() {
    local answer
    read -r -p "Do you want to create an env file for Managed/Workload identity development? " answer
    if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
        create_miwi_env_file
    else 
        create_regular_env_file
    fi
}

get_platform_workloadIdentity_role_sets() {   
    # Parse the JSON data using jq
    platformWorkloadIdentityRoles=$(echo "${PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS}" | jq -c '.[].platformWorkloadIdentityRoles[]')

    echo "${platformWorkloadIdentityRoles}"
}

assign_role_to_identity() {
    local objectId=$1
    local roleId=$2
    local scope="/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCEGROUP}"
    
    result=$(az role assignment list --assignee "${objectId}" --role "${roleId}" --scope "${scope}" 2>/dev/null | wc -l)
    if [[ $result -gt 1 ]]; then
        echo "INFO: Role already assigned to identity: ${objectId}"
        echo ""
        return
    fi

    echo "INFO: Assigning roles to identity: ${objectId}"
    az role assignment create --assignee-object-id "${objectId}" --assignee-principal-type "ServicePrincipal" --role "${roleId}"  --scope "${scope}" --output json
    echo ""
}

create_platform_identity_and_assign_role() {
    local operatorName="${1}"
    local roleDefinitionId="${2}"
    local identityName="aro-${operatorName}"
    
    identity=$(az identity show --name "${identityName}" --resource-group "${RESOURCEGROUP}" --subscription "${AZURE_SUBSCRIPTION_ID}" --output json 2>/dev/null)
    if [[ -n ${identity} ]]; then
        echo "INFO: Platform identity ${identityName} already exists for operator: ${operatorName}"
        echo ""
    else
        echo "INFO: Creating platform identity for operator: ${operatorName}"
        result=$(az identity create --name "${identityName}" --resource-group "${RESOURCEGROUP}" --subscription "${AZURE_SUBSCRIPTION_ID}" --output json)
    fi

    # Extract the client ID, principal Id, resource ID and name from the result
    clientID=$(jq -r .clientId <<<"${result}")
    principalId=$(jq -r .principalId <<<"${result}")
    resourceId=$(jq -r .id <<<"${result}")
    name=$(jq -r .name <<<"${result}")

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
    local platformWorkloadIdentityRoles
    
    platformWorkloadIdentityRoles=$(get_platform_workloadIdentity_role_sets)

    echo "INFO: Creating platform identities under RG ($RESOURCEGROUP) and Sub Id ($AZURE_SUBSCRIPTION_ID)"
    echo ""

    # Loop through each element under platformWorkloadIdentityRoles
    while read -r role; do
        operatorName=$(echo "$role" | jq -r '.operatorName')
        roleDefinitionId=$(echo "$role" | jq -r '.roleDefinitionId' | awk -F'/' '{print $NF}')

        echo "INFO: Setting up platform identity for operator: ${operatorName}, role: ${roleDefinitionId}"

        create_platform_identity_and_assign_role "${operatorName}" "${roleDefinitionId}"

    done <<< "$platformWorkloadIdentityRoles"
}

cluster_msi_role_assignment() {
    local clusterMSIAppID="${1}"
    local FEDERATED_CREDENTIAL_ROLE_ID="ef318e2a-8334-4a05-9e4a-295a196c6a6e"
    local clusterMSIObjectID
    
    clusterMSIObjectID=$(az ad sp show --id "${clusterMSIAppID}" --query '{objectId: id}' | jq -r .objectId)

    echo "INFO: Assigning role to cluster MSI: ${clusterMSIAppID}"
    assign_role_to_identity "${clusterMSIObjectID}" "${FEDERATED_CREDENTIAL_ROLE_ID}"
}

create_miwi_env_file() {
    echo "INFO: Creating default env config file for managed/workload identity development..."

    mockMSI=$(create_mock_msi)
    mockClientID=$(get_mock_msi_clientID "$mockMSI")
    mockTenantID=$(get_mock_msi_tenantID "$mockMSI")
    mockCert=$(get_mock_msi_cert "$mockMSI")

    setup_platform_identity
    cluster_msi_role_assignment "${mockClientID}"

    cat >env <<EOF
export LOCATION=eastus
export ARO_IMAGE=arointsvc.azurecr.io/aro:latest
export RP_MODE=development # to use a development RP running at https://localhost:8443/
export MOCK_MSI_CLIENT_ID="$mockClientID"
export MOCK_MSI_TENANT_ID="$mockTenantID"
export MOCK_MSI_CERT="$mockCert"
export PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS="$PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"

source secrets/env
EOF
}

create_regular_env_file() {
    echo "INFO: Creating default env config file..."

    cat >env <<EOF
export LOCATION=eastus
export ARO_IMAGE=arointsvc.azurecr.io/aro:latest
export RP_MODE=development # to use a development RP running at https://localhost:8443/

source secrets/env
EOF
}


ask_to_create_Azure_deployment() {
    local answer
    read -p -r "Create Azure deployment in the current subscription ($AZURE_SUBSCRIPTION_ID)? (y / n / l (list existing deployments)) " answer

    if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
        create_Azure_deployment
    elif [[ "$answer" == "n" || "$answer" == "N" ]]; then
        echo "INFO: Skipping creation of Azure deployment..."
    elif [[ "$answer" == "l" ]]; then
        list_Azure_deployment_names
        ask_to_create_Azure_deployment
    else
        echo "INFO: Unknown option, skipping step..."
    fi
}

list_Azure_deployment_names() {
    echo "INFO: Existing deployment names in the current subscription ($AZURE_SUBSCRIPTION_ID):"
    az deployment group list --resource-group "$RESOURCEGROUP" | jq '[ .[] | {deployment_name: ( .id ) | split("/deployments/")[1] } | .deployment_name ]'
}

create_Azure_deployment() {
    echo "INFO: Creating Azure deployment..."

    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n "databases-development-$USER" \
        --template-file pkg/deploy/assets/databases-development.json \
        --parameters \
        "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
        "databaseName=$DATABASE_NAME" \
        >/dev/null
    
    echo "INFO: Azure deployment created."
}

source_env() {
    echo "INFO: Sourcing env file..."
    source ./env
}

run_the_RP() {
    echo "INFO: Running the ARO RP locally..."
    make runlocal-rp
}

main() {
    build_development_az_aro_extension
    verify_aro_extension
    set_storage_account
    ask_to_create_default_env_config
    source_env
    ask_to_create_Azure_deployment
    run_the_RP
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi
