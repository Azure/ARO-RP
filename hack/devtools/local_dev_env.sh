#!/bin/bash

# Local development environment script.
# Execute this script from the root folder of the repo (ARO-RP).
# This script is aimed to provide an automatic and easy way to prepare 
# the environment and execute the ARO RP locally.
# The steps here are the ones defined in docs/deploy-development-rp.md
# We recommend to use this script after you understand the steps of the process, not before.


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
    read -p "Do you want to create a default env file? (existing one will be overwritten, if any) (y / n) " answer

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
    az ad sp create-for-rbac --name $appName --create-cert --output json
}

get_mock_msi_clientID() {
    echo "$1" | jq -r .appId
}

get_mock_msi_tenantID() {
    echo "$1" | jq -r .tenant
}

get_mock_msi_cert() {
    certFilePath=$(echo "$1" | jq -r '.fileWithCertAndPrivateKey')
    base64EncodedCert=$(base64 -w 0 $certFilePath)
    rm $certFilePath
    echo $base64EncodedCert
}

create_env_file() {
    local answer
    read -p "Do you want to create an env file for Managed/Workload identity development? " answer
    if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
        create_miwi_env_file
    else 
        create_regular_env_file
    fi
}

create_miwi_env_file() {
    echo "INFO: Creating default env config file for managed/workload identity development..."

    mockMSI=$(create_mock_msi)
    mockClientID=$(get_mock_msi_clientID "$mockMSI")
    mockTenantID=$(get_mock_msi_tenantID "$mockMSI")
    mockCert=$(get_mock_msi_cert "$mockMSI")

    cat >env <<EOF
export LOCATION=eastus
export ARO_IMAGE=arointsvc.azurecr.io/aro:latest
export RP_MODE=development # to use a development RP running at https://localhost:8443/
export MOCK_MSI_CLIENT_ID="$mockClientID"
export MOCK_MSI_TENANT_ID="$mockTenantID"
export MOCK_MSI_CERT="$mockCert"

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
    read -p "Create Azure deployment in the current subscription ($AZURE_SUBSCRIPTION_ID)? (y / n / l (list existing deployments)) " answer

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
    az deployment group list --resource-group $RESOURCEGROUP | jq '[ .[] | {deployment_name: ( .id ) | split("/deployments/")[1] } | .deployment_name ]'
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
