# PR Scan Agent

Automated pull request review agent for ARO-RP that checks for correctness, security, reliability, and quality issues.

## Overview

The PR scan agent is a specialized Claude Code agent that performs systematic code review of pull requests against production safety criteria. It's designed for the ARO-RP codebase and understands:

- Azure Red Hat OpenShift architecture (frontend/backend async pattern, CosmosDB state, operator controllers)
- ARO-specific constraints (VMSize type system, two-module build, API versioning)
- Production safety requirements (customer impact, backwards compatibility, security)

## What It Checks

### Seven Categories (All Applied)

1. **Correctness & Logic**: nil pointers, race conditions, resource leaks, error handling
2. **API Compatibility**: breaking changes, ARM API contracts, CosmosDB schema migrations
3. **Security**: authz checks, input validation, secret handling, injection vulnerabilities
4. **Reliability**: retries, timeouts, graceful degradation, panic recovery
5. **Test Coverage**: unit tests for new code, two-module test verification, edge cases
6. **Observability**: logging levels, PII scrubbing, metrics, tracing
7. **Performance**: allocations, database query optimization, lock contention

### ARO-RP Specific Checks

- VMSize type confusion (three distinct types - see `api-type-system.md`)
- Package deployment context (control plane vs customer cluster vs CI)
- Admin API underscore pattern compliance
- Frontend-backend async mutation pattern (architecture invariant)
- Multi-module build considerations (root + pkg/api)

## How to Use

### Prerequisites

