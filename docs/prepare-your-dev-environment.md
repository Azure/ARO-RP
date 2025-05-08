# Prepare Your Development Environment

> [!NOTE]
> This document outlines the development dependencies required to build the RP code.

## Table Of Contents
- [Prepare Your Development Environment](#prepare-your-development-environment)
  - [Table Of Contents](#table-of-contents)
  - [Containerized RP Software Required](#containerized-rp-software-required)
  - [Local RP Dependencies](#local-rp-dependencies)
    - [Install Package Dependencies Fedora/RHEL](#install-package-dependencies-fedorarhel)
    - [Install Go 1.22](#install-go-122)
    - [Install Python (`pyenv`)](#install-python-pyenv)
    - [Install AZ Client](#install-az-client)
    - [Install OpenVPN](#install-openvpn)
    - [Install Podman and Podman Docker](#install-podman-and-podman-docker)
      - [Configure Podman](#configure-podman)
    - [Install GolangCI Lint](#install-golangci-lint)
    - [Install YAMLLint](#install-yamllint)
  - [Other OS Requirements](#other-os-requirements)
    - [RHEL](#rhel)
    - [Debian](#debian)
    - [MacOS](#macos)
  - [Getting Started](#getting-started)
  - [Getting Started With Docker Compose](#getting-started-with-docker-compose)
  - [Troubleshooting](#troubleshooting)


## Containerized RP Software Required

> [!TIP]
> For a minimal development environment, the recommended approach is to use the containerized setup, which requires only the locally installed binaries listed below.

* az
* make
* podman
* openvpn

> [!NOTE]
> Instructions for these binaries are provided below. Refer to the [Podman](#install-podman-and-podman-docker) section for setup details specific to your operating system (Linux or macOS with Podman Machine).

> [!IMPORTANT]
> With the local binaries installed you can then refer to the [Getting Started](#getting-started) section below to obtain the source code before deploying a development RP.
>
> Instead of running `make runlocal-rp`, use `make run-rp` to run a containerized version of the application without requiring additional local binaries.

---

## Local RP Dependencies

> [!NOTE]
> To run an RP instance as a Go process using `go run` locally, additional tools are required and are outlined below.

### Install Package Dependencies Fedora/RHEL

> [!IMPORTANT]
> For other OS specific requirements, refer to the [Other OS Requirements](#other-os-requirements) section.

1. General dependencies

    ```sh
    sudo dnf install gpgme-devel libassuan-devel openssl
    ```
2. Dependencies for Fedora 37+

    ```sh
    sudo dnf install lvm2 lvm2-devel golang-github-containerd-btrfs-devel
    ```
3. Dependencies for `pyenv`

    ```sh
    sudo dnf install bzip2-devel ncurses-devel libffi-devel readline-devel sqlite-devel tk-devel xz-devel zlib-devel gcc make
    ```

### Install Go 1.22

1. [Download Go 1.22](https://golang.org/dl)
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

### Install AZ Client

> [!NOTE]
> Due to the `az` client requiring a specific Python version, you will find the instructions to install the `az` client in the [Getting Started](#getting-started) section. This will use `pyenv` to ensure the correct Python version limited to the local ARO-RP environment.
>
> ARO-RP comes with `make pyenv`, this will set up the environment and install the `az` client after setting the local Python version via `pyenv`.

### Install OpenVPN

1. Find the client you require [here](https://openvpn.net/community-downloads/)
2. **Or:** on RHEL/Fedora run the following

    ```sh
    sudo dnf install openvpn
    ```

> [!NOTE]
> You can also use the built in Network Manager to add `.ovpn` configuration files.

### Install Podman and Podman Docker

> [!NOTE]
> Podman is used for building container images and running the installer.

1. Install Podman

    ```sh
    sudo dnf install podman
    ```
2. Install Podman Docker

    ```sh
    sudo dnf install podman-docker
    ```

#### Configure Podman

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

### Install GolangCI Lint

1. Find latest version [here](https://github.com/golangci/golangci-lint/releases)
2. Run the install

    ```sh
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.0.2
    ```

### Install YAMLLint

```sh
sudo dnf install yamllint
```

## Other OS Requirements

### RHEL

1. Register the system with `subscription-manager register`
2. Enable the [CodeReady Linux Builder](https://access.redhat.com/articles/4348511) repository to install *-devel packages
3. Enable the [EPEL repository](https://docs.fedoraproject.org/en-US/epel/#_quickstart) for packages not in the base repositories (such as OpenVPN)

### Debian

1. Install the required dependencies

    ```sh
    sudo apt install libgpgme-dev libbtrfs-dev libdevmapper-dev
    ```
2. Make sure that `PKG_CONFIG_PATH` contains the `pkgconfig` files of the above packages. For example:

    ```sh
    export PKG_CONFIG_PATH:/usr/lib/x86_64-linux-gnu/pkgconfig
    ```

> [!IMPORTANT]
> Your actual `pkgconfig` path may differ; please adjust it accordingly.

### MacOS

> [!NOTE]
> Developers using macOS are encouraged to contribute to this repository. To ensure compatibility, macOS users should install GNU utilities on their systems.
>
> The goal is to minimize shell scripting and other platform-specific variations within the repository. Installing GNU utilities on macOS helps reduce discrepancies in command-line flags, usage and more, ensuring a consistent development experience across environments.

1. Install the required dependencies

    ```sh
    brew install coreutils findutils gnu-tar grep gettext gpgme diffutils
    ```
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

## Getting Started

1. Clone the repository

    ```sh
    git clone https://github.com/Azure/ARO-RP.git
    ```
2. Go to project

    ```sh
    cd /path/to/ARO-RP
    ```
3. Configure `pyenv` Python version

    ```sh
    pyenv local 3.10.0
    pyenv rehash

    python --version
    ```
4. Make environment

    ```sh
    make pyenv
    ```

> [!TIP]
> This will install the `az` client. However, if the install fails you can attempt a re-install with:
>
> ```sh
> source pyenv/bin/activate
> pip install azure-cli
> ```

5. Login to Azure

    ```sh
    az login
    ```
6. Configure local `git`

    ```sh
    # Set pre-commit hook
    make init-contrib

    # Set GitHub username globally
    git config --global github.user "<USERNAME>"

    # OR: Set GitHub username locally to repo
    git config github.user "<USERNAME>"
    ```

> [!IMPORTANT]
> Running `make init-contrib` enforces a necessary branch naming convention for your commits.
>
> The convention is: `<USERNAME>/<JIRA_NUMBER>`
> You can also append a description after the `<JIRA_NUMBER>` e.g: `<USERNAME>/<JIRA_NUMBER>/my-description-here`

## Getting Started With Docker Compose

1. Install [Docker Compose](https://docs.docker.com/compose/install/linux/#install-using-the-repository)
   1. Fedora/RHEL

        ```sh
        sudo dnf install docker-compose-plugin
        ```
   2. Debian

        ```sh
        sudo apt install docker-compose-plugin
        ```
   3. MacOS
        ```sh
        brew install docker-compose
        ```

> [!WARNING]
> Pay attention to the notes after the `brew` installer runs as there will be instructions to follow to complete setup on MacOS.

2. Check the `env.example` file and copy it to create your own

    ```sh
    cp env.example env
    ```
3. Source the `env` file

    ```sh
    . ./env
    ```
4. Run VPN, RP, and Portal services using Docker Compose

    ```sh
    docker compose up
    ```

## Troubleshooting
| Issue                                                                                            | Resolution                                                                                                                                                                                                                                                                                         |
| ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Error ./env:.:11: no such file or directory: secrets/env.`                                      | Run `SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets` to resolve.                                                                                                                                                                                                                              |
| `az -v` does not return `aro` as a dependency.                                                   | Ensure the environment file parameters are correctly set, following the `env.example` file.                                                                                                                                                                                                        |
| Git commit fails due to branch naming error: `There is something wrong with your branch name...` | Ensure the branch name follows the required pattern: <br><br> ```^${USERNAME}\/(ARO-[0-9]{4}[a-z0-9._-]*\|hotfix-[a-z0-9._-]+\|gh-issue-[0-9]+[a-z0-9._-]*)$``` <br><br> If the PR is not tied to a Jira ticket, GitHub issue, or hotfix, use `--no-verify` with `git commit` to bypass the check. |
