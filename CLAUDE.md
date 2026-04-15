# CLAUDE.md

Azure Red Hat OpenShift RP — ARM resource provider for OpenShift clusters on Azure. Single `aro` binary, multiple modes: rp, monitor, portal, gateway, operator, deploy, mirror, mimo-actuator.

## Architecture Invariant

Cluster mutations (PUT/DELETE) are **async**: Frontend writes to CosmosDB with non-terminal state → Backend polls and processes → updates with terminal state. Never bypass this: frontend handlers must NOT perform cluster operations directly.

## Two Go Modules (critical)

| Module | Path | `go.mod` |
|--------|------|----------|
| Root | `github.com/Azure/ARO-RP` | `go.mod` |
| API | `github.com/Azure/ARO-RP/pkg/api` | `pkg/api/go.mod` |

Root imports API via `replace` directive. **`./...` from root excludes `pkg/api/` tests.** `make unit-test-go` only tests root. To test API: `cd pkg/api && go test ./...`.

> Read `docs/agent-guides/multi-module-build.md` when changing build, test, or formatting targets.

## Essential Commands

```bash
make fmt                 # Format BOTH modules (golangci-lint, NOT gofmt)
make unit-test-go        # Unit tests (root module only)
make lint-go             # Lint
make generate            # Code generation (go generate, swagger)
make go-tidy             # go mod tidy for BOTH modules
go test -v ./pkg/frontend/... -run TestSpecificFunction   # Single test
```

## Safety Rails

**STOP — read before touching these areas:**

| Trigger | Required reading |
|---------|-----------------|
| Modifying `pkg/api/v*` types | `docs/agent-guides/api-type-system.md` |
| Adding/changing VM sizes | `docs/agent-guides/azure-product-constraints.md` |
| Touching `pkg/cluster`, `pkg/util/cluster`, or `pkg/deploy` | `docs/agent-guides/package-deployment-context.md` |
| Changing Makefile, CI, or build targets | `docs/agent-guides/multi-module-build.md` |

**Three VMSize types exist** — they are NOT interchangeable:
1. `api.VMSize` — internal, stored in CosmosDB
2. `vms.VMSize` — utility type with metadata, used by admin API and validate
3. Local `VMSize` in each `pkg/api/v*/` — external ARM-facing, matches swagger

Conversion files (`_convert.go`) bridge them with explicit casts. Getting casts wrong → compile errors.

**`client-generate` is destructive** — it deletes all generated SDK clients before regenerating. If Docker/autorest fails mid-run, restore with `git checkout -- pkg/client/ python/client/`.

## Where Code Runs

| Runtime context | Packages |
|----------------|----------|
| RP control plane (Azure VMSS) | `pkg/frontend`, `pkg/backend`, `pkg/cluster`, `pkg/monitor`, `pkg/gateway`, `pkg/portal` |
| Customer OpenShift cluster | `pkg/operator/controllers` (26 controllers) |
| CI/dev only (NOT production) | `pkg/util/cluster`, `hack/cluster`, `test/e2e` |
| RP infra deployment | `pkg/deploy` |

> Read `docs/agent-guides/package-deployment-context.md` for the full context map.

## Admin API Handler Pattern ("Underscore Pattern")

Admin API endpoints decouple HTTP parsing from business logic. The main handler extracts parameters and calls an underscore-prefixed function with raw values:

```go
func (f *frontend) postAdminFoo(w http.ResponseWriter, r *http.Request) {
    // HTTP layer: extract params, get logger
    err := f._postAdminFoo(log, ctx, r)
    adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminFoo(log *logrus.Entry, ctx context.Context, r *http.Request) error {
    // Business logic: testable without HTTP mocking
}
```

This allows business logic to be invoked by other Go packages without HTTP mocking. When adding admin APIs, follow this pattern and register routes in `pkg/frontend/frontend.go`.

## Code Style (enforced by CI)

- **Imports**: 9-tier ordering enforced by gci. See `.golangci.yml`.
- **Formatting**: `make fmt` (not `gofmt`). Pre-commit hook runs `make fmt`.
- **Pointer utils**: Use `pkg/util/pointerutils`, not `autorest/to` or `k8s.io/utils/ptr`.
- **Error handling**: Wrap with `fmt.Errorf("...: %w", err)`. Use `errors.Is`/`errors.As`.
- **No files directly in `pkg/util/`** — must use subpackages.

## Definition of Done

Before considering any change complete:
1. `make fmt` passes
2. `make unit-test-go` passes
3. If `pkg/api/` changed: `cd pkg/api && go test ./...`
4. If swagger-facing types changed: `make generate-swagger` and verify output
5. No new lint violations: `make lint-go`

Files per API version:

