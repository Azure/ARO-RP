#!/bin/bash
set -e

# Dev environment entrypoint script
# Auto-detects whether to use a host-mounted Podman socket or start
# Podman inside the container (macOS / Docker path).

if [ -S /podman/podman.sock ]; then
    # Linux path: host Podman socket is mounted
    export ARO_PODMAN_SOCKET="unix:///podman/podman.sock"
    echo "Using host Podman socket at /podman/podman.sock"
else
    # macOS / Docker path: start Podman inside the container
    echo "No host Podman socket found, starting Podman service inside container..."
    mkdir -p /run/podman
    podman system service --time=0 unix:///run/podman/podman.sock &
    # Wait for the socket to be ready
    for i in $(seq 1 30); do
        if [ -S /run/podman/podman.sock ]; then
            break
        fi
        sleep 0.5
    done
    if [ ! -S /run/podman/podman.sock ]; then
        echo "ERROR: Podman service did not start in time"
        exit 1
    fi
    export ARO_PODMAN_SOCKET="unix:///run/podman/podman.sock"
    echo "Podman service started at /run/podman/podman.sock"
fi

# Source environment and run the RP
. /workspace/env && make runlocal-rp
