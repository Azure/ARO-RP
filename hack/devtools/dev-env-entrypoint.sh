#!/bin/bash
set -e

# Dev environment entrypoint script.
# This script always runs inside a Linux container. It cannot use
# "uname -s" to detect the host OS because it will always return
# "Linux". Instead, it checks whether the host Podman socket was
# bind-mounted into the container (Linux host, via
# docker-compose.dev-env-linux.yml) or not (macOS host, where the
# macOS override does not mount a socket).

# /run/podman is pre-created and chowned in Dockerfile; re-create here in case /run is tmpfs
PODMAN_SOCK_DIR="/run/podman"
mkdir -p "${PODMAN_SOCK_DIR}" 2>/dev/null || {
	PODMAN_SOCK_DIR="/tmp/podman"
	mkdir -p "${PODMAN_SOCK_DIR}"
}

if [ -S "${PODMAN_SOCK_DIR}/podman.sock" ]; then
	echo "Using host Podman socket at ${PODMAN_SOCK_DIR}/podman.sock"
else
	echo "No host Podman socket found, starting Podman service inside container..."
	podman system service --time=0 "unix://${PODMAN_SOCK_DIR}/podman.sock" &
	for i in $(seq 1 30); do
		if [ -S "${PODMAN_SOCK_DIR}/podman.sock" ]; then
			break
		fi
		sleep 0.5
	done
	if [ ! -S "${PODMAN_SOCK_DIR}/podman.sock" ]; then
		echo "ERROR: Podman service did not start in time"
		exit 1
	fi
	echo "Podman service started at ${PODMAN_SOCK_DIR}/podman.sock"
fi

export ARO_PODMAN_SOCKET="unix://${PODMAN_SOCK_DIR}/podman.sock"

# Source environment and run the RP
. /workspace/env && make runlocal-rp
