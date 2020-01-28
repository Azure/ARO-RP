#!/bin/bash -e
#
# This script is intended to be run from the CI pipeline as follows:
#
#   # Define secrets in the envionment
#   source hack/e2e.sh
#   aro_ci_setup
#   make e2e

get_random_location() {
    array[0]="australiasoutheast"
    array[1]="eastus"
    array[2]="westeurope"

    size=${#array[@]}
    index=$(($RANDOM % $size))
    echo ${array[$index]}
}

aro_ci_setup() {
    echo "====== starting dev RP =========================="
    go build ./cmd/aro
    ./aro rp &
    trap 'return_id=$?; aro_ci_teardown_handler; exit $return_id' EXIT
    while [[ "$(curl -s -o /dev/null -w '%{http_code}' localhost:8443)" == "000" ]]; do
        sleep 2
    done

    if $CLUSTER_CREATE; then
        echo "====== setup subnets =========================="
        for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
            az network vnet subnet create \
                -g "$RESOURCEGROUP" \
                --vnet-name dev-vnet \
                -n "$subnet" \
                --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24 \
                --service-endpoints Microsoft.ContainerRegistry \
                >/dev/null
        done
        az network vnet subnet update \
            -g "$RESOURCEGROUP" \
            --vnet-name dev-vnet \
            -n "$CLUSTER-master" \
            --disable-private-link-service-network-policies true \
            >/dev/null

        echo "====== aro create =========================="
        az aro create -g "$RESOURCEGROUP" --cluster-resource-group "$CLUSTER" -n "$CLUSTER" --vnet dev-vnet --master-subnet "$CLUSTER-master" --worker-subnet "$CLUSTER-worker"
    fi
    echo "====== get admin kubeconfig =========================="
    go run ./hack/kubeadminkubeconfig "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER" >$KUBECONFIG
}

aro_ci_teardown_handler() {
    set +ex
    if $CLUSTER_DELETE; then
        echo "====== delete cluster =========================="
        az aro delete -g "$RESOURCEGROUP" -n "$CLUSTER" --yes
        rm -f $KUBECONFIG

        echo "====== delete subnets =========================="
        for subnet in "$CLUSTER-master" "$CLUSTER-worker"; do
            az network vnet subnet delete \
                -g "$RESOURCEGROUP" \
                --vnet-name dev-vnet \
                -n "$subnet" \
                >/dev/null
        done
    fi
    echo "====== kill the RP =========================="
    kill $(lsof -t -i :8443)
    wait $(lsof -t -i :8443)
}

# allow overriding these variables
if [[ -z "$CLUSTER_CREATE" ]]; then
    export CLUSTER_CREATE=true
fi
if [[ -z "$CLUSTER_DELETE" ]]; then
    export CLUSTER_DELETE=true
fi
if [[ -z "$LOCATION" ]]; then
    export LOCATION=$(get_random_location)
fi
if [[ -z "$CLUSTER" ]]; then
    export CLUSTER=v4-e2e-$(git log --format=%h -n 1 HEAD)
fi

export RESOURCEGROUP="v4-$LOCATION"
export COSMOSDB_ACCOUNT="$RESOURCEGROUP"
export DOMAIN_NAME="$RESOURCEGROUP"
export KEYVAULT_NAME="$RESOURCEGROUP"
export PROXY_HOSTNAME="vm0.aroproxy.$LOCATION.cloudapp.azure.com"
export DATABASE_NAME="e2e-$(git log --format=%h -n 1 HEAD)"
export KUBECONFIG=$(pwd)/$CLUSTER.kubeconfig

echo "LOCATION=$LOCATION"
echo "RESOURCEGROUP=$RESOURCEGROUP"
echo "CLUSTER=$CLUSTER"
if $CLUSTER_CREATE; then
    echo " > will be created"
fi
if $CLUSTER_DELETE; then
    echo " > will be deleted"
fi
echo "KUBECONFIG=$KUBECONFIG"
