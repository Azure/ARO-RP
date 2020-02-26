#!/bin/bash -e

create-rg() {
    echo "########## Creating RG $RESOURCEGROUP in $LOCATION ##########"
    az group create -g "$RESOURCEGROUP" -l "$LOCATION" >/dev/null
    echo "## Done"
}

deploy-int-nsg() {
    az group deployment create -n rp-nsg -g $RESOURCEGROUP --template-file deploy/rp-production-nsg.json >/dev/null
}

deploy-int-rp() {
    RP_SPN_ID=$(az group deployment show -n rp-nsg -g $RESOURCEGROUP --query properties.outputs.rpServicePrincipalId.value -o tsv)

    echo "########## Deploying rp-production in RG $RESOURCEGROUP ##########"
    echo "### domainName            = $DOMAIN_NAME.$PARENT_DOMAIN_NAME"
    echo "### keyvaultPrefix        = $KEYVAULT_PREFIX"
    echo "### rpServicePrincipalId  = $RP_SPN_ID"
    az group deployment create \
    -g "$RESOURCEGROUP" \
    -n rp-int \
    --template-file deploy/rp-production.json \
    --parameters @secrets/parameters.json \
    --parameters "rpServicePrincipalId=$RP_SPN_ID" \
    --parameters "pullSecret=$PULL_SECRET" \
    --parameters "sshPublicKey=$SSH_PUBLIC_KEY"  >/dev/null
    ARM_TEMPLATE_STATE=$(az group deployment show -n rp-int -g $RESOURCEGROUP --query properties.provisioningState -o tsv)
    if [[ $ARM_TEMPLATE_STATE == "Failed" ]]; then
    echo "##[error] Error deploying env-development $(az group deployment show -n rp-int -g $RESOURCEGROUP --query properties.error.details -o tsv)"
    exit 1
    fi
    echo "## Done"
}