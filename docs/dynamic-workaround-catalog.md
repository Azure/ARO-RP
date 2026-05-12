# Dynamic Workaround Catalog

The dynamic workaround catalog lets the ARO operator apply MachineConfig-shaped
hotfixes to a cluster **without** waiting for a new RP release.

It exists because some bugs that require a `MachineConfig` workaround (kernel
quirks, ipsec timing issues, ignition snippets, etc.) are discovered in the
wild before a clean upstream fix lands, and ARO needs a way to roll mitigations
out across the fleet faster than the RP release cadence allows.

> **Status: opt-in.** Default operator flags ship with this feature disabled
> and no workarounds opted-in. A catalog only takes effect on clusters where
> the RP (or a MIMO task) explicitly turns it on AND configures a per-cluster
> predicate for the specific workaround.

## How it works

There are two halves to the design, and BOTH must be set for a workaround to
land on a cluster:

1. **The catalog** is the *menu* of available workarounds. It is published
   centrally to an Azure Key Vault secret and lists every MachineConfig the
   operator could conceivably apply. The catalog itself does NOT decide which
   clusters get which workaround.
2. **The per-cluster predicate flag** is the *opt-in*. The operator flag
   `aro.dynamicworkaround.predicates` is a JSON map from a catalog workaround Name
   to a CEL boolean expression. A workaround applies on this cluster iff its
   Name appears in this map AND the expression evaluates true against the
   local cluster's facts.

This split lets the same catalog roll out to different cluster cohorts
independently — the cohort is just whichever clusters have the right flag
value set, and the CEL expression refines further (e.g. "only on 4.17.x
clusters in eastus that are running IPSec").

The reconcile loop:

1. The ARO operator runs the `DynamicWorkaround` controller (see
   [pkg/operator/controllers/dynamicworkaround](../pkg/operator/controllers/dynamicworkaround/)).
2. On a configurable poll interval (default 5 minutes), the controller reads a
   Key Vault secret whose URI is configured via the operator flag
   `aro.dynamicworkaround.catalog.secretURI`. The secret value is the catalog JSON.
3. The operator authenticates to Key Vault using the service-principal
   credentials already mounted into the master operator pod (the
   `azure-cloud-credentials` Secret → `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET`
   / `AZURE_TENANT_ID` env vars), or workload identity in MIWI mode.
4. The controller parses `aro.dynamicworkaround.predicates` and compiles each CEL
   expression.
5. For each catalog entry: look up the workaround Name in the parsed
   predicates map. If absent, skip. If present, evaluate the expression
   against the local cluster's facts (OpenShift version, region, ipsec mode,
   architecture). If true, apply.
6. Matching entries are applied as `MachineConfig` objects labelled
   `aro.openshift.io/dynamic-workaround=true`.
7. On each reconcile, any catalog-managed MachineConfig that is no longer in
   the catalog, or whose predicate no longer matches, is **deleted**. Foreign
   MachineConfigs (without the managed-by label) are never touched.

## Operator flags

