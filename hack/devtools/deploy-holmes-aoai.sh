#!/bin/bash
set -euo pipefail

# deploy-holmes-aoai.sh - Deploy Azure OpenAI resource, model, and workload
# identity infrastructure for the Holmes investigation admin API.
#
# Investigation pods authenticate to Azure OpenAI using workload identity.
# A User-Assigned Managed Identity (UAMI) is created, assigned the
# "Cognitive Services OpenAI User" role, and linked to a Kubernetes
# ServiceAccount on the Hive AKS cluster via a federated credential.
#
# Prerequisites:
#   - az CLI logged in with Contributor + User Access Administrator roles
#   - Source env file first (provides RESOURCEGROUP, LOCATION, AZURE_RP_CLIENT_ID)
#   - aks.kubeconfig generated: make aks.kubeconfig
#   - VPN connected to the Hive AKS cluster
#
# Usage:
#   source env && source secrets/env
#   ./hack/devtools/deploy-holmes-aoai.sh

# Validate required env vars
for var in RESOURCEGROUP LOCATION AZURE_RP_CLIENT_ID; do
    if [[ -z "${!var:-}" ]]; then
        echo "Error: ${var} is not set. Source your env file first."
        exit 1
    fi
done

# Constants
HOLMES_AOAI_ACCOUNT_NAME="${RESOURCEGROUP}-holmes-aoai"
HOLMES_AOAI_DEPLOYMENT_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_NAME="gpt-5.2"
HOLMES_AOAI_MODEL_VERSION="2025-12-11"
HOLMES_AOAI_SKU="S0"
HOLMES_AOAI_DEPLOYMENT_SKU_NAME="GlobalStandard"
HOLMES_AOAI_DEPLOYMENT_SKU_CAPACITY=50
HOLMES_API_VERSION="2025-04-01-preview"
COGNITIVE_SERVICES_OPENAI_USER_ROLE="5e0bd9bd-7b93-4f28-af87-19fc36ad61bd"

HOLMES_UAMI_NAME="${RESOURCEGROUP}-holmes-investigator"
HOLMES_NAMESPACE="holmes-system"
HOLMES_SA_NAME="holmes-investigator"
HIVE_AKS_CLUSTER_NAME="${HOLMES_HIVE_AKS_CLUSTER:-aro-aks-cluster-001}"
HIVE_KUBECONFIG="${HIVE_KUBE_CONFIG_PATH:-aks.kubeconfig}"

deploy_holmes_aoai_account() {
    echo "########## Deploying Azure OpenAI account ${HOLMES_AOAI_ACCOUNT_NAME} ##########"

    if az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        &>/dev/null; then
        echo "Azure OpenAI account already exists, skipping creation."
    else
        az cognitiveservices account create \
            --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --location "${LOCATION}" \
            --kind "OpenAI" \
            --sku "${HOLMES_AOAI_SKU}" \
            --custom-domain "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --yes >/dev/null
        echo "Azure OpenAI account created."
    fi

    # Ensure custom domain is set (required for Entra ID token auth).
    local current_domain
    current_domain=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "properties.customSubDomainName" -o tsv)
    if [[ -z "${current_domain}" || "${current_domain}" == "None" ]]; then
        echo "Setting custom domain on ${HOLMES_AOAI_ACCOUNT_NAME}..."
        az cognitiveservices account update \
            --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --custom-domain "${HOLMES_AOAI_ACCOUNT_NAME}" >/dev/null
        echo "Custom domain set."
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
        echo "Model deployment already exists, skipping."
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
        echo "Model deployment created."
    fi
}

deploy_holmes_uami() {
    echo "########## Creating managed identity ${HOLMES_UAMI_NAME} ##########"

    if az identity show \
        --name "${HOLMES_UAMI_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        &>/dev/null; then
        echo "Managed identity already exists, skipping creation."
    else
        az identity create \
            --name "${HOLMES_UAMI_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --location "${LOCATION}" >/dev/null
        echo "Managed identity created."
    fi

    local uami_principal_id
    uami_principal_id=$(az identity show \
        --name "${HOLMES_UAMI_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "principalId" -o tsv)

    local aoai_resource_id
    aoai_resource_id=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "id" -o tsv)

    echo "Assigning Cognitive Services OpenAI User role to UAMI..."
    if az role assignment list \
        --assignee "${uami_principal_id}" \
        --role "${COGNITIVE_SERVICES_OPENAI_USER_ROLE}" \
        --scope "${aoai_resource_id}" \
        --query "[0].id" -o tsv 2>/dev/null | grep -q .; then
        echo "Role assignment already exists, skipping."
    else
        az role assignment create \
            --assignee-object-id "${uami_principal_id}" \
            --assignee-principal-type ServicePrincipal \
            --role "${COGNITIVE_SERVICES_OPENAI_USER_ROLE}" \
            --scope "${aoai_resource_id}" >/dev/null
        echo "Role assigned."
    fi
}

