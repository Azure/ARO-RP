# Deploy production cluster

export VNET_RESOURCEGROUP=$RESOURCEGROUP-vnet

export CLUSTER=cluster

az group create -g "$VNET_RESOURCEGROUP" -l "$LOCATION"
az network vnet create -g "$VNET_RESOURCEGROUP" -n vnet --address-prefixes 10.0.0.0/9
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24

az role assignment create --role "ARO v4 Development Subnet Contributor" --assignee-object-id "$(az ad sp list --all --query "[?appId=='f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875'].objectId" -o tsv)" --scope "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$VNET_RESOURCEGROUP/providers/Microsoft.Network/virtualNetworks/vnet"

az aro create --resource-group $RESOURCEGROUP --name $CLUSTER --client-id $AZURE_CLUSTER_CLIENT_ID --client-secret $AZURE_CLUSTER_CLIENT_SECRET --vnet-rg-name $VNET_RESOURCEGROUP
