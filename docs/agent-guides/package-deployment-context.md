# Package Deployment Context Guide

Read this when modifying packages whose behavior depends on WHERE they run. Getting the runtime context wrong causes bugs that look correct locally but break in production.

## Runtime Context Map

### RP Control Plane (Azure VMSS)

These packages run on RP infrastructure VMs managed by Microsoft:

<<<<<<< HEAD
| Package        | Role                                                                                    |
| -------------- | --------------------------------------------------------------------------------------- |
| `pkg/frontend` | ARM REST API handlers (PUT/GET/DELETE/PATCH/POST)                                       |
| `pkg/backend`  | Async cluster operations worker (polls CosmosDB, processes lifecycle)                   |
| `pkg/cluster`  | **Production** cluster lifecycle orchestrator (Azure infra, certs, operator deployment) |
| `pkg/monitor`  | Cluster health monitoring                                                               |
| `pkg/gateway`  | Gateway service                                                                         |
| `pkg/portal`   | Portal backend                                                                          |
| `pkg/database` | CosmosDB wrapper layer                                                                  |
| `pkg/env`      | Environment shims (prod/dev/test)                                                       |
=======
| Package | Role |
|---------|------|
| `pkg/frontend` | ARM REST API handlers (PUT/GET/DELETE/PATCH/POST) |
| `pkg/backend` | Async cluster operations worker (polls CosmosDB, processes lifecycle) |
| `pkg/cluster` | **Production** cluster lifecycle orchestrator (Azure infra, certs, operator deployment) |
| `pkg/monitor` | Cluster health monitoring |
| `pkg/gateway` | Gateway service |
| `pkg/portal` | Portal backend |
| `pkg/database` | CosmosDB wrapper layer |
| `pkg/env` | Environment shims (prod/dev/test) |
>>>>>>> cc22b18c7 (Restructure CLAUDE.md with progressive disclosure via agent guides)

### Customer OpenShift Cluster

These packages run inside the customer's OpenShift cluster as the ARO operator:

<<<<<<< HEAD
| Package                    | Role                                                     |
| -------------------------- | -------------------------------------------------------- |
=======
| Package | Role |
|---------|------|
>>>>>>> cc22b18c7 (Restructure CLAUDE.md with progressive disclosure via agent guides)
| `pkg/operator/controllers` | 26 Kubernetes controllers managing cluster configuration |

Controllers include: alertwebhook, autosizednodes, banner, checkers, cloudproviderconfig, clusteroperatoraro, cpms, dnsmasq, etchosts, genevalogging, guardrails, imageconfig, ingress, machine, machinehealthcheck, machineset, monitoring, muo, node, previewfeature, pullsecret, rbac, routefix, storageaccounts, subnets, workaround.

### CI/Dev Only (NOT in production binary)

These packages exist solely for testing and development:

<<<<<<< HEAD
| Package            | Role                                                   |
| ------------------ | ------------------------------------------------------ |
| `pkg/util/cluster` | Test cluster creation tooling (Viper config, env vars) |
| `hack/cluster`     | CLI tool for manual cluster creation                   |
| `test/e2e`         | E2E test suite (Ginkgo v2 + Gomega)                    |

### RP Infrastructure Deployment

| Package      | Role                                                                   |
| ------------ | ---------------------------------------------------------------------- |
=======
| Package | Role |
|---------|------|
| `pkg/util/cluster` | Test cluster creation tooling (Viper config, env vars) |
| `hack/cluster` | CLI tool for manual cluster creation |
| `test/e2e` | E2E test suite (Ginkgo v2 + Gomega) |

### RP Infrastructure Deployment

| Package | Role |
|---------|------|
>>>>>>> cc22b18c7 (Restructure CLAUDE.md with progressive disclosure via agent guides)
| `pkg/deploy` | RP VMSS, CosmosDB, DNS, network infra deployment config (`aro deploy`) |

## The Three "Cluster" Packages

This is the most common source of confusion:

<<<<<<< HEAD
| Package                | `pkg/cluster`                              | `pkg/util/cluster`                 | `pkg/deploy`              |
| ---------------------- | ------------------------------------------ | ---------------------------------- | ------------------------- |
| **Used in production** | Yes                                        | No                                 | Yes (deploy only)         |
| **Called by**          | `pkg/backend`                              | `test/e2e`, `hack/cluster`         | `cmd/aro deploy`          |
| **Does what**          | Orchestrates cluster install/update/delete | Creates test clusters              | Deploys RP infrastructure |
| **VM size type**       | `api.VMSize`                               | `string` (cast at usage)           | N/A                       |
| **Config source**      | CosmosDB documents                         | Viper (env vars)                   | Azure Live Config         |
| **Requires**           | Running RP                                 | `CI=true` or `RP_MODE=development` | Azure credentials         |
=======
| Package | `pkg/cluster` | `pkg/util/cluster` | `pkg/deploy` |
|---------|--------------|-------------------|-------------|
| **Used in production** | Yes | No | Yes (deploy only) |
| **Called by** | `pkg/backend` | `test/e2e`, `hack/cluster` | `cmd/aro deploy` |
| **Does what** | Orchestrates cluster install/update/delete | Creates test clusters | Deploys RP infrastructure |
| **VM size type** | `api.VMSize` | `string` (cast at usage) | N/A |
| **Config source** | CosmosDB documents | Viper (env vars) | Azure Live Config |
| **Requires** | Running RP | `CI=true` or `RP_MODE=development` | Azure credentials |
>>>>>>> cc22b18c7 (Restructure CLAUDE.md with progressive disclosure via agent guides)

