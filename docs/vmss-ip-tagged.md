# VMSS Public IP Tagging

## Release Status

### Pending

| | ARO-RP | RP-Config |
|---|--------|-----------|
| **Repo** | `Azure/ARO-RP` | `Azure/RP-Config` |
| **Branch** | `preetisht/ARO-20272-from-master` | TBD |
| **Commit** | `c0924893ce60a6b0f441a66d48d386308ad1a20d` | TBD |
| **Tag** | `v20260217.01-vmss-ip-tagged` | — |

---

## Motivation

Azure VMSS instances can have per-instance public IPs. Like load balancer PIPs, these can carry **IP tags** — metadata key-value pairs such as `FirstPartyUsage`. The ARO RP and Gateway VMSS instances use public IPs for outbound connectivity and direct access. To comply with Azure networking requirements for first-party services, these VMSS PIPs need to be tagged with the appropriate `FirstPartyUsage` values.

Unlike the [Load Balancer Tagged IP Migration](load-balancer-tagged-ip.md), which uses a dual-frontend approach with separate tagged/untagged IPs, VMSS PIPs are tagged **in-place** using conditional ARM expressions. This is safe because:

- VMSS PIPs are ephemeral — they are recreated on scale-out and reimaging
- The tags are applied conditionally via ARM template parameters, so a region disable list provides instant rollback
- An empty tag array is functionally equivalent to no tags

## Overview

This change adds **configurable IP tags** to the per-instance public IPs on both the RP VMSS (`rp-vmss-pip`) and Gateway VMSS (`gateway-vmss-pip`). Tags are injected at ARM template generation time using a conditional fixup in `templates.go`. Each VMSS has its own independent tag configuration, allowing different `FirstPartyUsage` values for inbound (RP) vs. outbound (Gateway) traffic.

## Architecture

### Before

VMSS instances had public IPs with no tags:

```
RP VMSS instance ──► rp-vmss-pip (no tags)
Gateway VMSS instance ──► gateway-vmss-pip (no tags)
```

### After

VMSS instance PIPs conditionally carry IP tags based on configuration:

```
RP VMSS instance ──► rp-vmss-pip (tagged with rpVmssIpTags, if configured)
Gateway VMSS instance ──► gateway-vmss-pip (tagged with gwyVmssIpTags, if configured)
```

If the tag arrays are empty or the region is in the disabled list, the PIPs are created without tags (identical to the "Before" state).

## What Changed

### 1. Configuration Parameters

Four new fields and one new struct in `pkg/deploy/config.go`:

| Field | Type | Description |
|-------|------|-------------|
| `RPVmssIpTags` | `[]IPTag` | IP tag objects for RP VMSS instance PIPs |
| `RPVmssIpTagsDisabledRegions` | `[]string` | Regions where RP VMSS tags are skipped |
| `GwyVmssIpTags` | `[]IPTag` | IP tag objects for Gateway VMSS instance PIPs |
| `GwyVmssIpTagsDisabledRegions` | `[]string` | Regions where Gateway VMSS tags are skipped |

The `IPTag` struct:

```go
type IPTag struct {
    Type  string `json:"type,omitempty"`
    Value string `json:"value,omitempty"`
}
```

These are passed as ARM template parameters with empty-array defaults.

### 2. VMSS Resource Changes

Empty `IPTags: []` placeholders are added to the public IP configuration of both VMSS resources:

| VMSS | PIP Name | File |
|------|----------|------|
| RP VMSS | `rp-vmss-pip` | `pkg/deploy/generator/resources_rp.go` |
| Gateway VMSS | `gateway-vmss-pip` | `pkg/deploy/generator/resources_gateway.go` |

These empty arrays serve as anchors for the template fixup.

### 3. Template Fixup (Conditional Tag Injection)

In `pkg/deploy/generator/templates.go`, the empty `"ipTags": []` is replaced with a conditional ARM expression at template generation time. The fixup uses the PIP name to determine which parameter set to apply:

| PIP Name | Parameters Used | Purpose |
|----------|----------------|---------|
| `rp-vmss-pip` | `rpVmssIpTags`, `rpVmssIpTagsDisabledRegions` | Inbound FirstPartyUsage tag |
| `gateway-vmss-pip` | `gwyVmssIpTags`, `gwyVmssIpTagsDisabledRegions` | Outbound FirstPartyUsage tag |

The generated ARM expression:

```
if(
  or(
    contains(parameters('<disabledRegions>'), resourceGroup().location),
    equals(length(parameters('<ipTags>')), 0)
  ),
  createArray(),
  createArray(
    createObject('ipTagType', parameters('<ipTags>')[0].type, 'tag', parameters('<ipTags>')[0].value)
  )
)
```

