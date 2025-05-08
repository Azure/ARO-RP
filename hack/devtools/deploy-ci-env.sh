#!/bin/bash -e
######## Helper file to set up CI Environment ########

create_infra_rg() {
    echo "########## Creating RG $RESOURCEGROUP in $LOCATION ##########"
    az group create -g "$RESOURCEGROUP" -l "$LOCATION" --tags persist=true >/dev/null
}

deploy_aro_ci_acr() {
    echo "########## Creating CI ACR in RG $RESOURCEGROUP ##########"
    az deployment group create \
        --name aro-ci-acr \
        --resource-group $RESOURCEGROUP \
        --template-file pkg/deploy/assets/ci-development.json
}

echo "##########################################"
echo "##### ARO V4 CI Env helper sourced ######"
echo "##########################################"
echo "########## Current settings : ############"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo
echo "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID"
echo
echo "LOCATION=$LOCATION"
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; exit 128 )
[ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
