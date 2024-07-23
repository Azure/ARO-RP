#!/bin/bash

# This script creates a mock MSI object to use for local development
# We use a service principal and certificate as the mock object

scriptPath=$(realpath "$0")
scriptDir=$(dirname "$scriptPath")

source "$scriptDir/local_dev_env.sh"

sp=$(create_mock_msi)
if [[ -z "$sp" ]]; then
    echo "Failed to create mock MSI object"
    exit 1
fi
clientID=$(get_mock_msi_clientID "$sp")
tenantID=$(get_mock_msi_tenantID "$sp")
base64EncodedCert=$(get_mock_msi_cert "$sp")

# Print the extracted values
echo "Client ID: $clientID"
echo "Tenant ID: $tenantID"
echo "Base64 Encoded Certificate: $base64EncodedCert"