#!/bin/bash -e

if [[ "$#" -ne 1 ]]; then
    echo "usage: $0 resourceid" >&2
    exit 1
fi

cleanup() {
    rm -f id_rsa
}

trap cleanup EXIT
go run ./hack/db "$1" | jq -r .openShiftCluster.properties.sshKey | base64 -d | openssl rsa -inform der -outform pem >id_rsa 2>/dev/null
chmod 0600 id_rsa

RG=$(go run ./hack/db "$1" | jq -r .openShiftCluster.properties.clusterProfile.resourceGroupId | cut -d/ -f5)

IP=$(az network nic list -g "$RG" --query "[?ends_with(name, '-bootstrap-nic')].ipConfigurations[0].privateIpAddress" -o tsv)

ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i id_rsa -l core "$IP"
