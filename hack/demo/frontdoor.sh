#!/bin/bash
# run after setting up clusters 1 and 2

location=eastus
group=bcdrfrontdoor
profile=fdprofile
endpoint=staticweb
origingroup=og
mainroute=$(cat cluster1/cluster1_route.txt)
failoverroute=$(cat cluster2/cluster2_route.txt)

az group create -l $location -g $group
az afd profile create -g $group -n $profile --sku Standard_AzureFrontDoor
az afd endpoint create -g $group --endpoint-name $endpoint --profile-name $profile --enabled-state Enabled
az afd origin-group create -g $group --origin-group-name $origingroup --profile-name $profile --probe-request-type GET --probe-protocol Http --probe-interval-in-seconds 60 --probe-path / --sample-size 4 --successful-samples-required 3 --additional-latency-in-milliseconds 50
az afd origin create -g $group --profile-name $profile --origin-group-name $origingroup --origin-name main --host-name $mainroute --origin-host-header $mainroute --priority 1 --weight 1000 --enabled-state Enabled
az afd origin create -g $group --profile-name $profile --origin-group-name $origingroup --origin-name failover --host-name $failoverroute --origin-host-header $failoverroute --priority 2 --weight 1000 --enabled-state Enabled
az afd route create -g $group --profile-name $profile --endpoint-name $endpoint --forwarding-protocol MatchRequest --route-name route --origin-group $origingroup --supported-protocols Http --link-to-default-domain Enabled

echo "curl this hostname"
az afd endpoint list -g $group --profile-name $profile -o json | jq -r ".[0].hostName"
