#!/bin/bash -e

# TODO - check if needed
export USER=dummy
# Export the SECRET_SA_ACCOUNT_NAME environment variable and run make secrets
az account set -s fe16a035-e540-4ab7-80d9-373fa9a3d6ae
export SECRET_SA_ACCOUNT_NAME=rharosecretsdev && make secrets

# Define the expected directory and file names
expected_dir="secrets"
files=("env" "dev-ca.crt" "dev-client.crt")

# Validate that the secrets directory and required files exist
[ -d "$expected_dir" ] || { echo "Directory '$expected_dir' has not been created."; exit 1; }
for file in "${files[@]}"; do
  [ -f "$expected_dir/$file" ] || { echo "File '$file' does not exist inside the directory '$expected_dir'."; exit 1; }
done

export LOCATION=eastus
# Source environment variables from the secrets file
source secrets/env
echo "Success step 1 ✅ - Directory '$expected_dir' has been created with files - ${files[@]}"

export AZURE_PREFIX=$1 NO_CACHE=true ARO_INSTALL_VIA_HIVE=true ARO_ADOPT_BY_HIVE=true DATABASE_NAME=ARO

azure_resource_name=${AZURE_PREFIX}-aro-$LOCATION
# TODO truncate to 20 characters
export RESOURCEGROUP=$azure_resource_name DATABASE_ACCOUNT_NAME=$azure_resource_name KEYVAULT_PREFIX=$azure_resource_name
gitCommit=$(git rev-parse --short=7 HEAD)
export ARO_IMAGE=${AZURE_PREFIX}aro.azurecr.io/aro:$gitCommit

# Run the make command to generate dev-config.yaml
make dev-config.yaml

# Check if the dev-config.yaml file exists
[ -f "dev-config.yaml" ] || { echo "File dev-config.yaml does not exist."; exit 1; }
echo "Success step 2 ✅ - Config file dev-config.yaml has been created"
