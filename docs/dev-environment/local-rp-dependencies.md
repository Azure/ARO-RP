# Local RP Dependency Setup

- [Local RP Dependency Setup](#local-rp-dependency-setup)
  - [Install Package Dependencies](#install-package-dependencies)
    - [Fedora Dependencies](#fedora-dependencies)
    - [MacOS Dependencies](#macos-dependencies)
      - [Optional MacOS Dependencies](#optional-macos-dependencies)
  - [Install Go](#install-go)
  - [Install Python (`pyenv`)](#install-python-pyenv)
  - [Install AZ Client](#install-az-client)
  - [Install OpenVPN](#install-openvpn)
  - [Install Podman and Podman Docker](#install-podman-and-podman-docker)
    - [Configure Podman](#configure-podman)
  - [Containerized RP Software Required](#containerized-rp-software-required)

> [!NOTE]
> To run an RP instance as a Go process using `go run` locally, additional tools are required and are outlined below.

## Install Package Dependencies

### Fedora Dependencies

1. General dependencies
    ```sh
    sudo dnf install -y \
        gpgme-devel \
        libassuan-devel \
        openssl \
        nodejs
    ```
2. Dependencies for Fedora 37+
    ```sh
    sudo dnf install -y \
        lvm2 \
        lvm2-devel \
        golang-github-containerd-btrfs-devel
    ```
3. Dependencies for `pyenv`
    ```sh
    sudo dnf install -y \
        bzip2-devel \
        ncurses-devel \
        libffi-devel \
        readline-devel \
        sqlite-devel \
        tk-devel \
        xz-devel \
        zlib-devel \
        gcc \
        make
    ```

### MacOS Dependencies

1. Install the required dependencies
    ```sh
    brew install coreutils \
        findutils \
        gnu-tar \
        grep \
        gettext \
        gpgme diffutils \
        node
    ```

#### Optional MacOS Dependencies
> [!WARNING]
> Pay attention to the notes after the `brew` installer runs as there will be instructions to follow to complete setup on MacOS.
1. Install `docker-compose`
    ```sh
    brew install docker-compose
    ```

> [!NOTE]
> Developers using macOS are encouraged to contribute to this repository. To ensure compatibility, macOS users should install GNU utilities on their systems.
>
> The goal is to minimize shell scripting and other platform-specific variations within the repository. Installing GNU utilities on macOS helps reduce discrepancies in command-line flags, usage and more, ensuring a consistent development experience across environments.

1. Ensure you have installed all [MacOS dependencies](#macos-dependencies)
2. Link `gettext` to make commands available system-wide
    ```sh
    brew link gettext
    ```
3. Update your `PATH` in your shell's RC file to prepend your `PATH` with GBU Utils paths
    ```sh
    export PATH=$(find $(brew --prefix)/opt -type d -follow -name gnubin -print | paste -s -d ':' -):\$PATH
    ```
4. Add the following to your shell's RC file
    ```sh
    export LDFLAGS="-L$(brew --prefix)/lib"
    export CFLAGS="-I$(brew --prefix)/include"
    export CGO_LDFLAGS=$LDFLAGS
    export CGO_CFLAGS=$CFLAGS
    ```
5. Login to ACR
    > [!TIP]
    > The following steps ***may*** be applicable where you symlink `docker` to `podman` location.

    ```sh
    ### CHECK SYMLINK ###
    ls -la $(whereis -q docker)

    # Example Output: /Users/<USER>/.local/bin/docker -> /opt/homebrew/bin/podman

    ### LOGIN TO ACR ###
    az acr login --name <TARGET_ACR>
    ```

## Install Go

If you do not have a compatible Golang version available through your system's package manager:

1. Follow the [gvm installation instructions](https://github.com/moovweb/gvm#installing).
2. Install the required Golang version using `gvm`
    ```sh
    gvm install go1.26.4
    ```

### Install Python (`pyenv`)

1. Follow the [pyenv installation instructions](https://github.com/pyenv/pyenv#installation).
2. Install required Python version using `pyenv`
    ```sh
    pyenv install 3.12
    ```

## Install AZ Client

> [!NOTE]
> Due to the `az` client requiring a specific Python version, you will find the instructions to install the `az` client in the [Getting Started](#getting-started) section. This will use `pyenv` to ensure the correct Python version limited to the local ARO-RP environment.
>
> ARO-RP comes with `make pyenv`, this will set up the environment and install the `az` client after setting the local Python version via `pyenv`.

## Install OpenVPN

1. Find the client you require [here](https://openvpn.net/community-downloads/)
2. **Or:** on RHEL/Fedora run the following
    ```sh
    sudo dnf install openvpn
    ```

> [!NOTE]
> You can also use the built in Network Manager to add `.ovpn` configuration files.

## Install Podman and Podman Docker

> [!NOTE]
> Podman is used for building container images and running the installer.

1. Install Podman
    ```sh
    sudo dnf install -y \
        podman \
        podman-docker
    ```

### Configure Podman

> [!IMPORTANT]
> Podman needs to be running in daemon mode when running the RP locally.

1. On Linux, you can enable socket activation to start Podman in daemon mode
    ```sh
    systemctl --user enable podman.socket
    ```

> [!WARNING]
> If you are using `podman-machine`, you will need to export the socket:
>
> ```sh
> export ARO_PODMAN_SOCKET=unix://$HOME/.local/share/containers/podman/machine/qemu/podman.sock
> ```
>
> You will also need to ensure that `podman-machine` has enough resources:
>
> ```sh
> podman machine stop
> podman machine rm
> podman machine init --cpus 4 --memory 5000
> podman machine start
> ```

2. Disable Docker compatibility mode for `az acr login` support
    ```sh
    sudo touch /etc/containers/nodocker
    ```

### Configure Podman on macOS (Apple Silicon)

> [!IMPORTANT]
> On Apple Silicon Macs (M1/M2/M3/M4), the `aro-installer` container image is only built for `amd64`. You must configure Podman with Rosetta emulation to run amd64 containers.

1. Install Podman

    ```sh
    brew install podman
    ```

2. Initialize the Podman machine with Rosetta support and sufficient resources

    ```sh
    podman machine init --cpus 4 --memory 5000 --rootful --rosetta
    podman machine start
    ```

    > [!NOTE]
    > If you already have a Podman machine without Rosetta, recreate it:
    >
    > ```sh
    > podman machine stop
    > podman machine rm
    > podman machine init --cpus 4 --memory 5000 --rootful --rosetta
    > podman machine start
    > ```

3. Set the Podman socket environment variable

    The socket path varies by system. Find yours with:

    ```sh
    podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}'
    ```

    Then add to your shell RC file or `env` file:

    ```sh
    export ARO_PODMAN_SOCKET="unix://$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')"
    ```

4. Verify the setup

    ```sh
    # Confirm Rosetta is enabled
    podman machine inspect | grep -i rosetta

    # Test pulling an amd64 image
    podman pull --platform linux/amd64 alpine
    podman run --rm alpine uname -m  # Should output: x86_64
    ```

---

## Containerized RP Software Required

> [!TIP]
> For a minimal development environment, the recommended approach is to use the containerized setup. This runs the RP inside a container with your local workspace mounted, facilitating debugging and quick code changes.
>
> **See [Containerized Development Environment](../containerized-dev-environment.md) for a complete setup guide.**

The containerized development environment requires only these locally installed tools:

1. az
2. make
3. podman
4. openvpn (Optional for Hive cluster deployments)

> [!NOTE]
> Instructions for installing these tools are provided in the sections below. Refer to the [Podman](#install-podman-and-podman-docker) section for setup details specific to your operating system.
