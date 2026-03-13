# Containerized Development Environment — macOS (including Apple Silicon / ARM64)

This guide walks you through setting up the containerized ARO-RP development environment on macOS.

For an overview of the setup and how the workspace is mounted, see [Containerized Development Environment](containerized-dev-environment.md).

## Prerequisites

1. Docker Desktop (or an equivalent container runtime) installed.
2. Azure CLI installed on your host system (`brew install azure-cli`).
3. You've followed the steps to [prepare your development environment](prepare-your-dev-environment.md).

**Note:** On macOS the dev environment runs under Docker. Podman runs _inside_ the container automatically — no host Podman installation is needed. The `Makefile` detects macOS and uses the correct compose tool and overrides.

## Setup Steps

Follow these steps from the **root directory** of the ARO-RP repository.

### 1. Set up your environment variables

Copy the example environment file and edit it with your specific configuration.

```bash
cp env.example env
# Edit the newly created 'env' file with your settings
```

### 2. Get the required secrets and source your environment

Use the project's Makefile to fetch necessary secrets from Azure storage. The secrets will be downloaded and extracted to the `./secrets` directory.

```bash
SECRET_SA_ACCOUNT_NAME=<secrets_storage_account_name> make secrets
```

Replace `<secrets_storage_account_name>` with the actual storage account name for your environment.

Then source your environment file to load the configuration (including secrets):

```bash
source ./env
```

### 3. Configure platform (ARM64 only)

If you are on Apple Silicon, add the following to your `env` file:

```bash
export PLATFORM=linux/arm64
```

If you don't have ACR access, also add:

```bash
export FEDORA_REGISTRY=registry.fedoraproject.org
```

### 4. Build the container image

```bash
make dev-env-build
```

### 5. Start the container

```bash
make dev-env-start
```

The container runs an entrypoint script that sources your environment variables, starts Podman inside the container, and starts the RP in local development mode.

Verify the container is running:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml ps
```

### 6. View RP Logs (Optional)

Check the logs to see the RP startup output.

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml logs aro-dev-env
```

### 7. Enter the container shell

To interact with the environment inside the container (e.g., run other commands, debug):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml exec aro-dev-env bash
```

### 8. Run other development commands (inside container shell)

From inside the container, you can run project-specific `make` commands or scripts that expect the Go environment to be set up.

```bash
# Example: Run tests
make test-go

# Example: Build all components
make build-all
```

## Running E2E Tests

On Linux, the dev-env container uses `network_mode: host`, which gives the container direct access to the host network. On macOS with Docker Desktop, `network_mode: host` actually means the container shares the network of Docker Desktop's hidden Linux VM — not your Mac's network directly. In practice, Docker Desktop forwards ports from the VM to `localhost` on your Mac, so the RP at port `8443` is still reachable from your Mac's terminal.

### Running tests from inside the container

The simplest approach is to run e2e tests from inside the container shell, where `localhost:8443` always reaches the RP directly:

```bash
# Enter the container
docker compose -f docker-compose.yml -f docker-compose.dev-env-macos.yml exec aro-dev-env bash

# Inside the container
source /workspace/env
make test-e2e
```

### Reaching the RP from your Mac host

Docker Desktop forwards ports from the Linux VM to your Mac, so you can verify the RP is running from your Mac terminal:

```bash
curl -k https://localhost:8443/healthz/ready
```

If this does not work, ensure the container is running and check that no other process on your Mac is already using port `8443`.

### VPN access for private clusters

E2e tests against private clusters require a VPN connection (via `openvpn`). On macOS, `openvpn` cannot run directly inside the Docker Desktop VM because `/dev/net/tun` is not available. Options include:

- **Run OpenVPN on the Mac host** using a macOS-compatible OpenVPN client (e.g., Tunnelblick, or `brew install openvpn`) with the `.ovpn` config from `secrets/`. The VPN tunnel on the host is accessible from inside the container through Docker Desktop's networking.
- **Run tests entirely inside the container** after establishing the VPN from the host, since the container can reach the VPN-routed networks through Docker Desktop's VM networking.

> **Note:** This VPN-through-Docker-Desktop setup has not been extensively tested. If you encounter connectivity issues between the container and VPN-routed networks, try running `openvpn` directly on the Mac host and the e2e tests also on the host (outside the container), using a locally built `aro` binary.

## Testing Geneva Actions from a Windows VM

When testing Geneva Actions, you need a Windows VM running the .NET extension client to reach the RP running in Docker on your Mac. This is done by creating an SSH tunnel from the Windows VM to the macOS host.

### 1. Enable SSH on macOS

Check if Remote Login (SSH) is enabled:

```bash
sudo systemsetup -getremotelogin
```

If it shows `Remote Login: Off`, enable it:

```bash
sudo systemsetup -setremotelogin on
```

> **Note:** If you get an error about "Full Disk Access privileges", go to **System Settings > Privacy & Security > Full Disk Access** and grant access to Terminal (or the application you are using).

### 2. Get your Mac's IP address

```bash
ipconfig getifaddr en0
```

Use the returned IP address (e.g., `192.168.x.x`) in the SSH command below. Alternatively, go to **Apple menu > System Settings > Network > Wi-Fi or Ethernet > Details > IP Address**.

### 3. Open the SSH tunnel from the Windows VM

On the Windows VM (using Git Bash, PowerShell, or Command Prompt), create an SSH tunnel that forwards the VM's `localhost:8443` to the Mac's `localhost:8443`:

```bash
ssh YOUR_MACOS_USERNAME@YOUR_MAC_IP -L 8443:localhost:8443
```

When prompted, confirm the connection and enter your macOS account password.

Once the tunnel is established, the RP running in Docker on your Mac is reachable at `https://localhost:8443` from inside the Windows VM.

### 4. Run the Geneva Actions extension

With the SSH tunnel active, follow the Geneva Actions testing instructions to run the extension client on the Windows VM. The extension will connect to `https://localhost:8443`, which is tunneled through to the RP in the container on your Mac.

## Using Local Azure CLI with the Development RP

To use your local Azure CLI (`az`) to interact with the RP running in the container, you need to configure your local environment:

1. **Exit the container shell** if you are in it.
2. **Set the `RP_MODE` environment variable** in your local host terminal:

   ```bash
   export RP_MODE="development"
   # Or set it in your local .env file if your local az setup loads it
   ```

3. **Install the development `az aro` extension** from your local source code:

   ```bash
   make az
   ```

   This should build the extension and configure your local `az` to use it when `RP_MODE` is set to `development`.

Now, when you run `az aro` commands on your local host (from the project root), they should be directed to the RP running in your container (accessible via `localhost:8443`).

## Cleanup

To stop and remove the containerized development environment:

```bash
make dev-env-stop
```

If you also want to remove the built image:

```bash
docker rmi aro-rp_aro-dev
```
