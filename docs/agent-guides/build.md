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
