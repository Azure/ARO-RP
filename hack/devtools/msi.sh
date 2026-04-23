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
require_non_empty_value "$mockClientID" "mock MSI client ID" || exit 1
mockTenantID=$(get_mock_msi_tenantID "$sp")
require_non_empty_value "$mockTenantID" "mock MSI tenant ID" || exit 1
base64EncodedCert=$(get_mock_msi_cert "$sp")
require_non_empty_value "$base64EncodedCert" "mock MSI certificate" || exit 1
if ! mockObjectID=$(get_mock_msi_objectID "$mockClientID"); then
    exit 1
fi

setup_platform_identity || exit 1
cluster_msi_role_assignment "${mockClientID}" || exit 1

# Print the extracted values
echo "Cluster MSI Client ID: $mockClientID"
echo "Cluster MSI Object ID: $mockObjectID"
echo "Cluster MSI Tenant ID: $mockTenantID"
echo "Cluster MSI Base64 Encoded Certificate: $base64EncodedCert"
echo "Platform workload identity role sets: $PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"