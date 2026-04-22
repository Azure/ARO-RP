# Multi-Module Build Guide

Read this when changing Makefile targets, CI workflows, build steps, or test infrastructure.

## Two Go Modules

| Module | `go.mod` | Module path |
|--------|----------|-------------|
| Root | `go.mod` | `github.com/Azure/ARO-RP` |
| API | `pkg/api/go.mod` | `github.com/Azure/ARO-RP/pkg/api` |

Root imports API via `replace github.com/Azure/ARO-RP/pkg/api => ./pkg/api` in `go.mod`.

### The `./...` Exclusion Trap

Go's `./...` wildcard **excludes subdirectories with their own `go.mod`**. This means:

- `go build ./...` from root compiles `pkg/api` code transitively (via `replace` directive), so build errors surface
- `go test ./...` from root does **NOT** run tests inside `pkg/api/`
- `go mod tidy` from root does **NOT** update `pkg/api/go.mod`

**Consequence**: `make unit-test-go` only tests the root module. To test API module: `cd pkg/api && go test ./...`.

### Which Targets Handle Both Modules

| Target | Root | pkg/api | Notes |
|--------|------|---------|-------|
| `make fmt` | yes | yes | `golangci-lint fmt` in both |
| `make lint-go-fix` | yes | yes | Runs `--fix` in both |
| `make go-tidy` | yes | yes | `go mod tidy` in both |
| `make unit-test-go` | yes | **NO** | Only `gotestsum ./...` from root |
| `make lint-go` | yes | yes | Runs lint in both |

## Formatting: `make fmt` NOT `gofmt`

The `fmt` target runs `$(GOLANGCI_LINT) fmt` which uses **gci** (import ordering) + **gofumpt** (stricter gofmt). Plain `gofmt` will not match CI expectations.

The pre-commit hook (`.git/hooks/pre-commit`) calls `make fmt` and `make lint-go` on staged Go files.

## Import Ordering (enforced by gci)

Nine tiers, configured in `.golangci.yml`:

1. Standard library
2. Blank imports
3. Dot imports
4. Third-party (default)
5. `k8s.io/*`
6. `sigs.k8s.io/*`
7. `github.com/Azure/*`
8. `github.com/openshift/*`
9. Local module packages

### Required Import Aliases

The linter enforces specific aliases (see `.golangci.yml` for full list):

| Alias | Package |
|-------|---------|
| `kerrors` | `k8s.io/apimachinery/pkg/api/errors` |
| `metav1` | `k8s.io/apimachinery/pkg/apis/meta/v1` |
| `arov1alpha1` | ARO operator v1alpha1 API |
| `ctrl` | `sigs.k8s.io/controller-runtime` |

## Code Generation

60+ `//go:generate` directives producing:

| Category | Tool | Output |
|----------|------|--------|
| Swagger types | `hack/swagger/` | `swagger/{version}/redhatopenshift.json` |
| Controller code | `controller-gen` | deepcopy, CRDs, RBAC |
| Mocks | `mockgen` (uber-go/mock) | `pkg/util/mocks/` (40+ packages) |
| Enumerations | `enumer` | `zz_generated_*_enumer.go` |
| CosmosDB types | `go-cosmosdb` | `pkg/database/cosmosdb/zz_generated_*` |
| ARM templates | `hack/gendeploy` | deployment manifests |
| Binary data | `go-bindata` | embedded assets |

## Recommended Validation Order

When modifying API types (`pkg/api/v*`):

1. `make fmt`
2. `make unit-test-go`
3. `cd pkg/api && go test ./...`
4. If swagger-facing types changed: `make generate-swagger`
5. If clients need regeneration: `make image-autorest && make client`

## CI Workflows

| Workflow | What it checks |
|----------|---------------|
| `ci-go` | vendor-check, generate-check, golangci-lint, validate-go |
| `ci-python` | Python validation |
| `CodeQL` | Static analysis (Go, JS, Python) |
| `Test coverage` | 6 parallel suites: cmd, pkg-api, pkg-frontend, pkg-operator, pkg-util, pkg-other |
| `ci/prow/images` | OpenShift CI image build |

## Pointer Utilities

Use `github.com/Azure/ARO-RP/pkg/util/pointerutils`. Do NOT use:
- `github.com/Azure/go-autorest/autorest/to`
- `k8s.io/utils/ptr`
