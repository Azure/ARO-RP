#!/bin/bash

set -euxo pipefail

ACR_NAME="aropocsvc.azurecr.io"
CLUSTER_NAME="aro-rp-aks-poc"
DOCKERFILE="Dockerfile.aro-poc"
REGISTRY="REGISTRY=registry.access.redhat.com"
RESOURCE_GROUP="rp-aks-poc"
SUB="0cc1cafa-578f-4fa5-8d6b-ddfd8d82e6ea"

azure_login() {
    accountShow=$(az account show)
    if [ -z "$accountShow" ]; then
        az login
    fi
    az account set --subscription "$SUB"
}

build_container() {
    env GOOS=linux GOARCH=amd64 go build -o aro ./cmd/aro
    docker build --file "$DOCKERFILE" --tag "$DOCKERTAG" --build-arg="$REGISTRY" .
}

build_pkg() {
    helm package poc/pkg
}

clean() {
    rm -f ./aro > /dev/null 2>&1
    docker rmi $DOCKERTAG:latest > /dev/null 2>&1
    rm -f ./pkg-*.tgz > /dev/null 2>&1
}

deploy_pkg() {
    local namespace="$1-dev"
    local release="$1-dev"

    helm_list=$(helm list -q -f $release --namespace $namespace)
    if [[ $helm_list == *"$release"* ]]; then
        helm uninstall $release --namespace $namespace
    fi

    helm install $release ./pkg-0.1.0.tgz \
                --set image.repository=$DOCKERTAG \
                --set image.tag=latest \
                --namespace $namespace \
                --create-namespace
}

get_kubeconfig() {
    az aks get-credentials --resource-group "$RESOURCE_GROUP" --name "$CLUSTER_NAME" --admin
}

push_container() {
    az acr login --name "$ACR_NAME"
    docker push "$DOCKERTAG:latest"
}

# Begin script execution
alias="$1"
if [ -z "$alias" ]; then
    echo "Usage: $0 <alias>"
    exit 1
fi

DOCKERTAG="$ACR_NAME/dev/$alias"

# Build stage
build_container
build_pkg

# Deploy stage
azure_login
push_container
get_kubeconfig
deploy_pkg $alias

clean