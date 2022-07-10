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
            # after 40 secs return exit 1 to not block ci
            ELAPSED=$((ELAPSED+1))
            if [ $ELAPSED -eq 20 ]
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
    echo "########## 📦 Creating new DB $DATABASE_NAME in $DATABASE_ACCOUNT_NAME ##########"

    az deployment group create \
      -g "$RESOURCEGROUP" \
      -n "databases-development-$DATABASE_NAME" \
      --template-file deploy/databases-development.json \
      --parameters \
        "databaseAccountName=$DATABASE_ACCOUNT_NAME" \
        "databaseName=$DATABASE_NAME" \
        >/dev/null

}

register_sub() {
    echo "########## 🔑 Registering subscription ##########"
    curl -sko /dev/null -X PUT \
      -H 'Content-Type: application/json' \
      -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
      "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
}

clean_e2e_db(){
    echo "########## 🧹 Deleting DB $DATABASE_NAME ##########"
    az cosmosdb sql database delete --name $DATABASE_NAME \
        --yes \
        --account-name $DATABASE_ACCOUNT_NAME \
        --resource-group $RESOURCEGROUP >/dev/null
}

# TODO: CLUSTER and is also recalculated in multiple places
# in the billing pipelines :-(


# if LOCAL_E2E is set, set the value with the local test names
# If it it not set, it defaults to the build ID 
if [ -z "${LOCAL_E2E}" ] ; then 
    export CLUSTER="v4-e2e-V$BUILD_BUILDID-$LOCATION"
    export DATABASE_NAME="v4-e2e-V$BUILD_BUILDID-$LOCATION"
fi

if [ -z "${CLUSTER}" ] ; then
    echo "CLUSTER is not set, aborting"
    return 1
fi

if [ -z "${DATABASE_NAME}" ] ; then
    echo "DATABASE_NAME is not set, aborting"
    return 1
fi

echo "######################################"
echo "##### ARO V4 E2e helper sourced ######"
echo "######################################"
echo "######## Current settings : ##########"
echo
echo "LOCATION=$LOCATION"
echo "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID"
echo
echo "RP_MODE=$RP_MODE"
echo
echo "DATABASE_ACCOUNT_NAME=$DATABASE_ACCOUNT_NAME"
echo "DATABASE_NAME=$DATABASE_NAME"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo
echo "CLUSTER=$CLUSTER"
echo
echo "PROXY_HOSTNAME=$PROXY_HOSTNAME"
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; return 128 )
[ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set; please validate your ./secrets/env"; return 128 )
[ "$PROXY_HOSTNAME" ] || ( echo ">> PROXY_HOSTNAME is not set; please validate your ./secrets/env"; return 128 )
[ "$DATABASE_ACCOUNT_NAME" ] || ( echo ">> DATABASE_ACCOUNT_NAME is not set; please validate your ./secrets/env"; return 128 )
[ "$DATABASE_NAME" ] || ( echo ">> DATABASE_NAME is not set; please validate your ./secrets/env"; return 128 )
[ "$AZURE_SUBSCRIPTION_ID" ] || ( echo ">> AZURE_SUBSCRIPTION_ID is not set; please validate your ./secrets/env"; return 128 )

az account set -s $AZURE_SUBSCRIPTION_ID >/dev/null
