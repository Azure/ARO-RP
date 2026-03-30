# Claude Code Agents for ARO-RP

This directory contains specialized Claude Code agents for the Azure Red Hat OpenShift Resource Provider codebase.

## Available Agents

### pr-scan
**Purpose**: Automated pull request review for correctness, security, reliability, and quality.

**Quick Start**:
```bash
# Via Makefile (recommended)
make pr-scan PR=1234
make pr-scan BRANCH=fix/my-feature MODE=quick

# Via script
./hack/pr-scan.sh --pr 1234 --mode security
./hack/pr-scan.sh --branch fix/feature --mode pipeline

# Via Claude Code chat
"Scan PR 1234"
"Review branch fix/feature, quick mode"
```

**Modes**:
- `full` (default): Complete 7-category review, all severity levels
- `quick`: Blocker/High findings only, abbreviated summary
- `security`: Deep security focus (authz, secrets, injection, crypto)
- `pipeline`: CI/CD changes only (YAML, templates, scripts)

**Documentation**: See `docs/agent-guides/pr-scan-agent.md`

**What it checks**:
- Correctness & logic (nil pointers, races, leaks)
- API compatibility & breaking changes
- Security (authz, secrets, injection)
- Reliability (retries, timeouts, error handling)
- Test coverage & edge cases
- Observability (logging, metrics)
- Performance (allocations, queries)

**Output**: Structured findings with severity levels (Blocker/High/Medium/Low)

**Tools**:
- `hack/pr-scan.sh`: Context gathering script (requires git, optional gh CLI)
- `make pr-scan`: Makefile target that wraps the script

## Adding New Agents

Create new agent definitions in this directory following the format:

```markdown
---
name: agent-name
description: Brief description
model: sonnet
---

[Agent instructions and behavior]
```

Update this README with the new agent's purpose and usage examples.