| Flag                                       | Default | Description                                                                                                                            |
| ------------------------------------------ | ------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `aro.dynamicworkaround.catalog.enabled`           | `false` | Master kill switch. When `false`, all catalog-managed MachineConfigs are removed and no Key Vault read is attempted.                   |
| `aro.dynamicworkaround.catalog.secretURI`         | `""`    | Full Key Vault secret URI: `https://<vault>.vault.azure.net/secrets/<name>[/<version>]`. Empty value is treated the same as disabled.  |
| `aro.dynamicworkaround.catalog.pollinterval`      | `5m`    | How often to re-read the secret. Values shorter than 1 minute are clamped up. Unparseable values fall back to 5m.                      |
| `aro.dynamicworkaround.predicates`                | `""`    | JSON object mapping workaround Name → CEL boolean expression. See [Predicates flag](#predicates-flag). Empty value disables all.       |

Including the secret version in the URI pins to a specific snapshot:
`https://<vault>.vault.azure.net/secrets/<name>/<version>`. Omitting the
version means "always read the latest", which is the normal mode of operation.

## Catalog manifest schema (`v1alpha1`)

The Key Vault secret value must be this JSON shape:

```json
{
  "schemaVersion": "v1alpha1",
  "catalogVersion": "2026-05-11.1",
  "workarounds": [
    {
      "name": "ipsec-mtu-fix",
      "description": "Reduces tunnel MTU on affected 4.16 IPSec clusters.",
      "machineConfigName": "99-aro-ipsec-mtu-fix",
      "role": "worker",
      "ignition": {
        "ignition": { "version": "3.2.0" },
        "storage": {
          "files": [
            {
              "path": "/etc/sysctl.d/99-aro-ipsec.conf",
              "contents": { "source": "data:,net.ipv4.tcp_mtu_probing%3D2%0A" },
              "mode": 420
            }
          ]
        }
      }
    }
  ]
}
```

### Field reference

**Top level**

- `schemaVersion` (required) — Must be `"v1alpha1"`. Future breaking changes
  bump this and force operators to upgrade before the catalog applies.
- `catalogVersion` (required) — Opaque, publisher-controlled identifier (date,
  semver, git sha…). Recorded as an annotation on every applied MachineConfig
  so on-call engineers can correlate a live MC to a catalog publication.
- `workarounds` (optional, max 64 entries) — List of workaround entries.
  Empty list is valid and means "remove any previously applied workarounds".

**Workaround entry**

- `name` (required, DNS label) — Stable identifier used as the cleanup key
  AND as the key the per-cluster predicates map references. If you remove a
  name from the catalog, the corresponding MachineConfig is deleted from
  every cluster on the next reconcile.
- `description` (optional) — Human-readable text written as an annotation on
  the applied MachineConfig.
- `machineConfigName` (required, DNS label) — The Kubernetes object name. Use a
  numeric prefix (e.g. `99-`) so the MachineConfig sorts correctly relative to
  OpenShift's own.
- `role` (required) — `"master"` or `"worker"`.
- `ignition` (required) — Raw Ignition config. Marshalled directly into
  `MachineConfig.spec.config`. The operator does not introspect it; the MCO
  performs Ignition-spec validation when rendering.

Note: the catalog itself does NOT carry a predicate field. Predicates live on
the cluster side, in the `aro.dynamicworkaround.predicates` operator flag.

## Predicates flag

`aro.dynamicworkaround.predicates` is the per-cluster opt-in. Its value is a JSON
object mapping a catalog workaround Name to a CEL boolean expression:

```json
{
  "ipsec-mtu-fix":     "ipsecMode == \"Full\" && region == \"eastus\"",
  "kernel-quirk-4-16": "versionAtLeast(clusterVersion, \"4.16.0\") && versionLessThan(clusterVersion, \"4.17.0\")"
}
```

A catalog entry applies on this cluster iff:

- Its `name` appears as a key in this map, AND
- The corresponding CEL expression returns `true` for the local cluster's
  facts.

Workarounds whose name is **not** in the map are silently skipped on this
cluster. Empty or absent flag value means "no workarounds enabled here",
which is the safe default.

### Available CEL variables

| Name                  | Type     | Value                                                                                                  |
| --------------------- | -------- | ------------------------------------------------------------------------------------------------------ |
| `clusterVersion`      | `string` | OpenShift version, e.g. `"4.17.0"`. Empty string if unknown — version helpers fail closed in that case. |
| `ipsecMode`           | `string` | Literal value of `ipsecConfig.mode`, or `""` if absent.                                                 |
| `region`              | `string` | Cluster Azure region, lowercased.                                                                       |
| `architectureVersion` | `int`    | 1 or 2.                                                                                                 |

### Helper functions

| Signature                                           | Returns                                                                              |
| --------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `versionAtLeast(facts: string, target: string)`     | `bool` — true iff `facts ≥ target` under semver ordering (NOT lexicographic).        |
| `versionLessThan(facts: string, target: string)`    | `bool` — true iff `facts < target` under semver ordering.                            |

### Common expression patterns

```text
# Always apply on this cluster (useful for fleet-wide rollouts that have
# already been narrowed by which clusters got the flag at all).
true

# Apply only when IPSec is something other than Disabled.
ipsecMode != "Disabled"

# Apply on 4.16.x.
versionAtLeast(clusterVersion, "4.16.0") && versionLessThan(clusterVersion, "4.17.0")

# Apply on 4.16+ in eastus or westeurope with IPSec Full.
versionAtLeast(clusterVersion, "4.16.0") &&
  (region == "eastus" || region == "westeurope") &&
  ipsecMode == "Full"
```

### Limits and safety

- Maximum expression length: 4096 characters per entry.
- Maximum flag value: 64 KiB.
- Per-evaluation timeout: 250 ms (catches pathological expressions).
- Expressions must return `bool`; non-bool or syntactically invalid
  expressions cause the controller to **skip the whole reconcile** rather
  than apply partial state. Existing managed MachineConfigs are left in
  place — a typo'd flag never tears down active mitigations.
- Workaround names in the map are validated against the same DNS-label
  regex the catalog uses, so a typo'd name fails fast.

## Trust model

Two trust boundaries:

1. **The catalog Key Vault secret is fully trusted.** The operator:
    - Only accepts `https://` secret URIs.
    - Caps the secret value at 1 MiB (well above Key Vault's own 25 KiB cap).
    - Validates the JSON shape and rejects unknown schema versions.

    There is no separate payload signature; integrity comes from Key Vault
    access control. Whoever holds `secrets/set` on the vault can publish a
    catalog.

2. **The predicates flag is RP-controlled, like every operator flag.** It
   reaches the cluster via adminUpdate / MIMO operator-flags, which require
   RP-side authorization. Cluster owners cannot grant themselves a
   workaround they were not approved for, because they cannot edit operator
   flags.

**Therefore:**

- The catalog secret URI is set via the operator flags, which are RP-controlled.
  Cluster owners cannot point a cluster at an attacker-controlled vault.
- The Key Vault holding the catalog must be locked down to RP-managed
  identities for `secrets/set`. The operator's service principal needs
  `secrets/get` on that vault.
- Treat catalog publication AND predicate flag changes with the same review
  rigor as an RP release — both land MachineConfig content on customer
  clusters.

## Operational guide

### Provisioning a vault for catalog publication

1. Create (or pick) a Key Vault dedicated to dynamic workarounds, in a
   subscription separate from any tenant data.
2. Grant `secrets/set` to the RP-managed identity that publishes catalogs.
3. Grant `secrets/get` to the operator's cluster service principal /
   workload-identity object IDs. ARO already grants the cluster identity to
   the RP gateway vault; the catalog vault should follow the same model.
4. Note the vault's DNS suffix — it's part of the secret URI you'll set in
   the operator flag.

### Rolling out a new workaround

1. Author the catalog entry locally and validate it against the schema.
2. Publish a new version of the catalog secret in a **staging** Key Vault.
   Note the returned secret version GUID.
3. Author the CEL predicate that targets the right cohort.
4. On a test cluster: set `aro.dynamicworkaround.catalog.secretURI` to the pinned
   staging URI (`.../secrets/<name>/<version>`) AND set
   `aro.dynamicworkaround.predicates` to include the new entry. Flip
   `aro.dynamicworkaround.catalog.enabled=true` if it isn't already.
5. Watch the operator logs (`kubectl logs -n openshift-azure-operator deploy/aro-operator-master`)
   for `applied MachineConfig ...` entries and confirm `oc get machineconfig`
   shows the expected object.
6. Once verified, publish the same secret value in the production Key Vault.
7. Roll out the predicate flag via adminUpdate / MIMO to the production
   cluster cohort. The catalog itself can stay unpinned (latest) in
   production — the staged rollout is driven by which clusters get the
   predicate, not by which clusters see which catalog.

### Killing a workaround

Four options, in order of blast radius (smallest first):

1. **Per-cluster, per-workaround:** remove the entry from
   `aro.dynamicworkaround.predicates` on that cluster. The controller deletes the
   corresponding MachineConfig on the next reconcile; every other workaround
   is unaffected.
2. **Per-cluster, all workarounds:** flip `aro.dynamicworkaround.catalog.enabled=false`
   on that cluster. All catalog-managed MachineConfigs are removed on the next
   reconcile. Foreign MachineConfigs are untouched.
3. **Single entry, fleet-wide:** publish a new version of the catalog
   secret (with a bumped `catalogVersion`) that omits the entry's `name`.
   Clusters that have already fetched will delete the corresponding
   MachineConfig on the next reconcile, even if their predicate flag still
   names it (no catalog entry → nothing to apply).
4. **Whole fleet:** publish a new secret version with `"workarounds": []`.
   All clusters tear down everything the catalog produced.

### Safe-by-design behaviour

The reconciler is deliberately tolerant of partial failure:

- If the Key Vault GET fails (network, auth, throttling), the controller logs
  the error, requeues, and **does not** delete existing managed MachineConfigs.
  A Key Vault outage cannot tear down active mitigations.
- If the secret URI is malformed, the controller logs and skips the reconcile
  — again without deleting anything.
- If the predicates flag is malformed (bad JSON, bad CEL syntax, non-bool
  return type), the controller logs and skips the reconcile without
  deleting anything. A typo'd flag never causes a teardown.
- If one workaround's predicate raises a runtime error (e.g. a bad version
  string passed to `versionAtLeast`), only that workaround is skipped; the
  rest of the reconcile continues.
- If the cluster's ClusterVersion CR is unreadable, version-gated predicates
  evaluate to false (fail closed) rather than apply with bad data.

### Inspecting state

```bash
# Catalog-managed MachineConfigs only
oc get mc -l aro.openshift.io/dynamic-workaround=true

# Which catalog produced a given MC?
oc get mc 99-aro-ipsec-mtu-fix \
  -o jsonpath='{.metadata.annotations.aro\.openshift\.io/dynamic-workaround-catalog-version}'

# Which workarounds is this cluster opted in to?
oc get cluster.aro.openshift.io cluster \
  -o jsonpath='{.spec.operatorflags.aro\.dynamicworkaround\.predicates}'

# Operator logs filtered to this controller
oc logs -n openshift-azure-operator deploy/aro-operator-master \
  | grep '"controller":"DynamicWorkaround"'
```

## Limitations (v1alpha1)

- **MachineConfig only.** Other resource types (DaemonSets, ConfigMaps,
  ClusterOperators) are intentionally out of scope. Adding a new resource
  type is a breaking change and bumps the schema version.
- **No signature verification.** Trust is established via Key Vault
  authentication on the catalog side, and via RP-controlled operator flags
  on the predicate side.
- **Single secret per cluster.** The operator reads exactly one
  `secretURI`; there is no merge-from-multiple-sources logic. If a future
  scenario needs that, model it as a federation layer on the publishing side.
- **Predicates flag is per-cluster.** There is no group/tag mechanism — a
  fleet-wide rollout is performed by setting the same predicate on each
  cluster individually (typically via MIMO).
