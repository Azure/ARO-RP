#!/bin/bash -e

# pr-scan.sh: Prepare PR metadata and diff for Claude Code PR scan agent
# This script gathers context (changed files, commits, diff) for the pr-scan agent.
# The actual analysis is performed by Claude Code using .claude/agents/pr-scan.md

set -o pipefail

usage() {
    cat >&2 <<EOF
usage: $0 [options]

Gather PR or branch diff for Claude Code pr-scan agent analysis.

Options:
  --pr NUMBER          Fetch and analyze GitHub PR number (e.g., --pr 5678)
  --branch NAME        Analyze branch (e.g., --branch fix/my-feature)
  --base BRANCH        Base branch for diff (default: master)
  --list-open          List open PRs and exit (requires gh CLI)
  --mode MODE          Agent mode: full (default), quick, security, pipeline
  -h, --help           Show this help

Examples:
  $0 --pr 5678                    # Scan PR 5678 against master
  $0 --branch fix/feature         # Scan branch against master
  $0 --pr 5678 --base master      # Explicit base branch
  $0 --list-open                  # List open PRs (requires gh)
  $0 --branch fix/feature --mode quick    # Quick scan (Blocker/High only)

Modes:
  full      - Complete 7-category review (default)
  quick     - Blocker and High severity findings only
  security  - Security-focused review only
  pipeline  - CI/CD pipeline changes only (YAML, templates)

Notes:
  - Requires git access to repository
  - --pr option requires gh CLI (GitHub CLI)
  - Output is printed to stdout for Claude Code to analyze
  - The pr-scan agent definition is in .claude/agents/pr-scan.md
EOF
    exit 1
}

log() {
    echo "[pr-scan] $*" >&2
}

abort() {
    echo "[pr-scan] ERROR: $*" >&2
    exit 1
}

check_gh_cli() {
    if ! command -v gh &>/dev/null; then
        abort "gh CLI not found. Install from https://cli.github.com/ or use --branch instead of --pr"
    fi
}

# Parse arguments
PR=""
BRANCH=""
BASE="master"
LIST_OPEN=false
MODE="full"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --pr)
            PR="$2"
            shift 2
            ;;
        --branch)
            BRANCH="$2"
            shift 2
            ;;
        --base)
            BASE="$2"
            shift 2
            ;;
        --list-open)
            LIST_OPEN=true
            shift
            ;;
        --mode)
            MODE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage
            ;;
    esac
done

# Validate mode
case "$MODE" in
    full|quick|security|pipeline)
        ;;
    *)
        abort "Invalid mode: $MODE (must be: full, quick, security, pipeline)"
        ;;
esac

# Handle --list-open
if [ "$LIST_OPEN" = true ]; then
    check_gh_cli
    log "Fetching open PRs..."
    gh pr list --state open --limit 50
    exit 0
fi

# Validate inputs
if [ -z "$PR" ] && [ -z "$BRANCH" ]; then
    abort "Must specify --pr or --branch"
fi

if [ -n "$PR" ] && [ -n "$BRANCH" ]; then
    abort "Cannot specify both --pr and --branch"
fi

# Ensure we're in a git repository
if ! git rev-parse --git-dir &>/dev/null; then
    abort "Not in a git repository"
fi

# Fetch and determine target branch
if [ -n "$PR" ]; then
    check_gh_cli

    log "Fetching PR #${PR}..."

    # Fetch PR into local ref
    if ! git fetch origin "pull/${PR}/head:pr-${PR}" 2>&1 | grep -v "^From"; then
        abort "Failed to fetch PR #${PR}. Does it exist?"
    fi

    TARGET_BRANCH="pr-${PR}"
    log "Fetched PR #${PR} to local branch ${TARGET_BRANCH}"
else
    TARGET_BRANCH="$BRANCH"

    # Check if branch exists locally or remotely
    if ! git rev-parse --verify "$TARGET_BRANCH" &>/dev/null && \
       ! git rev-parse --verify "origin/$TARGET_BRANCH" &>/dev/null; then
        abort "Branch '$TARGET_BRANCH' not found locally or on origin. Try: git fetch origin"
    fi

    log "Analyzing branch: ${TARGET_BRANCH}"
fi

# Determine base branch
if ! git rev-parse --verify "origin/$BASE" &>/dev/null; then
    abort "Base branch 'origin/$BASE' not found. Try: git fetch origin"
fi

log "Base branch: origin/${BASE}"
log "Mode: ${MODE}"

# Gather context
echo ""
echo "========================================="
echo "PR/Branch Scan Context"
echo "========================================="
echo ""

if [ -n "$PR" ]; then
    echo "PR Number: #${PR}"

    # Get PR details if gh is available
    if command -v gh &>/dev/null; then
        echo ""
        echo "--- PR Details ---"
        gh pr view "$PR" --json title,author,createdAt,url,body \
            --template '{{.title}}
Author: {{.author.login}}
Created: {{.createdAt}}
URL: {{.url}}

{{.body}}
' || log "Warning: Could not fetch PR details via gh"
    fi
fi

echo ""
echo "--- Changed Files ---"
git diff --name-status "origin/${BASE}...${TARGET_BRANCH}" || abort "Failed to diff against origin/${BASE}"

echo ""
echo "--- Commit Log ---"
git log "origin/${BASE}..${TARGET_BRANCH}" --oneline --no-decorate || abort "Failed to get commit log"

echo ""
echo "--- Full Diff ---"
git diff "origin/${BASE}...${TARGET_BRANCH}" || abort "Failed to generate diff"

echo ""
echo "========================================="
echo "End of Context"
echo "========================================="
echo ""

# Print mode-specific guidance
case "$MODE" in
    quick)
        echo "[Mode: quick] Focus on Blocker and High severity findings only."
        echo "Skip Low/Nit findings. Provide abbreviated summary."
        ;;
    security)
        echo "[Mode: security] Security-focused review:"
        echo "- Authorization checks (authz, RBAC)"
        echo "- Input validation and injection risks"
        echo "- Secret handling and credential management"
        echo "- Cryptographic operations"
        echo "Skip non-security findings unless they create security risks."
        ;;
    pipeline)
        echo "[Mode: pipeline] CI/CD pipeline review:"
        echo "- YAML syntax and Azure Pipelines conventions"
        echo "- Template parameters and variable usage"
        echo "- Script error handling (set -e, validation)"
        echo "- Secret exposure in logs"
        echo "Skip non-pipeline code unless it affects CI."
        ;;
esac

echo ""
log "Context gathered. Paste this output into Claude Code for analysis."
log "Or run: make pr-scan PR=${PR:-\$PR_NUMBER} MODE=${MODE}"
