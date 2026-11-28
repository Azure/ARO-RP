# Admin API

All need `api-version=admin`.

## GET /admin/RESOURCE_ID/maintenancemanifests

Returns a list of MIMO maintenance manifests for the cluster.

### Query parameters

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | int | `100` | Max items to return |
| `$skipToken` | string | | Continuation token for paging |
| `after` | int64 (unix seconds) | `now - 30 days` (`0` when only `before` is set) | Inclusive lower bound on `runAfter` |
| `before` | int64 (unix seconds) | none | Inclusive upper bound on `runAfter` |

When neither `after` nor `before` is provided, `after` defaults to `now - 30 days`. When only `before` is provided, `after` defaults to `0` (full history up to `before`). Manifests in active states (**Pending** or **InProgress**) are always returned regardless of the time window, so that fleet cancellation tooling (e.g. `fleet-certificate-rotation.ps1 -action cancel`) that lists via this endpoint never misses a live task. Pass `after=0` to retrieve the full history.

The `after` and `before` values are **Unix epoch seconds** (seconds since 1970-01-01 UTC). To convert a date to epoch seconds:

```sh
# Linux / WSL (UTC)
date -u -d "2024-11-01" +%s          # → 1730419200

# macOS (UTC)
TZ=UTC date -j -f "%Y-%m-%d" "2024-11-01" +%s

# PowerShell (UTC)
[DateTimeOffset]::Parse("2024-11-01T00:00:00Z").ToUnixTimeSeconds()
```

### Examples

#### List manifests within a specific date range

```sh
# after=1700000000 (2023-11-14) to before=1730000000 (2024-10-27)
curl -k "https://localhost:8443/admin/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME/maintenancemanifests?api-version=admin&after=1700000000&before=1730000000"
```

#### List manifests from the last 7 days (dynamic)

```sh
curl -k "https://localhost:8443/admin/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME/maintenancemanifests?api-version=admin&after=$(date -u -d '7 days ago' +%s)"
```

#### Retrieve full history

```sh
curl -k "https://localhost:8443/admin/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME/maintenancemanifests?api-version=admin&after=0"
```

## PUT /admin/RESOURCE_ID/maintenancemanifests

Creates a new manifest. Returns the created manifest.

### Example

```sh
curl -X PUT -k "https://localhost:8443/admin/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME/maintenancemanifests?api-version=admin" \
  --header "Content-Type: application/json" \
  -d '{"maintenanceTaskID": "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6"}'
```

Replace `SUBSCRIPTION_ID`, `RESOURCE_GROUP`, and `CLUSTER_NAME` with the target cluster's values. Registered task IDs are defined in [`pkg/mimo/const.go`](../../pkg/mimo/const.go).

### Common task IDs

- TLS cert rotation: `9b741734-6505-447f-8510-85eb0ae561a2`
- Operator flags update: `b41749fc-af26-4ab7-b5a1-e03f3ee4cba6`
- Restore cluster service principal RBAC: `3d2a7f16-5c48-4b91-9e03-7a1f8cbe4d52`

The operator-flags task is used for fleet rollouts of
`properties.operatorFlags` changes (for example OTEL profile and source-field
settings).

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

The schedule `id` field is the schedule's own unique identifier (distinct from `maintenanceTaskID`). If `id` is omitted from the request body, a new schedule is created with an auto-generated ID. If `id` is provided and matches an existing schedule, that schedule is updated. If `id` is provided but does not match any existing schedule, a new schedule is created with that ID.

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
