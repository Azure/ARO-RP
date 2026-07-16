# constdup plugin spike

## Scope

PR3 adds a narrow `golangci-lint` module plugin in `tools/constdup`.

- Source of truth is limited to packages whose import path starts with `github.com/Azure/ARO-RP/pkg/util/version`.
- The analyzer only inspects raw Go string literals.
- It only reports literals that exactly match, or embed, registry-worthy shared constants discovered in that source package.
- It ignores generated files, excluded repo paths such as `pkg/client`, and the source-of-truth package itself.
- It does not try to enforce broad string deduplication or general style rules.

In the current repo state this intentionally targets the MUO case first, where `version.MUOImageTag` is the shared constant that other packages should compose from instead of retyping.

## Why goconst is insufficient

`goconst` is useful for repeated literals inside the code it sees, but it is not a good fit for the MUO problem.

- The desired source of truth already lives in another package: `pkg/util/version`.
- The problematic literal is often embedded inside a larger pullspec string, not repeated as a standalone exact literal.
- The goal is not "this string appears N times", it is "this specific shared constant should be referenced from the authoritative package".

The `constdup` plugin is therefore intentionally cross-package and source-of-truth-aware, while still staying narrow.

## Why go.work stays out

This spike keeps `tools/constdup` as a standalone Go module and relies on `golangci-lint` module-plugin support plus the local path in `.custom-gcl.yml`.

That is enough for:

- local `make lint-go`
- `golangci-lint-action` in CI
- `analysistest` coverage under `tools/constdup`

Adding `go.work` would widen module resolution across the repo for a v1 experiment without being required for the plugin build or the tests.

## Version sync

PR1 already made the GitHub Actions `golangci-lint-action` version come from `.bingo/golangci-lint.mod`.

PR3 adds `.custom-gcl.yml` with the same pinned version and a `make validate-custom-gcl` check that compares:

- `.bingo/golangci-lint.mod`
- `.custom-gcl.yml`

That validation now runs in:

- `make lint-go`
- `make lint-go-fix` for the root-module fix pass
- `make validate-go-action`
- the `golangci` workflow job before `golangci-lint-action`

The result is one effective source of truth for the version pin while still keeping `.custom-gcl.yml` explicit for module-plugin builds.

PR3 also wires `unit-test-constdup` into `make validate-go-action`, so the plugin's analysistest coverage is exercised in an existing CI-facing validation path instead of being local-only.

## Exact test commands

Run these from the repo root:

```bash
make unit-test-constdup
make validate-custom-gcl
make lint-go
make lint-go-fix
make unit-test-go
```

`make unit-test-go` is still the root-module test only. The plugin module coverage lives under `make unit-test-constdup`, and `make validate-go-action` now includes that target on clean-tree validation runs.

## Slack blurb

```text
DRAFT PR3 proposes a narrow constdup golangci plugin for ARO-RP to catch cross-package constant duplication that goconst cannot see.
It keeps golangci-lint-action in CI, uses module-plugin support, and requires .custom-gcl.yml to stay in sync with the bingo-managed version pin from PR1.
```
