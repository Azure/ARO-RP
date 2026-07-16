# golangci-lint version sync (PR1)

## Change

The `golangci` job in `.github/workflows/ci-go.yml` no longer hardcodes `version: v2.12.2` on `golangci/golangci-lint-action`. A preceding shell step reads the semver pin from `.bingo/golangci-lint.mod` and passes it to the action `version` input.

Extracted value today: `v2.12.2` (field 3 on the bingo `require github.com/golangci/golangci-lint/v2 …` line).

## Why cache/install behavior is preserved

PR1 keeps `golangci/golangci-lint-action` unchanged apart from how the `version` input is supplied. The action still downloads/installs the linter binary and manages its GitHub Actions cache; only the duplicate hardcoded pin was removed so CI follows bingo as the source of truth.

## Validation

1. Confirmed the awk extraction from `.bingo/golangci-lint.mod` yields `v2.12.2`.
2. `make lint-go` — passed (exit 0, 0 issues, golangci-lint v2.12.2).

## Follow-up: goconst rollout

PR2 will add conservative `goconst` settings in `.golangci.yml` and depends on this workflow version-sync remaining the single CI pin.

## Slack blurb

```text
[Ready to Review] PR1 removes the duplicate golangci version pin in GitHub Actions.
The workflow still uses golangci-lint-action and its cache/install path, but now derives the tool version from bingo so CI matches the local source of truth.
```
