# Load Balancer Tagged IP Migration

## Overview

This change introduces **tagged public IP addresses** to the ARO RP load balancer using a **dual-frontend approach**. Tagged IPs carry metadata (IP tags) that allow Azure to route traffic with specific properties (e.g., `RoutingPreference` or `FirstPartyUsage`). The existing untagged IPs remain in place, ensuring a safe, rollback-friendly migration.

## Architecture

### Before

The RP load balancer (`rp-lb`) had two frontends, each with an untagged public IP:

```
rp-pip  (untagged) ──► rp-frontend     ──► port 443 ──► RP backend (port 8443)
portal-pip (untagged) ──► portal-frontend ──► port 443 ──► Portal backend (port 444)
```

### After

Two new tagged public IPs and corresponding frontends are added to the **same** load balancer. The original frontends remain untouched.

```
rp-pip          (untagged) ──► rp-frontend          ──► port 443  ──► backend (8443)
portal-pip      (untagged) ──► portal-frontend      ──► port 443  ──► backend (444)

rp-pip-tagged   (tagged)   ──► rp-frontend-tagged   ──► port 443  ──► backend (8443)
portal-pip-tagged (tagged) ──► portal-frontend-tagged──► port 443  ──► backend (8444)
```

DNS records are updated to point at the **tagged** IPs, making them the primary entry point. The untagged IPs serve as a fallback for rollback.

## What Changed

### 1. New Tagged Public IP Resources

Two new public IP addresses are created:

| Resource              | Config Parameter   | Purpose                          |
|-----------------------|--------------------|----------------------------------|
| `rp-pip-tagged`       | `rpLbIpTags`       | Tagged IP for RP traffic         |
| `portal-pip-tagged`   | `portalLbIpTags`   | Tagged IP for Portal traffic     |

IP tags are injected at template generation time via a regex-based fixup in `templates.go`. If the tags array is empty or the region is in the disabled list, the IP is created without tags (equivalent to untagged behavior).

### 2. Configuration Parameters

Three new fields in `pkg/deploy/config.go` (`Configuration` struct):

| Field                    | Type            | Description                                                  |
|--------------------------|-----------------|--------------------------------------------------------------|
| `rpLbIpTags`             | `[]interface{}` | IP tag objects (`{type, value}`) for the RP tagged IP        |
| `portalLbIpTags`         | `[]interface{}` | IP tag objects (`{type, value}`) for the Portal tagged IP    |
| `lbIpTagsDisabledRegions`| `[]string`      | Regions where tagged IPs should NOT have tags applied         |

These are passed as ARM template parameters with empty-array defaults.

### 3. Load Balancer Changes

#### New Frontend IP Configurations

| Frontend Name             | Public IP            |
|---------------------------|----------------------|
| `rp-frontend-tagged`     | `rp-pip-tagged`      |
| `portal-frontend-tagged` | `portal-pip-tagged`  |

#### New Load Balancing Rules

| Rule Name            | Frontend              | Frontend Port | Backend Port | Purpose                     |
|----------------------|-----------------------|---------------|--------------|-----------------------------|
| `rp-lbrule-8443`     | `rp-frontend`        | 8443          | 8443         | RP traffic on untagged IP   |
| `portal-lbrule-8444` | `rp-frontend`        | 8444          | 8444         | Portal on untagged IP       |
| (tagged RP rule)     | `rp-frontend-tagged` | 443           | 8443         | RP traffic on tagged IP     |
| (tagged Portal rule) | `portal-frontend-tagged` | 443       | 8444         | Portal on tagged IP         |

#### New Health Probes

| Probe Name            | Port | Protocol | Path            |
|-----------------------|------|----------|-----------------|
| `rp-probe-tagged`     | 8443 | HTTPS    | `/healthz/ready`|
| `portal-probe-tagged` | 8444 | HTTPS    | `/healthz/ready`|

Separate probes allow independent health monitoring of tagged vs. untagged paths.

### 4. VMSS and Firewall Changes

- **Firewall ports opened:** `8443/tcp` and `8444/tcp` added to the RP VMSS firewall allow list.
- **Container port mapping:** RP container gets `-p 8443:8443` and `-p 8444:8444` in addition to the existing `-p 443:8443`.

### 5. NSG (Network Security Group) Rules

Two new inbound rules are added to the RP NSG in both development and production predeploy templates:

| Rule Name              | Priority | Port | Source Service Tag     |
|------------------------|----------|------|------------------------|
| `rp_in_arm_tagged`     | 121      | 8443 | `AzureResourceManager` |
| `rp_in_geneva_tagged`  | 131      | 8443 | `GenevaActions`        |

Additionally, NSG deployment is now **forced on every predeploy** (not only on initial creation) to ensure new rules are always applied.

### 6. DNS Update

`deploy_rp.go` → `configureDNS()` now resolves the tagged IPs instead of the untagged ones:

- `rp-pip` → `rp-pip-tagged`
- `portal-pip` → `portal-pip-tagged`

This makes the tagged IPs the primary DNS entry point.

## Rollback Strategy

Because both tagged and untagged frontends exist on the same load balancer:

1. **DNS rollback** — Point DNS back to the untagged IPs (`rp-pip`, `portal-pip`). Traffic immediately flows through the original, untagged path on port 443.
2. **Health probes** — Separate probes (`rp-probe-tagged`, `portal-probe-tagged`) allow monitoring tagged paths independently. If tagged probes fail, the untagged paths remain unaffected.
3. **Region disable list** — Add a region to `lbIpTagsDisabledRegions` to create the tagged IPs without tags in that region, effectively making them behave like untagged IPs.

## Files Changed

| File | What |
|------|------|
| `pkg/deploy/config.go` | New config fields for IP tags |
| `pkg/deploy/generator/resources.go` | `publicLBIPAddressTagged()` resource builder |
| `pkg/deploy/generator/templates.go` | Regex-based IP tag injection in template fixup |
| `pkg/deploy/generator/templates_rp.go` | New parameters and resources in RP template |
| `pkg/deploy/generator/resources_rp.go` | LB frontends, rules, probes, and NSG rules |
| `pkg/deploy/generator/scripts/rpVMSS.sh` | Firewall ports 8443, 8444 |
| `pkg/deploy/generator/scripts/util-services.sh` | Container port mappings |
| `pkg/deploy/deploy_rp.go` | DNS points to tagged IPs |
| `pkg/deploy/predeploy.go` | Force NSG deployment on every predeploy |
| `pkg/deploy/assets/rp-production.json` | Generated ARM template |
| `pkg/deploy/assets/rp-production-parameters.json` | Generated parameters |
| `pkg/deploy/assets/rp-development-predeploy.json` | Dev NSG rules |
| `pkg/deploy/assets/rp-production-predeploy.json` | Prod NSG rules |
