# Prepare Your Development Environment

> [!NOTE]
> This document outlines the development dependencies required to build the RP code.

> [!WARNING]
> Failure to complete all prerequisites in [Local RP Development Setup](dev-environment/local-rp-dependencies.md) will cause failures

- [Prepare Your Development Environment](#prepare-your-development-environment)
- [Install Local RP Dependencies](#install-local-rp-dependencies)
  - [Getting Started](#getting-started)
  - [Getting Started With Docker Compose](#getting-started-with-docker-compose)
    - [Bash Helpers](#bash-helpers)
  - [How to use ARO-RP Makefile](#how-to-use-aro-rp-makefile)
  - [Troubleshooting](#troubleshooting)

# Install Local RP Dependencies

Complete all steps located in [Local RP Development Setup](dev-environment/local-rp-dependencies.md)

## Getting Started

Setting up your ARO-RP development environment.

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
5. Login to Azure
    > [!WARNING]
    > This will install the `az` client. However, if the install fails you can attempt a re-install with:
    >
    > ```sh
    > source pyenv/bin/activate
    > pip install azure-cli
    > ```
    ```sh
    az login
    ```
6. Configure `git`
   1. Set pre-commit hook
    ```sh
    make init-contrib
    ```
    2. Set GitHub username globally
    ```sh
    git config --global github.user "<USERNAME>"
    ```
    > [!TIP]
    > Your GitHub username can optionally be set locally to this repository
    > ```sh
    > git config github.user "<USERNAME>"
    > ```

## Getting Started With Docker Compose

1. Install optional dependencies
   1. [Fedora/RHEL Optional Dependencies](#fedorarhel-optional-dependencies)
   2. [Debian Optional Dependencies](#debian-optional-dependencies)
   3. [MacOS Optional Dependencies](#optional-macos-dependencies)
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
    docker compose up \
        vpn \
        rp \
        portal
    ```

### Bash Helpers

See [Bash Environment Helpers](dev-environment/bash-environment.md)

## How to use ARO-RP Makefile

See [Makefile Usage](dev-environment/makefile.md)

## Troubleshooting
| Issue                                                                                            | Resolution                                                                                                                                                                                                                                                                                         |
| ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Error ./env:.:11: no such file or directory: secrets/env.`                                      | Run `SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets` to resolve.                                                                                                                                                                                                                              |
| `az -v` does not return `aro` as a dependency.                                                   | Ensure the environment file parameters are correctly set, following the `env.example` file.                                                                                                                                                                                                        |
| Git commit fails due to branch naming error: `There is something wrong with your branch name...` | Ensure the branch name follows the required pattern: <br><br> ```^${USERNAME}\/(ARO-[0-9]{4,}[a-z0-9._-]*\|hotfix-[a-z0-9._-]+\|gh-issue-[0-9]+[a-z0-9._-]*)$``` <br><br> If the PR is not tied to a Jira ticket, GitHub issue, or hotfix, use `--no-verify` with `git commit` to bypass the check. |
