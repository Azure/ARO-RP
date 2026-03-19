#!/bin/bash

if [[ "$#" -ne 1 ]]; then
    echo "usage: $0 resourceid" >&2
    exit 1
fi

if [[ $CI ]]; then
    # Build db binary if it doesn't exist
    [ ! -f ./db ] && go build ./hack/db
    ./db "$1" | jq -r .openShiftCluster.properties.adminKubeconfig | base64 -d | sed -e 's|https://api-int\.|https://api\.|'
else
    go run ./hack/db "$1" | jq -r .openShiftCluster.properties.adminKubeconfig | base64 -d | sed -e 's|https://api-int\.|https://api\.|'
fi
