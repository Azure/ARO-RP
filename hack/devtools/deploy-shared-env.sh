#!/bin/bash -e
######## Helper file to run E2e either locally or using Azure DevOps Pipelines ########

create_infra_rg() {
    echo "########## Creating RG $RESOURCEGROUP in $LOCATION ##########"
    az group create -g "$RESOURCEGROUP" -l "$LOCATION" --tags persist=true >/dev/null
}

deploy_rp_dev_predeploy() {
    echo "########## Deploying rp-development-predeploy in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n rp-development-predeploy \
        --template-file pkg/deploy/assets/rp-development-predeploy.json \
        --parameters \
            "adminObjectId=$ADMIN_OBJECT_ID" \
            "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)" \
            "keyvaultPrefix=$KEYVAULT_PREFIX" \
            "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_RP_CLIENT_ID'" --query '[].id' -o tsv)" >/dev/null
}

deploy_rp_dev() {
    echo "########## Deploying rp-development in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n rp-development \
        --template-file pkg/deploy/assets/rp-development.json \
        --parameters \
            "clusterParentDomainName=$PARENT_DOMAIN_NAME" \
            "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
            "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)" \
            "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_RP_CLIENT_ID'" --query '[].id' -o tsv)" \
            "globalDevopsServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_DEVOPS_ID'" --query '[].id' -o tsv)" >/dev/null
}

deploy_env_dev() {
    echo "########## Deploying env-development in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n env-development \
        --template-file pkg/deploy/assets/env-development.json \
        --parameters \
            "proxyCert=$(base64 -w0 <secrets/proxy.crt)" \
            "proxyClientCert=$(base64 -w0 <secrets/proxy-client.crt)" \
            "proxyDomainNameLabel=$(cut -d. -f2 <<<$PROXY_HOSTNAME)" \
            "proxyImage=arointsvc.azurecr.io/proxy:latest" \
            "proxyImageAuth=$(jq -r '.auths["arointsvc.azurecr.io"].auth' <<<$PULL_SECRET)" \
            "proxyKey=$(base64 -w0 <secrets/proxy.key)" \
            "sshPublicKey=$(<secrets/proxy_id_rsa.pub)" \
            "vpnCACertificate=$(base64 -w0 <secrets/vpn-ca.crt)" >/dev/null
}

deploy_miwi_infra_dev() {
    echo "########## Deploying OIDC storage account, cluster MSI key vault, and role assignments in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n rp-development-miwi \
        --template-file pkg/deploy/assets/rp-development-miwi.json \
        --parameters \
            "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_RP_CLIENT_ID'" --query '[].id' -o tsv)" \
            "oidcStorageAccountName=$(echo $OIDC_STORAGE_ACCOUNT_NAME)" >/dev/null
    echo "########## Enabling Static Website for OIDC storage account in RG $RESOURCEGROUP ##########"
    az storage blob service-properties update --static-website true --account-name ${OIDC_STORAGE_ACCOUNT_NAME} --auth-mode login >/dev/null
}

deploy_rp_managed_identity() {
    echo "########## Deploying RP Managed Identity (for hive/aks) in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n rp-managed-identity \
        --template-file pkg/deploy/assets/rp-production-managed-identity.json
}

deploy_aks_dev() {
    echo "########## Deploying aks-development in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n aks-development \
        --template-file pkg/deploy/assets/aks-development.json \
        --parameters \
            "dnsZone=$DOMAIN_NAME" \
            "keyvaultPrefix=$KEYVAULT_PREFIX" \
            "sshRSAPublicKey=$(<secrets/proxy_id_rsa.pub)" >/dev/null
}

deploy_hive_acr_dev() {
    echo "########## Deploying Hive ACR in RG $RESOURCEGROUP ##########"
    local acr_name="${HIVE_ACR_NAME:-arolocaldev${LOCATION}}"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n hive-acr \
        --template-file pkg/deploy/assets/ci-development.json \
        --parameters "acrName=$acr_name" >/dev/null
    echo "########## Created ACR: $acr_name ##########"
}

deploy_hive_acr_cache_and_access() {
    echo "########## Deploying Hive artifact cache and AKS access in RG $RESOURCEGROUP ##########"
    local acr_name="${HIVE_ACR_NAME:-arolocaldev${LOCATION}}"
    local aks_cluster="${AKS_CLUSTER_NAME:-aro-aks-cluster-001}"
    
    if [ -z "$HIVE_PULL_USERNAME" ] || [ -z "$HIVE_PULL_PASSWORD" ]; then
        echo "ERROR: HIVE_PULL_USERNAME and HIVE_PULL_PASSWORD must be set"
        echo "See team wiki for Hive pull secret credentials"
        return 1
    fi
    
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n hive-acr-cache-and-access \
        --template-file pkg/deploy/assets/hive-acr-cache-and-access.bicep \
        --parameters \
            "acrName=$acr_name" \
            "aksClusterName=$aks_cluster" \
            "hiveRegistryUsername=$HIVE_PULL_USERNAME" \
            "hiveRegistryPassword=$HIVE_PULL_PASSWORD" >/dev/null
    echo "########## Hive artifact cache and AKS access configured for $acr_name ##########"
}

