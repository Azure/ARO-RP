## Useful links

https://github.com/Azure/azure-resource-manager-rpc

https://github.com/microsoft/api-guidelines

https://docs.microsoft.com/en-gb/rest/api/cosmos-db

https://docs.microsoft.com/en-gb/rest/api/storageservices/queue-service-rest-api

https://github.com/jim-minter/go-cosmosdb

## Getting started

* Copy env.example to env, edit the values and source it (`. ./env`)

* Deploy a CosmosDB SQL database and a v2 storage account to a resource group:
  `az group create -g "$RP_RESOURCEGROUP" -l "$LOCATION"`
  `az group deployment create -g "$RP_RESOURCEGROUP" --mode complete --template-file deploy/rp.json --parameters "location=$LOCATION" "storageAccountName=$STORAGE_ACCOUNT" "databaseAccountName=$COSMOSDB_ACCOUNT"`

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
  with an Updating/Deleting provisioningState and unqueued flag set.

* pkg/queue/forwarder is a worker which spots unqueued documents, writes their
  IDs into a queue and unsets the flag.  It is intended that a document ID may
  be accidently written to the queue more than once without ill effect.

* pkg/backend reads IDs off the queue and asynchronously updates desired state,
  finally updating the database document with a terminal provisioningState
  (Succeeded/Failed).  The backend sends a heartbeat to the queue - if this
  fails, the ID will be picked up by a different worker.

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
