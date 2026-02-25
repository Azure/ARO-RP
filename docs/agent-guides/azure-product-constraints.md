# Azure Product Constraints Guide

Read this when adding VM sizes, modifying validation logic, or changing cluster topology rules. These are hard constraints from the [Azure Red Hat OpenShift support policy](https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4) — violating them is a production bug.

## Cluster Topology

- **Exactly 3 master nodes** — cannot be added, removed, or replaced by customers
- **3-250 worker nodes** — customer-configurable
- **SLA**: 99.95% availability

## VM Size Constraints

### Control Plane (Masters)

Minimum: 8 vCPU / 32 GiB RAM (e.g., `Standard_D8s_v3`).

Full list: `pkg/api/validate/vm.go` → `supportedMasterVmSizes` map.

### Workers

Minimum: 4 vCPU (e.g., `Standard_D4s_v3`).

Includes: general purpose, memory optimized, compute optimized, storage optimized, and GPU families.

Full list: `pkg/api/validate/vm.go` → `supportedWorkerVmSizes` map.

### Version-Gated VM Sizes

These families require OpenShift **4.19+**:

- Dsv6, Ddsv6, Dlsv6, Dldsv6, Lsv4

Enforced in `pkg/api/validate/vm.go` via:
- `masterVmSizesWithMinimumVersion`
- `workerVmSizesWithMinimumVersion`

### Special Restrictions

| VM Size | Restriction |
|---------|------------|
| `Standard_M128ms` | Does NOT support encryption at host |
| `ND96asr_v4`, `ND96amsr_A100_v4`, `NC*ads_A100_v4` | Day-2 only — not available at install time |

### Disk Size

Minimum **128 GiB** for worker nodes. Enforced in `validate.DiskSizeIsValid()`.

## Authentication Modes

Two mutually exclusive modes, determined by `UsesWorkloadIdentity()` on the cluster object:

1. **Service Principal** — traditional client ID + secret
2. **Workload Identity** — managed identity (platform workload identity role sets)

## Networking

- Clusters require direct outbound internet access, or `UserDefinedRouting`
- NSGs are managed by the service — customers cannot modify them unless using bring-your-own NSG feature

## Adding New VM Sizes

See `docs/agent-guides/api-type-system.md` → "Adding New VM Sizes Checklist" for the file-by-file checklist.

Also see: `docs/adding-new-instance-types.md` for the full procedure.