deploy_federated_credential() {
    echo "########## Creating federated identity credential ##########"

    local oidc_issuer
    oidc_issuer=$(az aks show \
        --name "${HIVE_AKS_CLUSTER_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "oidcIssuerProfile.issuerUrl" -o tsv)

    if [[ -z "${oidc_issuer}" || "${oidc_issuer}" == "None" ]]; then
        echo "Error: OIDC issuer not enabled on AKS cluster ${HIVE_AKS_CLUSTER_NAME}."
        echo "Enable it: az aks update -n ${HIVE_AKS_CLUSTER_NAME} -g ${RESOURCEGROUP} --enable-oidc-issuer --enable-workload-identity"
        exit 1
    fi

    local fedcred_name="holmes-investigator-fedcred"
    local subject="system:serviceaccount:${HOLMES_NAMESPACE}:${HOLMES_SA_NAME}"

    if az identity federated-credential show \
        --identity-name "${HOLMES_UAMI_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --name "${fedcred_name}" \
        &>/dev/null; then
        echo "Federated credential already exists, skipping."
    else
        az identity federated-credential create \
            --identity-name "${HOLMES_UAMI_NAME}" \
            --resource-group "${RESOURCEGROUP}" \
            --name "${fedcred_name}" \
            --issuer "${oidc_issuer}" \
            --subject "${subject}" \
            --audiences "api://AzureADTokenExchange" >/dev/null
        echo "Federated credential created."
    fi

    echo "  Issuer:  ${oidc_issuer}"
    echo "  Subject: ${subject}"
}

deploy_k8s_resources() {
    echo "########## Creating K8s namespace and service account on Hive AKS ##########"

    local uami_client_id
    uami_client_id=$(az identity show \
        --name "${HOLMES_UAMI_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "clientId" -o tsv)

    if ! [[ "${uami_client_id}" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
        echo "Error: UAMI client ID '${uami_client_id}' is not a valid GUID."
        exit 1
    fi

    if ! KUBECONFIG="${HIVE_KUBECONFIG}" kubectl get namespace "${HOLMES_NAMESPACE}" &>/dev/null; then
        KUBECONFIG="${HIVE_KUBECONFIG}" kubectl create namespace "${HOLMES_NAMESPACE}"
        echo "Namespace ${HOLMES_NAMESPACE} created."
    else
        echo "Namespace ${HOLMES_NAMESPACE} already exists."
    fi

    KUBECONFIG="${HIVE_KUBECONFIG}" kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${HOLMES_SA_NAME}
  namespace: ${HOLMES_NAMESPACE}
  annotations:
    azure.workload.identity/client-id: "${uami_client_id}"
EOF
    echo "ServiceAccount ${HOLMES_SA_NAME} applied with client-id ${uami_client_id}."
}

update_secrets_env() {
    echo "########## Updating secrets/env ##########"

    cd "$(git rev-parse --show-toplevel)"

    local api_base
    api_base=$(az cognitiveservices account show \
        --name "${HOLMES_AOAI_ACCOUNT_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "properties.endpoint" -o tsv)

    local uami_client_id
    uami_client_id=$(az identity show \
        --name "${HOLMES_UAMI_NAME}" \
        --resource-group "${RESOURCEGROUP}" \
        --query "clientId" -o tsv)

    if [[ -z "${api_base}" ]]; then
        echo "Error: failed to retrieve endpoint for ${HOLMES_AOAI_ACCOUNT_NAME}."
        exit 1
    fi

    local secrets_file="secrets/env"

    if [[ ! -f "${secrets_file}" ]]; then
        echo "Error: ${secrets_file} not found. Run 'make secrets' first."
        exit 1
    fi

    local tmp_file
    tmp_file=$(mktemp -p "$(dirname "${secrets_file}")")

    grep -v -E '^export HOLMES_(AZURE_API_(KEY|BASE|VERSION)|UAMI_CLIENT_ID)=|^# Holmes Azure OpenAI' \
        "${secrets_file}" > "${tmp_file}" || true

    # Remove trailing blank lines (portable for macOS/BSD)
    local size_before size_after
    while true; do
        size_before=$(wc -c < "${tmp_file}")
        sed -e '${/^[[:space:]]*$/d;}' "${tmp_file}" > "${tmp_file}.tmp" && mv "${tmp_file}.tmp" "${tmp_file}"
        size_after=$(wc -c < "${tmp_file}")
        [[ "${size_before}" -eq "${size_after}" ]] && break
    done

    cat >> "${tmp_file}" <<EOF

# Holmes Azure OpenAI config (workload identity auth)
export HOLMES_AZURE_API_BASE=$(printf %q "${api_base}")
export HOLMES_AZURE_API_VERSION=$(printf %q "${HOLMES_API_VERSION}")
export HOLMES_UAMI_CLIENT_ID=$(printf %q "${uami_client_id}")
EOF

    mv "${tmp_file}" "${secrets_file}"

    echo "secrets/env updated with HOLMES_AZURE_API_BASE, HOLMES_AZURE_API_VERSION, HOLMES_UAMI_CLIENT_ID."
}

main() {
    deploy_holmes_aoai_account
    deploy_holmes_aoai_model
    deploy_holmes_uami
    deploy_federated_credential
    deploy_k8s_resources
    update_secrets_env

    echo ""
    echo "########## Holmes deployment complete ##########"
    echo "Account:    ${HOLMES_AOAI_ACCOUNT_NAME}"
    echo "Deployment: ${HOLMES_AOAI_DEPLOYMENT_NAME}"
    echo "Model:      ${HOLMES_AOAI_MODEL_NAME}"
    echo "UAMI:       ${HOLMES_UAMI_NAME}"
    echo "Namespace:  ${HOLMES_NAMESPACE}"
    echo "SA:         ${HOLMES_SA_NAME}"
    echo "Auth:       Workload Identity (local auth disabled)"
    echo ""
    echo "Run 'make secrets-update' to push updated secrets/env to shared storage."
    echo "Then 'source env' to reload configuration."
}

main "$@"
