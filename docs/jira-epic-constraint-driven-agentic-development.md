# Jira Epic Draft: Constraint-Driven Agentic Development for ARO-RP

## Summary

Establish a layered enforcement model for agentic development in ARO-RP so that architectural invariants are encoded in committed guidance, local execution constraints, static analysis, CI policy, and protected-branch governance instead of relying primarily on human memory and review.

## Epic Goal

Make agent-generated changes safer to merge into `master` by:

- requiring isolated worktree-based execution for agent writes;
- making the host checkout read-only to the agent through a local harness;
- encoding high-value ARO-RP architectural invariants in `go/analysis` and Rego policy; and
- attaching those checks to mandatory protected-branch merge controls.

## User Story

**As an ARO-RP maintainer**  
**I want** a deterministic, layered harness for agentic development  
**So that** agent-authored changes cannot quietly bypass architectural rules that matter more than formatting, lint, or ordinary unit tests.

## Problem Statement

ARO-RP already has strong repo guidance and multiple technical gates, but the current enforcement model is still open in key places:

- repo guidance for agent workflows is not yet committed on `master`;
- no authoritative mechanism proves that an agent ran from an isolated worktree and treated the host checkout as read-only;
- root-module unit tests do not cover `pkg/api`, so dual-module validation is easy to miss;
- several architecture rules remain review-only concerns; and
- policy that should block unsafe changes can still be weakened in the same repo change unless it is independently governed.

The result is a class of failures where a change looks mechanically healthy but still violates ARO-RP’s operating model.

## Goals

- Commit a reusable, team-safe baseline of agent workflow rules to the repo.
- Introduce a local harness that gives the agent read-only host checkout access and read-write worktree access only.
- Produce execution attestation that CI can validate.
- Enforce the first set of ARO-specific invariants through `go/analysis`.
- Introduce repository policy checks for agent-execution and merge-governance invariants.
- Define which policy remains repo-local and which must move into an external signed bundle.

## Non-Goals

- Replacing all human review with automation.
- Building a fully remote-only agent platform as the first delivery.
- Solving every possible architecture invariant in the first analyzer set.
- Replacing existing runtime safeguards or product validation logic.
- Standardizing one single editor or agent product for the whole team forever.

## Proposed Workstreams

### Workstream 1: Invariant Catalog and Enforcement Firing Map

Deliverables:

- A written catalog of ARO-RP invariants and their earliest enforceable layer.
- A gap analysis of which invariants are still advisory or review-only.

Expected measurable impact:

- The team has an explicit map of what is currently enforced, what is not, and where new controls should fire.

Definition of done:

- The invariant catalog exists in repo docs.
- Each invariant is tagged to one or more layers: repo context, local harness, analyzer, CI policy, protected branch, or runtime guard.
- The catalog includes at least the async mutation model, dual-module validation, runtime-context boundaries, and API type boundaries.

### Workstream 2: Cursor-First Repo Baseline

Deliverables:

- Committed `.cursor` workflow rules and related guidance adapted from prior work such as `ARO-25801`.

Expected measurable impact:

- Agent workflow guidance becomes versioned and reviewable on `master` instead of existing only in local worktrees.

Definition of done:

- The baseline rules are committed to the repo.
- The rules are portable and do not depend on one developer’s local machine layout.
- The rules align with existing ARO-RP guidance in `CLAUDE.md` and `docs/agent-guides`.

### Workstream 3: Local Agent Harness and Worktree Attestation

Deliverables:

- A launcher or containerized harness that mounts the host checkout read-only and one task worktree read-write.
- An execution attestation artifact emitted into the worktree.

Expected measurable impact:

- 100% of supported agent sessions produce verifiable evidence of worktree-scoped execution.

Definition of done:

- The harness refuses to run without a git worktree.
- The host checkout is mounted read-only to the agent.
- The writable path is limited to the selected worktree.
- The attestation artifact is defined, documented, and consumed by a validation step.

### Workstream 4: Initial ARO-Specific `go/analysis` Pack

Deliverables:

- A repo-owned multichecker with an initial pass set.

Initial pass candidates:

- `frontendasyncmutation`
- `runtimecontextimports`
- `cienvinprod`
- `apitypeboundary`

