# CLAUDE.md

Azure Red Hat OpenShift RP — ARM resource provider for OpenShift clusters on Azure. A single `aro` binary is produced with both RP-side services and the in-cluster ARO Operator, with the service selected in the command line (e.g. `aro rp`, `aro portal`, `aro operator master`)

All Golang code in this repository is written for Go 1.25+.

## Architecture Invariant

Cluster mutations (PUT/DELETE) are **async**: Frontend writes to CosmosDB with non-terminal state → Backend polls and processes → updates with terminal state. Never bypass this: public API frontend handlers must NOT perform cluster operations directly.

## Two Go Modules (critical)

| Module | Path | `go.mod` |
|--------|------|----------|
| Root | `github.com/Azure/ARO-RP` | `go.mod` |
| API | `github.com/Azure/ARO-RP/pkg/api` | `pkg/api/go.mod` |

Root imports API via `replace` directive. **`./...` from root excludes `pkg/api/` tests.**

> Read `docs/agent-guides/multi-module-build.md` when changing build, test, or formatting targets.

## Essential Commands

```bash
make fmt                 # Format BOTH modules (golangci-lint, NOT gofmt)
make unit-test-go        # Unit tests
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
| Changing Makefile, CI, or build targets | `docs/agent-guides/multi-module-build.md` |


**`client-generate` is destructive** — it deletes all generated SDK clients before regenerating. If Docker/autorest fails mid-run, restore with `git checkout -- pkg/client/ python/client/`.

## Where Code Runs

| Runtime context | Packages |
|----------------|----------|
| RP control plane (Azure VMSS) | `pkg/frontend`, `pkg/backend`, `pkg/cluster`, `pkg/monitor`, `pkg/gateway`, `pkg/portal` |
| Customer OpenShift cluster | `pkg/operator/controllers` |
| CI/dev only (NOT production) | `pkg/util/cluster`, `hack/cluster`, `test/e2e` |
| RP infra deployment | `pkg/deploy` |

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

- **Imports**: `gci` is used to order Golang imports. See `.golangci.yml`.
- **Formatting**: `make fmt` (not `gofmt`). Pre-commit hook runs `make fmt`.
- **Pointer utils**: Use `pkg/util/pointerutils`, not `autorest/to` or `k8s.io/utils/ptr`.
- **Error handling**: Wrap with `fmt.Errorf("...: %w", err)`. Use `errors.Is`/`errors.As`.
- **No files directly in `pkg/util/`** — must use subpackages.

## Definition of Done

Before considering any change complete:
1. `make fmt` passes
2. `make unit-test-go` passes
3. No new lint violations: `make lint-go`