- `openshiftcluster.go` - External struct definitions
- `openshiftcluster_convert.go` - `ToExternal()` / `ToInternal()` conversion with type casts
- `openshiftcluster_validatestatic.go` - Request validation
- `register.go` - API version registration into `api.APIs` map
- `generate.go` - `//go:generate` directives for swagger generation

### Adding New VM Sizes

When adding new instance types, update these files (see `docs/adding-new-instance-types.md`):

1. `pkg/api/openshiftcluster.go` - Internal VMSize constants
2. `pkg/api/admin/openshiftcluster.go` - Admin API constants
3. `pkg/api/validate/vm.go` - Supported VM size maps (master and worker separately)
4. `pkg/validate/dynamic/quota.go` - Required resources per VM size
5. Older versioned APIs in `pkg/api/v*` had their own VMSize constants but this was deprecated in v20220401

## Swagger and Client Generation Pipeline

### Swagger Generation Flow

```
pkg/api/v*/openshiftcluster.go  →  hack/swagger/  →  swagger/{stable|preview}/{version}/redhatopenshift.json
```

- **Generator tool**: `hack/swagger/swagger.go` calls `pkg/swagger/swagger.go`
- **Makefile target**: `make generate-swagger`
- Each API version has a `generate.go` with `//go:generate go run ../../../hack/swagger ...`
- Swagger JSONs are **owned by Microsoft ARM team** and live in `swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/`

### Client Generation Flow

```
swagger/{version}/redhatopenshift.json  →  AutoRest (Docker)  →  pkg/client/services/redhatopenshift/mgmt/{version}/
```

- **Script**: `hack/apiclients/build-dev-api-clients.sh`
- **Requires**: Docker image `arointsvc.azurecr.io/autorest:3.7.2` (build locally with `make image-autorest`)
- **WARNING**: The script **deletes all existing generated clients** before regenerating. If it fails mid-run (e.g., Docker not available), generated files will be missing.
- Generates both Go SDK (`pkg/client/`) and Python SDK (`python/client/`)
- Generated Go clients are consumed by `pkg/util/azureclient/mgmt/redhatopenshift/{version}/`
- Only generates clients for versions listed in the `client-generate` Makefile target (currently `2024-08-12-preview` and `2025-07-25`)
- The `.sha256sum` file tracks client freshness for CI validation

## Frontend Request Flow

### Route Structure

The frontend (`pkg/frontend/frontend.go`) defines routes in `chiAuthenticatedRoutes()` and `chiUnauthenticatedRoutes()`.

**Public ARM routes** (authenticated via MISE + TLS cert fallback):

- `PUT/PATCH/GET/DELETE /subscriptions/.../openShiftClusters/{name}` - CRUD operations
- `POST .../listcredentials` / `POST .../listadmincredentials` - Credentials
- `GET /providers/.../openshiftversions` - Available OCP versions
- `GET /providers/.../platformworkloadidentityrolesets` - Role sets
- `POST .../deployments/.../preflight` - Preflight validation

**Admin routes** (authenticated via mutual TLS certificate):

- `GET /admin/supportedvmsizes?vmRole={master|worker}` - Lists supported VM sizes. **Hidden dependency**: `pkg/frontend/admin_supportvmsizes_list.go` calls `validate.SupportedVMSizesByRole()` and directly JSON-marshals `map[api.VMSize]api.VMSizeStruct`, so the `VMSizeStruct` JSON tags in `pkg/api/openshiftcluster.go` directly affect this API's response format.
- `GET/PUT /admin/versions` - Manage OCP versions in CosmosDB
- `POST .../admin/.../redeployvm|stopvm|startvm` - VM lifecycle (emit maintenance signals)
- `GET .../admin/.../kubernetesobjects` - Direct k8s API access

### PUT/PATCH Handler Flow (`openshiftcluster_putorpatch.go`)

1. Validate subscription state (must be `Registered`)
2. Get existing cluster document from CosmosDB (or create skeleton for new cluster)
3. Check provisioning state is terminal (not in-progress)
4. Enrich document with current cluster data (10s timeout)
5. Convert internal → external, strip read-only fields, unmarshal request body
6. **For CREATE**: Run full validation chain (static + SKU + quota + providers)
7. **For UPDATE**: Run static validation only
8. Convert external → internal, preserve immutable fields (ID, Name, Type, SystemData)
9. Set provisioning state to `Creating`/`Updating`/`AdminUpdating`
10. Persist to CosmosDB (with `RetryOnPreconditionFailed` for optimistic concurrency)
11. Return 201 Created / 200 OK with `Location` and `Azure-AsyncOperation` headers

### Frontend Handler Pattern

