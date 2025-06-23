#!/bin/bash

# Source environment variables from the main env file
if [ -f "/workspace/env" ]; then
  set -a
  . /workspace/env
  set +a
fi

# Ensure Go version matches go.mod
GO_MOD_PATH="/workspace/go.mod"
REQ_GO_VERSION=$(awk '/^go / {print $2}' "$GO_MOD_PATH")
CURRENT_GO_VERSION=$(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
if [ "$REQ_GO_VERSION" != "$CURRENT_GO_VERSION" ]; then
  echo "Installing Go $REQ_GO_VERSION..."
  wget -q "https://go.dev/dl/go${REQ_GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
fi
export PATH="/usr/local/go/bin:$PATH"
export PATH="/root/go/bin:$PATH"

# Install Go tools using bingo if .bingo directory exists
if [ -d "/workspace/.bingo" ]; then
  if ! command -v bingo &> /dev/null; then
    go install github.com/bwplotka/bingo@latest
  fi
  cd /workspace
  bingo get
  export PATH="/workspace/.bingo/bin:$PATH"
  cd -
fi

# Run the command provided by docker-compose
exec "$@"