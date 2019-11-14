#!/bin/bash

if [[ "$#" -ne 1 ]]; then
    echo "usage: $0 resourceid" >&2
    exit 1
fi

go run ./hack/db "$1" | jq -r .openShiftCluster.properties.adminKubeconfig | base64 -d >admin.kubeconfig