- Claude Code CLI, desktop app, or VS Code extension
- Git access to ARO-RP repository
- Working directory: `/path/to/ARO-RP`
- Optional: `gh` CLI for fetching PR metadata (install from https://cli.github.com/)

### Invocation Methods

#### Method 1: Via Makefile (Recommended)

The easiest way to scan a PR or branch:

```bash
# Auto-detect current branch (no PR/BRANCH needed!)
make pr-scan
make pr-scan MODE=quick

# Scan a specific PR by number
make pr-scan PR=1234

# Scan a specific branch
make pr-scan BRANCH=fix/my-feature

# Scan with custom base branch
make pr-scan BASE=master MODE=security

# All together
make pr-scan PR=1234 BASE=master MODE=pipeline
```

**Auto-detection**: If you don't specify `PR=` or `BRANCH=`, the script automatically scans your **current branch** against master. This is the fastest way to check your work-in-progress!

The Makefile target runs `hack/pr-scan.sh` which gathers context (diff, commits, changed files) and outputs it for Claude Code to analyze.

#### Method 2: Via Shell Script Directly

```bash
# Auto-detect current branch
./hack/pr-scan.sh
./hack/pr-scan.sh --auto --mode quick

# Scan specific PR
./hack/pr-scan.sh --pr 1234

# Scan specific branch
./hack/pr-scan.sh --branch fix/my-feature

# With all options
./hack/pr-scan.sh --branch fix/feature --base master --mode security

# List open PRs (requires gh CLI)
./hack/pr-scan.sh --list-open
```

**Auto-detection**: Running `./hack/pr-scan.sh` with no arguments automatically scans your current branch. Perfect for quick pre-commit checks!

The script outputs context to stdout. Pipe it to a file or copy/paste into Claude Code.

#### Method 3: Via Claude Code Chat (Manual)

Ask Claude Code directly without the script:

```
Scan PR 1234

Review branch fix/e2e-pipeline-install-go, pipeline mode

Check https://github.com/Azure/ARO-RP/pull/5678 for security issues

Scan PR #2222 in quick mode
```

Claude Code will fetch the diff and analyze it. The script methods provide richer context (PR metadata, formatted output).

### Input Formats Supported

| Format | Example | Script Option | Make Variable |
|--------|---------|---------------|---------------|
| Auto-detect | Current branch | `--auto` (default) | (none - auto) |
| PR number | `1234`, `#1234` | `--pr 1234` | `PR=1234` |
| Branch name | `fix/my-feature` | `--branch fix/feature` | `BRANCH=fix/feature` |
| Base branch | `master`, `main` | `--base master` | `BASE=master` |
| Mode | `full`, `quick`, `security`, `pipeline` | `--mode quick` | `MODE=quick` |
| GitHub URL | `https://github.com/Azure/ARO-RP/pull/5678` | N/A (extract PR number) | `PR=5678` |

### Review Modes

The agent supports four modes for different use cases:

#### Full Mode (Default)
Complete 7-category review with all severity levels:
- Correctness, API compatibility, security, reliability, tests, observability, performance
- Reports Blocker/High/Medium/Low/Nit findings
- Use for: Comprehensive pre-merge review

```bash
make pr-scan PR=1234
# or
make pr-scan PR=1234 MODE=full
```

#### Quick Mode
Rapid review focusing on critical issues only:
- Reports only Blocker and High severity findings
- Abbreviated summary (2-3 bullets)
- Skips Low/Nit findings
- Use for: Fast pre-merge checks, large PRs, time-sensitive reviews

```bash
make pr-scan PR=1234 MODE=quick
```

#### Security Mode
Deep security-focused review:
- Focus: Authz, secrets, injection, crypto, input validation
- Also checks correctness issues that create security risks
- Skips performance/style unless security-relevant
- Reports all severity levels for security findings
- Use for: PRs touching auth, customer data, security-sensitive packages

```bash
make pr-scan PR=1234 MODE=security
```

#### Pipeline Mode
CI/CD pipeline changes only:
- Target files: `.pipelines/**/*.yml`, `.github/workflows/*.yml`, CI scripts
- Focus: YAML syntax, templates, error handling, secret exposure, portability
- Skips application code unless it affects CI
- Use for: Build/test/deployment automation changes

```bash
make pr-scan PR=1234 MODE=pipeline
```

### Example Session

**Full mode:**
```
$ make pr-scan PR=1234

[pr-scan] Fetching PR #1234...
[pr-scan] Mode: full

=========================================
PR/Branch Scan Context
=========================================

PR Number: #1234
--- PR Details ---
feat: add macOS development environment support
Author: tuxerrante
Created: 2026-03-08T10:30:00Z
...

--- Changed Files ---
M docs/prepare-dev-environment.md
A hack/devtools/setup-macos.sh
...

[pr-scan] Context gathered. Paste this output into Claude Code for analysis.
```

Then paste into Claude Code or pipe directly.

**Quick mode:**
```
$ make pr-scan PR=1234 MODE=quick

[Mode: quick] Focus on Blocker and High severity findings only.
...

You: [Paste to Claude Code]

Agent Output:
### Summary
- Adds macOS dev environment support
- 1 High finding: error handling gap
- Recommendation: Fix error handling, then merge

### Findings

#### High
- **hack/devtools/setup-macos.sh:42**: Missing error check after brew install
  ...
```

## Output Format

The agent always produces structured output:

```
### Summary
[3-5 bullets: changes, risk level, recommendation]

### Findings
#### Blocker
[Must fix before merge: security holes, customer bugs, data loss]

#### High
[Significant issues: reliability, error handling, critical test gaps]

#### Medium
[Quality issues: edge cases, error messages, minor races]

#### Low / Nit
[Style, optimizations, clarity]

### Questions for Author
[Assumptions needing validation, missing context]

### Not Reviewed / Out of Scope
[Generated files, binaries, external deps, insufficient context]
```

## What It Doesn't Do

**Read-only inspection** - the agent will NOT:
- Modify your code or push changes
- Bypass security checks or commit secrets
- Run builds or tests (it checks if tests exist)
- Rewrite commit history
- Automatically comment on GitHub PRs

**Requirements**:
- Local git repository with branch available
- Ability to run `git diff`, `git log`, `git show`
- For remote PRs: `git fetch origin` to get latest refs
- **Optional**: `gh` CLI for PR metadata (title, author, description)
  - `hack/pr-scan.sh --pr` requires `gh`
  - `hack/pr-scan.sh --branch` works without `gh`
  - Script provides clear error if `gh` is missing when needed

## Limitations & Edge Cases

1. **Generated files**: Agent flags them as "not reviewed" (e.g., `zz_generated_*`, swagger output)
2. **Binary files**: Images, PDFs marked as out of scope
3. **External dependencies**: If PR depends on unreleased library changes, agent may miss context
4. **Complex Hive/MIMO logic**: Agent will flag "needs domain expert review" for deep operator/actuator changes
5. **Performance benchmarks**: Agent can spot obvious issues but won't run profiling

## Tips for Best Results

### Before Scanning
- Ensure branch is pushed and up to date: `git fetch origin`
- For large PRs, consider focus areas: "scan PR 1234, focus on security and tests"

### Interpreting Results
- **Blockers**: Do not merge until fixed
- **High**: Fix before merge unless exceptional circumstances (document why in PR)
- **Medium**: Fix if time permits, or create follow-up issue
- **Low/Nit**: Optional, team discretion

### Following Up
- **Questions for Author**: Add answers as PR comments for reviewers
- **Not Reviewed**: Human reviewer should examine those areas

## Integration with Workflow

### During Development
```bash
# Before pushing for review
git push origin my-feature-branch
# Ask Claude Code: "Scan branch my-feature-branch"
# Address findings, push fixes
```

### During PR Review
```bash
# Reviewer uses agent as first pass
# "Scan PR 5678"
# Focuses human review on areas agent flagged or couldn't assess
```

### Before Merge
```bash
# Final check after addressing review comments
# "Re-scan PR 5678, focus on changed files since last review"
```

## Customization

### Focus Areas
Request specific focus to narrow scope:
- "security only" - skips style/performance, emphasizes injection/authz
- "tests only" - checks coverage, edge cases, two-module verification
- "frontend changes" - emphasizes async pattern, ARM compatibility
- "operator changes" - focuses on k8s client usage, controller patterns

### Severity Threshold
Ask for filtered output:
- "Show only blockers and high severity"
- "What are the security findings?"

## Comparison with CI Checks

| Check | CI (Automated) | PR Scan Agent |
|-------|----------------|---------------|
| Linting (golangci-lint) | ✅ Yes | ✅ Validates ran |
| Unit tests pass | ✅ Yes | ✅ Checks coverage |
| Formatting (make fmt) | ✅ Yes | ✅ Validates ran |
| Security (CodeQL) | ✅ Yes | ✅ Deeper + context |
| Logic correctness | ❌ No | ✅ Reviews diffs |
| API compatibility | ❌ No | ✅ Breaking changes |
| Test coverage gaps | ❌ No | ✅ Missing edge cases |
| Production safety | ❌ No | ✅ Customer impact |

**Agent complements CI** - it catches logic issues, breaking changes, and missing test cases that pass automated checks.

## Troubleshooting

### "gh CLI not found"
**Error**: `hack/pr-scan.sh --pr 1234` fails with "gh CLI not found"

**Solution**:
- Install gh: `brew install gh` (macOS) or https://cli.github.com/
- Or use branch mode: `./hack/pr-scan.sh --branch pr-4649` (after `git fetch`)
- Or manual: Copy PR diff and paste into Claude Code

### "Could not find branch"
**Error**: `Branch 'fix/feature' not found locally or on origin`

**Solution**:
- Run `git fetch origin` to update remote refs
- Verify branch name: `git branch -a | grep <name>`
- For PRs: `./hack/pr-scan.sh --pr 1234` fetches automatically

### "Failed to diff against origin/master"
**Error**: Base branch not found

**Solution**:
- Check base branch name: this repo uses `master` (not `main`)
- Fetch: `git fetch origin master`
- Or specify: `make pr-scan PR=1234 BASE=main` (if different)

### "Diff too large to analyze"
**Solution**:
- Use quick mode: `make pr-scan PR=X MODE=quick`
- Use pipeline mode if only CI changes: `MODE=pipeline`
- Or ask Claude to focus: "scan PR X, focus on pkg/frontend only"

### "Not enough context"
**Solution**:
- Agent will list assumptions in "Questions for Author"
- Provide context: "Scan PR X; note: this depends on #Y being merged first"
- Use `--pr` with gh CLI to include PR description

### Script fails silently
**Solution**:
- Run with bash -x: `bash -x hack/pr-scan.sh --pr 1234`
- Check git status: `git status` (must be in repo)
- Check network: PR fetch requires GitHub access

## Examples

### Example 1: Quick Pre-Merge Check
```bash
$ make pr-scan PR=9999 MODE=quick

# Output provides context, paste into Claude Code
# Agent responds with Blocker/High findings only
```

**Result**: "1 High finding: missing error handling in admin API. Fix before merge."

### Example 2: Security Review for Auth Changes
```bash
$ make pr-scan BRANCH=feature/workload-identity MODE=security

# Agent focuses on authz, secrets, input validation
```

**Result**: Identified missing role check in new admin endpoint, flagged potential credential leak in logs.

### Example 3: Pipeline Changes Review
```bash
$ make pr-scan PR=5678 MODE=pipeline

# Reviews only .pipelines/*.yml and CI scripts
```

**Result**: Found `grep -Po` portability issue, suggested more portable alternative.

### Example 4: Full Review with gh Metadata
```bash
$ ./hack/pr-scan.sh --pr 1234

# Includes PR title, author, description in context
# Paste output into Claude Code
```

**Result**: Complete review with context from PR description informing the analysis.

### Example 5: Branch Diff Without gh CLI
```bash
$ git fetch origin
$ ./hack/pr-scan.sh --branch fix/my-feature --base master

# Works without gh CLI, uses git only
```

**Result**: Full diff and commit log for review.

### Example 6: List Open PRs Then Scan
```bash
$ ./hack/pr-scan.sh --list-open
#9999  feat: add workload identity support   feature/wli
#1234  ci: improve E2E job setup             fix/e2e-install-go
...

$ make pr-scan PR=9999 MODE=security
```

**Result**: Quick discovery of PRs to review, then targeted security scan.

## Contributing to the Agent

The agent definition lives at `.claude/agents/pr-scan.md`. To improve it:

1. **Add ARO-specific patterns**: Update checklist with recurring issues
2. **Refine severity calibration**: Adjust what counts as blocker vs high
3. **Extend focus areas**: Add new categories (e.g., "MIMO changes", "Hive integration")
4. **Update for new constraints**: When product requirements change, update API compatibility section

Changes to the agent definition take effect immediately in new Claude Code sessions.

## Related Documentation

- `docs/agent-guides/api-type-system.md` - Required reading for API changes
- `docs/agent-guides/multi-module-build.md` - Two-module test gotchas
- `docs/agent-guides/azure-product-constraints.md` - VMSize and quota rules
- `docs/agent-guides/package-deployment-context.md` - Where code runs
- `CLAUDE.md` - Architecture invariants and code style

## FAQ

**Q: Does the agent replace human code review?**
A: No. It catches common issues and provides first-pass analysis. Human reviewers provide domain expertise, architectural judgment, and context the agent can't access.

**Q: Can I run this in CI?**
A: Not yet. This is an interactive agent for local/manual use. CI automation would require API integration (future work).

**Q: What if the agent is wrong?**
A: Agents can miss context or flag false positives. Use judgment. If a finding is incorrect, that's good feedback to refine the agent.

**Q: How long does a scan take?**
A: Depends on PR size. Small PRs (5-10 files): 1-2 minutes. Large PRs (50+ files): 5-10 minutes. Use focus areas to speed up large scans.

**Q: Can I scan merged PRs?**
A: Yes, provide the commit SHA or tag: "Scan commit a8959fc" or "Review changes in v20260315.0"
