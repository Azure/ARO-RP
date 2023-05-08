# Prepare Your Development Environment

This document goes through the development dependencies one requires in order to build the RP code.

## Software Required

1. Install [Go 1.18](https://golang.org/dl) or later, if you haven't already.
   1. After downloading follow the [Install instructions](https://go.dev/doc/install), replacing the tar archive with your download.
   1. Append `export PATH="${PATH}:/usr/local/go/bin"` to your shell's profile file.

1. Configure `GOPATH` as an OS environment variable in your shell (a requirement of some dependencies for `make generate`). If you want to keep the default path, you can add something like `GOPATH=$(go env GOPATH)` to your shell's profile/RC file.

1. Append `export GO111MODULE=auto` to your shell's profile file.
    1. Read [New module changes in Go 1.16](https://go.dev/blog/go116-module-changes) for more information.

1. Install [Python 3.6+](https://www.python.org/downloads), if you haven't already.  You will also need `python-setuptools` installed, if you don't have it installed already.

1. Install the [az client](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli), if you haven't already.

1. Install [OpenVPN](https://openvpn.net/community-downloads) if it is not already installed

1. Install the relevant packages required for your OS defined below.

1. Install [Podman](https://podman.io/getting-started/installation) and [podman-docker](https://developers.redhat.com/blog/2019/02/21/podman-and-buildah-for-docker-users#) if you haven't already, used for building container images.

1. Run for `az acr login` compatability

    ```bash
    sudo touch /etc/containers/nodocker
    ```

1. Install [golangci-lint](https://golangci-lint.run/) and [yamllint](https://yamllint.readthedocs.io/en/stable/quickstart.html#installing-yamllint) (optional but your code is required to comply to pass the CI)

### Fedora / RHEL Packages

1. Install the `gpgme-devel`, `libassuan-devel`, and `openssl` packages.
    > `sudo dnf install -y gpgme-devel libassuan-devel openssl`

> __NOTE:__: If using RHEL, register the system with `subscription-manager register`, and then enable the [CodeReady Linux Builder](https://access.redhat.com/articles/4348511) repository to install *-devel packages. For other packages not in the base repositories, such as OpenVPN, you can [enable the EPEL repository](https://docs.fedoraproject.org/en-US/epel/#_quickstart) to install them.

1. Optionally install [Docker 17.05+](https://docs.docker.com/engine/install/fedora/) or later as an alternative to podman.

### Debian Packages

Install the `libgpgme-dev` package.

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
    git clone https://github.com/Azure/ARO-RP.git $GOPATH/src/github.com/Azure/ARO-RP
    ```

1. Go to project:

    ```bash
    cd ${GOPATH:-$HOME/go}/src/github.com/Azure/ARO-RP
    ```
