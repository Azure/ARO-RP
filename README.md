# github.com/jim-minter/rp

## Install

1. Install the following:

   * go 1.12 or later
   * az client

1. Log in to Azure:

   ```
   az login
   ```

1. You will need a publicly resolvable **DNS zone** resource in your Azure
   subscription.  *RH ARO engineering*: use the `osadev.cloud` zone in the `dns`
   resource group.

1. You will need an **ARM AAD application** with client secret authentication
   enabled.  *RH ARO engineering*: use `aro-v4-arm-shared` in the shared
   `secrets/env` file.

1. You will need an **RP AAD application** with client secret authentication
   enabled.  *RH ARO engineering*: use `aro-v4-rp-shared` in the shared
   `secrets/env` file.

1. You will need a **"first party" AAD application** with client certificate
   authentication enabled.  A suitable key/certificate file can be generated
   using the following helper utility:

   ```
   # Non-RH ARO engineering only
   go run ./hack/genkey -extKeyUsage client "$RP_AAD_APPLICATION_NAME"
   ```

   *RH ARO engineering*: use the `aro-v4-fp-shared` AAD application in the
   shared `secrets/env` and `secrets/aro-v4-fp-shared.pem` files.

1. You will need to set up the **RP role definitions and assignments** in your
   Azure subscription.  This mimics the RBAC that ARM sets up.  With at least
   `User Access Administrator` permissions on your subscription, do:

   ```
   # Non-RH ARO engineering only
   ARM_SERVICEPRINCIPAL_ID=$(az ad sp list --all --query "[?appId=='$AZURE_ARM_CLIENT_ID'].objectId" -o tsv)
   FP_SERVICEPRINCIPAL_ID=$(az ad sp list --all --query "[?appId=='$AZURE_FP_CLIENT_ID'].objectId" -o tsv)

   az deployment create -l eastus --template-file deploy/rbac-development.json --parameters "armServicePrincipalId=$ARM_SERVICEPRINCIPAL_ID" "fpServicePrincipalId=$FP_SERVICEPRINCIPAL_ID"
   ```

   *RH ARO engineering*: the above step has already been done.

1. You will need an RP serving key/certificate.  A suitable key/certificate file
   can be generated using the following helper utility:

   ```
   # Non-RH ARO engineering only
   go run ./hack/genkey localhost
   ```

   *RH ARO engineering*: use the `localhost` key and certificate in the shared
   `secrets/localhost.pem` file.

1. You will need your own **cluster AAD application** with client secret
   authentication enabled.

   ```
   AZURE_CLUSTER_CLIENT_ID=$(az ad app create --display-name user-$USER-v4 --query appId -o tsv)
   az ad sp create --id "$AZURE_CLUSTER_CLIENT_ID"
   AZURE_CLUSTER_CLIENT_SECRET=$(az ad app credential reset --id $AZURE_CLUSTER_CLIENT_ID --query password -o tsv)
   ```

