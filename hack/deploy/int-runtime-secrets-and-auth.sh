#!/bin/bash -e

upload-kv-secrets() {
    echo "########## Adding access polices to -svc keyvault to be able to upload secrets ##########"
    az keyvault set-policy -n "$KEYVAULT_PREFIX-svc" --spn $DEVOPS_CLIENT_ID -g $RESOURCEGROUP --secret-permissions get set --certificate-permissions get import

    echo "########## Uploading required runtime secrets to -svc keyvault ##########"
    if $(az keyvault secret show  --vault-name "$KEYVAULT_PREFIX-svc" -n encryption-key >/dev/null) ; then
    echo "## Encryption key found; skipping uploading"
    else
    echo "## Encryption key not found; uploading"
    az keyvault secret set \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name encryption-key \
        --value $ENCRYPTION_KEY >/dev/null
    echo "## Done"
    fi

    if $(az keyvault secret show  --vault-name "$KEYVAULT_PREFIX-svc" -n rp-firstparty >/dev/null) ; then
    echo "## First party client Cert found; skipping uploading"
    else
    echo "## First party client Cert not found; uploading"
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-firstparty   \
        --file secrets/fp-client-cert.pem >/dev/null
    echo "## Done"
    fi

    if $(az keyvault secret show  --vault-name "$KEYVAULT_PREFIX-svc" -n rp-server >/dev/null) ; then
    echo "## SSL Cert found; skipping uploading"
    else
    echo "## SSL Cert not found; uploading"
    az keyvault certificate import \
        --vault-name "$KEYVAULT_PREFIX-svc" \
        --name rp-server   \
        --file secrets/ssl-cert.pem >/dev/null
    echo "## Done"
    fi
}

add-ns-record-parent-dns-zone() {
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