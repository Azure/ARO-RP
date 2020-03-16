#!/bin/bash

if [[ "$#" -ne 1 ]]; then
    echo "usage: $0 resourceid" >&2
    exit 1
fi

RID="/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/${1}"

go run ./hack/db "${RID}" | jq -r .openShiftCluster.properties.adminKubeconfig | base64 -d | sed -e 's|https://api-int\.|https://api\.|'
