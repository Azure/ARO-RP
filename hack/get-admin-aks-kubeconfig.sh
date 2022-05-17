#!/bin/bash

if [ -z "$RESOURCEGROUP" ]; then
    echo "the environment variable RESOURCEGROUP is required" >&2
    exit 1
fi

if az aks get-credentials --admin -g "$RESOURCEGROUP" -n aro-aks-cluster --public-fqdn -f aks.kubeconfig 2>/dev/null; then
    chmod 600 aks.kubeconfig
fi
