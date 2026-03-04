# ARO-RP/python

[draft]

Welcome to the ARO command-line extension.

Code in these directories make up the commands and logic for the `az aro` CLI.

## Structure

### `python/az/aro/azext_aro`

This is where the majority of human written code lives.

highlights:

- `__init__.py` - Azure CLI entrypoint
- `commands.py` - ARO extension command structure definitions
- `custom.py` - Logic and helper methods for subcommands
- `_help.py` - Help output definitions

### `python/az/aro/build`

Locally generated code

### `python/az/aro/azext_aro/aaz`

Generated code vendored from AZ tooling (upstream?) that we occasionally change
when we need new classes or functions.

### `python/az/client`

More vendored code

## Prerequisites

Ensure you have `python` installed. I recommend using `asdf`
([ref](https://asdf-vm.com/)) if you aren't already.

## Setup

From the repo root, run `make pyenv` to create and setup our local python
virtual environment.
