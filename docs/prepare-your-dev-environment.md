# Prepare Your Development Environment

This document goes through the development dependencies one requires in order to build the RP code.

## Software Required
1. Install [Go 1.16](https://golang.org/dl) or later, if you haven't already.

1. Install [Python 3.6+](https://www.python.org/downloads), if you haven't already.  You will also need `python-setuptools` installed, if you don't have it installed already.

1. Install `virtualenv`, a tool for managing Python virtual environments.
> The package is called `python-virtualenv` on both Fedora and Debian-based systems.

1. Install the [az client](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli), if you haven't already. You will need `az` version 2.0.72 or greater, as this version includes the `az network vnet subnet update --disable-private-link-service-network-policies` flag.

1. Install [OpenVPN](https://openvpn.net/community-downloads) if it is not already installed

1. Install the relevant packages required for your OS defined below.

### Fedora Packages

1. Install the `gpgme-devel`, `libassuan-devel`, and `openssl` packages.
> `sudo dnf install -y gpgme-devel libassuan-devel openssl`

### Debian Packages
1. Install the `libgpgme-dev` package.

### MacOS Packages
1. We are open to developers on MacOS working on this repository.  We are asking MacOS users to setup GNU utils on their machines.

We are aiming to limit the amount of shell scripting, etc. in the repository, installing the GNU utils on MacOS will minimise the chances of unexpected differences in command line flags, usages, etc., and make it easier for everyone to ensure compatibility down the line.

Install the following packages on MacOS:
```bash
# GNU Utils
brew install coreutils
brew install findutils
brew install gnu-tar
brew install grep

# Install envsubst
brew install gettext
brew link --force gettext

# Install
brew install gpgme

# GNU utils
# Ref: https://web.archive.org/web/20190704110904/https://www.topbug.net/blog/2013/04/14/install-and-use-gnu-command-line-tools-in-mac-os-x
# gawk, diffutils, gzip, screen, watch, git, rsync, wdiff
export PATH="/usr/local/bin:$PATH"
# coreutils
export PATH="/usr/local/opt/coreutils/libexec/gnubin:$PATH"
# findutils
export PATH="/usr/local/opt/findutils/libexec/gnubin:$PATH"

#grep
export PATH="/usr/local/opt/grep/libexec/gnubin:$PATH"

#python-virtualenv
sudo pip3 install virtualenv
```

## Getting Started
1. Login to Azure:
    ```bash
    az login
    ```

1. Clone the repository to your local machine:
    ```bash
    go get -u github.com/Azure/ARO-RP/...
    cd ${GOPATH:-$HOME/go}/src/github.com/Azure/ARO-RP
    ```