This means:
- If the region is in the disabled list → no tags
- If the tag array is empty → no tags
- Otherwise → apply the first tag from the array

### 4. ARM Template Parameters

New parameters added to both RP and Gateway templates:

**RP template** (`templates_rp.go`):
- `rpVmssIpTags` (array, default `[]`)
- `rpVmssIpTagsDisabledRegions` (array, default `[]`)

**Gateway template** (`templates_gateway.go`):
- `gwyVmssIpTags` (array, default `[]`)
- `gwyVmssIpTagsDisabledRegions` (array, default `[]`)

### 5. Deploy-time Parameter Passing

| File | Parameters Passed |
|------|------------------|
| `pkg/deploy/deploy_rp.go` | `rpVmssIpTags`, `rpVmssIpTagsDisabledRegions` |
| `pkg/deploy/deploy_gateway.go` | `gwyVmssIpTags`, `gwyVmssIpTagsDisabledRegions` |

## Rollback Strategy

1. **Region disable list** — Add the region to `RPVmssIpTagsDisabledRegions` or `GwyVmssIpTagsDisabledRegions` in RP-Config. The next VMSS deployment in that region will create PIPs without tags.
2. **Empty tag array** — Set `RPVmssIpTags` or `GwyVmssIpTags` to `[]` in RP-Config. All regions will create untagged PIPs.
3. **VMSS reimage** — Since VMSS PIPs are ephemeral, reimaging an instance will pick up the new (untagged) configuration immediately.

## Comparison with Load Balancer Tagged IP Migration

| Aspect | LB Tagged IPs | VMSS Tagged IPs |
|--------|--------------|-----------------|
| **Scope** | Load balancer frontend PIPs | VMSS per-instance PIPs |
| **Approach** | Dual-frontend (additive) | In-place conditional |
| **Resources** | `rp-pip-tagged`, `portal-pip-tagged` | `rp-vmss-pip`, `gateway-vmss-pip` |
| **Rollback** | DNS switch + region disable | Region disable + empty array |
| **Risk** | Low (untagged IPs remain) | Low (empty array = no tags) |
| **Jira** | ARO-20087 | ARO-20272 |

Both features use the same pattern: regex-based fixup in `templates.go`, conditional ARM expressions, and per-region disable lists. See [load-balancer-tagged-ip.md](load-balancer-tagged-ip.md) for the LB counterpart.

## Cross-Repository Dependency

This feature spans two repositories that must be deployed together:

| Repository | What it provides |
|------------|-----------------|
| **Azure/ARO-RP** | ARM templates with the VMSS PIP tag placeholders, conditional fixup logic, and parameter definitions. |
| **Azure/RP-Config** | Per-environment configuration values: the actual IP tag objects (`RPVmssIpTags`, `GwyVmssIpTags`) and the disabled regions lists. Without these values, the PIPs are created without tags. |

**Deployment order:** RP-Config should be deployed first (or simultaneously) so that the IP tag values are available when the ARM template is evaluated.

## Files Changed

| File | What |
|------|------|
| `pkg/deploy/config.go` | New config fields (`RPVmssIpTags`, `RPVmssIpTagsDisabledRegions`, `GwyVmssIpTags`, `GwyVmssIpTagsDisabledRegions`) and `IPTag` struct |
| `pkg/deploy/generator/resources_rp.go` | Empty `IPTags` placeholder on `rp-vmss-pip` |
| `pkg/deploy/generator/resources_gateway.go` | Empty `IPTags` placeholder on `gateway-vmss-pip` |
| `pkg/deploy/generator/templates.go` | Conditional ARM expression fixup for both PIP names |
| `pkg/deploy/generator/templates_rp.go` | New `rpVmssIpTags` and `rpVmssIpTagsDisabledRegions` parameters |
| `pkg/deploy/generator/templates_gateway.go` | New `gwyVmssIpTags` and `gwyVmssIpTagsDisabledRegions` parameters |
| `pkg/deploy/deploy_rp.go` | Pass RP VMSS IP tag config at deploy time |
| `pkg/deploy/deploy_gateway.go` | Pass Gateway VMSS IP tag config at deploy time |
| `pkg/deploy/assets/rp-production.json` | Generated RP ARM template |
| `pkg/deploy/assets/rp-production-parameters.json` | Generated RP parameters |
| `pkg/deploy/assets/rp-development.json` | Generated dev ARM template |
| `pkg/deploy/assets/gateway-production.json` | Generated Gateway ARM template |
| `pkg/deploy/assets/gateway-production-parameters.json` | Generated Gateway parameters |
