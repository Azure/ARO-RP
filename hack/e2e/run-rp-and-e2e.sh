#!/bin/bash -e
######## Helper file to run E2e either locally or using Azure DevOps Pipelines ########

validate_rp_running() {
    echo "########## ？Checking ARO RP Status ##########"
    ELAPSED=0
    while true; do
        http_code=$(curl -k -s -o /dev/null -w '%{http_code}' https://localhost:8443/healthz/ready)
        case $http_code in
            "200")
            echo "########## ✅ ARO RP Running ##########"
            break
            ;;
            *)
            echo "Attempt $ELAPSED - local RP is NOT up. Code : $http_code, waiting"
            sleep 2
            # after 20 secs return exit 1 to not block ci
            (( ELAPSED++ ))
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

    az group deployment create \
      -g "$RESOURCEGROUP" \
      -n "databases-development-$DATABASE_NAME" \
      --template-file deploy/databases-development.json \
      --parameters \
        "databaseAccountName=$COSMOSDB_ACCOUNT" \
        "databaseName=$DATABASE_NAME" \
        >/dev/null
}

deploy_e2e_deps() {
    export RESOURCEGROUP=$CLUSTER_RESOURCEGROUP
    echo "🚀 Creating new RG: $RESOURCEGROUP and Vnet for cluster : $CLUSTER"

    echo "########## Create ARO RG : $RESOURCEGROUP ##########"
    az group create -g "$RESOURCEGROUP" -l $LOCATION >/dev/null

    echo "########## Create ARO Vnet ##########"
    az network vnet create \
      -g "$RESOURCEGROUP" \
      -n dev-vnet \
      --address-prefixes 10.0.0.0/9 >/dev/null

    echo "########## Create ARO Subnet ##########"
    for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
    az network vnet subnet create \
        -g "$RESOURCEGROUP" \
        --vnet-name dev-vnet \
        -n "$subnet" \
        --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24 \
        --service-endpoints Microsoft.ContainerRegistry >/dev/null
    done

    echo "########## Update ARO Subnet ##########"
    az network vnet subnet update \
      -g "$RESOURCEGROUP" \
      --vnet-name dev-vnet \
      -n "$CLUSTER-master" \
      --disable-private-link-service-network-policies true >/dev/null
}

register_sub() {
    echo "########## 🔑 Registering subscription ##########"
    curl -k -X PUT \
      -H 'Content-Type: application/json' \
      -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}' \
      "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0"
}

run_e2e() {
    export RESOURCEGROUP=$CLUSTER_RESOURCEGROUP
    echo "########## 🚀 Create ARO Cluster $CLUSTER ##########"
    az aro create \
      -g "$RESOURCEGROUP" \
      -n "$CLUSTER" \
      --vnet dev-vnet \
      --master-subnet "$CLUSTER-master" \
      --worker-subnet "$CLUSTER-worker"

    echo "########## CLI : ARO List ##########"
    az aro list -o table
    echo "########## CLI : ARO list-creds ##########"
    az aro list-credentials -g "$RESOURCEGROUP" -n "$CLUSTER"
    echo "########## Run E2E ##########"
    go run ./hack/kubeadminkubeconfig "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER" >$KUBECONFIG
    make e2e
    echo "########## CLI : ARO delete cluster ##########"
    az aro delete -g "$RESOURCEGROUP" -n "$CLUSTER" --yes
}

clean_e2e_db(){
    echo "########## 🧹 Deleting DB $DATABASE_NAME ##########"
    az cosmosdb sql database delete --name $DATABASE_NAME \
        --account-name $COSMOSDB_ACCOUNT \
        --resource-group $RESOURCEGROUP >/dev/null
}

clean_e2e() {
    export RESOURCEGROUP=$CLUSTER_RESOURCEGROUP
    echo "########## 🧹 Cleaning Cluster RG : $RESOURCEGROUP "
    az group delete -n $RESOURCEGROUP -y
    rm -f $KUBECONFIG
}

export CLUSTER="v4-e2e-$(git log --format=%h -n 1 HEAD)"
export CLUSTER_RESOURCEGROUP="v4-e2e-rg-$(git log --format=%h -n 1 HEAD)-$LOCATION"
export KUBECONFIG=$(pwd)/$CLUSTER.kubeconfig

echo "######################################"
echo "##### ARO V4 E2e helper sourced ######"
echo "######################################"
echo "######## Current settings : ##########"
echo
echo "LOCATION=$LOCATION"
echo "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID"
echo
echo "COSMOSDB_ACCOUNT=$COSMOSDB_ACCOUNT"
echo "DATABASE_NAME=$DATABASE_NAME"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo
echo "CLUSTER=$CLUSTER"
echo "CLUSTER_RESOURCEGROUP=$CLUSTER_RESOURCEGROUP"
echo "KUBECONFIG=$KUBECONFIG"
echo
echo "PROXY_HOSTNAME=$PROXY_HOSTNAME"
echo "######################################"

[ "$LOCATION" ] || ( echo ">> LOCATION is not set please validate your ./secrets/env"; exit 128 )
[ "$RESOURCEGROUP" ] || ( echo ">> RESOURCEGROUP is not set please validate your ./secrets/env"; exit 128 )
[ "$PROXY_HOSTNAME" ] || ( echo ">> PROXY_HOSTNAME is not set please validate your ./secrets/env"; exit 128 )
[ "$COSMOSDB_ACCOUNT" ] || ( echo ">> COSMOSDB_ACCOUNT is not set please validate your ./secrets/env"; exit 128 )
[ "$DATABASE_NAME" ] || ( echo ">> DATABASE_NAME is not set please validate your ./secrets/env"; exit 128 )
[ "$AZURE_SUBSCRIPTION_ID" ] || ( echo ">> AZURE_SUBSCRIPTION_ID is not set please validate your ./secrets/env"; exit 128 )

az account set -s $AZURE_SUBSCRIPTION_ID >/dev/null
