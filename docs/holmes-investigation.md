# Holmes Investigation API

The Holmes investigation API is an admin endpoint that runs [HolmesGPT](https://github.com/robusta-dev/holmesgpt) diagnostic investigations on ARO clusters. It creates a short-lived pod on the Hive AKS cluster that connects to the target cluster, runs diagnostic queries, and streams the results back to the caller.

**Endpoint:** `POST /admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroup}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{clusterName}/investigate`

## Configuration Reference

| Config | Env Var | Key Vault Secret (prod) | Default | Required |
|--------|---------|------------------------|---------|----------|
| Azure OpenAI API key | `HOLMES_AZURE_API_KEY` | `holmes-azure-api-key` | — | Yes |
| Azure OpenAI endpoint | `HOLMES_AZURE_API_BASE` | `holmes-azure-api-base` | — | Yes |
| HolmesGPT container image | `HOLMES_IMAGE` | — | `quay.io/haoran/holmesgpt:latest` | No |
| Azure OpenAI API version | `HOLMES_AZURE_API_VERSION` | — | `2025-04-01-preview` | No |
| LLM model name | `HOLMES_MODEL` | — | `azure/gpt-5.2` | No |
| Pod timeout (seconds) | `HOLMES_DEFAULT_TIMEOUT` | — | `600` | No |
| Max concurrent investigations per RP | `HOLMES_MAX_CONCURRENT` | — | `20` | No |

## Config Loading

Configuration is loaded once at RP startup in `NewFrontend` (`pkg/frontend/frontend.go`).

**Development mode** (`RP_MODE=development`): All values are read from environment variables via `NewHolmesConfigFromEnv()`.

**Production mode**: Sensitive values (API key, API base) are read from the service Key Vault (`{KEYVAULT_PREFIX}-svc`). Non-secret values (image, model, timeout, concurrency) are read from environment variables. This uses `NewHolmesConfig(ctx, serviceKeyvault)`.

**Soft-load behavior**: If loading fails (e.g., Key Vault secrets not provisioned), the RP logs a warning and starts normally. Only the investigate endpoint returns an error ("Holmes investigation is not configured"). This allows the RP to operate without Holmes configured.

The loaded config is stored on the `frontend` struct as `holmesConfig *holmes.HolmesConfig` and reused for all investigation requests.

## How Config Reaches the Pod

When an investigation request arrives, the RP creates three Kubernetes resources in the cluster's Hive namespace:

1. **Secret** (`holmes-kubeconfig-{id}`) — Contains:
   - `config`: Short-lived (1h) kubeconfig for `system:aro-diagnostics` identity
   - `azure-api-key`: From `holmesConfig.AzureAPIKey`
   - `azure-api-base`: From `holmesConfig.AzureAPIBase`
   - `azure-api-version`: From `holmesConfig.AzureAPIVersion`

2. **ConfigMap** (`holmes-config-{id}`) — Embedded toolset config from `pkg/hive/staticresources/holmes-config.yaml` (defines which kubectl commands Holmes can use)

3. **Pod** (`holmes-investigate-{id}`) — Runs:
   ```
   python holmes_cli.py ask "<question>" -n --model=<Model> --config=/etc/holmes/config.yaml
   ```
   - Image from `holmesConfig.Image`
   - `ActiveDeadlineSeconds` from `holmesConfig.DefaultTimeout`
   - Azure credentials injected as environment variables from the Secret
   - Kubeconfig mounted at `/etc/kubeconfig/config`

All three resources are cleaned up after the investigation completes (or fails).

## Development Setup

1. Ensure prerequisites: VPN connected, `secrets/env` generated, `aks.kubeconfig` generated

2. Export Holmes environment variables:
   ```bash
   source env && source secrets/env
   export HIVE_KUBE_CONFIG_PATH=$(realpath aks.kubeconfig)
   export ARO_INSTALL_VIA_HIVE=true
   export ARO_ADOPT_BY_HIVE=true
   export HOLMES_IMAGE="quay.io/haoran/holmesgpt:latest"
   export HOLMES_AZURE_API_KEY="<your-azure-openai-key>"
   export HOLMES_AZURE_API_BASE="<your-azure-openai-endpoint>"
   ```

3. Start the local RP: `make runlocal-rp`

4. Run an investigation:
   ```bash
   ./hack/test-holmes-investigate.sh <cluster-name> "what is the cluster health status?"
   ```

## Key Vault Provisioning (Staging/Production)

Create the following secrets in the service Key Vault (`{KEYVAULT_PREFIX}-svc`):

| Secret Name | Value |
|-------------|-------|
| `holmes-azure-api-key` | Azure OpenAI API key |
| `holmes-azure-api-base` | Azure OpenAI endpoint URL (e.g., `https://<resource>.openai.azure.com`) |

Non-secret config (`HOLMES_IMAGE`, `HOLMES_MODEL`, etc.) is set via ARM deployment parameters in `pkg/deploy/generator/resources_rp.go` when added to the deployment template.

## Security

- **Cluster access**: Investigation pods use a `system:aro-diagnostics` identity with read-only RBAC (get/list/watch only). The kubeconfig certificate expires after 1 hour.
- **Pod security**: Runs as non-root (UID 1000), no privilege escalation, all capabilities dropped, service account token not mounted.
- **Toolset restrictions**: Destructive commands (`kubectl delete`, `kubectl apply`, `kubectl exec`, `rm`) are blocked in the Holmes toolset config.
- **Rate limiting**: Per-RP-instance atomic counter limits concurrent investigations (default 20).
- **Input validation**: Question limited to 1000 characters, control characters rejected, model name validated against safe character pattern.

## Code Locations

| Component | File |
|-----------|------|
| Config struct and loaders | `pkg/util/holmes/config.go` |
| Config loading at startup | `pkg/frontend/frontend.go` (search `holmesConfig`) |
| Admin API handler | `pkg/frontend/admin_openshiftcluster_investigate.go` |
| Kubeconfig generation | `pkg/frontend/admin_openshiftcluster_investigate_kubeconfig.go` |
| Pod creation and streaming | `pkg/hive/investigate.go` |
| Kubeconfig transformation (dev) | `pkg/util/holmes/kubeconfig.go` |
| Holmes toolset config | `pkg/hive/staticresources/holmes-config.yaml` |
| RBAC ClusterRole | `pkg/operator/controllers/rbac/staticresources/clusterrole-diagnostics.yaml` |
| RBAC ClusterRoleBinding | `pkg/operator/controllers/rbac/staticresources/clusterrolebinding-diagnostics.yaml` |
| E2E test script | `hack/test-holmes-investigate.sh` |
