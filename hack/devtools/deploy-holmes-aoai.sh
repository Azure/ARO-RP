#!/bin/bash
set -euo pipefail

# deploy-holmes-aoai.sh - Deploy Azure OpenAI resource and GPT model for Holmes admin API
#
# Prerequisites:
#   - az CLI logged in with Cognitive Services Contributor role
#   - Source env file first (provides RESOURCEGROUP, LOCATION)
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

# Constants
HOLMES_AOAI_ACCOUNT_NAME="${RESOURCEGROUP}-holmes-aoai"
HOLMES_AOAI_DEPLOYMENT_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_VERSION="2025-12-11"
HOLMES_AOAI_SKU="S0"
HOLMES_AOAI_DEPLOYMENT_SKU_NAME="GlobalStandard"
HOLMES_AOAI_DEPLOYMENT_SKU_CAPACITY=10
HOLMES_API_VERSION="2025-04-01-preview"

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
            --yes >/dev/null
        echo "Azure OpenAI account ${HOLMES_AOAI_ACCOUNT_NAME} created."
    fi
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

update_secrets_env() {
    echo "########## Updating secrets/env with Holmes Azure OpenAI credentials ##########"

    local api_key
    api_key=$(az cognitiveservices account keys list \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "key1" -o tsv)

    local api_base
    api_base=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "properties.endpoint" -o tsv)

    local secrets_file="secrets/env"

    if [[ ! -f "${secrets_file}" ]]; then
        echo "Error: ${secrets_file} not found. Run 'make secrets' first."
        exit 1
    fi

    # Remove existing Holmes lines
    sed -i'' -e '/^export HOLMES_AZURE_API_KEY=/d' "${secrets_file}"
    sed -i'' -e '/^export HOLMES_AZURE_API_BASE=/d' "${secrets_file}"
    sed -i'' -e '/^export HOLMES_AZURE_API_VERSION=/d' "${secrets_file}"
    sed -i'' -e '/^# Holmes Azure OpenAI/d' "${secrets_file}"

    # Append Holmes secrets
    cat >> "${secrets_file}" <<EOF

# Holmes Azure OpenAI credentials
export HOLMES_AZURE_API_KEY='${api_key}'
export HOLMES_AZURE_API_BASE='${api_base}'
export HOLMES_AZURE_API_VERSION='${HOLMES_API_VERSION}'
EOF

    echo "secrets/env updated with HOLMES_AZURE_API_KEY, HOLMES_AZURE_API_BASE, HOLMES_AZURE_API_VERSION."
}

main() {
    deploy_holmes_aoai_account
    deploy_holmes_aoai_model
    update_secrets_env

    echo ""
    echo "########## Holmes Azure OpenAI deployment complete ##########"
    echo "Account:    ${HOLMES_AOAI_ACCOUNT_NAME}"
    echo "Deployment: ${HOLMES_AOAI_DEPLOYMENT_NAME}"
    echo "Model:      ${HOLMES_AOAI_MODEL_NAME}"
    echo ""
    echo "Run 'make secrets-update' to push updated secrets/env to shared storage."
    echo "Then 'source env' to reload configuration."
}

main "$@"
