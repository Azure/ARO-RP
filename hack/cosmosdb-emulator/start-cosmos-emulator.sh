#!/bin/bash
# Start Cosmos DB emulator container for local development

set -e

CONTAINER_NAME="${COSMOSDB_EMULATOR_CONTAINER_NAME:-aro-cosmosdb-emulator}"
CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-$(command -v podman || command -v docker)}"
COSMOSDB_EMULATOR_IMAGE="${COSMOSDB_EMULATOR_IMAGE:-mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:vnext-preview}"
EMULATOR_ENDPOINT="${COSMOSDB_EMULATOR_ENDPOINT:-localhost:8081}"
EMULATOR_PORT=$(echo "$EMULATOR_ENDPOINT" | cut -d ':' -f2)
EMULATOR_PORT="${EMULATOR_PORT:-8081}"

echo "Starting Cosmos DB emulator..."

if ${CONTAINER_RUNTIME} ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo "Container ${CONTAINER_NAME} already exists. Checking if it's running..."
    
    if ${CONTAINER_RUNTIME} ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Container ${CONTAINER_NAME} is already running."
        exit 0
    else
        echo "Starting existing container ${CONTAINER_NAME}..."
        ${CONTAINER_RUNTIME} start ${CONTAINER_NAME}
    fi
else
    echo "Creating and starting new container ${CONTAINER_NAME}..."
    
    mkdir -p "${HOME}/.aro-rp/cosmos-certs"
    
    ${CONTAINER_RUNTIME} run -d \
        --name ${CONTAINER_NAME} \
        -p ${EMULATOR_PORT}:8081 \
        -p 1234:1234 \
        -p 8080:8080 \
        -v "${HOME}/.aro-rp/cosmos-certs:/certs:Z" \
        -m 8g \
        --cpus=6.0 \
        -e PORT=8081 \
        -e PROTOCOL=https \
        -e ENABLE_EXPLORER=true \
        -e EXPLORER_PORT=1234 \
        -e CERT_PATH=/scripts/certs/domain.pfx \
        -e CERT_SECRET=secret \
        -e ENABLE_TELEMETRY=false \
        -e LOG_LEVEL=debug \
        ${COSMOSDB_EMULATOR_IMAGE}
fi

echo "Waiting for Cosmos DB emulator to be ready..."
max_attempts=60
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -s http://localhost:1234 > /dev/null 2>&1; then
        echo ""
        echo "Cosmos DB emulator is ready!"
        echo ""
        echo "Connection details:"
        echo "  Gateway Endpoint: https://${EMULATOR_ENDPOINT}"
        echo "  Data Explorer: http://localhost:1234"
        echo "To initialize database structure, run: make init-cosmos-structure"
        echo ""
        exit 0
    fi
    
    echo -n "."
    sleep 2
    attempt=$((attempt + 1))
done

echo ""
echo "ERROR: Cosmos DB emulator failed to start within ${max_attempts} attempts"
echo "Check logs with: ${CONTAINER_RUNTIME} logs ${CONTAINER_NAME}"
exit 1
