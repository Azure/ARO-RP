# MIMO Troubleshooting Guide

This guide is designed for ARO Site Reliability Engineers (SREs) who troubleshoot MIMO issues using PowerShell scripts that wrap the Admin API. SREs do not have direct access to Kubernetes clusters, SSH, or actuator logs - all diagnostics must be performed via PowerShell scripts, internal tooling, and metrics.

## Table of Contents

- [MIMO Architecture & Infrastructure](#mimo-architecture--infrastructure)
  - [What is MIMO?](#what-is-mimo)
  - [Key Components](#key-components)
  - [Core Concepts](#core-concepts)
  - [Data Flow](#data-flow)
  - [Concurrency & Timeouts](#concurrency--timeouts)
  - [Deployment](#deployment)
- [Monitoring & Working Standards](#monitoring--working-standards)
  - [Monitor 1: Actuator Heartbeat](#monitor-1-actuator-heartbeat)
  - [Monitor 2: Queue Length](#monitor-2-queue-length)
  - [Monitor 3: Active Worker Count](#monitor-3-active-worker-count)
- [Quick Reference](#quick-reference)
- [Admin API Reference](#admin-api-reference)
- [Common Issues & Resolutions](#common-issues--resolutions)
- [Diagnostic Procedures](#diagnostic-procedures)
- [Recovery Procedures](#recovery-procedures)
- [Related Documentation](#related-documentation)

---

## MIMO Architecture & Infrastructure

### What is MIMO?

MIMO (Managed Infrastructure Maintenance Operator) is an automated maintenance system that performs routine operational tasks on ARO clusters. It replaces the legacy manual "PUCM" (Pre-Upgrade Cluster Maintenance) process.

**What MIMO maintains:**
- Hosted infrastructure: Private Link Services, Gateway documents, CosmosDB records
- Customer infrastructure: Private Link endpoints, VNET configurations
- In-cluster resources: ARO Operator updates, TLS certificates, configurations
- Critical recurring tasks: TLS/MDSD certificate rotation, ACR token rotation

### Key Components

#### 1. Actuator (Implemented)
The actuator is the worker component that executes maintenance tasks. It runs as a 3-VM Virtual Machine Scale Set (VMSS) within the ARO infrastructure.

**How it works:**
- Continuously polls CosmosDB for pending maintenance manifests
- Uses lease-based concurrency to prevent multiple actuators from processing the same manifest
- Distributes work across VMs using bucket-based partitioning (256 buckets total)

**Work Distribution (Bucket Partitioning):**
- Each cluster is assigned to one of 256 buckets (numbered 0-255)
- Assignment is deterministic and **permanent**: `bucket = hash(clusterResourceId) % 256`
  - Example: Cluster `/subscriptions/.../cluster-A` → hash → 12345 → 12345 % 256 → **bucket 89**
  - **The same cluster ALWAYS maps to the same bucket** (never changes)
  - As long as the cluster resource ID doesn't change, the bucket number stays the same
- The 3 VMs in the VMSS divide these 256 buckets among themselves:
  - VM 0: processes buckets 0-85 (86 buckets)
  - VM 1: processes buckets 86-170 (85 buckets)
  - VM 2: processes buckets 171-255 (85 buckets)
- Each VM only polls CosmosDB for manifests belonging to its assigned buckets
- This prevents multiple VMs from attempting the same manifest simultaneously

**Why this matters for SREs:**
- **Any given cluster is always processed by the same VM** (its bucket determines which VM)
- If one VM is down, only ~1/3 of clusters are affected (those in that VM's buckets)
- If all manifests for a specific cluster are stuck, the VM handling that cluster's bucket may have issues
- You cannot control which VM processes which cluster (it's determined automatically by the cluster's resource ID)

#### 2. Scheduler (Not Yet Implemented)
Planned component that will automatically create manifests based on time triggers, cluster updates, or maintenance windows. Currently, manifests are created manually via Admin API.

#### 3. Reporter
Emits metrics (heartbeat, queue length, active workers), logs to Geneva Monitor, and provides error information in manifest `statusText` field.

### Core Concepts

#### Maintenance Manifest
A maintenance manifest is a work item stored in CosmosDB that represents a specific maintenance task to be performed on a cluster.

**Manifest Lifecycle:**
```
Pending → InProgress → Completed
                    ↓
                   Failed / RetriesExceeded / TimedOut / Cancelled
```

**Key Fields:**
- `id`: Unique manifest identifier (UUID)
- `clusterResourceID`: Full ARM resource ID of target cluster
- `maintenanceTaskID`: Type of task to execute (see Task IDs below)
- `state`: Current execution state
- `priority`: Lower number = higher priority (0 is highest)
- `runAfter`: Unix timestamp - earliest time to execute
- `runBefore`: Unix timestamp - deadline (becomes `TimedOut` if missed)
- `statusText`: Error messages or status information
- `dequeues`: Number of execution attempts (max 5)

#### Task
A task performs specific maintenance work on a cluster. Each task is composed of multiple **Steps** that execute sequentially. Tasks can run both off-cluster operations (CosmosDB, Azure resources) and in-cluster operations (delegated to ARO Operator).

**Implemented Tasks:**

⚠️ **Important:** These task IDs are hard-coded constants in `pkg/mimo/const.go`. You MUST use these exact UUIDs when creating manifests.

| Task ID | Purpose |
|---------|---------|
| `9b741734-6505-447f-8510-85eb0ae561a2` | TLS Certificate Rotation |
| `b41749fc-af26-4ab7-b5a1-e03f3ee4cba6` | Operator Flags Update |
| `082978ce-3700-4972-835f-53d48658d291` | ACR Token Checker |
| `a4477c3a-ddbb-41a0-88e8-b5cda67b623a` | MDSD Certificate Rotation |

#### Error Handling

**Transient Errors:**
- Temporary failures that may succeed on retry
- Manifest returns to `Pending` state for retry
- Max 5 attempts (`dequeues` counter)
- After 5 failures → `RetriesExceeded` state

**Terminal Errors:**
- Permanent failures that won't succeed on retry
- Manifest immediately moves to `Failed` state
- No automatic retry

### Data Flow

```
User/Scheduler → Admin API → CosmosDB → Actuator → Kubernetes API
                                             ↓
                                        Task Execution
                                             ↓
                                   CosmosDB (update state)
```

1. Manifest is created via Admin API (or future scheduler)
2. Manifest stored in CosmosDB with `Pending` state
3. Actuator polls CosmosDB for manifests matching its bucket assignment
4. Actuator acquires lease on manifest (60-minute TTL)
5. Manifest state updated to `InProgress`
6. Task executes against cluster (via Kubernetes API)
7. Result written back to CosmosDB (`Completed`, `Failed`, etc.)
8. Lease released

### Concurrency & Timeouts

**Lease-Based Locking:**
- Each manifest has a `leaseExpiry` timestamp and `leaseOwner` UUID
- Lease expires after 60 minutes (same as task timeout)
- Prevents multiple actuators from processing the same manifest

**Task Timeout:**
- 60-minute execution limit per task
- Timeout causes manifest to return to `Pending` and increments `dequeues`

### Deployment

- Runs in Microsoft-hosted first-party subscription (same as ARO RP)
- Deployed per region, not per cluster
- Not publicly accessible
- Admin API requires Azure AD authentication

---

## Monitoring & Working Standards

MIMO exposes three metrics for monitoring via MDM (statsd). This section describes each monitor, how it works, and alert thresholds.

---

### Monitor 1: Actuator Heartbeat

#### What It Is
- **Metric Name:** `actuator.heartbeat`
- **Type:** Counter
- **Update Frequency:** Every 60 seconds per actuator VM

#### How It Works
- Each of the 3 actuator VMs emits this metric independently
- Emitted ONLY when actuator is "ready":
  - Successfully connected to CosmosDB
  - Successfully connected to Kubernetes API
  - Actively polling for work
- Code location: `pkg/mimo/actuator/service.go:156`
- Emission condition checked via `checkReady()` function

**Normal Behavior:**
- You should see heartbeat from each VM every 60 seconds
- If all 3 VMs are healthy, you'll see 3 heartbeats per minute
- Missing heartbeat from one VM = that VM is down (expected during rolling updates)
- Missing heartbeat from all VMs = MIMO is completely down

#### Alert Thresholds

| Severity | Condition | Meaning |
|----------|-----------|---------|
| **Warning** | No heartbeat for 2-5 minutes | One or more VMs may be restarting |
| **Critical** | No heartbeat for > 5 minutes | All actuator VMs are down or unhealthy |

---

### Monitor 2: Queue Length

#### What It Is
- **Metric Name:** `database.maintenancemanifests.queue.length`
- **Type:** Gauge
- **Update Frequency:** Every 60 seconds

#### How It Works
- Calculated from CosmosDB query: `COUNT(*) WHERE state IN ('Pending', 'InProgress')`
- Includes ALL clusters globally across entire fleet
- Code location: `pkg/database/metrics.go:31-44`
- Emitted by a background goroutine in the database package

**Normal Behavior:**
- Baseline: 0-100 manifests (varies by fleet size)
- Spikes during maintenance windows are expected
- Should trend downward if no new manifests are being created
- Sustained growth = actuator cannot keep up with creation rate

**Healthy Pattern:**
- Manifests enter queue → processed within 30 minutes → exit queue
- Queue grows during maintenance window → drains after window ends

#### Alert Thresholds

| Severity | Condition | Meaning |
|----------|-----------|---------|
| **Warning** | 500-1000 manifests | Processing slower than creation rate |
| **Critical** | > 1000 manifests | Actuator significantly behind or stopped |

---

### Monitor 3: Active Worker Count

#### What It Is
- **Metric Name:** `mimo.actuator.workers.active.count`
- **Type:** Gauge
- **Update Frequency:** On worker start/stop (real-time)

#### How It Works
- Incremented when actuator starts processing a manifest
- Decremented when task completes (success or failure)
- Tracks concurrent task executions across all actuator VMs
- Code location: `pkg/mimo/actuator/service.go:303,307`

**Normal Behavior:**
- Range: 0-10 (depends on current workload)
- Should be > 0 when queue length > 0
- Stays at 0 when queue is empty (no work to do)

**Anomaly Pattern:**
- Worker count = 0 AND queue length > 100 → Actuator not picking up work

#### Alert Thresholds

| Severity | Condition | Meaning |
|----------|-----------|---------|
| **Warning** | 0 workers for > 10 min while queue > 50 | Actuator may be stuck |
| **Critical** | 0 workers for > 30 min while queue > 100 | Actuator not processing |

---

### Working Standards Summary

#### Normal Operating Conditions

| Metric | Healthy | Warning | Critical |
|--------|---------|---------|----------|
| `actuator.heartbeat` | Present every 60s | Missing 2-5 min | Missing > 5 min |
| `queue.length` | 0-100 | 500-1000 | > 1000 |
| `workers.active` | 0-10 (varies) | 0 while queue>50 | 0 while queue>100 |

#### Manifest State Distribution (Healthy System)

Based on historical data:
- 90%+ of manifests should reach `Completed` state
- < 5% in `Failed` or `RetriesExceeded`
- `Pending` manifests should clear within 30 minutes
- `InProgress` manifests should complete within 60 minutes

---

## Quick Reference

### Manifest States

| State | Meaning | SRE Action |
|-------|---------|------------|
| `Pending` | Waiting to execute | Normal - monitor if > 30 min |
| `InProgress` | Currently running | Normal - escalate if > 60 min |
| `Completed` | Success | None |
| `Failed` | Terminal error | Investigate `statusText`, may recreate |
| `RetriesExceeded` | Failed 5 times | Investigate root cause, recreate after fix |
| `TimedOut` | Missed `runBefore` | Delete or recreate if still needed |
| `Cancelled` | Manually cancelled | None |

### Task IDs (Hard-Coded in Codebase)

⚠️ **These are NOT examples - they are the actual UUIDs hard-coded in `pkg/mimo/const.go`**

| Task ID | Purpose |
|---------|---------|
| `9b741734-6505-447f-8510-85eb0ae561a2` | TLS Certificate Rotation |
| `b41749fc-af26-4ab7-b5a1-e03f3ee4cba6` | Operator Flags Update |
| `082978ce-3700-4972-835f-53d48658d291` | ACR Token Checker |
| `a4477c3a-ddbb-41a0-88e8-b5cda67b623a` | MDSD Certificate Rotation |

---

## Admin API Reference

### Base URL
```
https://management.azure.com/admin
```

All endpoints require `?api-version=admin` parameter and proper Azure authentication (Bearer token).

### Authentication

All requests require Azure AD authentication token in the `Authorization` header:
```
Authorization: Bearer {access_token}
```

### Available Admin API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/admin/{resourceId}/maintenanceManifests` | List all manifests for a specific cluster |
| PUT | `/admin/{resourceId}/maintenanceManifests` | Create new maintenance manifest |
| POST | `/admin/{resourceId}/maintenanceManifests/{id}/cancel` | Cancel a manifest |
| DELETE | `/admin/{resourceId}/maintenanceManifests/{id}` | Delete a manifest |

**Note:** `{resourceId}` is the full ARM resource ID for an ARO cluster:
```
/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{clusterName}
```

⚠️ **Important Limitations:**
- **Global queue queries** (e.g., viewing all queued manifests across all clusters) are NOT available via Admin API
- **Single manifest detail queries** (e.g., getting specific manifest by ID) are NOT available via Admin API
- For these operations, use internal tooling: **Geneva Actions**, **Admin Portal**, **Kusto/DGrep**, or **monitoring dashboards**

### Admin API Requests via PowerShell

**Script Location:** All PowerShell scripts referenced below are located in the [ARO-Scripts](https://msazure.visualstudio.com/DefaultCollection/AzureRedHatOpenShift/_git/ARO-Scripts) repository under the **powerShellActions** folder.

**Note:** ARO SREs do not have direct access to call Admin API during oncall. Instead, use the PowerShell scripts provided in the ARO-Scripts codebase as a wrapper to perform these operations.

#### 1. List Manifests for a Cluster

**PowerShell Script:**
```powershell
.\list-cluster-maintenancesets.ps1 `
  -location 'eastus2euap' `
  -resourceID '/subscriptions/60bf318d-6914-4105-a3b5-d0d2c10388c8/resourcegroups/my-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/my-cluster' `
  -icm 691536013
```

**Parameters:**
- `-location`: Azure region where the cluster is located (e.g., 'eastus2euap', 'eastus', 'westus')
- `-resourceID`: Full ARM resource ID of the target cluster
- `-icm`: ICM ticket number for tracking

**Response:**
Returns an array containing all maintenance manifests belonging to the specified cluster.

```json
{
  [
    {
      "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
      "state": "Completed",
      "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
      "runAfter": 6990500560,
      "runBefore": 7990500560
    },
    {
      "id": "a3c4d5e6-f7g8-9h0i-1j2k-3l4m5n6o7p8q",
      "state": "Pending",
      "maintenanceTaskID": "a4477c3a-ddbb-41a0-88e8-b5cda67b623a",
      "runAfter": 6990600000,
      "runBefore": 7990600000
    }
  ]
}
```

#### 2. Create New Manifest

**PowerShell Script:**
```powershell
.\create-maintenance-manifest.ps1 `
  -location 'eastus2euap' `
  -resourceId '/subscriptions/60bf318d-6914-4105-a3b5-d0d2c10388c8/resourcegroups/my-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/my-cluster' `
  -kubernetesObject '{"maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2"}' `
  -icm 691536013
```

**Parameters:**
- `-location`: Azure region where the cluster is located (e.g., 'eastus2euap', 'eastus', 'westus')
- `-resourceId`: Full ARM resource ID of the target cluster
- `-kubernetesObject`: JSON string containing manifest properties:
  - `maintenanceTaskID` (required): Task UUID to execute
  - `runAfter` (optional): Epoch time integer (Unix timestamp in seconds) - earliest time to execute
  - `runBefore` (optional): Epoch time integer (Unix timestamp in seconds) - deadline for execution
  - **Note:** `runAfter` and `runBefore` can both be present, both be omitted, or only one can be specified
  - ⚠️ **Important:** `runAfter` and `runBefore` must be provided as epoch time integers. You need to manually convert your desired datetime to epoch integer format. (This will be improved in future versions to accept human-readable datetime formats)
- `-icm`: ICM ticket number for tracking

**Converting to Epoch Time:**
- Use online tools like [epochconverter.com](https://www.epochconverter.com/) or PowerShell:
  ```powershell
  # Get current epoch time
  [int][double]::Parse((Get-Date -UFormat %s))

  # Convert specific datetime to epoch
  [int][double]::Parse((Get-Date "2024-12-31 23:59:59" -UFormat %s))
  ```

**Example with runAfter and runBefore:**
```powershell
.\create-maintenance-manifest.ps1 `
  -location 'eastus2euap' `
  -resourceId '/subscriptions/60bf318d-6914-4105-a3b5-d0d2c10388c8/resourcegroups/my-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/my-cluster' `
  -kubernetesObject '{"maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2", "runAfter": 6990500560, "runBefore": 7990500560}' `
  -icm 691536013
```

**Response:**
```json
{
  "id": "2b65ac7b-cbf5-4fdc-829d-a737e646d492",
  "state": "Pending",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "runAfter": 6990500560,
  "runBefore": 7990500560
}
```

#### 3. Cancel Manifest

**PowerShell Script:**
```powershell
.\cancel-maintenance-manifest.ps1 `
  -location 'eastus2euap' `
  -resourceId '/subscriptions/fe16a035-e540-4ab7-80d9-373fa9a3d6ae/resourcegroups/my-rg/providers/microsoft.redhatopenshift/openshiftclusters/my-cluster' `
  -manifestId 'b41749fc-af26-4ab7-b5a1-e03f3ee4cba6'
```

**Parameters:**
- `-location`: Azure region where the cluster is located
- `-resourceId`: Full ARM resource ID of the target cluster
- `-manifestId`: The ID of the manifest to cancel

**Important Notes:**
- Only manifests in `Pending` state can be cancelled
- Attempting to cancel a manifest in any other state (e.g., `InProgress`, `Completed`, `Failed`) will result in an error

**Response:**
```json
{
  "id": "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "state": "Cancelled",
  "priority": 0,
  "runAfter": 6990500560,
  "runBefore": 7990500560
}
```

#### 4. Delete Manifest

**PowerShell Script:**
```powershell
.\delete-maintenance-manifest.ps1 `
  -location 'eastus2euap' `
  -resourceId '/subscriptions/fe16a035-e540-4ab7-80d9-373fa9a3d6ae/resourcegroups/my-rg/providers/microsoft.redhatopenshift/openshiftclusters/my-cluster' `
  -manifestId 'b41749fc-af26-4ab7-b5a1-e03f3ee4cba6'
```

**Parameters:**
- `-location`: Azure region where the cluster is located
- `-resourceId`: Full ARM resource ID of the target cluster
- `-manifestId`: The ID of the manifest to delete

**Response:**
```
204 No Content
```

**Note:** Deleting a manifest is permanent and cannot be undone. This is typically used to clean up terminal-state manifests (`Completed`, `Failed`, `Cancelled`, `TimedOut`) that are no longer needed.

---

## Common Issues & Resolutions

### Issue 1: Manifests Stuck in Pending

**Symptoms:**
- Manifests in `Pending` state for > 30 minutes
- Queue length metric increasing

**Diagnosis Steps:**

1. Check global queue using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
   - Query for all manifests in `Pending` state
   - Get fields: `id`, `clusterResourceID`, `state`, `dequeues`, `runAfter`, `runBefore`
   - Retrieve at least 200 results

   **Look for in query results:**
   - Are all manifests showing `dequeues: 0`?
   - Are `runAfter` timestamps in the past?
   - Is one specific cluster dominating the queue?

2. Check actuator heartbeat metric in your monitoring system
   - Is heartbeat present within last 2 minutes?

**Resolution:**

| Situation | Action |
|-----------|--------|
| `actuator.heartbeat` missing > 5 min | **Contact platform team** - actuator is down |
| `dequeues: 0` on all manifests AND heartbeat present | **Contact platform team** - actuator not polling |
| Only one cluster affected | Investigate that cluster via PowerShell (list its manifests) |
| Manifests past `runBefore` timestamp | Wait for auto-timeout, then delete via PowerShell |

**Example Query Result Analysis:**
- Manifest ID `abc123` for cluster1: `state: "Pending"`, `dequeues: 0`, `runAfter: 1704067200`
  - ⚠️ Never attempted - indicates actuator issue
- Manifest ID `def456` for cluster1: `state: "Pending"`, `dequeues: 3`, `runAfter: 1704063600`
  - ✓ Being retried - indicates transient error

---

### Issue 2: Manifests Stuck in InProgress

**Symptoms:**
- Manifest in `InProgress` > 60 minutes
- Cluster may be stuck in `AdminUpdating` state

**Diagnosis Steps:**

1. Get manifest details using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
   - Query for the specific manifest by ID
   - Get fields: `id`, `state`, `statusText`, `dequeues`, `leaseExpiry`, `leaseOwner`

   **Example Query Result:**
   - Manifest ID: `stuck-manifest-id`
   - State: `InProgress`
   - StatusText: `Rotating certificates on control plane nodes`
   - Dequeues: `2`
   - LeaseExpiry: `1704067200` (compare with current time)
   - LeaseOwner: `actuator-vm-2`

2. Calculate how long manifest has been in InProgress:
   - Compare `leaseExpiry` to current Unix timestamp
   - If lease already expired, manifest should auto-transition back to `Pending`
   - If lease still valid and > 60 min since started, task is taking too long

**Resolution:**

| Time in InProgress | Lease Status | Action |
|-------------------|--------------|--------|
| < 60 minutes | Valid | Wait - task still executing |
| > 60 minutes | Valid | Cancel manifest to prevent further retries |
| > 60 minutes | Expired | Should auto-recover, monitor for transition to `Pending` |

**To Cancel Stuck Manifest:**

```powershell
.\cancel-maintenance-manifest.ps1 `
  -location '{region}' `
  -resourceId '/subscriptions/{sub}/resourceGroups/{rg}/providers/microsoft.redhatopenshift/openshiftclusters/{cluster}' `
  -manifestId '{manifestId}'
```

**Important:** Cancelling does NOT stop the currently executing task, it only prevents the manifest from being retried after completion.

---

### Issue 3: Tasks Repeatedly Failing

**Symptoms:**
- Manifest state cycling `Pending` → `InProgress` → `Pending`
- Eventually reaches `RetriesExceeded` or `Failed` state
- `statusText` field contains error messages

**Diagnosis Steps:**

1. Get manifest details using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
   - Query for the specific manifest by ID
   - Get fields: `id`, `maintenanceTaskID`, `state`, `statusText`, `dequeues`, `runAfter`

   **Example Query Result:**
   - Manifest ID: `failing-manifest`
   - MaintenanceTaskID: `9b741734-6505-447f-8510-85eb0ae561a2`
   - State: `RetriesExceeded`
   - StatusText: `TransientError: timeout connecting to kube-apiserver after 30s`
   - Dequeues: `5`
   - RunAfter: `1704067200`

2. Identify error type from `statusText`:

**Common Error Patterns:**

| statusText Pattern | Error Type | Meaning | SRE Action |
|-------------------|------------|---------|------------|
| `"TransientError: timeout"` | Transient | Network/API temporarily unavailable | Wait for auto-retry or investigate cluster connectivity |
| `"TransientError: 5xx"` | Transient | Cluster API temporary failure | Wait for auto-retry |
| `"TerminalError: 404"` | Terminal | Resource not found | Investigate cluster configuration |
| `"TerminalError: forbidden"` | Terminal | Permission denied | Check RBAC/service principal |
| `"certificate generation failed"` | Varies | Cert creation issue | Investigate cluster certificate authority |
| `"token expired"` (ACR task only) | Expected | Token nearing expiration detected | Normal - not an actual error, task is working correctly |

**Resolution:**

| Situation | Action |
|-----------|--------|
| `dequeues` < 5 and `TransientError` | Wait - will auto-retry |
| `dequeues` = 5 (RetriesExceeded) | Investigate root cause, fix cluster issue, recreate manifest |
| `TerminalError` at any attempt | Root cause needs fixing before recreate |
| All manifests of same `maintenanceTaskID` failing | **Contact platform team** - likely code bug |

---

### Issue 4: High Queue Length

**Symptoms:**
- `database.maintenancemanifests.queue.length` metric > 500

**Diagnosis Steps:**

1. Get queue overview using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
   - Query for all manifests in `Pending` or `InProgress` state
   - Retrieve at least 500 results
   - Get fields: `id`, `clusterResourceID`, `state`, `dequeues`, `runAfter`, `runBefore`, `maintenanceTaskID`, `statusText`

2. Analyze the query results to answer:
   - How many manifests are in each state? (count `state` field values)
   - Which clusters have the most manifests? (count by `clusterResourceID`)
   - Are there old manifests with past `runBefore` times?
   - What is the `dequeues` distribution? (0 = never tried, 5 = max retries)

**Example Analysis:**
After querying, you might find:
- 200 manifests with state: `Pending`, dequeues: `0`
- 50 manifests with state: `InProgress`
- 100 manifests with state: `TimedOut`
- 50 manifests with state: `Cancelled`

**Resolution:**

| Pattern | Root Cause | Action |
|---------|------------|--------|
| Most manifests have `dequeues: 0` | Actuator not processing | Check heartbeat metric, contact platform team if missing |
| Many `TimedOut` or `Cancelled` states | Queue not cleaned up | Delete terminal-state manifests |
| One cluster has 100+ manifests | Cluster-specific issue | Investigate that cluster, may need to cancel its manifests |
| Recent spike (check `runAfter` timestamps) | Temporary burst | Monitor, likely temporary |
| 50+ manifests with `dequeues: 5` | Systematic failure | Investigate common `statusText` patterns

---

## Diagnostic Procedures

### Procedure 1: Full MIMO Health Check

Use this procedure when you need to assess overall MIMO system health.

**Step 1: Check Global Queue Status**

Use internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards) to query global queue:
- Query for all manifests in `Pending` or `InProgress` state
- Retrieve at least 500 results
- Get fields: `id`, `clusterResourceID`, `state`, `dequeues`, `runAfter`, `runBefore`, `maintenanceTaskID`, `statusText`

Analyze the query results:
- Total number of manifests
- Distribution by `state` (group and count)
- Distribution by `clusterResourceID` (identify hot spots)
- Identify old manifests (compare `runAfter` with current timestamp)

**Step 2: Count Manifests by State**

From the query results above, manually or with tools count how many manifests are in each state:
- Pending: ?
- InProgress: ?
- Completed: ?
- Failed: ?
- RetriesExceeded: ?
- TimedOut: ?
- Cancelled: ?

**Step 3: Identify Top Clusters**

Count which `clusterResourceID` values appear most frequently in the queue. Clusters with 10+ manifests may have issues.

**Step 4: Find Old Manifests**

Compare current Unix timestamp with manifest `runAfter` values:
- Current time - runAfter > 7200 seconds (2 hours) = old manifest
- Count how many old manifests exist
- Identify if they're stuck or just waiting

**Step 5: Check Metrics**

In your monitoring system, verify:
- `actuator.heartbeat`: Present within last 60 seconds?
- `database.maintenancemanifests.queue.length`: Matches internal tooling query count?
- `mimo.actuator.workers.active.count`: > 0 if queue has work?

**Health Assessment:**

| Metric | Healthy | Unhealthy |
|--------|---------|-----------|
| Total queue | < 100 | > 500 |
| Pending with dequeues=0 | < 10 | > 50 |
| InProgress | < 10 | > 20 |
| Failed/RetriesExceeded | < 5% of total | > 10% of total |
| Heartbeat | Present | Missing > 5 min |

---

### Procedure 2: Investigate Specific Cluster

Use this when one cluster has multiple MIMO issues or is stuck in AdminUpdating state.

**Step 1: List All Manifests for Cluster**

```powershell
.\list-cluster-maintenancesets.ps1 `
  -location '{region}' `
  -resourceID '/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{cluster}' `
  -icm {icmNumber}
```

Example Response:
```json
{
  "value": [
    {
      "id": "manifest-1",
      "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
      "state": "Completed",
      "dequeues": 1
    },
    {
      "id": "manifest-2",
      "maintenanceTaskID": "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6",
      "state": "InProgress",
      "dequeues": 2,
      "leaseExpiry": 1704067200
    },
    {
      "id": "manifest-3",
      "maintenanceTaskID": "082978ce-3700-4972-835f-53d48658d291",
      "state": "RetriesExceeded",
      "statusText": "TransientError: connection timeout",
      "dequeues": 5
    }
  ]
}
```

**Step 2: Identify Problem Manifests**

From the response, look for:
- Any in `InProgress` state
- Any in `Failed` or `RetriesExceeded` state
- Any with high `dequeues` count (3-5)

**Step 3: Get Details for Problem Manifests**

For each problem manifest, get full details using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
- Query for specific manifest IDs
- Get fields: `id`, `state`, `statusText`, `leaseExpiry`, `dequeues`, `maintenanceTaskID`

Review `statusText` for error messages and `leaseExpiry` for stuck tasks.

**Step 4: Assess Cluster Pattern**

- Are all tasks failing for this cluster? → Cluster connectivity/health issue
- Is only one task type failing? → Task-specific issue
- Multiple manifests stuck in InProgress? → Cluster may be unresponsive

---

## Recovery Procedures

### Recovery 1: Cancel Stuck Manifest

**When to use:** Manifest stuck in `InProgress` state for > 60 minutes with valid lease

**Step 1: Verify Manifest is Stuck**

Use internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards) to query the manifest:
- Query for the specific manifest by ID
- Get fields: `state`, `leaseExpiry`, `leaseOwner`, `dequeues`

Verify:
- `state`: "InProgress"
- `leaseExpiry`: Still in future (Unix timestamp > current time)
- Duration: Lease acquired > 60 minutes ago

**Step 2: Cancel the Manifest**

```powershell
.\cancel-maintenance-manifest.ps1 `
  -location '{region}' `
  -resourceId '/subscriptions/{sub}/resourceGroups/{rg}/providers/microsoft.redhatopenshift/openshiftclusters/{cluster}' `
  -manifestId '{manifestId}'
```

Example Response:
```json
{
  "id": "{manifestId}",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "state": "Cancelled",
  "priority": 0,
  "runAfter": 6990500560,
  "runBefore": 7990500560
}
```

**Step 3: Verify Cancellation**

Wait 10 seconds, then verify using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
- Query for the manifest by ID
- Confirm `state` is now `Cancelled`

**Important:** Cancelling does NOT stop the currently executing task. It only prevents the manifest from being retried after the current execution completes or fails.

---

### Recovery 2: Clean Up Terminal-State Manifests

**When to use:** High queue length with many `TimedOut`, `Cancelled`, `Completed`, or old `Failed` manifests

**Step 1: Identify Manifests to Delete**

Use internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards) to query manifests:
- Query for manifests across all clusters
- Get at least 500 results
- Get fields: `id`, `clusterResourceID`, `state`, `runAfter`, `statusText`

From the query results, identify manifests where:
- `state` = "TimedOut" OR
- `state` = "Cancelled" OR
- `state` = "Completed" (if older than 7 days) OR
- `state` = "Failed" AND not being investigated

**Step 2: Delete Each Manifest**

For each manifest identified for deletion:

```powershell
.\delete-maintenance-manifest.ps1 `
  -location '{region}' `
  -resourceId '{clusterResourceID}' `
  -manifestId '{manifestId}'
```

Response: `204 No Content`

**Step 3: Verify Queue Length Decrease**

After deletion, check the queue length metric in your monitoring system or re-query using internal tooling to confirm the count has decreased.

**Note:** Deleting manifests is permanent and cannot be undone. Only delete manifests in terminal states that are no longer needed.

---

### Recovery 3: Recreate Failed Manifest

**When to use:**
- Manifest in `Failed` or `RetriesExceeded` state
- Root cause has been fixed
- Task needs to be attempted again

**Step 1: Verify Root Cause is Fixed**

Before recreating, ensure the issue that caused failure has been resolved. Check the original manifest's `statusText` using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
- Query for the failed manifest by ID
- Get fields: `statusText`, `state`, `dequeues`, `maintenanceTaskID`

Review `statusText` and confirm the underlying issue (network, permissions, cluster health) is resolved.

**Step 2: Create New Manifest**

```powershell
.\create-maintenance-manifest.ps1 `
  -location '{region}' `
  -resourceId '/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{cluster}' `
  -kubernetesObject '{"maintenanceTaskID": "{task-id-from-failed-manifest}"}' `
  -icm {icmNumber}
```

**Note:** You can optionally specify `runAfter` and/or `runBefore` in the `kubernetesObject` if needed. See the Create Manifest section for details.

Response:
```json
{
  "id": "new-manifest-id",
  "state": "Pending",
  "maintenanceTaskID": "{task-id}",
  "runAfter": 6990500560,
  "runBefore": 7990500560
}
```

**Step 3: Monitor New Manifest**

Wait 5-10 minutes and check the new manifest's progress using internal tooling (Geneva Actions, Admin Portal, Kusto, or monitoring dashboards):
- Query for the new manifest by ID
- Get fields: `state`, `statusText`, `dequeues`

Verify:
- `state` transitions from `Pending` → `InProgress` → `Completed`
- `statusText` does not contain errors
- `dequeues` increments as expected

**Step 4: Delete Old Failed Manifest (Optional)**

Once the new manifest completes successfully, you can delete the old failed one:

```powershell
.\delete-maintenance-manifest.ps1 `
  -location '{region}' `
  -resourceId '/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{cluster}' `
  -manifestId '{failedManifestId}'
```

---

## Related Documentation

For additional MIMO information, see the following documentation in this folder:

### For SREs and Operators:
- **[MIMO Overview (README.md)](./README.md)** - High-level overview of MIMO architecture, components, and concepts. Read this first to understand the fundamentals of how MIMO works.
- **[Admin API Reference (admin-api.md)](./admin-api.md)** - Technical reference for Admin API endpoints. Note that SREs should use PowerShell scripts instead of calling these APIs directly.
- **[Actuator Documentation (actuator.md)](./actuator.md)** - Detailed explanation of the Actuator component, including a flowchart showing how tasks are processed from the queue.

### For Developers:
- **[Writing MIMO Tasks (writing-tasks.md)](./writing-tasks.md)** - Guide for developers who need to create new MIMO maintenance tasks. Covers writing Steps, assembling Tasks, testing, and error handling patterns.
- **[Local Development (local-dev.md)](./local-dev.md)** - Instructions for running and testing MIMO locally during development.
- **[Scheduler Documentation (scheduler.md)](./scheduler.md)** - Information about the planned Scheduler component (not yet implemented).

### External Resources:
- **[ARO-Scripts Repository](https://msazure.visualstudio.com/DefaultCollection/AzureRedHatOpenShift/_git/ARO-Scripts)** - PowerShell scripts used by SREs for MIMO operations (located in `powerShellActions` folder).