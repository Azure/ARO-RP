# Containerized Development Environment

This document describes how to set up and use a containerized development environment for ARO-RP.

Choose the guide for your platform:

- [Linux](containerized-dev-environment-linux.md)
- [macOS (including Apple Silicon / ARM64)](containerized-dev-environment-macos.md)

## Files for this setup

The following files, located at the project root, are used for this setup:

- `Dockerfile.dev-env`: Defines the container image with necessary dependencies and tools.
- `docker-compose.yml`: Contains the definition for the `aro-dev-env` service.
- `docker-compose.dev-env-linux.yml`: Linux-specific override (adds host Podman socket mount).
- `docker-compose.dev-env-macos.yml`: macOS-specific override (privileged mode, no SELinux labels).
- `hack/devtools/dev-env-entrypoint.sh`: Entrypoint script that auto-detects the Podman runtime.

## How the Local Workspace is Mounted

The containerized development environment mounts your local workspace and configuration into the container for seamless development:

- **Local repository**: Your entire ARO-RP repository (`.`) is mounted to `/workspace` inside the container
- **Azure CLI config**: Your local `~/.azure` directory is mounted read-only to `/home/aro-dev/.azure` for authentication
- **SSH keys**: Your local `~/.ssh` directory is mounted read-only to `/home/aro-dev/.ssh` for Git operations
- **Secrets**: Your local `./secrets` directory is mounted read-only to `/workspace/secrets` for RP configuration

This means any changes you make to files in your local repository are immediately available inside the container, and vice versa. The container's working directory is set to `/workspace`, so you're working directly with your local code.
