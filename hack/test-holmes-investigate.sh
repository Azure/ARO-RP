#!/bin/bash
# Test script for the Holmes investigation admin API endpoint.
#
# Prerequisites:
#   1. VPN connected to the dev environment
#   2. secrets/ folder generated: SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
#   3. AKS kubeconfig generated: make aks.kubeconfig
#   4. A test cluster created via: CLUSTER=<name> go run ./hack/cluster create
#   5. Local RP running with Hive enabled (see below)
#
# Usage:
#   ./hack/test-holmes-investigate.sh <cluster-name> [question]
#
# Examples:
#   ./hack/test-holmes-investigate.sh haowang-holmes-test
#   ./hack/test-holmes-investigate.sh haowang-holmes-test "why is pod X crashing?"
#   ./hack/test-holmes-investigate.sh haowang-holmes-test "check node memory usage"
#
# To start the local RP with Hive + Holmes enabled:
#
#   source env && source secrets/env
#   export HIVE_KUBE_CONFIG_PATH=$(realpath aks.kubeconfig)
#   export ARO_INSTALL_VIA_HIVE=true
#   export ARO_ADOPT_BY_HIVE=true
#   export ARO_PODMAN_SOCKET="unix://$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')"
#   export HOLMES_IMAGE="quay.io/haoran/holmesgpt:latest"
#   export HOLMES_AZURE_API_KEY="<your-azure-openai-key>"
#   export HOLMES_AZURE_API_BASE="<your-azure-openai-endpoint>"
#   export HOLMES_AZURE_API_VERSION="2025-04-01-preview"
#   export HOLMES_MODEL="azure/gpt-5.2"
#   make runlocal-rp

set -euo pipefail

CLUSTER_NAME="${1:-}"
QUESTION="${2:-what is the cluster health status?}"

if [[ -z "$CLUSTER_NAME" ]]; then
    echo "Usage: $0 <cluster-name> [question]"
    echo ""
    echo "Examples:"
    echo "  $0 haowang-holmes-test"
    echo "  $0 haowang-holmes-test 'why is pod X crashing?'"
    exit 1
fi

# Source env if not already loaded
if [[ -z "${AZURE_SUBSCRIPTION_ID:-}" ]]; then
    if [[ -f env ]] && [[ -f secrets/env ]]; then
        source env
        source secrets/env
    else
        echo "Error: AZURE_SUBSCRIPTION_ID not set and env files not found."
        echo "Run from the repo root, or source env && source secrets/env first."
        exit 1
    fi
fi

RESOURCEGROUP="${RESOURCEGROUP:-v4-eastus}"
RP_URL="https://localhost:8443"
API_PATH="/admin/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourcegroups/${RESOURCEGROUP}/providers/Microsoft.RedHatOpenShift/openShiftClusters/${CLUSTER_NAME}/investigate"

echo "============================================"
echo " Holmes Investigation Test"
echo "============================================"
echo " Cluster:  ${CLUSTER_NAME}"
echo " RG:       ${RESOURCEGROUP}"
echo " Question: ${QUESTION}"
echo " Endpoint: POST ${RP_URL}${API_PATH}"
echo "============================================"
echo ""

# Check RP is running
if ! curl -sk -o /dev/null -w '' "${RP_URL}/healthz" 2>/dev/null; then
    echo "Error: Local RP is not running at ${RP_URL}"
    echo "Start it with: make runlocal-rp (see header comments for full env setup)"
    exit 1
fi

echo "Sending investigation request..."
echo "Streaming results (this may take 1-5 minutes):"
echo "--------------------------------------------"

curl -sk --no-buffer -X POST \
    "${RP_URL}${API_PATH}" \
    -H "Content-Type: application/json" \
    -d "{\"question\": \"${QUESTION}\"}"

echo ""
echo "--------------------------------------------"
echo "Investigation complete."
