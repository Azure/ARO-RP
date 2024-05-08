#!/bin/bash
set -e

echo "Creating Resource Group: $RESOURCEGROUP"
az group create \
--name $RESOURCEGROUP \
--location $LOCATION

echo "Creating VNET"
az network vnet create \
--resource-group $RESOURCEGROUP \
--name aro-vnet \
--address-prefixes 10.0.0.0/22

echo "Creating Master Subnet"
az network vnet subnet create \
--resource-group $RESOURCEGROUP \
--vnet-name aro-vnet \
--name master-subnet \
--address-prefixes 10.0.0.0/23

echo "Creating Worker Subnet"
az network vnet subnet create \
--resource-group $RESOURCEGROUP \
--vnet-name aro-vnet \
--name worker-subnet \
--address-prefixes 10.0.2.0/23

echo "Creating a Public API visibility cluster with version $VERSION"
az aro create \
--resource-group $RESOURCEGROUP \
--name $CLUSTER \
--vnet aro-vnet \
--master-subnet master-subnet \
--worker-subnet worker-subnet \
--version $VERSION
