#!/bin/bash -e
######## Helper file to run E2e either locally or using Azure DevOps Pipelines ########

create_infra_rg() {
    echo "########## Creating RG $RESOURCEGROUP in $LOCATION ##########"
    az group create -g "$RESOURCEGROUP" -l "$LOCATION" >/dev/null
}

deploy_rp_dev_predeploy() {
    echo "########## Deploying rp-development-predeploy in RG $RESOURCEGROUP ##########"
    az group deployment create \
        -g "$RESOURCEGROUP" \
        -n rp-development-predeploy \
        --template-file deploy/rp-development-predeploy.json \
        --parameters \
            "adminObjectId=$ADMIN_OBJECT_ID" \
            "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)" \
            "keyvaultPrefix=$KEYVAULT_PREFIX" \
            "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].objectId' -o tsv)" >/dev/null
}

deploy_rp_dev() {
    echo "########## Deploying rp-development in RG $RESOURCEGROUP ##########"
    az group deployment create \
        -g "$RESOURCEGROUP" \
        -n rp-development \
        --template-file deploy/rp-development.json \
        --parameters \
            "databaseAccountName=$COSMOSDB_ACCOUNT" \
            "domainName=$DOMAIN_NAME.$PARENT_DOMAIN_NAME" \
            "fpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].objectId' -o tsv)" \
            "rpServicePrincipalId=$(az ad sp list --filter "appId eq '$AZURE_CLIENT_ID'" --query '[].objectId' -o tsv)" >/dev/null
}

deploy_env_dev() {
    echo "########## Deploying env-development in RG $RESOURCEGROUP ##########"
    az group deployment create \
        -g "$RESOURCEGROUP" \
        -n env-development \
        --template-file deploy/env-development.json \
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

deploy_env_dev_override() {
    echo "########## Deploying env-development in RG $RESOURCEGROUP ##########"
    az group deployment create \
        -g "$RESOURCEGROUP" \
        -n env-development \
        --template-file deploy/env-development.json \
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
        --name rp-firstparty \
        --file secrets/firstparty-development.pem
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-server \
        --file secrets/localhost.pem
   az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name cluster-mdsd \
        --file secrets/cluster-logging-int.pem
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name encryption-key \
        --value "$(openssl rand -base64 32)"
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name fe-encryption-key \
        --value "$(openssl rand -base64 32)"
}

update_parent_domain_dns_zone() {
    echo "########## Creating NS record to DNS Zone $DOMAIN_NAME in $PARENT_DOMAIN_NAME | RG $PARENT_DOMAIN_RESOURCEGROUP ##########"
    az network dns record-set ns create \
        --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
        --zone "$PARENT_DOMAIN_NAME" \
        --name "$DOMAIN_NAME" >/dev/null
    for ns in $(az network dns zone show \
        --resource-group "$RESOURCEGROUP" \
        --name "$DOMAIN_NAME.$PARENT_DOMAIN_NAME" \
        --query nameServers -o tsv); do
        az network dns record-set ns add-record \
          --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
          --zone "$PARENT_DOMAIN_NAME" \
          --record-set-name "$DOMAIN_NAME" \
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
    export PRIVATEKEY="$(openssl rsa -inform der -in secrets/vpn-client.key)"
    unzip -qc vpnclientconfiguration.zip 'OpenVPN\\vpnconfig.ovpn' \
        | envsubst \
        | grep -v '^log ' >"secrets/vpn-$LOCATION.ovpn"
    rm vpnclientconfiguration.zip
}

validate_arm_template_state() {
    ARM_TEMPLATE_STATE=$(az group deployment show -n $1 -g $RESOURCEGROUP --query properties.provisioningState -o tsv)
    if [[ $ARM_TEMPLATE_STATE == "Failed" ]]; then
        echo "##[error] Error deploying $1 $(az group deployment show -n $1 -g $RESOURCEGROUP --query properties.error.details -o tsv)"
        exit 1
    fi
}

clean_env() {
    echo "########## Deleting RG $RESOURCEGROUP in $LOCATION ##########"
    az group delete -g "$RESOURCEGROUP" -y
    az network dns record-set ns delete \
        --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
        --zone "$PARENT_DOMAIN_NAME" \
        --name "$DOMAIN_NAME"
    for ns in $(az network dns zone show \
        --resource-group "$RESOURCEGROUP" \
        --name "$DOMAIN_NAME.$PARENT_DOMAIN_NAME" \
        --query nameServers -o tsv); do
        az network dns record-set ns remove-record \
          --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
          --zone "$PARENT_DOMAIN_NAME" \
          --record-set-name "$DOMAIN_NAME" \
          --nsdname "$ns"
      done
}

echo "##########################################"
echo "##### ARO V4 Dev Env helper sourced ######"
echo "##########################################"
echo "########## Current settings : ############"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo
echo "LOCATION=$LOCATION"
echo
echo "COSMOSDB_ACCOUNT=$COSMOSDB_ACCOUNT"
echo "DATABASE_NAME=$DATABASE_NAME"
echo
echo "ADMIN_OBJECT_ID=$ADMIN_OBJECT_ID"
echo "AZURE_CLIENT_ID=$AZURE_CLIENT_ID"
echo "AZURE_FP_CLIENT_ID=$AZURE_FP_CLIENT_ID"
echo
echo "DOMAIN_NAME=$DOMAIN_NAME"
echo "PARENT_DOMAIN_NAME=$PARENT_DOMAIN_NAME"
echo "PARENT_DOMAIN_RESOURCEGROUP=$PARENT_DOMAIN_RESOURCEGROUP"
echo
echo "KEYVAULT_PREFIX=$KEYVAULT_PREFIX"
echo
echo "PROXY_HOSTNAME=$PROXY_HOSTNAME"
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; exit 128 )
[ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
[ "$PROXY_HOSTNAME" ] || ( echo ">> PROXY_HOSTNAME is not set please validate your ./secrets/env"; exit 128 )
[ "$COSMOSDB_ACCOUNT" ] || ( echo ">> COSMOSDB_ACCOUNT is not set please validate your ./secrets/env"; exit 128 )
[ "$DATABASE_NAME" ] || ( echo ">> DATABASE_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$ADMIN_OBJECT_ID" ] || ( echo ">> ADMIN_OBJECT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$DOMAIN_NAME" ] || ( echo ">> DOMAIN_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$PARENT_DOMAIN_NAME" ] || ( echo ">> PARENT_DOMAIN_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$AZURE_FP_CLIENT_ID" ] || ( echo ">> AZURE_FP_CLIENT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$KEYVAULT_PREFIX" ] || ( echo ">> KEYVAULT_PREFIX is not set please validate your ./secrets/env"; exit 128 )
[ "$AZURE_CLIENT_ID" ] || ( echo ">> AZURE_CLIENT_ID is not set please validate your ./secrets/env"; exit 128 )
[ "$PULL_SECRET" ] || ( echo ">> PULL_SECRET is not set please validate your ./secrets/env"; exit 128 )
[ "$PARENT_DOMAIN_RESOURCEGROUP" ] || ( echo ">> PARENT_DOMAIN_RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
