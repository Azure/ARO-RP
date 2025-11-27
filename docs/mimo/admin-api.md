# Admin API

All endpoints require `api-version=admin` query parameter and proper Azure AD authentication (Bearer token).

**Base URL:** `https://management.azure.com/admin`

**Authentication:** All requests require Azure AD authentication token in the `Authorization` header:

```text
Authorization: Bearer {access_token}
```

**Resource ID Format:** The `{resourceId}` parameter is the full ARM resource ID for an ARO cluster:

```text
/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{clusterName}
```

## Data Structures

### MaintenanceManifest

```json
{
  "id": "string (UUID)",
  "clusterResourceID": "string",
  "state": "Pending | InProgress | Completed | Failed | RetriesExceeded | TimedOut | Cancelled",
  "statusText": "string",
  "maintenanceTaskID": "string (UUID)",
  "priority": "integer",
  "runAfter": "integer (Unix timestamp in seconds)",
  "runBefore": "integer (Unix timestamp in seconds)"
}
```

### MaintenanceManifestList

```json
{
  "value": [
    {
      "id": "...",
      "state": "...",
      ...
    }
  ],
  "nextLink": "string (optional, for pagination)"
}
```

## Endpoints

### GET /admin/{resourceId}/maintenancemanifests

Returns a paginated list of maintenance manifests for a specific cluster.

**Query Parameters:**

- `limit` (optional): Maximum number of manifests to return. Default: 100. Maximum per page: 10.

**Response:** `200 OK`

```json
{
  "value": [
    {
      "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
      "clusterResourceID": "/subscriptions/.../openShiftClusters/...",
      "state": "Completed",
      "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
      "priority": 0,
      "runAfter": 1704067200,
      "runBefore": 1704672000,
      "statusText": ""
    }
  ],
  "nextLink": "https://management.azure.com/admin/...?skipToken=..."
}
```

**Error Responses:**

- `404 Not Found`: Cluster not found or cluster is being deleted
- `500 Internal Server Error`: Database or internal error

---

### GET /admin/maintenancemanifests/queued

Returns a paginated list of all queued (Pending or InProgress) maintenance manifests across all clusters.

**Query Parameters:**

- `limit` (optional): Maximum number of manifests to return. Default: 100. Maximum per page: 10.

**Response:** `200 OK`

```json
{
  "value": [
    {
      "id": "...",
      "clusterResourceID": "...",
      "state": "Pending",
      ...
    }
  ],
  "nextLink": "..."
}
```

**Note:** This endpoint does not require a resource ID and returns manifests from all clusters.

---

### GET /admin/{resourceId}/maintenancemanifests/{manifestId}

Returns a specific maintenance manifest by ID.

**Path Parameters:**

- `{resourceId}`: Full ARM resource ID of the cluster
- `{manifestId}`: UUID of the manifest

**Response:** `200 OK`

```json
{
  "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
  "clusterResourceID": "/subscriptions/.../openShiftClusters/...",
  "state": "InProgress",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "priority": 0,
  "runAfter": 1704067200,
  "runBefore": 1704672000,
  "statusText": "Rotating certificates on control plane nodes"
}
```

**Error Responses:**

- `404 Not Found`: Cluster or manifest not found, or cluster is being deleted
- `500 Internal Server Error`: Database or internal error

---

### PUT /admin/{resourceId}/maintenancemanifests

Creates a new maintenance manifest. Returns the created manifest with generated ID and default values.

**Request Body:**

```json
{
  "maintenanceTaskID": "string (UUID, required)",
  "priority": "integer (optional, default: 0)",
  "runAfter": "integer (optional, Unix timestamp, default: current time)",
  "runBefore": "integer (optional, Unix timestamp, default: current time + 7 days)"
}
```

**Note:**

- `maintenanceTaskID` is required
- If `runAfter` is omitted or 0, it defaults to the current Unix timestamp
- If `runBefore` is omitted or 0, it defaults to 7 days from creation time
- The `id` field is auto-generated and should not be included in the request
- The `state` field is automatically set to `"Pending"` and should not be included
- The `clusterResourceID` field is automatically populated from the URL path and should not be included in the request
- The `priority` field defaults to 0 if omitted

