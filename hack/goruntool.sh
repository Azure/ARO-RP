#!/bin/bash

set -e

declare -A packages=(
    ["goimports"]="golang.org/x/tools/cmd/goimports@v0.23.0"
    ["gotestsum"]="gotest.tools/gotestsum@v1.11.0"
    ["enumer"]="github.com/alvaroloes/enumer@v1.1.2"
    ["mockgen"]="github.com/golang/mock/mockgen@v1.6.0"
    ["gocosmosdb"]="github.com/jewzaam/go-cosmosdb/cmd/gencosmosdb@v0.0.0-20240723075448-058185e3c66d"
    ["controller-gen"]="sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0"
    ["client-gen"]="k8s.io/code-generator/cmd/client-gen@v0.25.16"
    ["go-bindata"]="github.com/go-bindata/go-bindata/go-bindata@v3.1.2+incompatible"
    ["gocov-xml"]="github.com/AlekSi/gocov-xml@v1.1.0"
)

# shift off the first argument that we will use
COMMAND=$1
shift

if [ "$COMMAND" == "install" ]; then
    echo "installing go tools..."

    for package in "${!packages[@]}"; do
        echo "installing $package - ${packages[$package]}"
        go install "${packages[$package]}"
    done
    exit 0
fi

PACKAGE="${packages[$COMMAND]}"

if [ "$PACKAGE" != "" ]; then
    go run "$PACKAGE" "$@"
else
    echo "$COMMAND - not a command that we know about, sorry"
    exit 1
fi