```go
func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
    converter := f.apis[r.URL.Query().Get(api.APIVersionKey)].OpenShiftClusterConverter
    b, err := f._getOpenShiftCluster(ctx, log, r, converter)
    reply(log, w, nil, b, err)
}
```

Handlers look up the version-specific converter from `f.apis[apiVersion]`, enabling version-agnostic request handling.

## CosmosDB Document Lifecycle

### Key Document Types

- `OpenShiftClusterDocument` - Main cluster state (partition key: subscription ID lowercased)
- `AsyncOperationDocument` - Async operation tracking
- `SubscriptionDocument` - Subscription registration state
- `OpenShiftVersionDocument` / `PlatformWorkloadIdentityRoleSetDocument` - Configuration (updated via changefeed)
- `BillingDocument`, `MaintenanceManifestDocument`, `MaintenanceScheduleDocument` - Operational data

### Dequeue Query

```sql
SELECT * FROM OpenShiftClusters doc
WHERE doc.openShiftCluster.properties.provisioningState IN ("Creating", "Deleting", "Updating", "AdminUpdating")
AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000
```

### Generated CosmosDB Code

`pkg/database/cosmosdb/` files prefixed with `zz_generated_` are auto-generated by `github.com/bennerv/go-cosmosdb`. The generator is invoked via `//go:generate` in `pkg/database/cosmosdb/generate.go`.

## Viper Config Flow (Test/CI Only)

`pkg/util/cluster/cluster.go` uses Viper for test/CI cluster creation config:

```go
type ClusterConfig struct {
    ClusterName    string   `mapstructure:"CLUSTER"`
    SubscriptionID string   `mapstructure:"AZURE_SUBSCRIPTION_ID"`
    IsCI           bool     `mapstructure:"CI"`
    RpMode         string   `mapstructure:"RP_MODE"`
    MasterVMSize   string   `mapstructure:"MASTER_VM_SIZE"`
    WorkerVMSize   string   `mapstructure:"WORKER_VM_SIZE"`
    // ... 15+ more fields
}
```

**Flow**: Environment variables → `viper.AutomaticEnv()` → `viper.Unmarshal(&conf)` → mapstructure tags → struct fields

**Callers** (only test/dev):

- `test/e2e/setup.go` - E2E test setup
- `hack/cluster/cluster.go` - Manual cluster creation tool

This is NOT used by the production RP. Production config comes from Azure Live Config (`pkg/util/liveconfig/`).

## Code Style Requirements

### Import Ordering

Imports must follow this order (enforced by golangci-lint with gci):

1. Standard library
2. Blank imports
3. Dot imports
4. Third-party packages
5. `k8s.io/*` packages
6. `sigs.k8s.io/*` packages
7. `github.com/Azure/*` packages
8. `github.com/openshift/*` packages
9. Local module packages

### Required Import Aliases

The linter enforces specific aliases for common packages. Key examples:

- `kerrors` for `k8s.io/apimachinery/pkg/api/errors`
- `metav1` for `k8s.io/apimachinery/pkg/apis/meta/v1`
- `arov1alpha1` for ARO operator v1alpha1 API
- `ctrl` for `sigs.k8s.io/controller-runtime`

See `.golangci.yml` for the complete list.

### Pointer Utilities

Use `github.com/Azure/ARO-RP/pkg/util/pointerutils` instead of:

- `github.com/Azure/go-autorest/autorest/to`
- `k8s.io/utils/ptr`

## Testing

### Test Framework

- Unit tests: Standard Go testing with gotestsum
- E2E tests: Ginkgo v2 with Gomega matchers
- Mocks: uber-go/mock (mockgen), generated via `//go:generate` directives in `generate.go` files
- Fake database: `test/database/inmemory.go` provides `NewFakeOpenShiftClusters()` for unit tests

### E2E Test Labels

```bash
make test-e2e E2E_LABEL="smoke"           # Run smoke tests
make test-e2e E2E_LABEL="!smoke"          # Exclude smoke tests
```

Default excludes: `!smoke&&!regressiontest`

## Environment Setup

### Required Tools

- Go 1.25.3
- Python 3.8-3.10 (for az aro CLI extension)
- Docker/Podman
- Azure CLI (`az`)

### Python Environment

```bash
make pyenv                  # Create Python venv with dependencies
make az                     # Build development az aro extension
```

### Secrets

Secrets are stored in Azure Storage and extracted locally:

```bash
SECRET_SA_ACCOUNT_NAME=<account> make secrets
```

## Release Process

Releases use annotated git tags: `vYYYYMMDD.nn` (e.g., `v20260205.00`).

