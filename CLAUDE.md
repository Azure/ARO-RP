# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Azure Red Hat OpenShift Resource Provider (ARO-RP) is a production-grade Azure Resource Manager resource provider for managing OpenShift clusters on Azure. It's jointly engineered by Microsoft and Red Hat.

The single `aro` binary serves multiple service modes: `rp` (resource provider), `monitor`, `portal`, `gateway`, `operator`, `deploy`, `mirror`, `mimo-actuator`, `update-versions`, and `update-role-sets`.

## Build and Development Commands

### Building

```bash
make aro                    # Build the ARO binary
make build-all              # Build all Go binaries
make generate               # Run code generation (go generate, swagger)
make fmt                    # Format with golangci-lint (gci, gofumpt) - both root and pkg/api modules
```

### Running Locally

```bash
make runlocal-rp            # Run the resource provider locally
make runlocal-monitor       # Run the monitor locally
make runlocal-portal        # Run the portal locally
```

### Testing

```bash
make test-go                # Full validation: generate, build, validate, lint, unit tests
make unit-test-go           # Run unit tests with coverage (uses gotestsum)
make lint-go                # Run golangci-lint
make lint-go-fix            # Run linter with auto-fix
make test-e2e               # Run end-to-end tests (requires cluster)

# Run a single test
go test -v ./pkg/frontend/... -run TestSpecificFunction
```

### Validation

```bash
make validate-go            # Format check, license check, client freshness
make imports                # Fix import ordering (runs lint-go-fix)
```

### Code Generation

```bash
make generate-swagger                 # Generate OpenAPI specs from API definitions
make generate-operator-apiclient      # Generate Kubernetes operator client
make client-generate                  # Regenerate Azure SDK clients (requires Docker + autorest image)
make client                           # Full pipeline: generate + client-generate + lint-go-fix + lint-go
make image-autorest                   # Build autorest Docker image locally (needed before client-generate)
```

### Recommended Change Validation Order

When modifying API types (`pkg/api/v*`):

1. `make fmt` - fix formatting
2. `make unit-test-go` - run unit tests
3. `make image-autorest && make client` - regenerate clients (only if swagger-facing types changed)

## Architecture

### Core Pattern: Async Update Model

- **Frontend** (`pkg/frontend`): Accepts PUT/DELETE requests, writes to CosmosDB with non-terminal provisioningState (Updating/Deleting)
- **Backend** (`pkg/backend`): Polls database for documents with non-terminal states, processes them, updates with terminal state (Succeeded/Failed)
- **Database**: CosmosDB with optimistic concurrency control (see `RetryOnPreconditionFailed`)

### Key Packages

| Package                    | Purpose                                                                               |
| -------------------------- | ------------------------------------------------------------------------------------- |
| `pkg/api`                  | Internal/external API definitions with versioning (v20240812preview, v20250725, etc.) |
| `pkg/frontend`             | RP REST API handlers                                                                  |
| `pkg/backend`              | Async cluster operations worker                                                       |
| `pkg/cluster`              | Cluster create/update/delete using OCP installer                                      |
| `pkg/database`             | CosmosDB wrapper layer                                                                |
| `pkg/env`                  | Environment shims (prod/dev/test)                                                     |
| `pkg/operator/controllers` | Kubernetes controllers for ARO operator                                               |
| `pkg/util`                 | Utilities (must use subpackages, no files directly in pkg/util)                       |

### Cluster Packages (important distinction)

