# Containerized Development Environment

This document describes how to set up and use a containerized development environment for ARO-RP using Docker Compose.

## Files for this setup

The following files, located at the project root, are used for this setup:

- `Dockerfile.dev-env`: Defines the container image with necessary dependencies and tools.
- `dev-container-entrypoint.sh`: Script executed when the container starts to set up the environment (Go version, source env).
- `docker-compose.yml`: Contains the definition for the `aro-dev-env` service.

## Prerequisites

1.  Docker installed on your host system ([https://docs.docker.com/get-docker/](https://docs.docker.com/get-docker/)).
2.  Docker Compose plugin installed.
3.  Azure CLI installed on your host system.

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
    docker compose build aro-dev-env
    ```

4.  **Start the container:**
    Start the `aro-dev-env` service. The container's main command will be to run the RP.

    ```bash
    docker compose up -d aro-dev-env
    ```
    Verify the container is running:
    ```bash
    docker compose ps
    ```

5.  **View RP Logs (Optional):**
    Check the logs to see the RP startup output.

    ```bash
    docker compose logs aro-dev-env
    ```

6.  **Enter the container shell:**
    To interact with the environment inside the container (e.g., run other commands, debug).

    ```bash
    docker compose exec aro-dev-env bash
    ```
    *Note: The entrypoint script has already ensured the correct Go version is installed and sourced environment variables from `/workspace/env`.*

7.  **Run other development commands (Inside container shell):**
    From inside the container, you can run project-specific `make` commands or scripts that expect the Go environment to be set up.

    ```bash
    # Example: Run tests
    make test
    
    # Example: Build a specific component
    make build-component
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
docker compose down aro-dev-env
```

If you also want to remove the built image:

```bash
docker rmi aro-rp_aro-dev:latest
```