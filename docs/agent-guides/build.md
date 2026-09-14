# Build Guide

Read this when changing Makefile targets, CI workflows, build steps, or test infrastructure.

## Two Go Modules

| Module | `go.mod` | Module path |
|--------|----------|-------------|
| Root | `go.mod` | `github.com/Azure/ARO-RP` |
| API | `pkg/api/go.mod` | `github.com/Azure/ARO-RP/pkg/api` |

Root imports API via `replace github.com/Azure/ARO-RP/pkg/api => ./pkg/api` in `go.mod`.

### Use Makefile targets

Makefile targets perform tasks on both modules where applicable and includes required environment variables/configuration. Use them instead of tools directly.

For example:

* `make fmt` instead of direct `gofmt` invocations (calls `golangci-lint` with configured `gci` and `gofumpt` plugins)
* `make unit-test-go` for running all tests instead of `go test ./...`
* `make lint-go` for lint checks

## Recommended Validation Order

When modifying API types (`pkg/api/v*`):

1. `make fmt`
2. `make unit-test-go`
3. If swagger-facing types changed: `make generate-swagger`
4. If clients need regeneration: `make client`

## CI Workflows

| Workflow | What it checks |
|----------|---------------|
| `ci-go` | vendor-check, generate-check, golangci-lint, validate-go |
| `ci-python` | Python validation |
| `CodeQL` | Static analysis (Go, JS, Python) |
| `Test coverage` | 6 parallel suites: cmd, pkg-api, pkg-frontend, pkg-operator, pkg-util, pkg-other |
| `ci/prow/images` | OpenShift CI image build |