1. Copy env.example to env, edit the values and source the env file.  This file
   holds (only) the environment variables necessary for the RP to run.

   * LOCATION:                      Azure location where RP and cluster(s) will run (default: `eastus`)
   * RESOURCEGROUP:                 Name of a new resource group which will contain the RP resources
   * RP_MODE:                       Set to `development` when not in production.

   * AZURE_TENANT_ID:               Azure tenant UUID
   * AZURE_SUBSCRIPTION_ID:         Azure subscription UUID
   * AZURE_ARM_CLIENT_ID:           ARM application client UUID
   * AZURE_ARM_CLIENT_SECRET:       ARM application client secret
   * AZURE_FP_CLIENT_ID:            RP "first party" application client UUID
   * AZURE_CLIENT_ID:               RP AAD application client UUID
   * AZURE_CLIENT_SECRET:           RP AAD application client secret
   * AZURE_CLUSTER_CLIENT_ID:       Cluster AAD application client UUID
   * AZURE_CLUSTER_CLIENT_SECRET:   Cluster AAD application client secret

   * PULL_SECRET:                   A cluster pull secret retrieved from [Red Hat OpenShift Cluster Manager](https://cloud.redhat.com/openshift/install/azure/installer-provisioned)

   ```
   cp env.example env
   vi env
   . ./env
   ```

1. Choose the RP deployment parameters:

   * COSMOSDB_ACCOUNT:       Name of a new CosmosDB account
   * DOMAIN:                 DNS subdomain shared by all clusters              (RH: $RESOURCEGROUP.osadev.cloud)
   * KEYVAULT_NAME:          Name of a new key vault
   * ADMIN_OBJECT_ID:        AAD object ID for key vault admin(s)              (RH: `az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv`)
   * RP_SERVICEPRINCIPAL_ID: AAD object ID for RP principal                    (RH: `az ad sp list --all --query "[?appDisplayName=='aro-v4-rp-shared'].objectId" -o tsv`)
   * FP_SERVICEPRINCIPAL_ID: AAD object ID for "first party" service principal (RH: `az ad sp list --all --query "[?appDisplayName=='aro-v4-fp-shared'].objectId" -o tsv`)

1. Create the resource group and with at least `Contributor` and `User Access
   Administrator` permissions on your subscription deploy the RP resources:

   ```
   COSMOSDB_ACCOUNT=$RESOURCEGROUP
   DOMAIN=$RESOURCEGROUP.osadev.cloud
   KEYVAULT_NAME=$RESOURCEGROUP
   ADMIN_OBJECT_ID=$(az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv)
   RP_SERVICEPRINCIPAL_ID=$(az ad sp list --all --query "[?appDisplayName=='aro-v4-rp-shared'].objectId" -o tsv)
   FP_SERVICEPRINCIPAL_ID=$(az ad sp list --all --query "[?appDisplayName=='aro-v4-fp-shared'].objectId" -o tsv)

   az group create -g "$RESOURCEGROUP" -l "$LOCATION"

   az group deployment create -g "$RESOURCEGROUP" --mode complete --template-file deploy/rp-development.json --parameters "location=$LOCATION" "databaseAccountName=$COSMOSDB_ACCOUNT" "domainName=$DOMAIN" "keyvaultName=$KEYVAULT_NAME" "adminObjectId=$ADMIN_OBJECT_ID" "rpServicePrincipalId=$RP_SERVICEPRINCIPAL_ID"
   ```

1. Load the application key/certificate into the key vault:

   ```
   AZURE_KEY_FILE=secrets/aro-v4-fp-shared.pem

   az keyvault certificate import --vault-name "$KEYVAULT_NAME" --name rp-firstparty --file "$AZURE_KEY_FILE"
   ```

1. Load the serving key/certificate into the key vault:

   ```
   TLS_KEY_FILE=secrets/localhost.pem

   az keyvault certificate import --vault-name "$KEYVAULT_NAME" --name rp-server --file "$TLS_KEY_FILE"
   ```

1. Create a glue record in the parent DNS zone:

   ```
   PARENT_DNS_RESOURCEGROUP=dns

   az network dns record-set ns create --resource-group "$PARENT_DNS_RESOURCEGROUP" --zone "$(cut -d. -f2- <<<"$DOMAIN")" --name "$(cut -d. -f1 <<<"$DOMAIN")"

   for ns in $(az network dns zone show --resource-group "$RESOURCEGROUP" --name "$DOMAIN" --query nameServers -o tsv); do az network dns record-set ns add-record --resource-group "$PARENT_DNS_RESOURCEGROUP" --zone "$(cut -d. -f2- <<<"$DOMAIN")" --record-set-name "$(cut -d. -f1 <<<"$DOMAIN")" --nsdname $ns; done
   ```

## Running the RP

```
go run ./cmd/rp
```

## Useful commands

```
export VNET_RESOURCEGROUP=$RESOURCEGROUP-vnet
az group create -g "$VNET_RESOURCEGROUP" -l "$LOCATION"
az network vnet create -g "$VNET_RESOURCEGROUP" -n vnet --address-prefixes 10.0.0.0/9
az role assignment create --role "ARO v4 Development Subnet Contributor" --assignee-object-id "$(az ad sp list --all --query "[?appId=='$AZURE_FP_CLIENT_ID'].objectId" -o tsv)" --scope "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$VNET_RESOURCEGROUP/providers/Microsoft.Network/virtualNetworks/vnet"
az role assignment create --role "ARO v4 Development Subnet Contributor" --assignee-object-id "$(az ad sp list --all --query "[?appId=='$AZURE_CLUSTER_CLIENT_ID'].objectId" -o tsv)" --scope "/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$VNET_RESOURCEGROUP/providers/Microsoft.Network/virtualNetworks/vnet"

export CLUSTER=cluster
```

* Register a subscription:

```
curl -k -X PUT "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0" -H 'Content-Type: application/json' -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}'
```

* Create a cluster:

```
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24

envsubst <examples/cluster-v20191231.json | curl -k -X PUT "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d @-
```

* Get a cluster:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview"
```

* Get a cluster's kubeadmin credentials:

```
curl -k -X POST "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/credentials?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{}'
```

* List clusters in resource group:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2019-12-31-preview"
```

* List clusters in subscription:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/providers/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2019-12-31-preview"
```

* Scale a cluster:

```
COUNT=4

curl -k -X PATCH "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{"properties": {"workerProfiles": [{"name": "worker", "count": '"$COUNT"'}]}}'
```

* Delete a cluster:

```
curl -k -X DELETE "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview"

az network vnet subnet delete -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master"
az network vnet subnet delete -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker"
```

* Delete a subscription:

```
curl -k -X PUT "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0" -H 'Content-Type: application/json' -d '{"state": "Deleted", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}'
```

* List operations:

```
curl -k "https://localhost:8443/providers/Microsoft.RedHatOpenShift/operations?api-version=2019-12-31-preview"
```

## Basic architecture

* pkg/frontend is intended to become a spec-compliant RP web server.  It is
  backed by CosmosDB.  Incoming PUT/DELETE requests are written to the database
  with an non-terminal (Updating/Deleting) provisioningState.

* pkg/backend reads documents with non-terminal provisioningStates,
  asynchronously updates them and finally updates document with a terminal
  provisioningState (Succeeded/Failed).  The backend updates the document with a
  heartbeat - if this fails, the document will be picked up by a different
  worker.

* As CosmosDB does not support document patch, care is taken to correctly pass
  through any fields in the internal model which the reader is unaware of (see
  `github.com/ugorji/go/codec.MissingFielder`).  This is intended to help in
  upgrade cases and (in the future) with multiple microservices reading from the
  database in parallel.

* Care is taken to correctly use optimistic concurrency to avoid document
  corruption through concurrent writes (see `RetryOnPreconditionFailed`).

* The pkg/api architecture differs somewhat from
  `github.com/openshift/openshift-azure`: the intention is to fix the broken
  merge semantics and try pushing validation into the versioned APIs to improve
  error reporting.

* Everything is intended to be crash/restart/upgrade-safe, horizontally
  scaleable, upgradeable...

## Debugging

* Get an admin kubeconfig:

  ```
  hack/get-admin-kubeconfig.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER
  export KUBECONFIG=admin.kubeconfig
  oc version
  ```

* SSH to the bootstrap node:

  ```
  hack/ssh-bootstrap.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER
  ```

## Useful links

* https://github.com/Azure/azure-resource-manager-rpc

* https://github.com/microsoft/api-guidelines

* https://docs.microsoft.com/en-gb/rest/api/cosmos-db

* https://github.com/jim-minter/go-cosmosdb
