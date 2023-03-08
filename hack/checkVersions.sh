#!/bin/bash


regions="westcentralus australiacentral australiacentral2 australiaeast australiasoutheast centralindia eastasia southindia"
regions="$regions japaneast japanwest koreacentral centralus eastus eastus2 northcentralus southcentralus westus westus2 westus3 canadacentral"
regions="$regions canadaeast germanywestcentral northeurope norwayeast norwaywest swedencentral switzerlandnorth switzerlandwest westeurope francecentral "
regions="$regions brazilsouth brazilsoutheast southeastasia southafricanorth uaenorth uksouth ukwest"


printf "%-23s %10s %10s\n" "Region" "4.10 OK?" "4.11 OK?"
echo  "---------------------------------------------"
for reg in $regions; do

    v410present="false"
    v411present="false"
    out=`az aro get-versions -l $reg `    
    if [[  "$out" == *"4.10.40"* ]]; then
           v410present="true"
    fi
    if [[  "$out" == *"4.11.26"* ]]; then
           v411present="true"
    fi
    printf "%-20s %10s %10s\n" $reg $v410present $v411present
done