deploy_vpn_for_dedicated_rp() {
    echo "########## Deploying Dev VPN in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n dev-vpn \
        --template-file pkg/deploy/assets/vpn-development.json \
        --parameters \
             "vpnCACertificate=$(base64 -w0 <secrets/vpn-ca.crt)" >/dev/null
}

deploy_miwi_infra_for_dedicated_rp() {
    echo "########## Deploying OIDC storage account, cluster MSI key vault, and role assignments in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n rp-development-miwi \
        --template-file pkg/deploy/assets/rp-development-miwi.json \
        --parameters \
            "rpServicePrincipalId=$(az identity show -g $RESOURCEGROUP -n aro-rp-$LOCATION | jq -r '.["principalId"]')" \
            "oidcStorageAccountName=$(yq '.rps[].configuration.oidcStorageAccountName' dev-config.yaml)" >/dev/null
    echo "########## Enabling Static Website for OIDC storage account in RG $RESOURCEGROUP ##########"
    az storage blob service-properties update --static-website true --account-name $(yq '.rps[].configuration.oidcStorageAccountName' dev-config.yaml) --auth-mode login >/dev/null
}

deploy_env_dev_override() {
    echo "########## Deploying env-development in RG $RESOURCEGROUP ##########"
    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n env-development \
        --template-file pkg/deploy/assets/env-development.json \
        --parameters \
            "proxyCert=$(base64 -w0 <secrets/proxy.crt)" \
            "proxyClientCert=$(base64 -w0 <secrets/proxy-client.crt)" \
            "proxyDomainNameLabel=$(cut -d. -f2 <<<$PROXY_HOSTNAME)" \
            "proxyImage=arointsvc.azurecr.io/proxy:latest" \
            "proxyImageAuth=$(jq -r '.auths["arointsvc.azurecr.io"].auth' <<<$PULL_SECRET)" \
            "proxyKey=$(base64 -w0 <secrets/proxy.key)" \
            "sshPublicKey=$(<secrets/proxy_id_rsa.pub)" \
            "vpnCACertificate=$(base64 -w0 <secrets/vpn-ca.crt)" \
            "publicIPAddressSkuName=Basic" \
            "publicIPAddressAllocationMethod=Dynamic" >/dev/null
}

import_certs_secrets() {
    echo "########## Import certificates to $KEYVAULT_PREFIX-svc KV ##########"
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-server \
        --file secrets/localhost.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name dev-arm \
        --file secrets/arm.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-server \
        --file secrets/localhost.pem >/dev/null
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-client \
        --file secrets/portal-client.pem >/dev/null
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --query '[].name' \
        -o tsv | grep -q ^encryption-key$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name encryption-key \
        --value "$(openssl rand -base64 32)" >/dev/null
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --query '[].name' \
        -o tsv | grep -q ^encryption-key-v2$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name encryption-key-v2 \
        --value "$(openssl rand -base64 64)"
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --query '[].name' \
        -o tsv | grep -q ^fe-encryption-key$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name fe-encryption-key \
        --value "$(openssl rand -base64 32)" >/dev/null
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --query '[].name' \
        -o tsv | grep -q ^fe-encryption-key-v2$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name fe-encryption-key-v2 \
        --value "$(openssl rand -base64 64)"
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --query '[].name' \
        -o tsv | grep -q ^portal-session-key$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-session-key \
        --value "$(openssl rand -base64 32)" >/dev/null
    az keyvault secret list \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --query '[].name' \
        -o tsv | grep -q ^portal-sshkey$ || \
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-por" \
        --name portal-sshkey \
        --value "$(openssl genpkey -algorithm rsa -pkeyopt rsa_keygen_bits:2048 -outform der | base64 -w0)" >/dev/null
}

update_parent_domain_dns_zone() {
    CHILD_DOMAIN_PREFIX="$(cut -d. -f1 <<<$DOMAIN_NAME)"
    echo "########## Creating NS record to DNS Zone $CHILD_DOMAIN_PREFIX ##########"
    az network dns record-set ns create \
        --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
        --zone "$PARENT_DOMAIN_NAME" \
        --name "$CHILD_DOMAIN_PREFIX" >/dev/null
    for ns in $(az network dns zone show \
        --resource-group "$RESOURCEGROUP" \
        --name "$DOMAIN_NAME" \
        --query nameServers -o tsv); do
        az network dns record-set ns add-record \
          --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
          --zone "$PARENT_DOMAIN_NAME" \
          --record-set-name "$CHILD_DOMAIN_PREFIX" \
          --nsdname "$ns" >/dev/null
      done
}

vpn_configuration() {
    echo "########## VPN Configuration ##########"
    curl -so vpnclientconfiguration.zip "$(az network vnet-gateway vpn-client generate \
        -g "$RESOURCEGROUP" \
        -n dev-vpn \
        -o tsv)"
    export CLIENTCERTIFICATE="$(openssl x509 -inform der -in secrets/vpn-client.crt)"
    export PRIVATEKEY="$(openssl pkey -inform der -in secrets/vpn-client.key)"
    unzip -qc vpnclientconfiguration.zip 'OpenVPN\\vpnconfig.ovpn' \
        | envsubst \
        | grep -v '^log ' >"secrets/vpn-$LOCATION.ovpn"
    rm vpnclientconfiguration.zip
}

