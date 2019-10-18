## Useful links

https://github.com/Azure/azure-resource-manager-rpc

https://github.com/microsoft/api-guidelines

https://docs.microsoft.com/en-gb/rest/api/cosmos-db

https://github.com/jim-minter/go-cosmosdb

## Getting started

* Copy env.example to env, edit the values and source it (`. ./env`)

* Deploy a CosmosDB SQL database to a resource group:
  `az group create -g "$RP_RESOURCEGROUP" -l "$LOCATION"`
  `az group deployment create -g "$RP_RESOURCEGROUP" --mode complete --template-file deploy/rp.json --parameters "location=$LOCATION" "databaseAccountName=$COSMOSDB_ACCOUNT"`

* `go run ./cmd/rp`

## Useful commands

`export CLUSTER=cluster`

`curl -X PUT "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/RedHat.OpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview" -H 'Content-Type: application/json' -d '{"location":"'"$LOCATION"'", "properties": {"pullSecret": "'"$(base64 -w0 <<<$PULL_SECRET)"'"}}'`

`curl "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/RedHat.OpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview"`

`curl "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/RedHat.OpenShift/OpenShiftClusters/$CLUSTER/credentials?api-version=2019-12-31-preview"`

`curl -X DELETE "localhost:8080/subscriptions/$AZURE_SUBSCRIPTION_ID/resourceGroups/$CLUSTER/providers/RedHat.OpenShift/OpenShiftClusters/$CLUSTER?api-version=2019-12-31-preview"`

## Basic architecture

* pkg/frontend is intended to become a spec-compliant RP web server.  It is
  backed by CosmosDB.  Incoming PUT/DELETE requests are written to the database
  with an Updating/Deleting provisioningState.

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

* Everything is intended to be crash/restart/upgrade-safe...
