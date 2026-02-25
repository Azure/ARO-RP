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
