#!/bin/bash -e
######## Helper file to run E2e either locally or using Azure DevOps Pipelines ########

validate_rp_running() {
    echo "########## ？Checking ARO RP Status ##########"
    ELAPSED=0
    while true; do
        http_code=$(curl -k -s -o /dev/null -w '%{http_code}' https://localhost:8443/healthz/ready || true)
        case $http_code in
            "200")
            echo "########## ✅ ARO RP Running ##########"
            break
            ;;
            *)
            echo "Attempt $ELAPSED - local RP is NOT up. Code : $http_code, waiting"
            sleep 2
            # after 20 secs return exit 1 to not block ci
            ELAPSED=$((ELAPSED+1))
            if [ $ELAPSED -eq 10 ]
            then
                exit 1
            fi
            ;;
        esac
    done
}

run_rp() {
    echo "########## 🚀 Run ARO RP in background ##########"
    ./aro rp &
}

kill_rp(){
    echo "########## Kill the RP running in background ##########"
    rppid=$(lsof -t -i :8443)
    kill $rppid
    wait $rppid
}

deploy_e2e_db() {
    echo "########## 📦 Creating new DB $DATABASE_NAME in $COSMOSDB_ACCOUNT ##########"

    az deployment group create \
      -g "$RESOURCEGROUP" \
      -n "databases-development-$DATABASE_NAME" \
      --template-file deploy/databases-development.json \
      --parameters \
        "databaseAccountName=$COSMOSDB_ACCOUNT" \
        "databaseName=$DATABASE_NAME" \
        >/dev/null

}

set_cli_context() {
    echo "########## Setting az cli context ##########"
    az account set -s $AZURE_SUBSCRIPTION_ID
}

register_sub() {
    echo "########## 🔑 Registering subscription ##########"
    curl -sko /dev/null -X PUT \
      -H 'Content-Type: application/json' \
      -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
      "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
}

run_e2e() {
    RESOURCEGROUP=$ARO_RESOURCEGROUP make test-e2e
}

clean_e2e_db(){
    echo "########## 🧹 Deleting DB $DATABASE_NAME ##########"
    az cosmosdb sql database delete --name $DATABASE_NAME \
        --yes \
        --account-name $COSMOSDB_ACCOUNT \
        --resource-group $RESOURCEGROUP >/dev/null
}

# TODO: CLUSTER and ARO_RESOURCEGROUP are also recalculated in multiple places
# in the billing pipelines :-(
export CLUSTER="v4-e2e-V$BUILD_BUILDID"
export DATABASE_NAME="v4-e2e-V$BUILD_BUILDID"
export ARO_RESOURCEGROUP="v4-e2e-rg-V$BUILD_BUILDID-$LOCATION"
export E2E_CREATE_CLUSTER=true
export E2E_DELETE_CLUSTER=true

echo "######################################"
echo "##### ARO V4 E2e helper sourced ######"
echo "######################################"
echo "######## Current settings : ##########"
echo
echo "LOCATION=$LOCATION"
echo "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID"
echo
echo "RP_MODE=$RP_MODE"
if [ "$RP_MODE" = "development" ]
then
    echo
    echo "COSMOSDB_ACCOUNT=$COSMOSDB_ACCOUNT"
    echo "DATABASE_NAME=$DATABASE_NAME"
    echo "RESOURCEGROUP=$RESOURCEGROUP"
fi
echo
echo "CLUSTER=$CLUSTER"
echo "ARO_RESOURCEGROUP=$ARO_RESOURCEGROUP"
if [ "$RP_MODE" = "development" ]
then
    echo
    echo "PROXY_HOSTNAME=$PROXY_HOSTNAME"
fi
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; exit 128 )
if [ "$RP_MODE" = "development" ]
then
    [ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set; please validate your ./secrets/env"; exit 128 )
    [ "$PROXY_HOSTNAME" ] || ( echo ">> PROXY_HOSTNAME is not set; please validate your ./secrets/env"; exit 128 )
    [ "$COSMOSDB_ACCOUNT" ] || ( echo ">> COSMOSDB_ACCOUNT is not set; please validate your ./secrets/env"; exit 128 )
    [ "$DATABASE_NAME" ] || ( echo ">> DATABASE_NAME is not set; please validate your ./secrets/env"; exit 128 )
fi
[ "$AZURE_SUBSCRIPTION_ID" ] || ( echo ">> AZURE_SUBSCRIPTION_ID is not set; please validate your ./secrets/env"; exit 128 )

az account set -s $AZURE_SUBSCRIPTION_ID >/dev/null
