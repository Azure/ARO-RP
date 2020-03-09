#!/bin/bash
set -u

RESOURCEID=$1
CONTEXT_NAME=$2

if [[ "$#" -ne 2 ]]; then
    echo "usage: $0 resourceid context_name" >&2
    exit 1
fi

if [[ "$CONTEXT_NAME" == "admin" ]]; then
  KEY="adminKubeconfig"
elif [[ "$CONTEXT_NAME" == "aro-service" ]]; then
  KEY="aroServiceKubeconfig"
else
  echo "usage: context name must be one of admin or aro-service"
  exit 1
fi

go run ./hack/db ${RESOURCEID} | jq -r .openShiftCluster.properties.${KEY} | base64 -d | sed -e 's|https://api-int\.|https://api\.|'
