# CustomDNS Controller Changes - ARO-22038

## Backward Compatibility Verification

The table below traces every cluster scenario to verify that the flag rename
(`aro.dnsmasq.enabled` -> `aro.dns.enabled`) and controller changes do not
break existing clusters.

`SetDefaults()` in `pkg/api/defaults.go` only applies `DefaultOperatorFlags()`
when the `OperatorFlags` map is **nil** (brand new cluster). Existing clusters
always have a non-nil map, so their flags are never overwritten. The
`IsDNSControllerEnabled()` helper in `dnstype.go` checks `aro.dns.enabled`
first, then falls back to the legacy `aro.dnsmasq.enabled` for clusters that
predate the rename.

| # | Scenario | Flags in CosmosDB | `IsDNSControllerEnabled()` | `GetEffectiveDNSType()` | Result |
|---|----------|-------------------|---------------------------|-------------------------|--------|
| 1 | Existing dnsmasq cluster (pre-4.21) | `aro.dnsmasq.enabled: true`, no `aro.dns.enabled` | `aro.dns.enabled` not found, falls back to `aro.dnsmasq.enabled: true` -> **true** | `aro.dns.type` blank -> **dnsmasq** | Dnsmasq MCs reconciled as before. **OK** |
| 2 | Existing cluster, disabled | `aro.dnsmasq.enabled: false` | Falls back to `false` -> **false** | N/A (skipped) | Controller does nothing. **OK** |
| 3 | Existing cluster, admin merge | `aro.dnsmasq.enabled: true` preserved | Falls back to `true` -> **true** | Unchanged -> **dnsmasq** | No change to existing behavior. **OK** |
| 4 | Existing cluster, admin reset | `aro.dnsmasq.enabled` removed, `aro.dns.enabled: true` added from defaults | `aro.dns.enabled` found -> **true** | `aro.dns.type` blank -> **dnsmasq** | Still works, new flag takes over. **OK** |
| 5 | New cluster (pre-4.21) | `aro.dns.enabled: true`, `aro.dns.type: ""` | `aro.dns.enabled: true` -> **true** | Blank -> **dnsmasq** | Dnsmasq path. **OK** |
| 6 | New cluster (4.21+) | `aro.dns.enabled: true`, `aro.dns.type` auto-set to `clusterhosted` by `setDNSDefaults()` | **true** | `clusterhosted`, version >=4.21 -> **clusterhosted** | Infrastructure CR path. **OK** |
| 7 | Existing 4.21+, admin sets clusterhosted | `aro.dnsmasq.enabled: true`, `aro.dns.type: clusterhosted` | Falls back to `true` -> **true** | Version >=4.21 -> **clusterhosted** | Infrastructure CR reconciled, dnsmasq MCs skipped. **OK** |
| 8 | Cluster has BOTH flags | `aro.dns.enabled: true`, `aro.dnsmasq.enabled: false` | `aro.dns.enabled` found, uses it -> **true** | Based on `aro.dns.type` | New flag takes precedence. **OK** |
| 9 | Existing cluster, `ensureDefaults()` runs in backend | `OperatorFlags != nil`, `SetDefaults()` skips, flags unchanged | Falls back to legacy flag correctly | Unchanged | No surprise mutations. **OK** |

### Monitoring impact

The `clusterflagsandbanner.go` monitoring code compares the cluster's flags
against `DefaultOperatorFlags()`. Since defaults now contain `aro.dns.enabled`
but not `aro.dnsmasq.enabled`, existing clusters will emit
`aro.dns.enabled: DNE` in the non-standard flags metric. This is a reporting
artifact that helps with TDR Goal #3 (fleet tracking), not a functional issue.

### EtcHosts controller

Completely independent. Uses `aro.etchosts.enabled` / `aro.etchosts.managed`.
No interaction with DNS flags. Works identically for both dnsmasq and CustomDNS.

---

## Changes by File

### New file: `pkg/operator/controllers/dns/infrastructure.go`

New file containing Infrastructure CR reconciliation logic for CustomDNS.

Since the vendored `openshift/api` (`v0.0.0-20240103200955`) does not include
`CloudLoadBalancerConfig` on `AzurePlatformStatus`, this file uses the
`unstructured` client to read and patch the Infrastructure CR.

Functions:

- `reconcileInfrastructureCR()` - Main entry point. Reads the Infrastructure CR
  as unstructured, compares the current `cloudLoadBalancerConfig` against the
  desired state from the ARO Cluster spec's `APIIntIP` and `IngressIP`, and
  patches via status subresource merge patch if they differ.
- `getInfrastructureCR()` - Gets Infrastructure CR as `unstructured.Unstructured`.
- `buildDesiredCloudLBConfig()` - Builds the desired config from `APIIntIP` and
  `IngressIP`. Per the TDR, `apiLoadBalancerIPs` uses the same value as
  `apiIntLoadBalancerIPs`.
- `getCurrentCloudLBConfig()` - Reads the current config from
  `status.platformStatus.azure.cloudLoadBalancerConfig` using nested field helpers.
