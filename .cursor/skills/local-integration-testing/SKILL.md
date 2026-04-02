---
name: local-integration-testing
description: >-
  Run local integration tests against the containerized ARO-RP dev environment.
  Use when testing admin API endpoints locally, starting/stopping local RP, running
  E2E against local RP, or verifying changes in a live dev container.
---

# Local Integration Testing (Linux-first)

This skill is Linux/Fedora-first because most of the team runs Linux. macOS arm64
guidance is included in a dedicated section.

## Canonical docs and drift check

Use `docs/` as the source of truth for environment setup and test workflows:

- `docs/containerized-dev-environment-linux.md`
- `docs/containerized-dev-environment-macos.md`
- `docs/testing.md`
- `docs/prepare-your-dev-environment.md`

If this skill and `docs/` diverge, treat `docs/` as authoritative and update this
skill in the same PR to prevent configuration drift.

## Prerequisites

1. **Container runtime** available on host (`docker` or `podman`)
2. **`env` file** at repo root (copy from `env.example`, fill in secrets)
3. **`secrets/` directory** with valid credentials (`secrets/env`, certs, kubeconfig)
4. **Azure CLI** authenticated (`az login`)

Verify runtime is available:

```bash
docker info --format '{{.OSType}}/{{.Architecture}}'
```

## Worktree setup (required)

Git worktrees do NOT contain gitignored files (`env`, `secrets/`). The container
bind-mounts the worktree as `/workspace`, so these files must be present there.
Always check and create links before starting the container from a worktree.

```bash
# Detect the host checkout root (parent repo of the worktree)
HOST_CHECKOUT="$(git worktree list | head -1 | awk '{print $1}')"

# Link env file if missing
[ ! -e env ] && ln -s "$HOST_CHECKOUT/env" env

# Link secrets/ directory if missing
[ ! -e secrets ] && ln -s "$HOST_CHECKOUT/secrets" secrets
```

Verify both exist and are readable before proceeding:

```bash
ls -la env secrets/env
```

> **Agent rules**:
> - Before running `make dev-env-start` in any worktree, check for the presence
>   of `env` and `secrets/`. If either is missing, create links to the host
>   checkout as shown above.
> - **NEVER overwrite, delete, modify, or replace existing `env` or `secrets/`
>   files/directories** — whether they are real files or symlinks. These contain
>   credentials and secrets that cannot be recovered. Only create links when
>   the targets do not already exist (the `[ ! -e ... ]` guard is mandatory).
> - Do NOT copy credential files — use links so they stay in sync with the
>   host checkout.

## Quick start (all platforms)

### 1. Build the dev container image (first time or after Dockerfile changes)

```bash
make dev-env-build
```

### 2. Start the local RP

```bash
make dev-env-start
```

This launches the `aro-dev-env` container in detached mode. Inside the container,
`hack/devtools/dev-env-entrypoint.sh` sources the `env` file and runs
`make runlocal-rp`, which starts the RP on port **8443** (HTTPS).

### 3. Wait for the RP to become healthy

```bash
until curl -ksSf https://localhost:8443/healthz/ready 2>/dev/null; do sleep 5; done
echo "RP is ready"
```

### 4. Export env variables for direct `go test -tags e2e`

When running E2E directly (not via `make test-e2e`), export variables from both
`env` and `secrets/env` in the current shell:

```bash
set -a
source env
source secrets/env
set +a
```

### 5. Run a focused E2E test

```bash
export RP_BASE_URL=https://localhost:8443
go test -v -tags=e2e ./test/e2e/... \
  -run "TestE2E" \
  --ginkgo.label-filter="Admin API" \
  --ginkgo.focus="Resize control plane" \
  --ginkgo.timeout=30m
```

### 6. Stop the local RP

```bash
make dev-env-stop
```

## Existing-cluster mode (fast feedback)

To avoid create/delete cycles during repeated admin API testing, run E2E against
an existing cluster as documented in `docs/testing.md`:

```bash
set -a
source env
source secrets/env
set +a

export CLUSTER=<existing-cluster-name>
export RESOURCEGROUP=<existing-cluster-resource-group>
unset CI
export E2E_DELETE_CLUSTER=false
```

Then run your focused test:

```bash
go test -v -tags=e2e ./test/e2e/... \
  -run "TestE2E" \
  --ginkgo.focus="Resize control plane"
```

## Linux/Fedora notes (default)

- Use the default compose setup from `make dev-env-build` / `make dev-env-start`.
- Keep SELinux/container runtime settings consistent with team docs.
- If using Podman, verify compose compatibility and socket availability first.

## macOS arm64 notes (separate)

Use this only when running on macOS arm64 (Docker Desktop or equivalent alternative).

- Platform should be `linux/arm64`.
- Compose override: `docker-compose.dev-env-macos.yml`.
- Podman socket is not mounted from host; Podman runs in-container.

Recommended env values:

```bash
export PLATFORM=linux/arm64
export FEDORA_REGISTRY=registry.fedoraproject.org
```

macOS build command (equivalent to `make dev-env-build`):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml build aro-dev-env
```

## Container behavior

1. Bind-mounts the workspace at `/workspace` and `~/.azure` + `~/.ssh` read-only
2. Runs as a non-root user matching the host UID
3. On macOS, starts an in-container Podman service when host socket is unavailable
4. Sources `/workspace/env` and runs `make runlocal-rp`
5. Exposes HTTPS on port 8443 (host network mode)

## Troubleshooting

### Container won't start

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml logs aro-dev-env
```

### RP fails health check

Check that `/workspace/env` and `/workspace/secrets/env` are readable:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml exec aro-dev-env cat /workspace/env
```

### Architecture mismatch

If you see `exec format error`, rebuild with explicit platform:

```bash
PLATFORM=linux/arm64 make dev-env-build
```

### Missing `env` or `secrets/` in worktree

The container sources `/workspace/env` on startup. If running from a worktree
and this file is missing, the container will fail immediately. See the
**Worktree setup** section above to link from the host checkout.

Signs of this problem:
- Container exits immediately after start
- Logs show `No such file or directory: /workspace/env`
- RP fails to initialize with missing credential errors

If E2E still reports unset variables, verify that required keys are exported
(without printing values):

```bash
env | cut -d= -f1 | rg '^(LOCATION|CLUSTER|AZURE_TENANT_ID|AZURE_SUBSCRIPTION_ID|AZURE_CLIENT_ID|AZURE_CLIENT_SECRET)$'
```

### Port conflict

If port 8443 is already in use (e.g., another RP instance):

```bash
lsof -i :8443
make dev-env-stop
```

## Integration Test Workflow for Admin API Changes

When developing a new admin API endpoint:

```
Task Progress:
- [ ] Step 1: Implement handler, register route, write unit tests
- [ ] Step 2: Run make fmt && make lint-go && make unit-test-go
- [ ] Step 3: If in a worktree, symlink env and secrets/ from host checkout
- [ ] Step 4: Build dev container: make dev-env-build
- [ ] Step 5: Start local RP: make dev-env-start
- [ ] Step 6: Wait for health: curl -ksSf https://localhost:8443/healthz/ready
- [ ] Step 7: Test endpoint via curl or E2E test
- [ ] Step 8: Check RP logs: docker compose ... logs -f aro-dev-env
- [ ] Step 9: Stop RP: make dev-env-stop
```
