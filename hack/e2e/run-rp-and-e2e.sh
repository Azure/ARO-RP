#!/bin/bash -e
######## Helper file to run E2e either locally or using Azure DevOps Pipelines ########

if [[ $CI ]]; then
    set -o pipefail

    . secrets/env
    echo "##vso[task.setvariable variable=RP_MODE]$RP_MODE"

    set -a
    HIVEKUBECONFIGPATH="secrets/e2e-aks-kubeconfig"
    HIVE_KUBE_CONFIG_PATH_1="secrets/aks.kubeconfig"
    CLUSTER="v4-e2e-V$BUILD_BUILDID-$LOCATION"
    DATABASE_NAME="v4-e2e-V$BUILD_BUILDID-$LOCATION"
    PRIVATE_CLUSTER=true
    E2E_DELETE_CLUSTER=false
    set +a
fi

validate_rp_running() {
    echo "########## ？Checking ARO RP Status ##########"
    ELAPSED=0
    while true; do
        sleep 5
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
            ELAPSED=$((ELAPSED + 1))
            if [ $ELAPSED -eq 20 ]; then
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

kill_rp() {
    echo "########## Kill the RP running in background ##########"
    rppid=$(lsof -t -i :8443)
    kill $rppid
    wait $rppid
}

validate_portal_running() {
    echo "########## ？Checking Admin Portal Status ##########"
    ELAPSED=0
    while true; do
        sleep 5
        http_code=$(curl -k -s -o /dev/null -w '%{http_code}' https://localhost:8444/api/info)
        case $http_code in
        "403")
            echo "########## ✅ ARO Admin Portal Running ##########"
            break
            ;;
        *)
            echo "Attempt $ELAPSED - local Admin Portal is NOT up. Code : $http_code, waiting"
            sleep 2
            # after 40 secs return exit 1 to not block ci
            ELAPSED=$((ELAPSED + 1))
            if [ $ELAPSED -eq 20 ]; then
                exit 1
            fi
            ;;
        esac
    done
}

run_portal() {
    echo "########## 🚀 Run Admin Portal in background ##########"
    export AZURE_ENVIRONMENT=AzurePublicCloud
    ./aro portal &
}

kill_portal() {
    echo "########## Kill the Admin Portal running in background ##########"
    rppid=$(lsof -t -i :8444)
    kill $rppid
    wait $rppid
}

run_vpn() {
    echo "########## 🚀 Run OpenVPN in background ##########"
    echo "Using Secret secrets/$VPN"
    sudo openvpn --config secrets/$VPN --daemon --writepid vpnpid
    sleep 10
}

kill_vpn() {
    echo "########## Kill the OpenVPN running in background ##########"
    while read pid; do sudo kill $pid; done <vpnpid
}

deploy_e2e_db() {
    echo "########## 📦 Creating new DB $DATABASE_NAME in $DATABASE_ACCOUNT_NAME ##########"

    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n "databases-development-$DATABASE_NAME" \
        --template-file pkg/deploy/assets/databases-development.json \
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

clean_e2e_db() {
    echo "########## 🧹 Deleting DB $DATABASE_NAME ##########"
    az cosmosdb sql database delete --name $DATABASE_NAME \
        --yes \
        --account-name $DATABASE_ACCOUNT_NAME \
        --resource-group $RESOURCEGROUP >/dev/null
}

delete_e2e_cluster() {
    echo "########## 🧹 Deleting Cluster $CLUSTER ##########"
    if [[ $CI ]]; then
        ./cluster delete
    else
        go run ./hack/cluster delete
    fi
}

# TODO: CLUSTER and is also recalculated in multiple places
# in the billing pipelines :-(

if [[ -z $CLUSTER ]]; then
    echo "CLUSTER is not set, aborting"
    return 1
fi

if [[ -z $DATABASE_NAME ]]; then
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

[[ $LOCATION ]] || (
    echo ">> LOCATION is not set please validate your ./secrets/env"
    return 128
)
[[ $RESOURCEGROUP ]] || (
    echo ">> RESOURCEGROUP is not set; please validate your ./secrets/env"
    return 128
)
[[ $PROXY_HOSTNAME ]] || (
    echo ">> PROXY_HOSTNAME is not set; please validate your ./secrets/env"
    return 128
)
[[ $DATABASE_ACCOUNT_NAME ]] || (
    echo ">> DATABASE_ACCOUNT_NAME is not set; please validate your ./secrets/env"
    return 128
)
[[ $DATABASE_NAME ]] || (
    echo ">> DATABASE_NAME is not set; please validate your ./secrets/env"
    return 128
)
[[ $AZURE_SUBSCRIPTION_ID ]] || (
    echo ">> AZURE_SUBSCRIPTION_ID is not set; please validate your ./secrets/env"
    return 128
)
