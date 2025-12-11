# Admin API

All need `api-version=admin`.

## GET /admin/RESOURCE_ID/maintenanceManifests

Returns a list of MIMO maintenance manifests.

## PUT /admin/RESOURCE_ID/maintenanceManifests

Creates a new manifest. Returns the created manifest.

### Example

```sh
curl -X PUT -k "https://localhost:8443/admin/subscriptions/fe16a035-e540-4ab7-80d9-373fa9a3d6ae/resourcegroups/v4-westeurope/providers/microsoft.redhatopenshift/openshiftclusters/abrownmimom1test/maintenanceManifests?api-version
=admin" -d '{"maintenanceTaskID": "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6"}' --header "Content-Type: application/json"
```

## GET /admin/RESOURCE_ID/maintenanceManifests/MANIFEST_ID

Returns a manifest.

## DELETE /admin/RESOURCE_ID/maintenanceManifests/MANIFEST_ID

Deletes a manifest. This is only to be used as a last resort.

## POST /admin/RESOURCE_ID/maintenanceManifests/MANIFEST_ID/cancel

Cancels the manifest (the state becomes CANCELLED). It does not stop a task that is in the current process of execution.