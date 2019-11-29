# github.com/jim-minter/rp

## Install

1. Install the following:

   * go 1.12 or later
   * az client

1. Log in to Azure:

   ```
   az login
   ```

1. You will need a publicly resolvable DNS zone resource in Azure.  For RH ARO
   engineering, this is the `osadev.cloud` zone in the `dns` resource group.

1. You will need an AAD application with:

   * client certificate and (for now) client secret authentication enabled
   * (for now) User Access Administrator role granted on the subscription
   * (for now) `Azure Active Directory Graph / Application.ReadWrite.OwnedBy`
     privileges granted

   For RH ARO engineering, this is the `aro-team-shared` AAD application. You
   will need the client ID, client secret, and a key/certificate file
   (`aro-team-shared.pem`) that can be loaded into your key vault.  Ask if you
   do not have these.

   For non-RH ARO engineering, a suitable key/certificate file can be generated
   using the following helper utility:

   ```
   # Non-RH ARO engineering only
   go run ./hack/genkey -extKeyUsage client "$AZURE_CLIENT_ID"
   ```

1. Copy env.example to env, edit the values and source the env file.  This file
   holds (only) the environment variables necessary for the RP to run.

   * AZURE_TENANT_ID:       Azure tenant UUID
   * AZURE_SUBSCRIPTION_ID: Azure subscription UUID
   * AZURE_CLIENT_ID:       Azure AD application client UUID
   * AZURE_CLIENT_SECRET:   Azure AD application client secret

   * LOCATION:              Azure location where RP and cluster(s) will run (default: `eastus`)
   * RESOURCEGROUP:         Name of a new resource group which will contain the RP resources

   * PULL_SECRET:           A cluster pull secret retrieved from [Red Hat OpenShift Cluster Manager](https://cloud.redhat.com/openshift/install/azure/installer-provisioned)

   * RP_MODE:               Set to `development` when not in production.
   ```
   cp env.example env
   vi env
   . ./env
   ```

1. Choose the RP deployment parameters:

   * COSMOSDB_ACCOUNT: Name of a new CosmosDB account
   * DOMAIN:           DNS subdomain shared by all clusters (RH: $something.osadev.cloud)
   * KEYVAULT_NAME:    Name of a new key vault
   * ADMIN_OBJECT_ID:  AAD object ID for key vault admin(s) (RH: `az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv`)
   * RP_OBJECT_ID:     AAD object ID for AAD application    (RH: `az ad app list --all --query "[?appId=='$AZURE_CLIENT_ID'].objectId" -o tsv`)

1. Create the resource group and deploy the RP resources:

   ```
   COSMOSDB_ACCOUNT=$RESOURCEGROUP
   DOMAIN=$RESOURCEGROUP.osadev.cloud
   KEYVAULT_NAME=$RESOURCEGROUP
   ADMIN_OBJECT_ID=$(az ad group list --query "[?displayName=='Engineering'].objectId" -o tsv)
   RP_OBJECT_ID=$(az ad sp list --all --query "[?appId=='$AZURE_CLIENT_ID'].objectId" -o tsv)

   az group create -g "$RESOURCEGROUP" -l "$LOCATION"

   az group deployment create -g "$RESOURCEGROUP" --mode complete --template-file deploy/rp.json --parameters "location=$LOCATION" "databaseAccountName=$COSMOSDB_ACCOUNT" "domainName=$DOMAIN" "keyvaultName=$KEYVAULT_NAME" "adminObjectId=$ADMIN_OBJECT_ID" "rpObjectId=$RP_OBJECT_ID"
   ```

1. Load the application key/certificate into the key vault:

   ```
   AZURE_KEY_FILE=aro-team-shared.pem

   az keyvault certificate import --vault-name "$KEYVAULT_NAME" --name azure --file "$AZURE_KEY_FILE"
   ```

1. Generate a self-signed serving key/certificate and load it into the key vault:

   ```
   TLS_KEY_FILE=localhost.pem
   go run ./hack/genkey localhost
   az keyvault certificate import --vault-name "$KEYVAULT_NAME" --name tls --file "$TLS_KEY_FILE"
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

export CLUSTER=cluster
```

* Register a subscription:

curl -k -X PUT "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID?api-version=2.0" -H 'Content-Type: application/json' -d '{"state": "Registered", "properties": {"tenantId": "'"$AZURE_TENANT_ID"'"}}'

* Create a cluster:

```
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24
az network vnet subnet create -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker" --address-prefixes 10.$((RANDOM & 127)).$((RANDOM & 255)).0/24

envsubst <examples/cluster-v20191231.json | curl -k -X PUT "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d @-
```

* Get a cluster:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview"
```

* Get a cluster's kubeadmin credentials:

```
curl -k -X POST "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER/credentials?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{}'
```

* List clusters in resource group:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2019-12-31-preview"
```

* List clusters in subscription:

```
curl -k "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/providers/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2019-12-31-preview"
```

* Scale a cluster:

```
COUNT=3

curl -k -X PATCH "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{"properties": {"workerProfiles": [{"name": "worker", "count": '"$COUNT"'}]}}'
```

* Delete a cluster:

```
curl -k -X DELETE "https://localhost:8443/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER?api-version=2019-12-31-preview"

az network vnet subnet delete -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-master"
az network vnet subnet delete -g "$VNET_RESOURCEGROUP" --vnet-name vnet -n "$CLUSTER-worker"
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
  hack/get-admin-kubeconfig.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER
  export KUBECONFIG=admin.kubeconfig
  oc version
  ```

* SSH to the bootstrap node:

  ```
  hack/ssh-bootstrap.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER
  ```

## Useful links

* https://github.com/Azure/azure-resource-manager-rpc

* https://github.com/microsoft/api-guidelines

* https://docs.microsoft.com/en-gb/rest/api/cosmos-db

* https://github.com/jim-minter/go-cosmosdb
