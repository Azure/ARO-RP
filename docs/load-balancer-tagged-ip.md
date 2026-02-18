# Load Balancer Tagged IP Migration

## Release Status

### INT Environment — Successful

| | ARO-RP | RP-Config |
|---|--------|-----------|
| **Repo** | `Azure/ARO-RP` | `Azure/RP-Config` |
| **Branch** | `preetisht/ARO-20087-from-master` | `b-ptripathi/vmss-ip-tags` |
| **Commit** | `f03cd0fc3c76db8acec034bdb12267a2220f428f` | `163e61e5fa09cbb388df700dc4180ce367b74959` |
| **Tag** | `v20260216.01-lb-tagged-ip-dual-frontend` | `v20260216.01-lb-tagged-ip-dual-frontend` |

Released to INT on **2026-02-16**. Deployment completed successfully.
[INT Release Pipeline](https://msazure.visualstudio.com/AzureRedHatOpenShift/_build/results?buildId=153392735&view=results)

### INT Environment (v2 — SSH tagged rule) — Successful

| | ARO-RP | RP-Config |
|---|--------|-----------|
| **Repo** | `Azure/ARO-RP` | `Azure/RP-Config` |
| **Branch** | `preetisht/ARO-20087-from-master` | `b-ptripathi/vmss-ip-tags` |
| **Commit** | `923f0af3679d5b952992f37ac9244110ee7d9fa1` | `163e61e5fa09cbb388df700dc4180ce367b74959` |
| **Tag** | `v20260217.01-lb-tagged-ip-dual-frontend` | `v20260217.01-lb-tagged-ip-dual-frontend` |

Released to INT on **2026-02-17**. Deployment completed successfully.
[INT Release Pipeline (v2)](https://msazure.visualstudio.com/AzureRedHatOpenShift/_build/results?buildId=153402072&view=results)

Changes in this release: renamed `portal-probe-tagged` to `portal-probe-https-tagged`, added `portal-lbrule-ssh-tagged` (port 22 → 2223), `portal-probe-ssh-tagged` (TCP/2223), firewall port 2223, and portal container `-p 2223:2223`.

---

## Motivation

Azure public IP addresses can carry **IP tags** — metadata key-value pairs such as `RoutingPreference` and `FirstPartyUsage`. These tags tell Azure's networking fabric how to handle traffic on those IPs (e.g., preferred routing paths, first-party billing attribution). The ARO RP and Portal services were originally deployed with untagged public IPs. To comply with Azure networking requirements for first-party services, we need to migrate to tagged IPs.

Replacing the IPs in-place would be risky: a misconfigured tag or a platform issue could take down the RP or Portal. Instead, we use a **dual-frontend approach** — add new tagged IPs alongside the existing untagged ones, point DNS to the tagged IPs, and keep the untagged IPs as an instant rollback path.

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
portal-pip      (untagged) ──► portal-frontend      ──► port 22   ──► backend (2222)

rp-pip-tagged   (tagged)   ──► rp-frontend-tagged   ──► port 443  ──► backend (8443)
portal-pip-tagged (tagged) ──► portal-frontend-tagged──► port 443  ──► backend (8444)
portal-pip-tagged (tagged) ──► portal-frontend-tagged──► port 22   ──► backend (2223)
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

| Rule Name                  | Frontend                   | Frontend Port | Backend Port | Purpose                          |
|----------------------------|----------------------------|---------------|--------------|----------------------------------|
| `rp-lbrule`               | `rp-frontend`              | 443           | 443          | RP traffic on untagged IP        |
| `rp-lbrule-8443`          | `rp-frontend-tagged`       | 443           | 8443         | RP traffic on tagged IP          |
| `portal-lbrule`           | `portal-frontend`          | 443           | 444          | Portal HTTPS on untagged IP      |
| `portal-lbrule-8444`      | `portal-frontend-tagged`   | 443           | 8444         | Portal HTTPS on tagged IP        |
| `portal-lbrule-ssh`       | `portal-frontend`          | 22            | 2222         | Portal SSH on untagged IP        |
| `portal-lbrule-ssh-tagged`| `portal-frontend-tagged`   | 22            | 2223         | Portal SSH on tagged IP          |

#### New Health Probes

| Probe Name                  | Port | Protocol | Path            |
|-----------------------------|------|----------|-----------------|
| `rp-probe-tagged`           | 8443 | HTTPS    | `/healthz/ready`|
| `portal-probe-https-tagged` | 8444 | HTTPS    | `/healthz/ready`|
| `portal-probe-ssh-tagged`   | 2223 | TCP      | —               |

Separate probes allow independent health monitoring of tagged vs. untagged paths.

### 4. VMSS and Firewall Changes

- **Firewall ports opened:** `8443/tcp`, `8444/tcp`, and `2223/tcp` added to the RP VMSS firewall allow list.
- **Container port mapping:** RP container gets `-p 8443:8443` and `-p 8444:8444` in addition to the existing `-p 443:8443`. Portal container gets `-p 2223:2223` in addition to the existing `-p 2222:2222`.

### 5. Container Routing

The VMSS runs multiple containers. Each container handles different traffic types. Understanding which container listens on which port is critical for debugging:

| Container        | Service File                     | Ports                                      | Traffic Type              |
|------------------|----------------------------------|--------------------------------------------|---------------------------|
| **aro-rp**       | `aro-rp.service`                 | `443:8443`, `8443:8443`, `8444:8444`       | RP API (ARM, Geneva)      |
| **aro-portal**   | `aro-portal.service`             | `444:8444`, `2222:2222`, `2223:2223`       | Portal HTTPS and SSH      |

The format is `host_port:container_port`. Key points:

- **RP traffic** (both untagged on 443 and tagged on 8443) routes to the **aro-rp** container, which internally listens on 8443.
- **Portal HTTPS** (untagged on 444, tagged on 8444) routes to the **aro-portal** container, which internally listens on 8444.
- **Portal SSH** (untagged on 2222, tagged on 2223) routes to the **aro-portal** container. The portal process handles SSH tunnelling to cluster nodes.

### 6. NSG (Network Security Group) Rules

Two new inbound rules are added to the RP NSG in both development and production predeploy templates:

| Rule Name              | Priority | Port | Source Service Tag     |
|------------------------|----------|------|------------------------|
| `rp_in_arm_tagged`     | 121      | 8443 | `AzureResourceManager` |
| `rp_in_geneva_tagged`  | 131      | 8443 | `GenevaActions`        |

Additionally, NSG deployment is now **forced on every predeploy** (not only on initial creation) to ensure new rules are always applied.

### 7. DNS Update

`deploy_rp.go` → `configureDNS()` now resolves the tagged IPs instead of the untagged ones:

- `rp-pip` → `rp-pip-tagged`
- `portal-pip` → `portal-pip-tagged`

This makes the tagged IPs the primary DNS entry point.

## Rollback Strategy

Because both tagged and untagged frontends exist on the same load balancer:

1. **DNS rollback** — Point DNS back to the untagged IPs (`rp-pip`, `portal-pip`). Traffic immediately flows through the original, untagged path on port 443.
2. **Health probes** — Separate probes (`rp-probe-tagged`, `portal-probe-https-tagged`, `portal-probe-ssh-tagged`) allow monitoring tagged paths independently. If tagged probes fail, the untagged paths remain unaffected.
3. **Region disable list** — Add a region to `lbIpTagsDisabledRegions` to create the tagged IPs without tags in that region, effectively making them behave like untagged IPs.

## Cross-Repository Dependency

This feature spans two repositories that must be deployed together:

| Repository | What it provides |
|------------|-----------------|
| **Azure/ARO-RP** | ARM templates with the dual-frontend LB, tagged IP resources, probes, rules, NSG rules, firewall/container config, and DNS logic. |
| **Azure/RP-Config** | Per-environment configuration values: the actual IP tag objects (`rpLbIpTags`, `portalLbIpTags`) and the disabled regions list (`lbIpTagsDisabledRegions`). Without these values, the tagged IPs are created but have no tags applied. |

**Deployment order:** RP-Config should be deployed first (or simultaneously) so that the IP tag values are available when the ARO-RP ARM template is evaluated. If ARO-RP is deployed before RP-Config provides the tag values, the tagged IPs will be created without tags (safe, but defeats the purpose).

## Port Convention

Tagged backend ports follow a predictable offset from their untagged counterparts:

| Service       | Untagged Backend Port | Tagged Backend Port | Offset     |
|---------------|-----------------------|---------------------|------------|
| RP API        | 443                   | 8443                | +8000      |
| Portal HTTPS  | 444                   | 8444                | +8000      |
| Portal SSH    | 2222                  | 2223                | +1         |

This convention allows the same VMSS to serve both tagged and untagged traffic on distinct ports, with the application process distinguishing traffic origin by the port it arrives on.

## Files Changed

| File | What |
|------|------|
| `pkg/deploy/config.go` | New config fields for IP tags |
| `pkg/deploy/generator/resources.go` | `publicLBIPAddressTagged()` resource builder |
| `pkg/deploy/generator/templates.go` | Regex-based IP tag injection in template fixup |
| `pkg/deploy/generator/templates_rp.go` | New parameters and resources in RP template |
| `pkg/deploy/generator/resources_rp.go` | LB frontends, rules, probes, and NSG rules |
| `pkg/deploy/generator/scripts/rpVMSS.sh` | Firewall ports 8443, 8444, 2223 |
| `pkg/deploy/generator/scripts/util-services.sh` | Container port mappings (RP: 8443, 8444; Portal: 2223) |
| `pkg/deploy/deploy_rp.go` | DNS points to tagged IPs |
| `pkg/deploy/predeploy.go` | Force NSG deployment on every predeploy |
| `pkg/deploy/assets/rp-production.json` | Generated ARM template |
| `pkg/deploy/assets/rp-production-parameters.json` | Generated parameters |
| `pkg/deploy/assets/rp-development-predeploy.json` | Dev NSG rules |
| `pkg/deploy/assets/rp-production-predeploy.json` | Prod NSG rules |
