# Deploy production cluster

VNET_RESOURCEGROUP=$RESOURCEGROUP-vnet
CLUSTER=cluster

az group create -g "$VNET_RESOURCEGROUP" -l "$LOCATION"
az network vnet create -g "$VNET_RESOURCEGROUP" -n vnet --address-prefixes 10.0.0.0/9
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24

az aro create -g "$RESOURCEGROUP" -n "$CLUSTER" --vnet-resource-group "$VNET_RESOURCEGROUP" --vnet vnet --master-subnet "$CLUSTER-master" --worker-subnet "$CLUSTER-worker"
