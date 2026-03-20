#!/bin/bash

if [ -z "$RESOURCEGROUP" ]; then
    echo "the environment variable RESOURCEGROUP is required" >&2
    exit 1
fi

[ -f aks.kubeconfig ] && rm -f aks.kubeconfig

if az aks get-credentials --admin -g "$RESOURCEGROUP" -n aro-aks-cluster-001 --public-fqdn -f aks.kubeconfig 2>/dev/null; then
    chmod 600 aks.kubeconfig
else
    echo "Error generating AKS kubeconfig"
    exit 1
fi
