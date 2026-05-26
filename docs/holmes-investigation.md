# Holmes Investigation API

The Holmes investigation API is an admin endpoint that runs [HolmesGPT](https://github.com/robusta-dev/holmesgpt) diagnostic investigations on ARO clusters. It creates a short-lived pod on the Hive AKS cluster that connects to the target cluster, runs diagnostic queries, and streams the results back to the caller.

**Endpoint:** `POST /admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroup}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{clusterName}/investigate`

## Configuration Reference

| Config | Env Var | Key Vault Secret (prod) | Default | Required |
|--------|---------|-------------------------|---------|----------|
| Azure OpenAI endpoint | `HOLMES_AZURE_API_BASE` | `holmes-azure-api-base` | — | Yes |
| HolmesGPT container image | `HOLMES_IMAGE` | — | `version.HolmesImage(acrDomain)` | No |
| Azure OpenAI API version | `HOLMES_AZURE_API_VERSION` | — | `2025-04-01-preview` | No |
| LLM model name | `HOLMES_MODEL` | — | `azure/gpt-5.2` | No |
| Pod timeout (seconds) | `HOLMES_DEFAULT_TIMEOUT` | — | `600` | No |
| Max concurrent investigations per RP | `HOLMES_MAX_CONCURRENT` | — | `20` | No |

## Authentication

Holmes uses **Entra ID token authentication** for Azure OpenAI (`disableLocalAuth=true` on the AOAI resource). No API keys are stored or used.

At RP startup, the RP obtains its managed identity credential (prod) or dev service principal credential (dev). At investigation request time, `HolmesConfig.AcquireToken()` acquires a short-lived (~1 hour) token scoped to `https://cognitiveservices.azure.com/.default`. The token is injected into the investigation pod.

**Requirements:**
- The Azure OpenAI resource must have a **custom subdomain** endpoint (e.g., `https://<name>.openai.azure.com/`)
- The RP identity must have the **Cognitive Services OpenAI User** role on the AOAI resource

## Config Loading

Configuration is loaded once at RP startup in `NewFrontend` (`pkg/frontend/frontend.go`).

**Development mode** (`RP_MODE=development`): The API base URL is read from the `HOLMES_AZURE_API_BASE` environment variable. The MSI credential is obtained via `NewMSITokenCredential()` (uses `AZURE_RP_CLIENT_ID`/`AZURE_RP_CLIENT_SECRET` in dev). This uses `NewHolmesConfigFromEnv(acrDomain, credential)`.

**Production mode**: The API base URL is read from the service Key Vault (`{KEYVAULT_PREFIX}-svc`). The MSI credential is the RP's managed identity. Non-secret values (image, model, timeout, concurrency) use code defaults from `pkg/util/version/const.go` and `pkg/util/holmes/config.go`, with env var overrides. This uses `NewHolmesConfig(ctx, acrDomain, serviceKeyvault, credential)`.

**Soft-load behavior**: If loading fails (e.g., MSI credential unavailable or Key Vault secrets not provisioned), the RP logs a warning and starts normally. Only the investigate endpoint returns an error ("Holmes investigation is not configured"). This allows the RP to operate without Holmes configured.

The loaded config is stored on the `frontend` struct as `holmesConfig *holmes.HolmesConfig` and reused for all investigation requests.

## How Config Reaches the Pod

When an investigation request arrives, the RP creates three Kubernetes resources in the cluster's Hive namespace:

1. **Secret** (`holmes-kubeconfig-{id}`) — Contains:
   - `config`: Short-lived (1h) kubeconfig for `system:aro-diagnostics` identity
   - `azure-ad-token`: Short-lived Entra ID token acquired per-request via `HolmesConfig.AcquireToken()`
   - `azure-api-base`: From `holmesConfig.AzureAPIBase`
   - `azure-api-version`: From `holmesConfig.AzureAPIVersion`

2. **ConfigMap** (`holmes-config-{id}`) — Embedded toolset config from `pkg/hive/staticresources/holmes-config.yaml` (defines which kubectl commands Holmes can use)

3. **Pod** (`holmes-investigate-{id}`) — Runs:
   ```
   python holmes_cli.py ask "<question>" -n --model=<Model> --config=/etc/holmes/config.yaml
   ```
   - Image from `holmesConfig.Image` (default: `version.HolmesImage(acrDomain)`)
   - `ActiveDeadlineSeconds` from `holmesConfig.DefaultTimeout`
   - Azure credentials injected as environment variables from the Secret
   - Kubeconfig mounted at `/etc/kubeconfig/config` (Secret Items filter)
   - `HostAliases` maps `api-int.*` hostname to the cluster's `APIServerPrivateEndpointIP`
   - `imagePullSecrets` references `hive-global-pull-secret` for ACR authentication

All three resources are cleaned up after the investigation completes (or fails).

## Development Setup

1. Ensure prerequisites: VPN connected, `secrets/env` generated, `aks.kubeconfig` generated

2. Export environment variables:
   ```bash
   source env && source secrets/env
   export HIVE_KUBE_CONFIG_PATH=$(realpath aks.kubeconfig)
   export ARO_INSTALL_VIA_HIVE=true
   export ARO_ADOPT_BY_HIVE=true
   export HOLMES_IMAGE="arointsvc.azurecr.io/holmesgpt:latest"
   # You can override HOLMES_IMAGE with a different image for testing
   ```

3. Start the local RP: `make runlocal-rp`

4. Run an investigation:
   ```bash
   ./hack/test-holmes-investigate.sh <cluster-name> "what is the cluster health status?"
   ```

## Provisioning (Staging/Production)

**Key Vault:** Create the following secret in the service Key Vault (`{KEYVAULT_PREFIX}-svc`):

| Secret Name | Value |
|-------------|-------|
| `holmes-azure-api-base` | Azure OpenAI endpoint URL (must use custom subdomain, e.g., `https://<name>.openai.azure.com`) |

**RBAC:** Assign the **Cognitive Services OpenAI User** role to the RP's managed identity on the Azure OpenAI resource.

**Azure OpenAI resource:** Must have `disableLocalAuth=true` and a custom subdomain configured.

Non-secret config uses code defaults defined in `pkg/util/version/const.go` (image) and `pkg/util/holmes/config.go` (model, timeout, concurrency). These can be overridden via environment variables.

## Security

- **Cluster access**: Investigation pods use a `system:aro-diagnostics` identity with read-only RBAC (get/list/watch only). The kubeconfig certificate expires after 1 hour.
- **Pod security**: Runs as non-root (UID 1000), no privilege escalation, all capabilities dropped, service account token not mounted, FSGroup set for writable emptyDir volumes.
- **DNS resolution**: Pod uses `HostAliases` to map `api-int.*` to the cluster's private endpoint IP, bypassing DNS and preserving TLS certificate validation.
- **Toolset restrictions**: Destructive commands (`kubectl delete`, `kubectl apply`, `kubectl exec`, `rm`) are blocked in the Holmes toolset config.
- **Rate limiting**: Per-RP-instance CAS-based atomic counter limits concurrent investigations (default 20).
- **Input validation**: Question limited to 1000 characters, control characters rejected (including DEL), model name validated against safe character pattern.

## Code Locations

| Component | File |
|-----------|------|
| Config struct and loaders | `pkg/util/holmes/config.go` |
| Holmes image constant | `pkg/util/version/const.go` |
| Config loading at startup | `pkg/frontend/frontend.go` (search `holmesConfig`) |
| Admin API handler | `pkg/frontend/admin_openshiftcluster_investigate.go` |
| Kubeconfig generation | `pkg/frontend/admin_openshiftcluster_investigate_kubeconfig.go` |
| Pod creation, HostAliases, and streaming | `pkg/hive/investigate.go` |
| Holmes toolset config | `pkg/hive/staticresources/holmes-config.yaml` |
| RBAC ClusterRole | `pkg/operator/controllers/rbac/staticresources/clusterrole-diagnostics.yaml` |
| RBAC ClusterRoleBinding | `pkg/operator/controllers/rbac/staticresources/clusterrolebinding-diagnostics.yaml` |
| E2E test script | `hack/test-holmes-investigate.sh` |
