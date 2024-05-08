#!/bin/bash
echo "Deleting the cluster."
az aro delete \
--resource-group $RESOURCEGROUP \
--name $CLUSTER \
--yes

echo "Deleting the resource group."
az group delete \
--name $RESOURCEGROUP \
--yes
