## Useful links

https://github.com/Azure/azure-resource-manager-rpc

https://github.com/microsoft/api-guidelines

https://docs.microsoft.com/en-gb/rest/api/cosmos-db

https://github.com/jim-minter/go-cosmosdb

## Prequisites

* Publicly resolvable DNS zone resource in Azure

* Service principal (client ID and secret) with (for now) User Access
  Administrator access to the subscription and (for now) `Azure Active Directory
  Graph / Application.ReadWrite.OwnedBy` privileges

## Installation

* Copy env.example to env and edit the values as follows:

  * AZURE_TENANT_ID:       Azure tenant UUID
  * AZURE_SUBSCRIPTION_ID: Azure subscription UUID
  * AZURE_CLIENT_ID:       Azure service principal client UUID
  * AZURE_CLIENT_SECRET:   Azure service principal secret

  * LOCATION:              Azure location where RP and cluster(s) will run (default: `eastus`)

  * RP_RESOURCEGROUP:      Name of resource group which will contain the CosmosDB resource
  * COSMOSDB_ACCOUNT:      CosmosDB account name
  * COSMOSDB_KEY:          CosmosDB master key (default: autopopulated)

  * PULL_SECRET:           A cluster pull secret retrieved from (Red Hat OpenShift Cluster Manager)[https://cloud.redhat.com/openshift/install/azure/installer-provisioned]

```
cp env.example env
vi env
```

* Source the env file

```
. ./env
```

* Deploy a CosmosDB SQL database and DNS zone to a resource group

```
DOMAIN=mydomain.osadev.cloud

az group create -g "$RP_RESOURCEGROUP" -l "$LOCATION"`

az group deployment create -g "$RP_RESOURCEGROUP" --mode complete --template-file deploy/rp.json --parameters "location=$LOCATION" "databaseAccountName=$COSMOSDB_ACCOUNT" "domainName=$DOMAIN"
```

* If appropriate, create a glue record in the parent DNS zone

```
DOMAIN=mydomain.osadev.cloud
PARENT_DNS_RESOURCEGROUP=dns
PARENT_DNS_ZONE=osadev.cloud
CHILD_DNS_NAME=mydomain

az network dns record-set ns create --resource-group "$PARENT_DNS_RESOURCEGROUP" --zone "$PARENT_DNS_ZONE" --name "$CHILD_DNS_NAME"

for ns in $(az network dns zone show --resource-group "$RP_RESOURCEGROUP" --name "$DOMAIN" --query nameServers -o tsv); do az network dns record-set ns add-record --resource-group "$PARENT_DNS_RESOURCEGROUP" --zone "$PARENT_DNS_ZONE" --record-set-name "$CHILD_DNS_NAME" --nsdname $ns; done
```

## Getting started

* Source the env file a second time so that $COSMOSDB_KEY is discovered and
  populated

```
. ./env
```

* Run the RP

```
go run ./cmd/rp
```

## Useful commands

```
CLUSTER=cluster
```

* Create a cluster

```
curl -X PUT "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{"location":"'"$LOCATION"'", "properties": {"servicePrincipalProfile": {"clientId": "'"$CLIENT_ID"'", "clientSecret": "'"$CLIENT_SECRET"'"}}}'
```

* Get a cluster

```
curl "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview"
```

* Get a cluster's credentials

```
curl -X POST "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER/credentials?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{}'
```

* List clusters in resource group

```
curl "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters?api-version=2019-12-31-preview"
```

* List clusters in subscription

```
curl "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/providers/Microsoft.RedHatOpenShift/OpenShiftClusters?api-version=2019-12-31-preview"
```

* Scale a cluster

```
COUNT=3

curl -X PATCH "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{"properties": {"workerProfiles": [{"name": "worker", "count": '"$COUNT"'}]}}'
```

* Delete a cluster

```
curl -X DELETE "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview"
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

* Get an admin kubeconfig

```
hack/get-admin-kubeconfig.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER
export KUBECONFIG=admin.kubeconfig
oc version
```

* SSH to the bootstrap node

```
hack/ssh-bootstrap.sh /subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/$CLUSTER
```
