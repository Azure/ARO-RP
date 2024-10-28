#!/bin/bash
set -o nounset
set -o pipefail

# This script creates a mock cluster MSI object and platform identities to use for local development
# We use a service principal and certificate as the mock cluster MSI

if [[ -z "${AZURE_SUBSCRIPTION_ID:-}" ]]; then
    echo "Error: AZURE_SUBSCRIPTION_ID is not set."
    exit 1
fi

if [[ -z "${CLUSTER_RESOURCEGROUP:-}" ]]; then
    echo "Error: CLUSTER_RESOURCEGROUP is not set."
    exit 1
fi

scriptPath=$(realpath "$0")
scriptDir=$(dirname "$scriptPath")

source "$scriptDir/local_dev_env.sh"

sp=$(create_mock_msi)
if [[ -z "$sp" ]]; then
    echo "Failed to create mock MSI object"
    exit 1
fi
mockClientID=$(get_mock_msi_clientID "$sp")
mockTenantID=$(get_mock_msi_tenantID "$sp")
base64EncodedCert=$(get_mock_msi_cert "$sp")
mockObjectID=$(get_mock_msi_objectID "$mockClientID")

setup_platform_identity
cluster_msi_role_assignment "${mockClientID}"

# Print the extracted values
echo "Cluster MSI Client ID: $mockClientID"
echo "Cluster MSI Object ID: $mockObjectID"
echo "Cluster MSI Tenant ID: $mockTenantID"
echo "Cluster MSI Base64 Encoded Certificate: $base64EncodedCert"
echo "Platform workload identity role sets: $PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"