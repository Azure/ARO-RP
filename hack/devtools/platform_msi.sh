#!/bin/bash

# This script creates a platform Identities to use for local development
# The script reads the operators file and creates platform identities for each operator

prompt_for_operators_file() {
    local operatorsFilePath
    read -p "Please provide the absolute path of operators file including the file name?" operatorsFilePath

    if [ -z "$operatorsFilePath" ] || ! [ -f "$operatorsFilePath" ]; then
        echo "ERROR: File ($operatorsFilePath) doesn't exists. Please provide the valid path of operators file."
        exit 1
    fi

    dos2unix "$operatorsFilePath"
    echo "$operatorsFilePath"
}

create_platform_identity() {
    local result=$(az identity create --name "$1" --resource-group "$RESOURCEGROUP" --subscription "$AZURE_SUBSCRIPTION_ID" --output json)

    # Extract the client ID, principal Id, resource ID and name from the result
    clientID=$(echo $result | jq -r .clientId)
    principalId=$(echo $result | jq -r .principalId)
    resourceId=$(echo $result | jq -r .id)
    name=$(echo $result | jq -r .name)

    echo "Client ID: $clientID"
    echo "Principal ID: $principalId"
    echo "Resource ID: $resourceId"
    echo "Name: $name"
}

process_operators_file() {
    local operatorsFilePath="$1"

    echo "INFO: Creating platform identities under RG ($RESOURCEGROUP) and Sub Id ($AZURE_SUBSCRIPTION_ID)"
    echo ""
    while read operator || [ -n "${operator}" ]; do
        echo "INFO: Creating platform identity for operator: ${operator}"
        create_platform_identity "$operator"
        echo ""
    done < "${operatorsFilePath}"
}

main() {
    operatorsFilePath=$(prompt_for_operators_file)
    process_operators_file "$operatorsFilePath"
}

main

