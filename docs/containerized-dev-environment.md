# Containerized Development Environment

This document describes how to set up and use a containerized development environment for ARO-RP using podman compose.

## Files for this setup

The following files, located at the project root, are used for this setup:

- `Dockerfile.dev-env`: Defines the container image with necessary dependencies and tools.
- `docker-compose.yml`: Contains the definition for the `aro-dev-env` service.

## How the Local Workspace is Mounted

The containerized development environment mounts your local workspace and configuration into the container for seamless development:

- **Local repository**: Your entire ARO-RP repository (`.`) is mounted to `/workspace` inside the container
- **Azure CLI config**: Your local `~/.azure` directory is mounted read-only to `/root/.azure` for authentication
- **SSH keys**: Your local `~/.ssh` directory is mounted read-only to `/root/.ssh` for Git operations
- **Secrets**: Your local `./secrets` directory is mounted read-only to `/workspace/secrets` for RP configuration

This means any changes you make to files in your local repository are immediately available inside the container, and vice versa. The container's working directory is set to `/workspace`, so you're working directly with your local code.

## Prerequisites

1.  Podman 4.7+ installed on your host system ([https://podman.io/docs/installation](https://podman.io/docs/installation)).
2.  Azure CLI installed on your host system.
3.  You've followed the steps to [prepare your development environment](prepare-your-dev-environment.md).

## Setup Steps

Follow these steps from the **root directory** of the ARO-RP repository:

1.  **Set up your environment variables:**
    Copy the example environment file and edit it with your specific configuration.

    ```bash
    cp env.example env
    # Edit the newly created 'env' file with your settings
    ```

2.  **Get the required secrets:**
    Use the project's Makefile to fetch necessary secrets, which are typically saved into the `./secrets` directory.

    ```bash
    SECRET_SA_ACCOUNT_NAME=<secrets_storage_account_name> make secrets
    # Replace <secrets_storage_account_name> with the actual storage account name
    # Ensure the secrets are placed in the ./secrets directory
    ```

3.  **Build the container image:**
    Build the `aro-dev-env` container image using the Dockerfile.

    ```bash
    podman compose build aro-dev-env
    ```

4.  **Start the container:**
    Start the `aro-dev-env` service. The container will automatically source your environment file and start the RP on port 8443.

    ```bash
    podman compose up -d aro-dev-env
    ```
    
    **Note:** The container runs the command `. /workspace/env && make runlocal-rp`, which sources your environment variables and starts the RP in local development mode.
    
    Verify the container is running:
    ```bash
    podman compose ps
    ```

5.  **View RP Logs (Optional):**
    Check the logs to see the RP startup output.

    ```bash
    podman compose logs aro-dev-env
    ```

6.  **Enter the container shell:**
    To interact with the environment inside the container (e.g., run other commands, debug).

    ```bash
    podman compose exec aro-dev-env bash
    ```

7.  **Run other development commands (Inside container shell):**
    From inside the container, you can run project-specific `make` commands or scripts that expect the Go environment to be set up.

    ```bash
    # Example: Run tests
    make test-go
    
    # Example: Build all components
    make build-all
    ```

## Using Local Azure CLI with the Development RP

To use your local Azure CLI (`az`) to interact with the RP running in the container, you need to configure your local environment:

1.  **Exit the container shell** if you are in it.
2.  **Set the `RP_MODE` environment variable** in your local host terminal:

    ```bash
    export RP_MODE="development"
    # Or set it in your local .env file if your local az setup loads it
    ```

3.  **Install the development `az aro` extension** from your local source code:

    ```bash
    make az
    ```
    This should build the extension and configure your local `az` to use it when `RP_MODE` is set to `development`.

Now, when you run `az aro` commands on your local host (from the project root), they should be directed to the RP running in your container (accessible via `localhost:8443`).

## Cleanup

To stop and remove the containerized development environment:

```bash
podman compose down aro-dev-env
```

If you also want to remove the built image:

```bash
podman rmi aro-rp_aro-dev
```