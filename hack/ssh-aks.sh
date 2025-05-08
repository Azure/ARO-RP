#!/bin/bash -e

# ssh-aks.sh allows you to ssh into the specifed AKS clusters worker nodes

usage() {
	echo -e "usage: ${0} <cluster-name> <worker-node-index>\n" >&2
	echo "       Examples: ${0} aks-test 0        # the first VM node in 'aks-test'" >&2
	echo "                 ${0} aro-aks-cluster 1 # The second VM node in 'aro-aks-cluster'" >&2
	exit 1
}

if [ "$#" -ne 2 ]; then
	usage
fi

if [ -z "$RESOURCEGROUP" ]; then
	echo "RESOURCEGROUP env variable must be set"
	exit 1
fi

NODE_RESOURCEGROUP=$( az aks list -g "$RESOURCEGROUP" --query "[?contains(name, '$1')].nodeResourceGroup" -o tsv )
if [[ $(grep -c . <<<"$NODE_RESOURCEGROUP") -ne 1 ]]; then
	echo "Cluster with pattern "${1}" not found in resource group ${RESOURCEGROUP}"
	echo "The following AKS clusters are available:"
	az aks list -g $RESOURCEGROUP -otable
	exit 1
fi

echo "Node RG:   ${NODE_RESOURCEGROUP}"

NODE_VMSS=$( az vmss list -g "$NODE_RESOURCEGROUP" --query "[?contains(name, 'system')].name" -otsv )
if [[ $(grep -c . <<<"$NODE_VMSS") -ne 1 ]]; then
	echo "VMSS not found in the node resource group ${NODE_RESOURCEGROUP}"
	usage
fi

echo "Node VMSS: ${NODE_VMSS}"

IPs=$( az vmss nic list -g "$NODE_RESOURCEGROUP" --vmss-name "$NODE_VMSS" --query '[?items[].ipConfigurations[].primary == true].ipConfigurations[0].privateIpAddress' )
if [ -z "$IPs" ]; then
	echo "Primary IPs not found for VMs in ${NODE_VMSS}"
	usage
fi

IP_COUNT=$( echo ${IPs} | jq -r ". | length" )
echo "Found ${IP_COUNT} worker node IPs"

IP=$( echo "$IPs" | jq -r ".[${2}]" )
if [ -z "$IP" ]; then
	echo "IP not found for worker index ${2}"
	usage
fi

ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i secrets/proxy_id_rsa -l cloud-user "$IP"
