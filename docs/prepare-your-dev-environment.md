# Prepare Your Development Environment

This document goes through the development dependencies one requires in order to build the RP code.

## Containerized RP Software Required

If you just want to get up and running with a minimal dev environment I recommend starting out with our conainerized setup. For a containerized setup, the only local bins you need are:

```text
az
make
podman
openvpn
```

> NOTE: Instructions for these binaries are found below. In particular, see the `Podman` instructions below for a setup based on your OS flavor (Linux vs MacOS + Podman Machine)

That's it! You can jump to [Getting Started](#getting-started) below to grab the source code before heading to [deploy your own development RP](./deploy-development-rp.md) - but **instead of `make runlocal-rp`, invoke `make run-rp` instead** to use a containerized version of the app without needing additional local binaries.

## Local RP Software Required

If you'd like to run an RP instance as a golang process (via `go run`) locally - you'll need additional tools:

1. Install [Go 1.22](https://golang.org/dl), if you haven't already.
   1. After downloading follow the [Install instructions](https://go.dev/doc/install), replacing the tar archive with your download.
   1. Append `export PATH="${PATH}:/usr/local/go/bin"` to your shell's profile file.

1. Configure `GOPATH` as an OS environment variable in your shell (a requirement of some dependencies for `make generate`). If you want to keep the default path, you can add something like `GOPATH=$(go env GOPATH)` to your shell's profile/RC file.

1. Install [Python 3.6-3.10](https://www.python.org/downloads), if you haven't already.  You will also need `python-setuptools` installed, if you don't have it installed already. Python versions earlier than 3.6 or later than 3.10 are not supported as of now.

1. Install the [az client](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli), if you haven't already.

    > Depending on the default version of Python available on your system, it may be convenient to set up the above within a virtual env. You can do so by running the `make pyenv` Makefile target within this repository.
    > Ensure that your `python3` command points to a valid version of Python in the above range, e.g. 3.10, when running the command. 
    > You can then install the Azure CLI via Pip: `pip install azure-cli`.

1. Install [OpenVPN](https://openvpn.net/community-downloads) if it is not already installed

1. Install the relevant packages required for your OS defined below.

1. Install [Podman](https://podman.io/getting-started/installation) and [podman-docker](https://developers.redhat.com/blog/2019/02/21/podman-and-buildah-for-docker-users#) if you haven't already. Podman is used for building container images and running the installer.
    1. Podman needs to be running in daemon mode when running the RP locally.
        
        On Linux, you can set this up to automatically start via socket activation with::
            
            ```bash
            systemctl --user enable podman.socket
            ```
        
        If you're using podman-machine, you will need to export the socket, for example::

            ```bash
            export ARO_PODMAN_SOCKET=unix://$HOME/.local/share/containers/podman/machine/qemu/podman.sock
            ```
        
        You will also need to ensure that podman machine has enough resources::

            ```bash
            podman machine stop
            podman machine rm
            podman machine init --cpus 4 --memory 5000
            podman machine start
            ```

> __NOTE:__ If using Fedora 37+ podman and podman-docker should already be installed and enabled.

1. Run for `az acr login` compatability

```sh
sudo touch /etc/containers/nodocker
```

If using a MAC, the following steps may be applicable where you symlink docker to podman location. 

```sh
ls -la $(whereis -q docker)
lrwxr-xr-x@ 1 domfinn  staff  24  7 Dec 14:10 /Users/domfinn/.local/bin/docker -> /opt/homebrew/bin/podman
az acr login -n domfinnaro
Login Succeeded!
```

1. Install [golangci-lint](https://golangci-lint.run/) and [yamllint](https://yamllint.readthedocs.io/en/stable/quickstart.html#installing-yamllint) (optional but your code is required to comply to pass the CI)

### Fedora / RHEL Packages

1. Install the `gpgme-devel`, `libassuan-devel`, and `openssl` packages.
```sh
sudo dnf install -y gpgme-devel libassuan-devel openssl podman
```

> __NOTE:__ If using RHEL, register the system with `subscription-manager register`, and then enable the [CodeReady Linux Builder](https://access.redhat.com/articles/4348511) repository to install *-devel packages. For other packages not in the base repositories, such as OpenVPN, you can [enable the EPEL repository](https://docs.fedoraproject.org/en-US/epel/#_quickstart) to install them.

1. For Fedora 37+ you will also need to install the packages: `lvm2`, `lvm2-devel` and `golang-github-containerd-btrfs-devel`
```sh
sudo dnf install -y lvm2 lvm2-devel golang-github-containerd-btrfs-devel
```
### Debian Packages

Install the `libgpgme-dev`, `libbtrfs-dev` and `libdevmapper-dev` packages.

Make sure that `PKG_CONFIG_PATH` contains the pkgconfig files of the above packages.  E.g. `export PKG_CONFIG_PATH:/usr/lib/x86_64-linux-gnu/pkgconfig` (your actual pkgconfig path may vary, so adjust accordingly).

### MacOS Packages

1. We are open to developers on MacOS working on this repository.  We are asking MacOS users to setup GNU utils on their machines.

    We are aiming to limit the amount of shell scripting, etc. in the repository, installing the GNU utils on MacOS will minimise the chances of unexpected differences in command line flags, usages, etc., and make it easier for everyone to ensure compatibility down the line.

    Install the following packages on MacOS:

    ```bash
    # GNU Utils
    brew install coreutils findutils gnu-tar grep

    # Install envsubst (provided with gettext)
    brew install gettext
    brew link gettext

    # Install gpgme
    brew install gpgme

    # Install diffutils to avoid errors during test runs
    brew install diffutils
    ```

1. Modify your `~/.zshrc` (or `~/.bashrc` for Bash): this prepends `PATH` with GNU Utils paths;

    ```bash
    echo "export PATH=$(find $(brew --prefix)/opt -type d -follow -name gnubin -print | paste -s -d ':' -):\$PATH" >> ~/.zshrc
    ```

1. Add the following into your `~/.zshrc`/`~/.bashrc` file:

    ```bash
    export LDFLAGS="-L$(brew --prefix)/lib"
    export CFLAGS="-I$(brew --prefix)/include"
    export CGO_LDFLAGS=$LDFLAGS
    export CGO_CFLAGS=$CFLAGS
    ```

## Getting Started

1. Login to Azure:

    ```bash
    az login
    ```

1. Clone the repository to your local machine:

    ```bash
    go get -u github.com/Azure/ARO-RP/...
    ```

    Alternatively you can also use:

    ```bash
    git clone https://github.com/Azure/ARO-RP.git ${GOPATH:-$HOME/go}/src/github.com/Azure/ARO-RP
    ```

1. Go to project:

    ```bash
    cd ${GOPATH:-$HOME/go}/src/github.com/Azure/ARO-RP
    ```

1. Configure local git

    ```bash
    make init-contrib
    git config --global github.user <<user_name>>
    ```
    - **NOTE**: ```make init-contrib``` will enforce a branch naming regex on your commits.

# Troubleshooting

- Error`./env:.:11: no such file or directory: secrets/env`.
To resolve, run `SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets`.

- `az -v` does not return `aro` as dependency.
To resolve, make sure it is being used the `env` file parameters as per the `env.example`

- If you get the following error when running `git commit`

    ```bash
    There is something wrong with your branch name. Branch names in this project must adhere to this contract: ^${USERNAME}\/(ARO-[0-9]{4}[a-z0-9._-]*|hotfix-[a-z0-9._-]+|gh-issue-[0-9]+[a-z0-9._-]*)$. Your commit will be rejected. Please rename your branch (git branch --move) to a valid name and try again.
    ```
    Make sure you adhere to the rule, unless the PR is not tied to a Jira ticket, a GitHub issue or a hotfix. For those cases you can append `--no-verify` to your `git commit` command.

## Getting Started with Docker Compose

1. Install [Docker Compose](https://docs.docker.com/compose/install/linux/#install-using-the-repository)

2. Check the `env.example` file and copy it by creating your own:

        ```bash
        cp env.example env
        ```

3. Source the `env` file

        ```bash
        . ./env
        ```
4. Run VPN, RP, and Portal services using Docker Compose

        ```bash
        docker compose up
        ```