Expected measurable impact:

- At least three high-value architectural invariants are enforced statically before merge.

Definition of done:

- The analyzer command exists in-tree.
- Analyzer diagnostics are deterministic and actionable.
- Unit tests cover both accepted and rejected patterns.
- CI runs the analyzer and fails on violations.

### Workstream 5: Repo-Local Rego Policy Gate

Deliverables:

- Rego policy and tests for validating attestation format and agent-governance prerequisites.

Expected measurable impact:

- Agent-execution policy becomes machine-checked in CI rather than inferred from PR text or reviewer assumption.

Definition of done:

- Policy sources and tests live in-tree.
- CI runs policy tests on every PR.
- The gate validates the attestation contract and required merge prerequisites.

### Workstream 6: External Signed Policy Bundle and Protected-Branch Attachment

Deliverables:

- A separately governed or signed policy bundle for non-negotiable rules.
- Protected-branch wiring that makes the policy gate mandatory.

Expected measurable impact:

- Unsafe PRs cannot weaken the exact policy that is judging them without crossing a separate governance boundary.

Definition of done:

- The external policy source is versioned and documented.
- The bundle is validated independently of the product diff.
- Protected-branch controls require the policy validation result before merge.

### Workstream 7: Review Reporting and Operational Rollout

Deliverables:

- A reviewer-visible summary of which agentic controls fired for a PR.
- Rollout guidance for contributors and maintainers.

Expected measurable impact:

- Reviewers spend less time rediscovering mechanical violations and more time evaluating design, scope, and risk.

Definition of done:

- The enforcement output is easy to inspect in PR checks.
- Maintainers know which gates are advisory, mandatory, or externally governed.
- Rollout documentation covers expected local workflow changes.

## Candidate Child Tasks

1. Build the invariant catalog and enforcement firing map for ARO-RP.
2. Upstream the committed Cursor-first baseline rules and local-integration guidance.
3. Design and prototype the local agent harness execution model.
4. Define the attestation schema and CI validation contract.
5. Implement the first ARO multichecker and analyzer test suite.
6. Wire analyzer execution into Make targets and GitHub Actions.
7. Create repo-local Rego policy and CI tests for agent-execution validation.
8. Define the external policy-bundle governance model and signing/versioning approach.
9. Attach the resulting gate set to protected-branch merge requirements.
10. Add PR reporting and maintainer-facing rollout guidance.

## Dependencies

- Agreement on the initial invariant catalog.
- Buy-in on the local execution model for agent sessions.
- Maintainer ownership for analyzer and policy upkeep.
- Protected-branch and ruleset updates where required.

## Acceptance Criteria

- Agent workflow guidance is committed and reusable on `master`.
- There is a supported path for worktree-only agent execution with a read-only host checkout.
- Execution attestation is produced and validated in CI.
- The first analyzer pack blocks at least three high-value architectural violations before merge.
- Repository policy validates the agent execution contract.
- Non-negotiable merge policy is not trivially weakenable in the same product diff.

## Definition of Done

This Epic is done when all of the following are true:

- ARO-RP has committed, reviewed, repo-local agent workflow rules.
- A supported local harness exists for isolated agent execution.
- The harness produces verifiable attestation and CI validates it.
- ARO-specific analyzers run in CI and block violations.
- Repo-local policy exists for fast-moving governance checks.
- External signed or separately governed policy protects the non-negotiable merge boundary.
- Protected-branch rules require the resulting enforcement gate.
- Maintainers have clear documentation for rollout, ownership, and review expectations.

## Expected Measurable Impact

- 100% of agent-authored PRs targeting `master` include valid execution attestation.
- 100% of agent-authored PRs touching `pkg/api` trigger dual-module validation.
- At least 3 architectural invariants are enforced statically before merge in the initial rollout.
- Reviewer comments about worktree misuse, CI-only imports into prod code, or frontend mutation-path violations trend toward zero after rollout.

## Breadcrumbs

- `docs/constraint-driven-agentic-development.md`
- `docs/adr-constraint-driven-agentic-development.md`
- `ARO-25801`
- `ARO-26090`
- `ARO-6541`
- `Template - Decision documentation`
