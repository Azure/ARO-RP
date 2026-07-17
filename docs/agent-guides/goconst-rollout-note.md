# goconst Rollout Note

This note records the conservative `goconst` rollout for PR2.

## Measured Findings

The counts below were measured from the PR2 worktree with the planned generated/generated-like exclusions in place:

- `min-len: 20`, `min-occurrences: 4`: 22 findings
- `min-len: 24`, `min-occurrences: 4`: 17 findings
- `min-len: 24`, `min-occurrences: 5`: 9 findings

At `24/5`, the findings set was small enough for a lint-infra PR and collapsed into a few obvious shared strings. Those findings were fixed in-scope, so the post-fix `goconst` result at the chosen threshold is 0 issues.

## Chosen Threshold

Chosen settings:

- `min-len: 24`
- `min-occurrences: 5`
- `match-constant: true`
- `eval-const-expressions: true`
- `ignore-tests: true`

Why this threshold:

- `20/4` and `24/4` still produced a broader cross-package cleanup than PR2 should carry.
- `24/5` was the lowest setting in the one-way sweep that produced a reviewable findings set after exclusions.
- Keeping the threshold conservative lets the repo start benefiting from `goconst` without turning the rollout into a large refactor.

## Excluded Paths

These exclusions stay in place because they are generated or generated-like code, or because they would add churn that is not useful for this rollout:

- `pkg/client/(.+)\.go`: generated SDK/client code
- `(.+/)?zz_generated_(.+)\.go`: Kubernetes/code-generator output
- `(.+/)?bindata.go`: generated bindata payloads
- `pkg/operator/(clientset|mocks)/(.+)\.go`: generated clientsets and mock-heavy support code
- `pkg/util/mocks/(.+)\.go`: generated mocks
- `pkg/util/graph/graphsdk/(.+)\.go`: generated Kiota/Graph SDK output
- `pkg/swagger/(.+)\.go`: generated swagger artifacts
- `test/e2e/(.+)\.go`: large test surface where duplicate strings are common and rollout noise would be high

## Why The Workflow Still Uses golangci-lint-action

PR2 does not replace `golangci-lint-action` because the action still provides the CI wrapper that installs and runs the configured linter cleanly in GitHub Actions, along with the existing caching and reporting behavior. This rollout only changes lint configuration; it does not require a custom plugin or custom workflow execution path. Related PR1 keeps the version-pin sync change separate by sourcing the workflow version from bingo.

## Follow-Up: constdup

If we want broader duplicate-constant detection later, evaluate the `constdup` plugin in a separate change after the baseline `goconst` rollout has settled. Keeping `constdup` out of PR2 avoids mixing lint-policy rollout with custom-plugin and workflow complexity.

## Slack Blurb

```text
DRAFT PR2 proposes a conservative goconst rollout for ARO-RP.
The thresholds are chosen empirically from a worktree run, generated and generated-like paths stay excluded, and the workflow continues to use golangci-lint-action. Related PR1 keeps the version-pin sync change separate.
```
