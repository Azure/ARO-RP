#!/bin/bash
set -euo pipefail

# deploy-holmes-aoai.sh - Deploy Azure OpenAI resource and GPT model for Holmes admin API
#
# Uses Entra ID authentication (no API keys). The RP's managed identity (prod)
# or dev service principal acquires tokens at request time via the
# "Cognitive Services OpenAI User" RBAC role.
#
# Prerequisites:
#   - az CLI logged in with Cognitive Services Contributor role
#   - Source env file first (provides RESOURCEGROUP, LOCATION, AZURE_RP_CLIENT_ID)
#
# Usage:
#   source env
#   ./hack/devtools/deploy-holmes-aoai.sh

# Validate required env vars
if [[ -z "${RESOURCEGROUP:-}" ]]; then
    echo "Error: RESOURCEGROUP is not set. Source your env file first."
    exit 1
fi

if [[ -z "${LOCATION:-}" ]]; then
    echo "Error: LOCATION is not set. Source your env file first."
    exit 1
fi

if [[ -z "${AZURE_RP_CLIENT_ID:-}" ]]; then
    echo "Error: AZURE_RP_CLIENT_ID is not set. Source your env file first."
    exit 1
fi

# Constants
HOLMES_AOAI_ACCOUNT_NAME="${RESOURCEGROUP}-holmes-aoai"
HOLMES_AOAI_DEPLOYMENT_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_VERSION="2025-12-11"
HOLMES_AOAI_SKU="S0"
HOLMES_AOAI_DEPLOYMENT_SKU_NAME="GlobalStandard"
HOLMES_AOAI_DEPLOYMENT_SKU_CAPACITY=10
HOLMES_API_VERSION="2025-04-01-preview"
# Cognitive Services OpenAI User — inference only, no management operations.
COGNITIVE_SERVICES_OPENAI_USER_ROLE="5e0bd9bd-7b93-4f28-af87-19fc36ad61bd"

deploy_holmes_aoai_account() {
    echo "########## Deploying Azure OpenAI account ${HOLMES_AOAI_ACCOUNT_NAME} in RG ${RESOURCEGROUP} ##########"

    if az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        &>/dev/null; then
        echo "Azure OpenAI account ${HOLMES_AOAI_ACCOUNT_NAME} already exists, skipping creation."
    else
        az cognitiveservices account create \
            --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --location "${LOCATION}" \
            --kind "OpenAI" \
            --sku "${HOLMES_AOAI_SKU}" \
            --custom-domain "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --yes >/dev/null
        echo "Azure OpenAI account ${HOLMES_AOAI_ACCOUNT_NAME} created."
    fi

    # Disable local (API key) auth — only Entra ID tokens are accepted.
    echo "Disabling local auth on ${HOLMES_AOAI_ACCOUNT_NAME}..."
    local aoai_id
    aoai_id=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "id" -o tsv)
    az resource update --ids "${aoai_id}" --set properties.disableLocalAuth=true >/dev/null
    echo "Local auth disabled."
}

deploy_holmes_aoai_model() {
    echo "########## Deploying model ${HOLMES_AOAI_MODEL_NAME} as ${HOLMES_AOAI_DEPLOYMENT_NAME} ##########"

    if az cognitiveservices account deployment show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --deployment-name "${HOLMES_AOAI_DEPLOYMENT_NAME}" \
        &>/dev/null; then
        echo "Model deployment ${HOLMES_AOAI_DEPLOYMENT_NAME} already exists, skipping."
    else
        az cognitiveservices account deployment create \
            --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --deployment-name "${HOLMES_AOAI_DEPLOYMENT_NAME}" \
            --model-name "${HOLMES_AOAI_MODEL_NAME}" \
            --model-version "${HOLMES_AOAI_MODEL_VERSION}" \
            --model-format "OpenAI" \
            --sku-name "${HOLMES_AOAI_DEPLOYMENT_SKU_NAME}" \
            --sku-capacity "${HOLMES_AOAI_DEPLOYMENT_SKU_CAPACITY}" >/dev/null
        echo "Model deployment ${HOLMES_AOAI_DEPLOYMENT_NAME} created."
    fi
}

assign_role() {
    echo "########## Assigning Cognitive Services OpenAI User role to RP service principal ##########"

    local aoai_resource_id
    aoai_resource_id=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "id" -o tsv)

    # Check if assignment already exists to avoid errors on re-run.
    if az role assignment list \
        --assignee "${AZURE_RP_CLIENT_ID}" \
        --role "${COGNITIVE_SERVICES_OPENAI_USER_ROLE}" \
        --scope "${aoai_resource_id}" \
        --query "[0].id" -o tsv 2>/dev/null | grep -q .; then
        echo "Role assignment already exists, skipping."
    else
        az role assignment create \
            --assignee "${AZURE_RP_CLIENT_ID}" \
            --role "${COGNITIVE_SERVICES_OPENAI_USER_ROLE}" \
            --scope "${aoai_resource_id}" >/dev/null
        echo "Cognitive Services OpenAI User role assigned to ${AZURE_RP_CLIENT_ID}."
    fi
}

update_secrets_env() {
    echo "########## Updating secrets/env with Holmes Azure OpenAI endpoint ##########"

    # Ensure we're working from repo root to handle relative paths correctly
    cd "$(git rev-parse --show-toplevel)"

    local api_base
    api_base=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "properties.endpoint" -o tsv)

    if [[ -z "${api_base}" ]]; then
        echo "Error: failed to retrieve endpoint for ${HOLMES_AOAI_ACCOUNT_NAME}."
        exit 1
    fi

    local secrets_file="secrets/env"

    if [[ ! -f "${secrets_file}" ]]; then
        echo "Error: ${secrets_file} not found. Run 'make secrets' first."
        exit 1
    fi

    # Remove existing Holmes lines and append new config via temp file for portability
    local tmp_file
    tmp_file=$(mktemp -p "$(dirname "${secrets_file}")")

    # Handle case where all lines might match the filter (use || true to avoid exit 1 with set -e)
    # Also remove blank lines that precede Holmes section to prevent accumulation
    grep -v -E '^export HOLMES_AZURE_API_(KEY|BASE|VERSION)=|^# Holmes Azure OpenAI|^[[:space:]]*$' \
        "${secrets_file}" > "${tmp_file}" || true

    # No API key — Entra ID tokens are acquired at runtime by the RP.
    cat >> "${tmp_file}" <<EOF
# Holmes Azure OpenAI config (Entra ID auth — no API key needed)
export HOLMES_AZURE_API_BASE=$(printf %q "${api_base}")
export HOLMES_AZURE_API_VERSION=$(printf %q "${HOLMES_API_VERSION}")
EOF

    mv "${tmp_file}" "${secrets_file}"

    echo "secrets/env updated with HOLMES_AZURE_API_BASE, HOLMES_AZURE_API_VERSION."
}

main() {
    deploy_holmes_aoai_account
    deploy_holmes_aoai_model
    assign_role
    update_secrets_env

    echo ""
    echo "########## Holmes Azure OpenAI deployment complete ##########"
    echo "Account:    ${HOLMES_AOAI_ACCOUNT_NAME}"
    echo "Deployment: ${HOLMES_AOAI_DEPLOYMENT_NAME}"
    echo "Model:      ${HOLMES_AOAI_MODEL_NAME}"
    echo "Auth:       Entra ID (local auth disabled)"
    echo ""
    echo "Run 'make secrets-update' to push updated secrets/env to shared storage."
    echo "Then 'source env' to reload configuration."
}

main "$@"
