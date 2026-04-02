# Admin API

All need `api-version=admin`.

## GET /admin/RESOURCE_ID/maintenancemanifests

Returns a list of MIMO maintenance manifests.

## PUT /admin/RESOURCE_ID/maintenancemanifests

Creates a new manifest. Returns the created manifest.

### Example

```sh
curl -X PUT -k "https://localhost:8443/admin/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME/maintenancemanifests?api-version=admin" \
  --header "Content-Type: application/json" \
  -d '{"maintenanceTaskID": "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6"}'
```

Replace `SUBSCRIPTION_ID`, `RESOURCE_GROUP`, and `CLUSTER_NAME` with the target cluster's values. Registered task IDs are defined in [`pkg/mimo/const.go`](../../pkg/mimo/const.go).

## GET /admin/RESOURCE_ID/maintenancemanifests/MANIFEST_ID

Returns a manifest.

## DELETE /admin/RESOURCE_ID/maintenancemanifests/MANIFEST_ID

Deletes a manifest. This is only to be used as a last resort.

## POST /admin/RESOURCE_ID/maintenancemanifests/MANIFEST_ID/cancel

Cancels the manifest (the state becomes CANCELLED). It does not stop a task that is in the current process of execution.

## POST /admin/maintenancemanifests/cancel?scheduleID=SCHEDULE_ID

Cancels all manifests created by the given schedule ID.

## GET /admin/maintenanceschedules

Returns a list of all schedules.

## PUT /admin/maintenanceschedules

Creates or updates a schedule. Returns `201 Created` for new schedules and `200 OK` for updates.

If `id` is omitted from the request body, a new schedule is created with an auto-generated ID. If `id` is provided and matches an existing schedule, that schedule is updated. If `id` is provided but does not match any existing schedule, a new schedule is created with that ID.

See [Scheduler Calendar and Selectors](./scheduler-calendar-and-selectors.md) for the calendar expression format and selector syntax.

### Example

```sh
curl -X PUT -k "https://localhost:8443/admin/maintenanceschedules?api-version=admin" \
  --header "Content-Type: application/json" \
  -d '{"state": "Enabled", "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2", "schedule": "Mon *-*-* 00:00", "lookForwardCount": 4, "scheduleAcross": "24h", "selectors": [{"key": "subscriptionState", "operator": "in", "values": ["Registered"]}]}'
```

Task IDs are defined in [`pkg/mimo/const.go`](../../pkg/mimo/const.go).

## GET /admin/maintenanceschedules/SCHEDULE_ID

Returns a schedule.

## GET /admin/RESOURCE_ID/selectors

Returns the selector key-value pairs for a specific cluster. Use this to verify which clusters a schedule's selectors will match.
