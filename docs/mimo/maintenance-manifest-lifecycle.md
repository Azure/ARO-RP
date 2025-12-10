# MIMO Maintenance Manifest Lifecycle

## What is a Maintenance Manifest?

A **Maintenance Manifest** is a work order that tells MIMO to execute a maintenance task on a specific cluster. It serves as the bridge between administrative intent (e.g., "rotate this cluster's certificates") and actual execution by the MIMO Actuator.

### Key Properties

| Property | Description |
|----------|-------------|
| **ID** | Unique identifier (UUID) for the manifest |
| **ClusterResourceID** | The ARM resource ID of the target cluster |
| **MaintenanceTaskID** | UUID of the task to execute |
| **State** | Current lifecycle state (Pending, InProgress, etc.) |
| **StatusText** | Human-readable result or error message |
| **Priority** | Execution order (lower number = higher priority, default: 0) |
| **RunAfter** | Earliest time the task can start (Unix timestamp) |
| **RunBefore** | Deadline - task times out if not started by this time (Unix timestamp) |

### Default Behavior

- **RunAfter**: Defaults to current time if not specified
- **RunBefore**: Defaults to 7 days from creation if not specified
- **Priority**: Defaults to 0 (highest priority)
- Manifests are stored in CosmosDB and scoped to a specific cluster

## Manifests vs Tasks

| Concept | Description |
|---------|-------------|
| **Task** | A reusable template defining *what* to do (e.g., rotate certificates) |
| **Manifest** | An instance specifying *when* and *where* to run a task |

One Task → Many Manifests (one per cluster/execution)

## Lifecycle Flow

```
                          +-------------+
                          |   CREATED   |
                          +------+------+
                                 |
                                 v
+----------------+         +-----+------+         +----------------+
|                |         |            |         |                |
|   TIMED OUT    |<--------|  PENDING   |-------->|   CANCELLED    |
|                |         |            |         |                |
+----------------+         +-----+------+         +----------------+
                                 |
                                 v
                          +------+------+
                          |             |
                          | IN PROGRESS |
                          |             |
                          +------+------+
                                 |
          +----------------------+----------------------+
          |                      |                      |
          v                      v                      v
   +------+------+        +------+------+        +------+------+
   |             |        |             |        |   PENDING   |
   |  COMPLETED  |        |   FAILED    |        |   (retry)   |
   |             |        |             |        +------+------+
   +-------------+        +-------------+               |
                                                        v (after 5 retries)
                                                 +------+------+
                                                 |   RETRIES   |
                                                 |   EXCEEDED  |
                                                 +-------------+
```

## States

| State | Description |
|-------|-------------|
| **Pending** | Waiting to be picked up by actuator |
| **InProgress** | Task is currently executing |
| **Completed** | Task succeeded *(terminal)* |
| **Failed** | Task hit a permanent error *(terminal)* |
| **RetriesExceeded** | Failed after 5 retry attempts *(terminal)* |
| **TimedOut** | Missed execution deadline *(terminal)* |
| **Cancelled** | Manually cancelled *(terminal)* |

## Error Types

| Error Type | Result |
|------------|--------|
| **Transient** | Returns to Pending, retries up to 5 times |
| **Terminal** | Immediately fails, no retry |
