# Scheduler Calendar and Selectors Reference

## Calendar Format

The Scheduler uses a calendar expression format based on [systemd calendar events](https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html#Calendar%20Events) to define when maintenance schedules trigger.

### Syntax

```
[Weekday] Year-Month-Day Hour:Minute[:Second]
```

The seconds component is optional. If provided, it must be `00`. Per-second granularity is not supported.

| Component | Position | Required | Format |
|-----------|----------|----------|--------|
| Weekday | Prefix (space-separated from date) | No | Three-letter abbreviation(s), comma-separated |
| Year | Before first `-` | Yes | Four-digit year, `*` for any, or comma-separated list |
| Month | Between first and second `-` | Yes | 1-12, `*` for any, or comma-separated list |
| Day | After second `-` | Yes | 1-31, `*` for any, or comma-separated list |
| Hour | Before first `:` | Yes | 0-23, `*` for any, or comma-separated list |
| Minute | After first `:` | Yes | `0`, `15`, `30`, or `45`, or `*` for any, or comma-separated list of allowed values |
| Second | After second `:` (if present) | No | Must be `00` if provided; per-second granularity is unsupported |

### Field Ranges

| Field | Min | Max | Notes |
|-------|-----|-----|-------|
| Year | 2026 | -- | Years before 2026 are rejected. There is no enforced upper bound. Matching is exact: specifying `2026` matches only 2026, not subsequent years. Use `*` for any year. |
| Month | 1 | 12 | |
| Day | 1 | 31 | Days beyond the month's range are handled gracefully |
| Hour | 0 | 23 | |
| Minute | 0, 15, 30, 45 | | Only these four values are allowed. The wildcard `*` matches any minute. |

### Weekday Abbreviations

| Abbreviation | Day |
|-------------|-----|
| `Mon` | Monday |
| `Tue` | Tuesday |
| `Wed` | Wednesday |
| `Thu` | Thursday |
| `Fri` | Friday |
| `Sat` | Saturday |
| `Sun` | Sunday |

### Wildcards and Lists

- **Wildcard (`*`)**: Matches any value for that field. For example, `*-*-* 00:00` means "every day at midnight."
- **Comma-separated list**: Matches any of the listed values. For example, `Mon,Wed,Fri` means Monday, Wednesday, or Friday. Multiple values can also be used in numeric fields: `*-*-1,15 00:00` means the 1st and 15th of every month. For minutes, only `0`, `15`, `30`, and `45` may appear in the list.

### Calendar Examples

| Expression | Meaning |
|------------|---------|
| `*-*-* 00:00` | Every day at midnight UTC |
| `*-*-* 06:00` | Every day at 06:00 UTC |
| `*-*-* *:00` | Every hour on the hour |
| `*-*-* *:15` | Every hour at quarter past |
| `Mon *-*-* 00:00` | Every Monday at midnight UTC |
| `Mon,Thu *-*-* 00:00` | Every Monday and Thursday at midnight UTC |
| `Mon,Wed,Fri *-*-* 12:00` | Monday, Wednesday, Friday at noon UTC |
| `*-*-1 00:00` | First day of every month at midnight UTC |
| `*-*-1,15 06:00` | 1st and 15th of every month at 06:00 UTC |
| `*-1-* 00:00` | Every day in January at midnight UTC |
| `*-3,6,9,12-1 00:00` | First day of each quarter at midnight UTC |
| `2026-*-* 00:00` | Every day in 2026 only, at midnight UTC |
| `Sat,Sun *-*-* 02:00` | Every weekend at 02:00 UTC |
| `*-*-* 00:00:00` | Equivalent to `*-*-* 00:00` (seconds are optional) |
| `*-*-* *:0,30` | Every hour at the top and bottom of the hour |

### Time Zone Handling

All times are in UTC. The Scheduler does not support time zone specifications in calendar expressions. If you need to target a specific local time, convert to UTC before defining the schedule.

| Target | Local Time | UTC Equivalent (Winter) | UTC Equivalent (Summer/DST) |
|--------|-----------|------------------------|---------------------------|
| US East business end | 18:00 EST | 23:00 UTC | 22:00 UTC (EDT) |
| US West business end | 18:00 PST | 02:00 UTC (+1 day) | 01:00 UTC (+1 day, PDT) |
| EU West business end | 18:00 CET | 17:00 UTC | 16:00 UTC (CEST) |

### Differences from systemd Calendar Events

The Scheduler's calendar format is inspired by but not identical to systemd's `OnCalendar` syntax:

| Feature | systemd | Scheduler |
|---------|---------|-----------|
| Weekday prefix | Supported | Supported |
| Comma-separated values | Supported | Supported |
| Seconds field | Required | Optional; must be `00` if present |
| Minute granularity | Any value 0-59 | Restricted to `0`, `15`, `30`, `45` |
| Range syntax (e.g., `Mon..Fri`) | Supported | Not supported |
| Repeat syntax (e.g., `*-*-* *:00/15:00`) | Supported | Not supported |
| Multiple expressions (separated by `;`) | Supported | Not supported |
| `~` (last day of month) | Supported | Not supported |

If you need functionality not supported by the Scheduler's calendar format, create multiple schedules with different expressions.

## Selectors

Selectors determine which clusters a `MaintenanceSchedule` applies to. They filter the fleet based on cluster and subscription properties.

### Selector Syntax

Each selector is a JSON object with the following fields:

```json
{
  "key": "SELECTOR_KEY",
  "operator": "OPERATOR",
  "value": "SINGLE_VALUE",
  "values": ["VALUE_1", "VALUE_2"]
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | The cluster property to match against (see [Well-Known Selector Keys](#well-known-selector-keys)) |
| `operator` | Yes | The comparison operator (see [Available Operators](#available-operators)) |
| `value` | Conditional | Single string value; required for `eq` operator, must not be provided for `in`/`notin` |
| `values` | Conditional | Array of string values; required for `in`/`notin` operators, must not be provided for `eq` |

Replace `SELECTOR_KEY`, `OPERATOR`, `SINGLE_VALUE`, `VALUE_1`, and `VALUE_2` with appropriate values.

### Available Operators

| Operator | Description | Requires |
|----------|-------------|----------|
| `eq` | Exact string equality match | `value` (single string) |
| `in` | Value is contained in the provided list | `values` (string array, at least one element) |
| `notin` | Value is not contained in the provided list | `values` (string array, at least one element) |

### Well-Known Selector Keys

The following keys are defined in the Scheduler's cluster cache (see [`pkg/mimo/scheduler/selectors.go`](../../pkg/mimo/scheduler/selectors.go)). All values are strings.

| Key | Description | Example Values |
|-----|-------------|---------------|
| `resourceID` | Full ARM resource ID of the cluster (lowercased) | `/subscriptions/.../openshiftclusters/mycluster` |
| `subscriptionID` | Azure subscription ID containing the cluster | `00000000-0000-0000-0000-000000000000` |
| `subscriptionState` | Registration state of the subscription | `Registered`, `Warned`, `Suspended` |
| `authenticationType` | Cluster authentication mechanism | `WorkloadIdentity`, `ServicePrincipal` |
| `architectureVersion` | Cluster architecture version (integer as string) | `1`, `2` |
| `provisioningState` | Current provisioning state of the cluster | `Succeeded`, `Failed`, `Creating` |
| `outboundType` | Network outbound routing type | `Loadbalancer`, `UserDefinedRouting` |
| `APIServerVisibility` | API server endpoint visibility | `Public`, `Private` |
| `isManagedDomain` | Whether the cluster uses an ARO-managed domain | `true`, `false` |

The per-cluster selectors diagnostic endpoint (`GET /admin/RESOURCE_ID/selectors`) can be used to inspect the actual selector values for a given cluster. See [Admin API](./admin-api.md).

### Selector Examples

#### Match all clusters in registered subscriptions

```json
[
  {
    "key": "subscriptionState",
    "operator": "in",
    "values": ["Registered"]
  }
]
```

#### Match clusters in registered or warned subscriptions

```json
[
  {
    "key": "subscriptionState",
    "operator": "in",
    "values": ["Registered", "Warned"]
  }
]
```

#### Exclude a specific subscription

```json
[
  {
    "key": "subscriptionState",
    "operator": "in",
    "values": ["Registered"]
  },
  {
    "key": "subscriptionID",
    "operator": "notin",
    "values": ["EXCLUDED_SUBSCRIPTION_ID"]
  }
]
```

Replace `EXCLUDED_SUBSCRIPTION_ID` with the subscription to exclude.

#### Target a single specific cluster

```json
[
  {
    "key": "resourceID",
    "operator": "eq",
    "value": "/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME"
  }
]
```

Replace `SUBSCRIPTION_ID`, `RESOURCE_GROUP`, and `CLUSTER_NAME` with the target cluster's values. The resource ID must be lowercased.

#### Target clusters in a specific subscription

```json
[
  {
    "key": "subscriptionID",
    "operator": "eq",
    "value": "TARGET_SUBSCRIPTION_ID"
  }
]
```

Replace `TARGET_SUBSCRIPTION_ID` with the target subscription ID.

#### Target only Workload Identity clusters on managed domains

```json
[
  {
    "key": "subscriptionState",
    "operator": "in",
    "values": ["Registered"]
  },
  {
    "key": "authenticationType",
    "operator": "eq",
    "value": "WorkloadIdentity"
  },
  {
    "key": "isManagedDomain",
    "operator": "eq",
    "value": "true"
  }
]
```

#### Exclude clusters in a non-terminal provisioning state

```json
[
  {
    "key": "subscriptionState",
    "operator": "in",
    "values": ["Registered"]
  },
  {
    "key": "provisioningState",
    "operator": "notin",
    "values": ["Creating", "Deleting"]
  }
]
```

### Selector Evaluation Rules

1. **All selectors use AND logic.** A cluster must match every selector in the list to be included.
2. **Empty selectors match no clusters.** A schedule with zero selectors is rejected by the API with `400 Bad Request`.
3. **Unknown keys cause an error.** If a selector references a key not present in the cluster's selector data, the cluster is skipped and an error is logged.
4. **String comparison is exact.** All comparisons are case-sensitive string matches. The `resourceID` key is always lowercased in the cluster cache.
5. **Selectors are evaluated per cluster, per schedule.** Each Scheduler poll cycle re-evaluates selectors against the current cluster cache, so changes to cluster or subscription state are reflected on the next cycle.

## Combining Calendar and Selectors

When designing a schedule, consider both the timing (calendar) and targeting (selectors) together. The task IDs used in these examples are defined in [`pkg/mimo/const.go`](../../pkg/mimo/const.go).

### Example: Conservative Weekly Rollout

A schedule that runs TLS certificate rotation every Monday, spread across 24 hours, targeting only registered subscriptions:

```json
{
  "state": "Enabled",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "schedule": "Mon *-*-* 00:00",
  "lookForwardCount": 4,
  "scheduleAcross": "24h",
  "selectors": [
    {
      "key": "subscriptionState",
      "operator": "in",
      "values": ["Registered"]
    }
  ]
}
```

The result: each Monday at midnight UTC, the Scheduler begins creating manifests. Each cluster's manifest has a `runAfter` time calculated as Monday 00:00 UTC plus its deterministic offset within the 24-hour `scheduleAcross` window. The [Actuator](./actuator.md) executes each manifest after its `runAfter` time.

### Example: Testing on a Single Cluster

A schedule targeting a single cluster for validation before fleet-wide rollout:

```json
{
  "state": "Enabled",
  "maintenanceTaskID": "9b741734-6505-447f-8510-85eb0ae561a2",
  "schedule": "*-*-* 12:00",
  "lookForwardCount": 1,
  "scheduleAcross": "0s",
  "selectors": [
    {
      "key": "resourceID",
      "operator": "eq",
      "value": "/subscriptions/SUBSCRIPTION_ID/resourcegroups/RESOURCE_GROUP/providers/microsoft.redhatopenshift/openshiftclusters/CLUSTER_NAME"
    }
  ]
}
```

Replace `SUBSCRIPTION_ID`, `RESOURCE_GROUP`, and `CLUSTER_NAME` with your test cluster's values.

The result: a manifest is created daily at noon UTC for the single specified cluster with `scheduleAcross` of `0s` (no spread, immediate execution).