validate_arm_template_state() {
    ARM_TEMPLATE_STATE=$(az deployment group show -n $1 -g $RESOURCEGROUP --query properties.provisioningState -o tsv)
    if [[ $ARM_TEMPLATE_STATE == "Failed" ]]; then
        echo "##[error] Error deploying $1 $(az deployment group show -n $1 -g $RESOURCEGROUP --query properties.error.details -o tsv)"
        exit 1
    fi
}

clean_env() {
    echo "########## Deleting RG $RESOURCEGROUP in $LOCATION ##########"
    az group delete -g "$RESOURCEGROUP" -y
    az network dns record-set ns delete \
        --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
        --zone "$PARENT_DOMAIN_NAME" \
        --name "$RESOURCEGROUP"
    for ns in $(az network dns zone show \
        --resource-group "$RESOURCEGROUP" \
        --name "$DOMAIN_NAME" \
        --query nameServers -o tsv); do
        az network dns record-set ns remove-record \
          --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
          --zone "$PARENT_DOMAIN_NAME" \
          --record-set-name "$RESOURCEGROUP" \
          --nsdname "$ns"
      done
}

deploy_global_secret_storage() {
    if [ -z "${SECRET_STORAGE_RESOURCEGROUP}" ]; then
        echo "SECRET_STORAGE_RESOURCEGROUP is unset"
        return
    fi
    az deployment group create \
        --name secretstorage \
        --resource-group $SECRET_STORAGE_RESOURCEGROUP \
        --parameters storageAccounts_arosecrets_name=$SECRET_SA_ACCOUNT_NAME \
        --template-file pkg/deploy/assets/shared-rp-secret-storage.json 
}

deploy_aro_spn_keyvault() {
    az deployment group create \
        --name aroe2eprincipals \
        --resource-group global-infra \
        --parameters \
            "vaults_aro_e2e_principals_name=$VAULTS_ARO_E2E_PRINCIPALS_NAME" \
            "tenant_id=$AZURE_TENANT_ID" \
        --template-file pkg/deploy/assets/e2e-aro-spn-keyvault.json
}

deploy_aro_e2e_global_keyvault() {
    az deployment group create \
        --name aroe2eprincipals \
        --resource-group global-infra \
        --parameters \
            "vault_name=$ARO_E2E_GLOBAL_VAULT_NAME" \
            "tenant_id=$AZURE_TENANT_ID" \
        --template-file pkg/deploy/assets/e2e-global-keyvault.json

}

echo "##########################################"
echo "##### ARO V4 Dev Env helper sourced ######"
echo "##########################################"
echo "########## Current settings : ############"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID"
echo
echo "LOCATION=$LOCATION"
echo
echo "DATABASE_ACCOUNT_NAME=$DATABASE_ACCOUNT_NAME"
echo
echo "ADMIN_OBJECT_ID=$ADMIN_OBJECT_ID"
echo "AZURE_RP_CLIENT_ID=$AZURE_RP_CLIENT_ID"
echo "AZURE_FP_CLIENT_ID=$AZURE_FP_CLIENT_ID"
echo
echo "PARENT_DOMAIN_NAME=$PARENT_DOMAIN_NAME"
echo "PARENT_DOMAIN_RESOURCEGROUP=$PARENT_DOMAIN_RESOURCEGROUP"
echo
echo "KEYVAULT_PREFIX=$KEYVAULT_PREFIX"
echo
echo "PROXY_HOSTNAME=$PROXY_HOSTNAME"
echo
echo "SECRET_SA_ACCOUNT_NAME=$SECRET_SA_ACCOUNT_NAME"
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; exit 128 )
[ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
[ "$PROXY_HOSTNAME" ] || ( echo ">> PROXY_HOSTNAME is not set please validate your ./secrets/env"; exit 128 )
[ "$DATABASE_ACCOUNT_NAME" ] || ( echo ">> DATABASE_ACCOUNT_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$ADMIN_OBJECT_ID" ] || ( echo ">> ADMIN_OBJECT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$PARENT_DOMAIN_NAME" ] || ( echo ">> PARENT_DOMAIN_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$AZURE_FP_CLIENT_ID" ] || ( echo ">> AZURE_FP_CLIENT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$KEYVAULT_PREFIX" ] || ( echo ">> KEYVAULT_PREFIX is not set please validate your ./secrets/env"; exit 128 )
[ "$AZURE_RP_CLIENT_ID" ] || ( echo ">> AZURE_RP_CLIENT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$PULL_SECRET" ] || ( echo ">> PULL_SECRET is not set please validate your ./secrets/env"; exit 128 )
[ "$PARENT_DOMAIN_RESOURCEGROUP" ] || ( echo ">> PARENT_DOMAIN_RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
[ "$SECRET_SA_ACCOUNT_NAME" ] || ( echo ">> SECRET_SA_ACCOUNT_NAME is not set please validate your ./secrets/env"; exit 128 )
