# Local RP Dependency Setup

- [Local RP Dependency Setup](#local-rp-dependency-setup)
  - [Install Package Dependencies](#install-package-dependencies)
    - [Fedora/RHEL Dependencies](#fedorarhel-dependencies)
      - [Fedora/RHEL Optional Dependencies](#fedorarhel-optional-dependencies)
    - [Debian Dependencies](#debian-dependencies)
      - [Debian Optional Dependencies](#debian-optional-dependencies)
    - [MacOS Dependencies](#macos-dependencies)
      - [Optional MacOS Dependencies](#optional-macos-dependencies)
  - [Install Go](#install-go)
    - [Install Go Manually](#install-go-manually)
    - [Install Python (`pyenv`)](#install-python-pyenv)
  - [Install AZ Client](#install-az-client)
  - [Install OpenVPN](#install-openvpn)
  - [Install Podman and Podman Docker](#install-podman-and-podman-docker)
    - [Configure Podman](#configure-podman)
  - [Install GolangCI Lint](#install-golangci-lint)
  - [Install YAMLLint](#install-yamllint)
- [Miscellaneous OS Requirements](#miscellaneous-os-requirements)
  - [RHEL](#rhel)
  - [Debian](#debian)
  - [MacOS](#macos)
  - [Containerized RP Software Required](#containerized-rp-software-required)

> [!NOTE]
> To run an RP instance as a Go process using `go run` locally, additional tools are required and are outlined below.

## Install Package Dependencies

### Fedora/RHEL Dependencies

> [!IMPORTANT]
> For other OS specific requirements, refer to the [Miscellaneous OS Requirements](#other-os-requirements) section.

1. General dependencies
    ```sh
    sudo dnf install -y \
        gpgme-devel \
        libassuan-devel \
        openssl
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

#### Fedora/RHEL Optional Dependencies
1. Install [Docker Compose](https://docs.docker.com/compose/install/linux/#install-using-the-repository)
    1. Fedora/RHEL
        ```sh
        sudo dnf install -y \
            docker-compose-plugin
        ```
    2. See [Install Go via `gvm`](#prepare-dev-environment/gvm.md)
    [gvm](#prepare-dev-environment/gvm.md)

### Debian Dependencies

1. Install the required dependencies
    ```sh
    sudo apt install -y \
        libgpgme-dev \
        libbtrfs-dev \
        libdevmapper-dev
    ```

#### Debian Optional Dependencies
   1. Install `docker-compose-plugin`
        ```sh
        sudo apt install -y
            docker-compose-plugin
        ```

### MacOS Dependencies

1. Install the required dependencies
    ```sh
    brew install coreutils \
        findutils \
        gnu-tar \
        grep \
        gettext \
        gpgme diffutils
    ```

#### Optional MacOS Dependencies
> [!WARNING]
> Pay attention to the notes after the `brew` installer runs as there will be instructions to follow to complete setup on MacOS.
1. Install `docker-compose`
    ```sh
    brew install docker-compose
    ```

## Install Go

### Install Go Manually

> [!TIP]
> Go versions installation and management can be simplified with `gvm`.
> See [Install Go With `gvm`](dev-environment/gvm.md)
1. [Download Go](https://golang.org/dl) matching the version in `go.mod`.
2. Extract the archive

   ```sh
   cd $HOME/Downloads
   sudo tar -C /usr/local -xzf go1.22.12.linux-amd64.tar.gz
   ```
3. Add Go to `PATH` in your shell's RC file

   ```sh
   export PATH="${PATH}:/usr/local/go/bin"
   ```
4. Configure `GOPATH` as an environment variable in your shell, as it is required by some dependencies for `make generate`. To use the default path, add the following to your shell's RC file

    ```sh
    export GOPATH=$(go env GOPATH)
    ```

### Install Python (`pyenv`)

> [!IMPORTANT]
> Python versions earlier than 3.6 or later than 3.10 are currently **not** supported.

1. Install `pyenv`
    ```sh
    curl https://pyenv.run | bash
    ```
2. Append the following to your shell's RC file
    ```sh
    export PATH="$HOME/.pyenv/bin:$PATH"
    eval "$(pyenv init --path)"
    eval "$(pyenv init -)"
    ```
3. Install required Python version using `pyenv`
    ```sh
    pyenv install 3.10.0
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

## Install GolangCI Lint

1. Find latest version [here](https://github.com/golangci/golangci-lint/releases)
2. Run the install
    ```sh
    # https://github.com/golangci/golangci-lint/releases
    GOLINT_VERSION="<REPLACE WITH LATEST>"

    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin "$GOLINT_VERSION"
    ```

## Install YAMLLint

```sh
sudo dnf install -y \
    yamllint
```

---

# Miscellaneous OS Requirements

## RHEL

1. Register the system with `subscription-manager register`
2. Enable the [CodeReady Linux Builder](https://access.redhat.com/articles/4348511) repository to install *-devel packages
3. Enable the [EPEL repository](https://docs.fedoraproject.org/en-US/epel/#_quickstart) for packages not in the base repositories (such as OpenVPN)

## Debian

> [!IMPORTANT]
> Your actual `pkgconfig` path may differ; please adjust it accordingly.
1. Ensure you have installed all [Debian dependencies](#debian-dependencies)
2. Make sure that `PKG_CONFIG_PATH` contains the `pkgconfig` files of the above packages. For example:
    ```sh
    export PKG_CONFIG_PATH:/usr/lib/x86_64-linux-gnu/pkgconfig
    ```

## MacOS

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

---

## Containerized RP Software Required

> [!TIP]
> For a minimal development environment, the recommended approach is to use the containerized setup. This runs the RP inside a container with your local workspace mounted, facilitating debugging and quick code changes.
>
> **See [Containerized Development Environment](containerized-dev-environment.md) for a complete setup guide.**

The containerized development environment requires only these locally installed tools:

1. az
2. make
3. podman
4. openvpn (Optional for Hive cluster deployments)

> [!NOTE]
> Instructions for installing these tools are provided in the sections below. Refer to the [Podman](#install-podman-and-podman-docker) section for setup details specific to your operating system.