**Rule**: CI-specific behavior (VM size retry on quota errors, VM size shuffling) belongs in `pkg/util/cluster/ClusterConfig`, driven by explicit config fields. Never add ad-hoc `os.Getenv("CI")` checks in `pkg/cluster` or `pkg/frontend`.

## Frontend Request Lifecycle

### Async Model (cluster mutations)

```
Client PUT/DELETE → pkg/frontend (validates, writes to CosmosDB with non-terminal state)
                              ↓
                    CosmosDB (Creating/Updating/Deleting)
                              ↓
                    pkg/backend (polls every 10s, dequeues, leases document)
                              ↓
                    pkg/cluster (orchestrates Azure infra + OpenShift install)
                              ↓
                    CosmosDB (Succeeded/Failed)
```

### PUT/PATCH Flow (`openshiftcluster_putorpatch.go`)

1. Validate subscription state (must be `Registered`)
2. Get existing cluster document from CosmosDB (or create skeleton)
3. Check provisioning state is terminal
4. Enrich with current cluster data (10s timeout)
5. Convert internal → external, strip read-only fields, unmarshal request body
6. **CREATE**: Full validation (static + SKU + quota + providers)
7. **UPDATE**: Static validation only
8. Convert external → internal, preserve immutable fields
9. Set provisioningState to `Creating`/`Updating`/`AdminUpdating`
10. Persist with `RetryOnPreconditionFailed` (optimistic concurrency)
11. Return 201/200 with `Location` and `Azure-AsyncOperation` headers

### Backend Dequeue

```sql
SELECT * FROM OpenShiftClusters doc
WHERE doc.openShiftCluster.properties.provisioningState IN ("Creating", "Deleting", "Updating", "AdminUpdating")
AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000
```

Max 100 concurrent workers. Lease renewed via `renewLease` pretrigger with optimistic concurrency.

## CosmosDB Document Types

| Document                                  | Partition key                | Purpose                               |
| ----------------------------------------- | ---------------------------- | ------------------------------------- |
| `OpenShiftClusterDocument`                | Subscription ID (lowercased) | Main cluster state                    |
| `AsyncOperationDocument`                  | —                            | Async operation tracking              |
| `SubscriptionDocument`                    | —                            | Subscription registration state       |
| `OpenShiftVersionDocument`                | —                            | OCP version config (changefeed)       |
| `PlatformWorkloadIdentityRoleSetDocument` | —                            | Workload identity config (changefeed) |
| `BillingDocument`                         | —                            | Billing data                          |
| `MaintenanceManifestDocument`             | —                            | Maintenance manifests                 |

Generated code: `pkg/database/cosmosdb/zz_generated_*` (auto-generated by `go-cosmosdb`).

Fake implementations for unit tests: `test/database/inmemory.go` → `NewFakeOpenShiftClusters()`.

## Viper Config Flow (test/CI only)

`pkg/util/cluster/ClusterConfig` uses Viper for env var-based configuration:

```
Environment variables → viper.AutomaticEnv() → viper.Unmarshal(&conf) → mapstructure tags → struct fields
```

Callers: `test/e2e/setup.go`, `hack/cluster/cluster.go`. NOT used by production RP.

Production config source: Azure Live Config (`pkg/util/liveconfig/`).

## Admin API: VM Resize Orchestration

The admin API exposes VM resize at `POST /admin/{resourceId}/resize?vmName={vm}&vmSize={sku}` (`pkg/frontend/admin_openshiftcluster_vmresize.go`). This endpoint resizes a single VM in the cluster resource group.

The full control plane resize is orchestrated externally by a Geneva Action (`Compute-ARO-GenevaActions` repo) called `ResizeControlPlanVMs`. It iterates through master nodes in reverse order and for each node:

1. **Cordon** node (disable scheduling)
2. **Drain** node (evacuate pods, retry 3x with 2s backoff)
3. **Stop** VM (Azure operation)
4. **Resize** VM (Azure operation, only works on stopped VM)
5. **Start** VM (Azure operation)
6. **Wait** for node Ready (poll every 5s, up to 30 min)
7. **Uncordon** node (re-enable scheduling)
8. **Update** Machine and Node objects with new SKU

Machines are processed in reverse order (infra-2 → infra-1 → infra-0) for quorum safety: if something fails, earlier machines remain unchanged.

**Pre-flight validations** should run _before_ any cordon/drain to avoid leaving the cluster degraded. The admin API provides dedicated `preResizeValidation` and `postResizeValidation` endpoints that check: service principal validity (via operator condition), API server health (kube-apiserver operator status), and SKU availability (Azure Compute ResourceSkusClient).

The Geneva Action disables HTTP timeouts for these operations. There are no explicit timeouts in the RP admin API context either.

## MIMO (Managed Infrastructure Maintenance Operator)

An internal maintenance automation system that handles tasks previously requiring manual intervention (certificates, tokens, operator flags). It focuses on "managed infrastructure" — components deployed by the RP and ARO Operator — rather than Hive-managed components.

The Actuator (`cmd/aro` mode `mimo-actuator`) executes maintenance tasks from a work queue of Maintenance Manifests stored in CosmosDB. As of [ARO-25080], the actuator no longer dequeues the `OpenShiftClusterDocument` when performing tasks — it reads the document without acquiring a lease to avoid contention as task volume scales up.

MIMO tasks are triggered via Geneva Actions calling the Admin API. A Scheduler component (for automating creation of new tasks) is currently in design for a future release.

Links:
- <https://github.com/Azure/ARO-RP/tree/master/docs/mimo>
- <https://eng.ms/docs/cloud-ai-platform/azure-core/azure-cloud-native-and-management-platform/control-plane-bburns/azure-red-hat-openshift/azure-redhat-openshift-team-doc/tsgs/mimo/mimo-101>
