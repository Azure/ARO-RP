#!/bin/bash

location=eastus2
group=bcdr2
cluster=bcdrfailover
vnet=aro-vnet
kubeconfig=$cluster.kubeconfig

az group create -l $location -g $group
az network vnet create -g $group -n $vnet --address-prefixes 10.0.0.0/22
az network vnet subnet create -g $group --vnet-name $vnet --name master-subnet --address-prefixes 10.0.0.0/23
az network vnet subnet create -g $group --vnet-name $vnet --name worker-subnet --address-prefixes 10.0.2.0/23
az aro create -g $group -n $cluster --vnet $vnet --master-subnet master-subnet --worker-subnet worker-subnet --master-vm-size Standard_D8s_v3 --worker-vm-size Standard_D8s_v3 --version 4.18.26

# wait for cluster to complete
az aro get-admin-kubeconfig -g $group -n $cluster -f $cluster.kubeconfig
export KUBECONFIG=$kubeconfig

oc create namespace staticweb
oc project staticweb
oc create deployment nginx --image=nginxinc/nginx-unprivileged:latest -r 2 --port 8080
pod=$(oc get pod -l app=nginx -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | head -1)
oc wait --for=condition=Ready pod/$pod
oc set volume deploy/nginx --add --name=staticweb --type=pvc --claim-size=1M --mount-path=/usr/share/nginx/html/
pod=$(oc get pod --sort-by .metadata.creationTimestamp -l app=nginx -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | tail -1)
oc wait --for=condition=Ready pod/$pod
echo "welcome to cluster2" > index.html
oc cp ./index.html $pod:/usr/share/nginx/html/ 
rm ./index.html
oc create service clusterip nginx --tcp=8080:8080
oc expose service nginx
oc get route nginx -o jsonpath='{.status.ingress[0].host}{"\n"}' > cluster2_route.txt
