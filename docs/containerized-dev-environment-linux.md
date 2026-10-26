# Containerized Development Environment — Linux

This guide walks you through setting up the containerized ARO-RP development environment on Linux.

For an overview of the setup and how the workspace is mounted, see [Containerized Development Environment](containerized-dev-environment.md).

## Prerequisites

1. Podman 4.7+ installed on your host system ([https://podman.io/docs/installation](https://podman.io/docs/installation)).
2. You've followed the steps to [prepare your development environment](prepare-your-dev-environment.md).

## Setup Steps

Follow these steps from the **root directory** of the ARO-RP repository.

### 1. Set up your environment variables

Copy the example environment file and edit it with your specific configuration.

```sh
cp env.example env
# Edit the newly created 'env' file with your settings
```

### 2. Get the required secrets and source your environment

Use the project's Makefile to fetch necessary secrets from Azure storage. The secrets will be downloaded and extracted to the `./secrets` directory.

```sh
SECRET_SA_ACCOUNT_NAME=<secrets_storage_account_name> make secrets
```

Replace `<secrets_storage_account_name>` with the actual storage account name for your environment.

Then source your environment file to load the configuration (including secrets):

```sh
. ./env
```

### 3. Enable the Podman socket

This is required when not using Hive for cluster deployment.

```sh
systemctl --user enable --now podman.socket
```

Verify the socket is running:

```sh
systemctl --user status podman.socket
```

### 4. Build the container image

```sh
make dev-env-build
```

### 5. Start the container

```sh
make dev-env-start
```

The container runs an entrypoint script that sources your environment variables and starts the RP in local development mode.

Verify the container is running:

```sh
podman compose ps
```

### 6. View RP Logs (Optional)

Check the logs to see the RP startup output.

```sh
podman compose logs aro-dev-env
```

### 7. Enter the container shell

To interact with the environment inside the container (e.g., run other commands, debug):

```sh
podman compose exec aro-dev-env bash
```

### 8. Run other development commands (inside container shell)

From inside the container, you can run project-specific `make` commands or scripts that expect the Go environment to be set up.

```sh
# Example: Run tests
make test-go

# Example: Build all components
make build-all
```

## Using Local Azure CLI with the Development RP

To use your local Azure CLI (`az`) to interact with the RP running in the container, you need to configure your local environment:

1. **Exit the container shell** if you are in it.
2. **Set the `RP_MODE` environment variable** in your local host terminal:

   ```sh
   export RP_MODE="development"
   # Or set it in your local .env file if your local az setup loads it
   ```

3. Set up local development for the `az aro` upstream extension, as described in [az-aro-python-development.md](az-aro-python-development.md#ext-dev)

Now, when you run `az aro` commands on your local host (from the azure-cli-extensions virtualenv), they should be directed to the RP running in your container (accessible via `localhost:8443`).

## Cleanup

To stop and remove the containerized development environment:

```bash
make dev-env-stop
```

If you also want to remove the built image:

```sh
podman rmi localhost/aro-rp_aro-dev
```
