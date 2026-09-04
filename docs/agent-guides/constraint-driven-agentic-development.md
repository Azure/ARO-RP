# Constraint-Driven Agentic Development for ARO-RP

## Purpose

This document captures the research needed to move ARO-RP from advisory agent guidance to a layered enforcement model that can protect `master` from agent-generated changes which pass normal checks but violate architectural invariants.

The intended use is threefold:

1. As the technical baseline for an ADR.
2. As the scope definition for a Jira Epic and its child tasks.
3. As the source of truth for which constraints should be enforced by docs, local harnessing, static analysis, CI policy, and protected-branch governance.

## Problem Statement

ARO-RP already has strong human-oriented guidance and multiple validation gates, but the enforcement set for agentic development is still open.

Today an agent can be given good instructions and still produce a change that:

- passes `make fmt`, `make lint-go`, and root-module unit tests;
- misses the separate `pkg/api` validation path;
- preserves syntactic correctness while violating runtime-context boundaries;
- weakens policy in the same PR that the policy is meant to judge; or
- edits from the host checkout instead of an isolated worktree with no authoritative proof that isolation was respected.

Constraint-driven agentic development closes that gap by pushing critical invariants into mechanisms that fire earlier and more deterministically than human review alone.

## Current Enforcement Map

### Advisory repo context

ARO-RP already encodes important agent-facing constraints in repo docs:

- `CLAUDE.md` defines the async mutation invariant, the two-module model, dangerous commands, runtime-context boundaries, and the definition of done.
- `docs/agent-guides/multi-module-build.md` documents the `./...` exclusion trap for `pkg/api`, the formatting/lint behavior across both modules, and the expected validation order.
- `docs/agent-guides/api-type-system.md` defines the internal-vs-external API boundary and the non-interchangeable VM size types.
- `docs/agent-guides/package-deployment-context.md` defines where packages run and explicitly prohibits CI-only behavior from leaking into production paths.

This layer is valuable because it narrows the solution space for both humans and agents, but it is still advisory. It can be ignored, misread, or weakened in the same change.

### Local developer and agent validation

The current repo already has strong local chokepoints:

- `make fmt` formats both the root module and `pkg/api`.
- `make lint-go` runs the root lint suite.
- `make unit-test-go` runs root-module tests only.
- `make validate-go-action` enforces import cleanup, pinned GitHub Actions, license validation, no Go files directly under `pkg/util`, no Windows-incompatible filenames, and generated client checksum freshness.
- `make test-go` chains `generate`, `build-all`, `validate-go`, `lint-go`, and `unit-test-go`.

This layer is preventive and fast, but it remains bypassable locally and still relies on the caller knowing when to run the additional `pkg/api` test path.

### CI and pre-merge enforcement

ARO-RP already uses multiple CI systems:

- `.github/workflows/ci-go.yml` runs `go-verify`, `make generate` plus cleanliness checks, `golangci-lint`, and `make validate-go-action`.
- `.github/workflows/check-coverage.yml` covers test execution including `pkg/api`.
- `.github/workflows/ci-guardrailpolicies.yml` runs OPA and Gatekeeper policy tests for operator guardrail policies.
- `.pipelines/ci.yml` handles containerized CI and E2E in Azure DevOps.

This layer is closer to authoritative, but it is fragmented. The practical release-quality gate is spread across GitHub Actions, Azure DevOps, and protected-branch settings rather than one single repo-owned workflow definition.

### Protected-branch and ruleset enforcement

The default branch already benefits from non-bypassable governance:

- an organization ruleset requiring pull requests, code-owner review, and last-push approval;
- a repository ruleset requiring Copilot review on the default branch; and
- branch protection that marks `master` as protected.

This is the right level for authoritative enforcement, but the current required-check surface is not yet explicitly aligned with agent-specific analyzers or policy gates.

### Existing policy-as-code precedent

ARO-RP already accepts policy-as-code as a normal engineering tool:

- `pkg/operator/controllers/guardrails/policies` contains Gatekeeper policies and tests;
- `pkg/operator/controllers/guardrails/policies/scripts/test.sh` runs both `opa test` and `gator verify`;
- `.github/workflows/ci-guardrailpolicies.yml` makes those policies part of CI.

This precedent matters because it means Rego is not a foreign addition to ARO-RP. What is missing is a comparable policy lifecycle for repository, workflow, and agent-execution invariants.

## Key Gaps

The most important gaps are:

