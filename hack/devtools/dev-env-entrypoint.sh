#!/bin/bash
set -e

# Dev environment entrypoint script.
# This script always runs inside a Linux container. It cannot use
# "uname -s" to detect the host OS because it will always return
# "Linux". Instead, it checks whether the host Podman socket was
# bind-mounted into the container (Linux host, via
# docker-compose.dev-env-linux.yml) or not (macOS host, where the
# macOS override does not mount a socket).

mkdir -p /run/podman

if [ -S /run/podman/podman.sock ]; then
	# Host Podman socket was mounted — use it directly
	echo "Using host Podman socket at /run/podman/podman.sock"
else
	# No host socket — start Podman inside the container
	echo "No host Podman socket found, starting Podman service inside container..."
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
	echo "Podman service started at /run/podman/podman.sock"
fi

export ARO_PODMAN_SOCKET="unix:///run/podman/podman.sock"

# Source environment and run the RP
. /workspace/env && make runlocal-rp
