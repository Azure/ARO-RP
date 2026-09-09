# Azure Product Constraints Guide

Read this when adding VM sizes, modifying validation logic, or changing cluster topology rules. These are hard constraints from the [Azure Red Hat OpenShift support policy](https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4) — violating them is a production bug.

## Cluster Topology

- **Exactly 3 master nodes** — cannot be added, removed, or replaced by customers
- **3 worker nodes minimum**
- **SLA**: 99.95% availability

## VM Size Constraints

### Control Plane (Masters)

Full list: `pkg/api/validate/vm.go` → `supportedMasterVmSizes` map.

### Workers

Full list: `pkg/api/validate/vm.go` → `supportedWorkerVmSizes` map.

### Disk Size

Minimum **128 GiB** for worker nodes. Enforced in `validate.DiskSizeIsValid()`.

## Authentication Modes

Two mutually exclusive modes, determined by `UsesWorkloadIdentity()` on the cluster object:

1. **Service Principal** — traditional client ID + secret
2. **Workload Identity** — managed identity (platform workload identity role sets)

**CLI helper**: `az aro identity get-required` outputs the identity and role assignment commands needed for creating a cluster with managed identities.

**Error handling**: When cluster MSI role assignments are missing over platform workload identities, the RP returns a `400 InvalidClusterMSIPermissions` error (not a 500 timeout). This tells the customer which identity and role are missing permissions. See `pkg/frontend/openshiftcluster_putorpatch.go` → `ValidateClusterUserAssignedIdentity()`.

## Networking

- Clusters require direct outbound internet access, or `UserDefinedRouting`
- NSGs are managed by the service — customers cannot modify them unless using bring-your-own NSG feature

## Adding New VM Sizes

See `docs/agent-guides/api-type-system.md` → "Adding New VM Sizes Checklist" for the file-by-file checklist.

Also see: `docs/adding-new-instance-types.md` for the full procedure.
