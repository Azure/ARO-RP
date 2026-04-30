# Decision: Constraint-Driven Agentic Development for ARO-RP

## Decision

ARO-RP will adopt a layered, phased harness for agentic development.

The target state is:

- tool-agnostic overall, so the governance model can apply to Cursor, Claude, Copilot, and future agents;
- Cursor-first in initial delivery, because the current repo-specific workflow guidance already exists there;
- worktree-based for all agent write operations;
- enforced by a local harness that treats the host checkout as read-only and the task worktree as read-write; and
- backed by repo-owned analyzers and policy gates that become mandatory through protected-branch governance.

## Context

### Problem

ARO-RP already has strong documentation, Makefile-based validation, CI, and protected-branch rulesets. Those controls catch many problems, but they do not yet close the enforcement set for agent-generated changes.

In particular:

- the repo guidance for agent workflows is not yet committed on `master`;
- no authoritative mechanism proves that an agent used a worktree or kept the host checkout read-only;
- root-module tests do not cover `pkg/api`, which makes dual-module validation easy to miss;
- important architecture constraints still depend heavily on human review; and
- policy that should be hard to weaken in the same PR is not yet separated from the code it is meant to judge.

### Constraints

The solution must respect the following ARO-RP invariants:

- the async mutation model, where `pkg/frontend` persists intent and `pkg/backend` performs cluster mutations asynchronously;
- the two-module model, where `pkg/api` requires separate test awareness;
- runtime-context boundaries between production RP code, operator code, CI-only code, and deployment tooling;
- internal/external/admin API type boundaries; and
- the existing GitHub and Azure DevOps release and merge model.

### Assumptions

- ARO-RP will continue to use GitHub as the review surface and protected-branch boundary for `master`.
- Local development and testing will continue to matter, so the first implementation cannot assume a remote-only execution model.
- Some constraints will evolve quickly during rollout and should initially stay in-tree.
- A smaller set of non-negotiable constraints should eventually move behind independently governed policy.

## Alternatives Considered

### Option 1: Advisory docs and review only

This option would rely on committed rules, `CLAUDE.md`, and human review without a local execution harness or new analyzers.

Why it was rejected:

- It improves clarity but not determinism.
- It does not prove worktree isolation.
- It leaves critical architecture rules as review-only concerns.

### Option 2: Host-checkout filesystem tricks

This option would use `chmod`, ACLs, or ownership changes to make the host checkout read-only to the agent.

Why it was rejected:

- It is brittle across operating systems.
- It can interfere with normal developer tooling.
- It is hard to standardize and audit.

### Option 3: Local dedicated-user execution

This option would run the agent as a separate local user with limited write access.

Why it was not chosen as the default:

- It is stronger than pure filesystem tricks.
- It is still more operationally awkward than a container harness.
- It creates more workstation setup burden than necessary for the first rollout.

### Option 4: Containerized local harness with attestation

This option runs the agent in a container or sandbox with:

- a read-only mount of the host checkout;
- a read-write mount of exactly one task worktree;
- minimal read-only credentials exposure; and
- execution attestation emitted into the writable worktree for CI validation.

Why it was chosen:

- It gives a clear, understandable boundary.
- It preserves normal human workflows on the host.
- It fits ARO-RP’s existing containerized development practices.
- It creates a practical bridge between local execution and CI enforcement.

### Option 5: Fully remote execution

This option would require all agent work to happen in a remote environment.

Why it was deferred:

- It is the strongest isolation option.
- It is much larger in cost, latency, and operational impact.
- It is not required to get a meaningful first enforcement win.

## Rationale

The chosen design balances determinism, usability, and rollout cost.

Docs alone are too weak. Filesystem tricks are too brittle. Fully remote execution is too heavy for the first step. A containerized local harness is the best trade-off because it creates an actual execution boundary without forcing the team to abandon the current local development model.

Just as importantly, the harness is only one layer. ARO-RP also needs repo-owned analyzers and policy checks so that architectural invariants are enforced before merge instead of being rediscovered in review or runtime.

This leads to a layered model:

1. committed repo context narrows the solution space;
2. the local harness constrains where and how the agent can write;
3. analyzers and policy checks validate the resulting change set;
4. GitHub rulesets and branch protection make the checks mandatory; and
5. runtime safeguards continue to protect residual risk.

## Consequences

### Positive

- Agent execution becomes materially more constrained and auditable.
- Reviewers spend less time checking mechanical architecture rules.
- High-value invariants move from tribal knowledge into deterministic enforcement.
- ARO-RP gets a reusable governance model that can outlive any single editor or agent product.
- The approach aligns with existing OPA/Gatekeeper precedent instead of inventing a parallel philosophy.

### Negative

- Local setup becomes more complex.
- The team must own and maintain a harness, analyzers, and policy lifecycle.
- Poorly designed analyzers could create false positives and drag on productivity.
- External policy governance adds process overhead and requires ownership outside the product diff.

## Cross-Cutting Concerns

### Security

- The host checkout should be mounted read-only to the agent.
- Credentials should be mounted minimally and preferably read-only.
- Policy that decides mergeability should not be trivially weakenable in the same PR.

### Developer Experience

- The first rollout should preserve the existing local inner loop as much as possible.
- The harness should integrate cleanly with worktrees, common shell flows, and containerized ARO-RP development.
- Diagnostics from analyzers must be specific and actionable.

### Governance

- Repo-local policy should be used for fast iteration.
- A smaller external signed bundle should hold non-negotiable merge policy.
- Protected-branch rulesets must require the policy gate that validates agent execution evidence.

### Measurement

Success should be measured through:

- valid execution attestation on agent-authored PRs;
- mandatory dual-module validation when `pkg/api` is touched;
- static enforcement of multiple high-value architectural invariants; and
- reduced review churn on rule-like comments.

## Initial Rollout

### Phase 1

- Commit Cursor-first workflow rules and supporting docs to `master`.

### Phase 2

- Deliver the local containerized harness with worktree-only write access and read-only host checkout access.

### Phase 3

- Add the first ARO-specific `go/analysis` passes for async mutation, runtime-context imports, CI branching in production code, and API type-boundary enforcement.

### Phase 4

- Add repo-local Rego policy for execution attestation and merge prerequisites.

### Phase 5

- Move non-negotiable policy into an externally governed or signed bundle and make its validation gate mandatory on the default branch.

## Related Documents

- `docs/constraint-driven-agentic-development.md`
- `docs/jira-epic-constraint-driven-agentic-development.md`