| Package            | Purpose                                                                                                                                                              |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/cluster`      | **Production** cluster manager used by the backend worker (`pkg/backend`) for real cluster create/update/delete via OCP installer                                    |
| `pkg/util/cluster` | **Test/dev/CI** tooling for creating test clusters via `hack/cluster` and E2E setup. Not part of the production RP path. Requires `CI=true` or `RP_MODE=development` |
| `pkg/deploy`       | RP infrastructure deployment config (`aro deploy`). `Configuration` struct governs RP VMSS, CosmosDB, DNS, etc. Not consumed by `pkg/util/cluster`                   |

CI/dev-specific behavior (VM size retry on quota errors, VM size shuffling) belongs in the test tooling config (`pkg/util/cluster/ClusterConfig`), driven by explicit config fields rather than ad-hoc env var checks in business logic. The deploy `Configuration` declares CI mode canonically but does not cross-depend with the cluster test config.

### Multi-module Structure

The repository has two Go modules:

- Root module (`github.com/Azure/ARO-RP`)
- API module (`pkg/api/go.mod` - `github.com/Azure/ARO-RP/pkg/api`)

When running `go mod tidy`, use `make go-tidy` to handle both modules.

**Multi-module gotchas:**

- `go build ./...` from the repo root only builds the root module. To build/test `pkg/api/` packages, run from the `pkg/api/` directory: `cd pkg/api && go build ./...`
- `make unit-test-go` handles both modules automatically via gotestsum
- `make fmt` runs `golangci-lint fmt` in both module roots (not `gofmt`)
- `make lint-go-fix` also runs in both modules: root and `cd pkg/api/`

## Common Pitfalls (learned from experience)

### Three distinct VMSize types

The codebase has three separate `VMSize` string types that are NOT interchangeable:

1. **`api.VMSize`** (`pkg/api/openshiftcluster.go`) — the internal type used in CosmosDB documents
2. **`vms.VMSize`** (`pkg/api/util/vms/types.go`) — the centralized VM size utility type with metadata, used by the `admin` API and `validate` package
3. **Local `VMSize`** (each `pkg/api/v*/openshiftcluster.go`) — the external ARM-facing type matching swagger

Conversion files (`_convert.go`) must explicitly cast between these types. Getting the cast direction wrong causes compile errors that look like `cannot use vms.VMSize as VMSize value`.

### Admin API is different from external APIs

The `pkg/api/admin/` package is an **internal** API (mutual TLS auth, not customer-facing). Its struct fields use `vms.VMSize` directly (same as internal types). Do NOT apply the same "use local VMSize" pattern to admin that applies to external `pkg/api/v*` packages.

### `validate.VMSizeIsValidForVersion` signature

This function's signature includes a `requireD2sWorkers` parameter that exists on `master` but may not exist on older branches. Always check the current signature before calling it. Similarly, `validate.VMSizeIsValid` has evolved over time.

### `client-generate` is destructive

`hack/apiclients/build-dev-api-clients.sh` **deletes all generated clients** (`rm -rf pkg/client/services/redhatopenshift/mgmt`) before regenerating them. If the Docker autorest image isn't available or the script fails mid-run, the generated Go/Python SDK files will be missing and unit tests will fail with `no required module provides package` errors. Fix by restoring from git: `git checkout -- pkg/client/ python/client/`

### `make fmt` vs `gofmt`

The Makefile `fmt` target uses `$(GOLANGCI_LINT) fmt` (which runs gci + gofumpt), NOT plain `gofmt`. Always use `make fmt` to match CI expectations. The pre-commit hook (`.git/hooks/pre-commit`) calls `make fmt`.

### Files that are NOT customer-facing (dev/CI only)

These files are used only for development and CI, not in the production RP binary:

- `pkg/util/cluster/` — Test cluster creation tooling (reads env vars via Viper)
- `hack/cluster/` — CLI tool for manual cluster creation
- `test/e2e/` — E2E test suite
- `pkg/util/cluster/cluster.go` `ClusterConfig` uses `vms.VMSize` for VM size fields — changes here don't affect customer API surface

## API Type System: Internal vs External Boundary

### Type Architecture Overview

The RP maintains a strict **internal/external type boundary**:

- **Internal types** (`pkg/api/openshiftcluster.go`): Single source of truth for cluster state, stored in CosmosDB
- **External types** (`pkg/api/v*/openshiftcluster.go`): ARM-facing types matching swagger definitions, one per API version
- **Admin types** (`pkg/api/admin/openshiftcluster.go`): Internal admin API, not customer-facing

### API Version Registration

Each API version registers itself via `init()` in its `register.go` file into the global `api.APIs` map (`pkg/api/register.go`):

```go
// pkg/api/v20250725/register.go
func init() {
    api.APIs["2025-07-25"] = &api.Version{
        OpenShiftClusterConverter:    openShiftClusterConverter{},
        OpenShiftClusterStaticValidator: openShiftClusterStaticValidator{},
        // ... other converters
    }
}
```

The `api.Version` struct defines all converters and validators for a version:

- `OpenShiftClusterConverter` - `ToExternal()`, `ToInternal()`, `ExternalNoReadOnly()`
- `OpenShiftClusterStaticValidator` - `Static()` validation
- `OpenShiftClusterCredentialsConverter` - Credentials response
- `OpenShiftVersionConverter` - OCP version info
- `PlatformWorkloadIdentityRoleSetConverter` - Workload identity role sets

### External Type Constraints

**External API types (`pkg/api/v*`) must match the swagger spec exactly.** The swagger defines `VMSize` as `"type": "string"`, so external types must use a local `type VMSize string`, not import types from internal utility packages. The conversion layer (`_convert.go`) bridges the gap with explicit type casts.

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
    ClusterName    string     `mapstructure:"CLUSTER"`
    SubscriptionID string     `mapstructure:"AZURE_SUBSCRIPTION_ID"`
    IsCI           bool       `mapstructure:"CI"`
    RpMode         string     `mapstructure:"RP_MODE"`
    MasterVMSize   vms.VMSize `mapstructure:"MASTER_VM_SIZE"`
    WorkerVMSize   vms.VMSize `mapstructure:"WORKER_VM_SIZE"`
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
- Python 3.6-3.10 (for az aro CLI extension)
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