**Response:** `201 Created`

```json
{
  "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
  "clusterResourceID": "/subscriptions/.../openShiftClusters/...",
  "state": "Pending",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "priority": 0,
  "runAfter": 1704067200,
  "runBefore": 1704672000,
  "statusText": ""
}
```

**Example Request:**

```bash
curl -X PUT \
  "https://management.azure.com/admin/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{cluster}/maintenancemanifests?api-version=admin" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
    "runAfter": 1704067200,
    "runBefore": 1704672000
  }'
```

**Error Responses:**

- `400 Bad Request`: Invalid request content or validation error
- `404 Not Found`: Cluster not found or cluster is being deleted
- `500 Internal Server Error`: Database or internal error

---

### POST /admin/{resourceId}/maintenancemanifests/{manifestId}/cancel

Cancels a maintenance manifest. Only manifests in `Pending` state can be cancelled.

**Path Parameters:**

- `{resourceId}`: Full ARM resource ID of the cluster
- `{manifestId}`: UUID of the manifest to cancel

**Response:** `200 OK`

```json
{
  "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
  "clusterResourceID": "/subscriptions/.../openShiftClusters/...",
  "state": "Cancelled",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "priority": 0,
  "runAfter": 1704067200,
  "runBefore": 1704672000,
  "statusText": ""
}
```

**Important Notes:**

- Only manifests in `Pending` state can be cancelled
- Cancelling does NOT stop a task that is currently executing (`InProgress` state)
- Attempting to cancel a manifest in any other state will return `406 Not Acceptable`

**Error Responses:**

- `404 Not Found`: Cluster or manifest not found, or cluster is being deleted
- `406 Not Acceptable`: Manifest is not in `Pending` state (cannot cancel)
- `500 Internal Server Error`: Database or internal error

---

### DELETE /admin/{resourceId}/maintenancemanifests/{manifestId}

Deletes a maintenance manifest permanently. This operation cannot be undone.

**Path Parameters:**

- `{resourceId}`: Full ARM resource ID of the cluster
- `{manifestId}`: UUID of the manifest to delete

**Response:** `200 OK`

```json
{}
```

**Warning:** This is a destructive operation. Only use this to clean up terminal-state manifests (`Completed`, `Failed`, `Cancelled`, `TimedOut`) that are no longer needed.

**Error Responses:**

- `404 Not Found`: Manifest not found
- `500 Internal Server Error`: Database or internal error

---

## Maintenance Manifest States

| State | Description |
|-------|-------------|
| `Pending` | Waiting to be executed by actuator |
| `InProgress` | Currently being executed by actuator |
| `Completed` | Successfully completed |
| `Failed` | Failed with terminal error (no retry) |
| `RetriesExceeded` | Failed after 5 retry attempts |
| `TimedOut` | Missed the `runBefore` deadline |
| `Cancelled` | Manually cancelled via API |

## Valid Maintenance Task IDs

These are the hard-coded task IDs defined in `pkg/mimo/const.go`. You must use these exact UUIDs when creating manifests:

| Task ID | Purpose |
|---------|---------|
| `9b741734-6505-447f-8510-85eb0ae561a2` | TLS Certificate Rotation |
| `b41749fc-af26-4ab7-b5a1-e03f3ee4cba6` | Operator Flags Update |
| `082978ce-3700-4972-835f-53d48658d291` | ACR Token Checker |
| `a4477c3a-ddbb-41a0-88e8-b5cda67b623a` | MDSD Certificate Rotation |

## Pagination

When a response includes a `nextLink` field, you can use it to retrieve the next page of results:

```bash
curl -X GET "{nextLink}" \
  -H "Authorization: Bearer {token}"
```

The `nextLink` contains all necessary query parameters including the `skipToken` for continuation.

## Error Response Format

All error responses follow the Azure Cloud Error format:

```json
{
  "error": {
    "code": "ErrorCode",
    "message": "Error message description"
  }
}
```

Common error codes:

- `NotFound`: Resource not found
- `InvalidRequestContent`: Request body validation failed
- `PropertyChangeNotAllowed`: Cannot perform operation in current state
- `InternalServerError`: Internal server error