- Before release: Tag the ADO RP-Config repository with the same tag
- Release pipeline: EV2 pipelines (manual via Azure DevOps)
- Image registries: `arosvc.azurecr.io` (production), `arointsvc.azurecr.io` (integration testing)
- GitHub release notes: Auto-generated via `.github/workflows/release-note.yml` on tag push

### Current Version Info

- Default OCP install stream: **4.17.44**
- Install architecture version: `ArchitectureVersionV2`
- Latest stable API: `2025-07-25`
- Latest preview API: `2024-08-12-preview`

## CI/CD

### Docker Images

```bash
make ci-rp                  # Build CI image (runs tests, generates coverage)
make image-aro-multistage   # Build production RP image
make image-e2e              # Build E2E test image
```

### Key CI Targets

- `make validate-go-action` - Full validation for CI (imports, lint, licenses, client freshness)
- `make unit-test-go-coverpkg` - Unit tests with full coverage package list

### CI Workflows

- `ci-go` - vendor-check, generate-check, golangci-lint, validate-go
- `ci-python` - validate-python
- `CodeQL` - Go, JavaScript, Python analysis
- `Test coverage` - 6 parallel test suites (cmd, pkg-api, pkg-frontend, pkg-operator, pkg-util, pkg-other)
- `ci/prow/images` - OpenShift CI image build
- `CI` (ADO) - Azure DevOps pipeline

## Code Generation Summary

60+ `//go:generate` directives producing:

- **Swagger types**: `hack/swagger/` → `swagger/{version}/redhatopenshift.json` (10 versions)
- **Controller code**: `controller-gen` → deepcopy, CRDs, RBAC
- **Mocks**: `mockgen` → `pkg/util/mocks/` (40+ packages)
- **Enumerations**: `enumer` → `zz_generated_*_enumer.go`
- **CosmosDB types**: `go-cosmosdb` → `pkg/database/cosmosdb/zz_generated_*`
- **Deployment manifests**: `hack/gendeploy` → ARM templates
- **Binary data**: `go-bindata` → embedded assets

## Azure Product Constraints

Source: [Azure Red Hat OpenShift support policy](https://learn.microsoft.com/en-us/azure/openshift/support-policies-v4)

These hard constraints are enforced by the RP and must be respected in code changes:

- **Cluster topology**: Exactly 3 master nodes, 3-250 worker nodes. Masters cannot be added/removed/replaced by customers.
- **SLA**: 99.95% availability
- **Control plane VM sizes**: Minimum 8 vCPU / 32 GiB RAM (e.g., Standard_D8s_v3). Full list in `pkg/api/validate/vm.go` `supportedMasterVmSizes` map.
- **Worker VM sizes**: Minimum 4 vCPU (e.g., Standard_D4s_v3). Includes general purpose, memory optimized, compute optimized, storage optimized, and GPU families.
- **Version-gated VM sizes**: Dsv6, Ddsv6, Dlsv6, Dldsv6, Lsv4 families require OpenShift **4.19+**. This is enforced in `pkg/api/validate/vm.go` via `masterVmSizesWithMinimumVersion` and `workerVmSizesWithMinimumVersion` maps.
- **Disk size**: Minimum 128 GiB for worker nodes (enforced in `validate.DiskSizeIsValid()`)
- **GPU nodes**: Some GPU sizes (ND96asr_v4, ND96amsr_A100_v4, NC*ads_A100_v4) are day-2 only — not available at install time
- **Standard_M128ms**: Does not support encryption at host
- **Authentication**: Service Principal or Workload Identity (managed identity) — determined by `UsesWorkloadIdentity()` on the cluster object
- **Networking**: Clusters require direct outbound internet access (or UserDefinedRouting). NSGs are managed by the service and cannot be customer-modified unless using bring-your-own NSG feature.

## Go Quality Guidelines

In addition to this project's existing style requirements (import ordering, pointer utilities, etc.), follow these general Go practices:

- **Error handling**: Return `error` and always check it. Wrap errors with context using `fmt.Errorf("...: %w", err)`. Use `errors.Is`/`errors.As` instead of string matching.
- **Context propagation**: Accept `context.Context` as the first argument in any function that performs I/O, RPCs, or may block. Propagate contexts through all layers with appropriate deadlines.
- **Resource cleanup**: Close files, connections, and response bodies with `defer` immediately after acquisition.
- **Concurrency**: Make goroutine lifetimes explicit. Every goroutine needs a clear exit path (context, done channel, or `WaitGroup`).
- **Interface design**: Keep interfaces narrowly focused. Start unexported and export only when real external consumers exist.
- **Test quality**: Keep tests small, fast, parallel, and deterministic. Use `go test -race` for important packages. Structure tests to run reliably in CI without timing dependencies.
- **Main package scope**: Restrict `main` packages to configuration, wiring, and process lifecycle. Place business logic in importable packages.