1. There is no committed `.cursor` baseline on `master`, so agent workflow guidance is not yet versioned as part of the repo contract.
2. There is no authoritative proof that an agent ran in an isolated worktree or treated the host checkout as read-only.
3. Dual-module validation is still easy to misapply because root tests do not cover `pkg/api`.
4. The repo does not yet contain ARO-specific `go/analysis` passes that encode architectural invariants.
5. Policy that should be hard to weaken in the same PR does not yet live behind an external or separately governed bundle.

## Worktree Isolation Mechanisms

### Option 1: Repo rules only

Mechanism:

- Commit `.cursor/rules/*`, `CLAUDE.md`, and related docs.
- Instruct agents to always work from a worktree.

Advantages:

- Fast to roll out.
- Portable across tools.
- Low engineering cost.

Limitations:

- Fully advisory.
- No protection against an agent ignoring the rule.
- No evidence for CI or reviewers.

Verdict:

- Necessary as a baseline, but insufficient as the main control.

### Option 2: Filesystem permissions on the host checkout

Mechanism:

- Use `chmod`, ACLs, or ownership changes to make the host checkout read-only to the agent.

Advantages:

- Conceptually simple.
- Does not require a container runtime.

Limitations:

- Fragile across macOS and Linux.
- Easy to misconfigure.
- Risks breaking normal human workflows and editor behavior.
- Hard to scope cleanly to one agent process.

Verdict:

- Useful as a local experiment, but too brittle as the default architecture.

### Option 3: Dedicated local user for agent execution

Mechanism:

- Run the agent as a different local user with read-only access to the host checkout and write access only to a designated worktree.

Advantages:

- Stronger boundary than plain docs.
- Works without full containerization.

Limitations:

- Operationally awkward on developer machines.
- More setup friction.
- Harder to distribute consistently across the team.

Verdict:

- Stronger than `chmod`, but still not the preferred default for broad adoption.

### Option 4: Containerized local harness

Mechanism:

- Launch the agent inside a container or sandbox.
- Mount the host checkout read-only.
- Mount one task worktree read-write.
- Mount only the minimum credentials and tooling needed, ideally read-only.
- Emit an attestation file into the worktree describing harness version, branch, repo SHA, worktree path, mount mode, and start time.

Advantages:

- Strong, understandable boundary.
- Preserves normal human workflows on the host checkout.
- Works well with existing containerized development patterns in ARO-RP.
- Provides an obvious place to generate evidence that CI can validate.

Limitations:

- Requires harness engineering and platform support.
- Needs careful credential-handling design.
- Some developer ergonomics work is required for macOS and Linux parity.

Verdict:

- Recommended primary mechanism for ARO-RP.

### Option 5: Fully remote executor

Mechanism:

- Run all agent changes in a remote environment and only submit PR output back to GitHub.

Advantages:

- Strongest isolation boundary.
- Simplifies local workstation trust assumptions.

Limitations:

- Highest cost and highest latency.
- Harder to integrate with ARO-RP’s local dev and E2E workflows.
- Much larger operational change than the current team needs.

Verdict:

- A possible future phase, but not the recommended first implementation.

## Recommended Worktree Architecture

ARO-RP should adopt a container-first local harness with the following properties:

- The host checkout is mounted read-only.
- A single task worktree is mounted read-write.
- The harness refuses to start if the selected writable path is not a git worktree.
- The harness records attestation metadata inside the writable worktree.
- CI validates the attestation format and the branch/worktree relationship before merge.

The key design point is that GitHub alone cannot prove how the agent ran. The local harness must produce evidence, and the protected branch must require the policy check that validates that evidence.

## Initial `go/analysis` Strategy

### Integration model

The first ARO-specific analyzers should be implemented as a repo-owned multichecker rather than as a large initial GolangCI plugin.

Recommended first integration steps:

1. Add a small analyzer module under `hack/analysis/` with one command that runs all ARO analyzers.
2. Add a dedicated Make target such as `lint-go-aro-analyzers`.
3. Invoke that target from `make validate-go-action` and from `.github/workflows/ci-go.yml`.
4. Once the rules stabilize, decide whether to expose the same analyzers through `.golangci.yml` as a custom plugin or keep them as an explicit repo-owned step.

This keeps the first rollout simple and debuggable while still enforcing the rules before merge.

### First analyzer set

#### 1. `frontendasyncmutation`

Intent:

- Enforce the core ARO invariant that `pkg/frontend` handlers do not perform direct cluster mutations.

What it should catch:

