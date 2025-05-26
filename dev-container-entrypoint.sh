#!/bin/bash

# Initialize pyenv
export PYENV_ROOT="$HOME/.pyenv"
export PATH="$PYENV_ROOT/bin:$PATH"

eval "$(pyenv init --path)"
eval "$(pyenv init -)"
eval "$(pyenv virtualenv-init -)"

# Create virtual environment outside of the mounted volume to avoid permissions issues
# We'll create it in /root/pyenv
export VIRTUAL_ENV_DIR="/root/pyenv"
if [ ! -d "$VIRTUAL_ENV_DIR" ]; then
  # Ensure the correct python3 (from pyenv) is used to create the venv
  /root/.pyenv/versions/3.10.0/bin/python3 -m venv "$VIRTUAL_ENV_DIR"
fi

# Activate the virtual environment
source "$VIRTUAL_ENV_DIR/bin/activate"

# Source environment variables from the main env file
if [ -f "/workspace/env" ]; then
  set -a
  . /workspace/env
  set +a
fi

# Run the command provided by docker-compose
exec "$@"