- `patchInfrastructureStatus()` - Builds and applies a JSON merge patch to the
  status subresource.

### Modified: `pkg/operator/flags.go`

**Flag rename with backward compatibility:**

- Added `DNSEnabled = "aro.dns.enabled"` as the new primary flag.
- Kept `DnsmasqEnabled = "aro.dnsmasq.enabled"` as a constant for backward
  compatibility (used by `IsDNSControllerEnabled()` fallback).
- `DefaultOperatorFlags()` now sets `DNSEnabled: FlagTrue` instead of
  `DnsmasqEnabled: FlagTrue` for new clusters.
- `RestartDnsmasqEnabled` kept as-is (dnsmasq-specific, only used in dnsmasq
  code path).

### Modified: `pkg/operator/controllers/dns/cluster_controller.go`

Two changes:

1. **Reconcile method**: When `effectiveDNSType == clusterhosted`, instead of
   returning early, it now calls `reconcileInfrastructureCR()` to ensure the
   Infrastructure CR has the correct LB IPs. The version >= 4.21 check is
   handled inside `GetEffectiveDNSType()`. For dnsmasq, the existing
   `reconcileMachineConfigs()` path is unchanged.

2. **SetupWithManager**: Added a watch on `configv1.Infrastructure` filtered to
   `name == "cluster"`. This provides drift protection - if someone manually
   edits the Infrastructure CR's `cloudLoadBalancerConfig`, the change triggers
   reconciliation which validates and restores the correct IPs.

3. **Enabled check**: Uses `IsDNSControllerEnabled()` instead of directly
   reading `operator.DnsmasqEnabled`.

### Modified: `pkg/operator/controllers/dns/dnstype.go`

Added `IsDNSControllerEnabled(flags)` helper function:

- Checks `aro.dns.enabled` first (new clusters).
- If not present, falls back to `aro.dnsmasq.enabled` (existing clusters).
- This is the single point of backward-compatible flag lookup used by all three
  DNS controllers.

### Modified: `pkg/operator/controllers/dns/machineconfig_controller.go`

- Uses `IsDNSControllerEnabled()` instead of `operator.DnsmasqEnabled`.
- Added `GetEffectiveDNSType()` check: when `clusterhosted`, skips dnsmasq
  MachineConfig drift reconciliation since those MachineConfigs do not exist
  for CustomDNS clusters.

### Modified: `pkg/operator/controllers/dns/machineconfigpool_controller.go`

- Uses `IsDNSControllerEnabled()` instead of `operator.DnsmasqEnabled`.
- Added `GetEffectiveDNSType()` check: when `clusterhosted`, skips per-pool
  dnsmasq MachineConfig creation since the Infrastructure CR is cluster-wide.

### Modified: `pkg/operator/controllers/dns/doc.go`

Updated package documentation to reflect:

- The new `aro.dns.enabled` flag and backward-compatible fallback.
- Infrastructure CR reconciliation for CustomDNS clusters.
- Infrastructure CR drift protection watch.
- MCO reading `cloudLoadBalancerConfig` from the Infrastructure CR.

### Modified: `pkg/operator/controllers/dns/cluster_controller_test.go`

Updated all test cases to use `operator.DNSEnabled` instead of
`operator.DnsmasqEnabled` (8 occurrences).

### Modified: `pkg/operator/controllers/dns/machineconfig_controller_test.go`

Updated all test cases to use `operator.DNSEnabled` instead of
`operator.DnsmasqEnabled` (9 occurrences).

### Modified: `pkg/operator/controllers/dns/machineconfigpool_controller_test.go`

Updated all test cases to use `operator.DNSEnabled` instead of
`operator.DnsmasqEnabled` (9 occurrences).

### Modified: `pkg/monitor/cluster/clusterflagsandbanner_test.go`

Updated test cases to use `operator.DNSEnabled` instead of
`operator.DnsmasqEnabled` (6 occurrences). These tests verify monitoring
metrics for non-standard and missing operator flags.

---

## Data Flow Diagrams

### dnsmasq clusters (unchanged)

```
ARO Cluster CR              ClusterReconciler           MachineConfigs
 (APIIntIP, IngressIP) --->  reconcileMachineConfigs --> 99-master-aro-dns
                                                         99-worker-aro-dns
                                                                |
                                                                v
                                                         dnsmasq.conf on nodes
                                                         (api, api-int, .apps)
```

### CustomDNS clusters (new)

```
ARO Cluster CR              ClusterReconciler           Infrastructure CR
 (APIIntIP, IngressIP) --->  reconcileInfrastructureCR -> status.platformStatus
                                                           .azure
                                                           .cloudLoadBalancerConfig
                                                                |
                                                                v
                                                         MCO reads IPs and
                                                         renders CoreDNS
                                                         static pod on nodes
```

### Drift protection (new)

```
Infrastructure CR modified ---> Watch triggers reconcile ---> IPs restored
```
