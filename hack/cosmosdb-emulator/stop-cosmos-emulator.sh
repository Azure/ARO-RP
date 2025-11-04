#!/bin/bash
# Stop Cosmos DB emulator container

set -e

# Configuration
CONTAINER_NAME="${COSMOSDB_EMULATOR_CONTAINER_NAME:-aro-cosmosdb-emulator}"
CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-$(command -v podman || command -v docker)}"

echo "Stopping Cosmos DB emulator..."

# Check if container exists
if ${CONTAINER_RUNTIME} ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo "Stopping container ${CONTAINER_NAME}..."
    ${CONTAINER_RUNTIME} stop ${CONTAINER_NAME}
    
    echo "Container stopped. To remove it completely, run:"
    echo "${CONTAINER_RUNTIME} rm ${CONTAINER_NAME}"
else
    echo "Container ${CONTAINER_NAME} does not exist."
fi