- direct calls from `pkg/frontend` into mutation orchestration paths in `pkg/cluster`;
- direct Azure ARM create/update/delete flows from frontend handlers; and
- new patterns that bypass the CosmosDB async model.

Why first:

- This is the highest-value architectural invariant already documented in `CLAUDE.md`.

#### 2. `runtimecontextimports`

Intent:

- Enforce runtime-context boundaries from `docs/agent-guides/package-deployment-context.md`.

What it should catch:

- production RP packages importing `pkg/util/cluster`, `hack/cluster`, or `test/e2e`;
- operator controllers importing control-plane-only or dev-only packages; and
- new cross-runtime dependencies that make code look valid locally but fail in production.

Why first:

- This is a classic agent failure mode: technically valid imports that violate deployment context.

#### 3. `cienvinprod`

Intent:

- Prevent ad-hoc CI branching from leaking into production code.

What it should catch:

- `os.Getenv("CI")` and similar environment checks inside `pkg/frontend`, `pkg/backend`, `pkg/cluster`, and other production runtime packages unless explicitly whitelisted.

Why first:

- The repo already documents this as forbidden behavior, which makes it a strong candidate for deterministic enforcement.

#### 4. `apitypeboundary`

Intent:

- Enforce the external/internal/admin API type boundaries described in `docs/agent-guides/api-type-system.md`.

What it should catch:

- external `pkg/api/v*` exported fields that use internal or admin types directly;
- conversions that happen outside the intended `_convert.go` boundary; and
- new version packages that do not follow the version-registration and conversion contract.

Why first:

- This is subtle, repetitive, and easy for an agent to violate while still producing compilable code in some cases.

### Analyzer acceptance bar

The initial analyzers should only ship if they meet all of the following:

- low false-positive rate on the current codebase;
- deterministic output in CI;
- clear remediation text in diagnostics; and
- unit tests that demonstrate both allowed and forbidden patterns.

## Rego Policy Governance

### What should remain repo-local first

The following policy should live in-tree during the early phases because it will evolve quickly with the implementation:

- the local harness attestation schema;
- the list of accepted agent-execution modes;
- PR metadata checks for worktree evidence;
- rules that couple tightly to current repo layout or current Make targets; and
- fast-moving policy that teams will tune based on early rollout feedback.

These rules should be testable in CI and owned with normal code review, but they are still mutable in the same repo.

### What should move to an external signed bundle

The following policy should move behind a separately governed, signed, or otherwise independently controlled bundle:

- the minimum mandatory gate set for default-branch merges;
- the attestation validation rules that decide whether agent execution was acceptable;
- protected-path ownership for `.cursor`, harness launchers, policy loaders, and enforcement scripts;
- approval requirements for weakening agent governance; and
- any rule whose purpose is to stop a PR from weakening the exact mechanism that judges it.

This is the right place to encode the "authoritative no" that should not be trivially editable inside the same product diff.

### Policy lifecycle recommendation

Adopt a two-tier policy model:

- Tier 1: repo-local policy for rapid iteration and fast feedback;
- Tier 2: externally governed policy bundle for non-negotiable merge requirements.

ARO-RP already has the cultural and technical precedent for this through Gatekeeper policy testing. The next step is to extend that pattern from cluster guardrails to repository governance.

## Measurable Outcomes

The phased rollout should target measurable outcomes rather than only new documents:

- 100% of agent-authored PRs to `master` carry valid execution attestation.
- 100% of agent changes that touch `pkg/api` trigger the `pkg/api` validation path.
- At least three high-value architectural invariants are enforced statically before merge.
- No normal contributor can weaken the mandatory policy gate in the same PR that is being judged.
- Reviewer effort shifts from checking mechanical invariants to judging design and risk.

## Recommended Delivery Sequence

1. Land the committed Cursor-first rules and supporting docs on `master`.
2. Build the containerized local harness and attestation format.
3. Ship the first ARO-specific analyzers through Make and CI.
4. Add repo-local Rego policy that validates the harness contract and merge prerequisites.
5. Move non-negotiable policy into an external signed bundle and attach it to the protected-branch path.
6. Add reporting so reviewers can see which controls fired for a given PR.

## Related Inputs

- Internal Jira reference: `ARO-25801`
- Internal Jira Epic reference: `ARO-6541`
- Internal Epic style reference: `ARO-26090`
- Internal Confluence template: `Template - Decision documentation`
- External proposal reference: <https://github.com/Azure/ARO-RP/pull/4739>
