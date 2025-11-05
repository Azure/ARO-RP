#!/bin/bash
# Start Cosmos DB emulator container for local development

set -e

CONTAINER_NAME="${COSMOSDB_EMULATOR_CONTAINER_NAME:-aro-cosmosdb-emulator}"
CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-$(command -v podman || command -v docker)}"
COSMOSDB_EMULATOR_IMAGE="${COSMOSDB_EMULATOR_IMAGE:-mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest}"
EMULATOR_ENDPOINT="${COSMOSDB_EMULATOR_ENDPOINT:-localhost:8081}"
EMULATOR_PORT=$(echo "$EMULATOR_ENDPOINT" | cut -d ':' -f2)
EMULATOR_PORT="${EMULATOR_PORT:-8081}"

echo "Starting Cosmos DB emulator..."
echo "Note: To connect to the emulator, you must set COSMOSDB_EMULATOR_ENDPOINT and COSMOSDB_EMULATOR_KEY environment variables."

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
    ${CONTAINER_RUNTIME} run -d \
        --name ${CONTAINER_NAME} \
        -p ${EMULATOR_PORT}:8081 \
        -p 10250-10255:10250-10255 \
        -e AZURE_COSMOS_EMULATOR_PARTITION_COUNT=25 \
        -e AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE=true \
        -e AZURE_COSMOS_EMULATOR_IP_ADDRESS_OVERRIDE=127.0.0.1 \
        -m 4g \
        --cpus=4.0 \
        ${COSMOSDB_EMULATOR_IMAGE}
fi

echo "Waiting for Cosmos DB emulator to be ready..."
max_attempts=60
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -k -s https://${EMULATOR_ENDPOINT}/_explorer/emulator.pem > /dev/null 2>&1; then
        echo "Cosmos DB emulator is ready!"
        echo ""
        echo "Connection details:"
        echo "  Endpoint: https://${EMULATOR_ENDPOINT}"
        echo "  Database Explorer: https://${EMULATOR_ENDPOINT}/_explorer/index.html"
        echo ""
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
exit 1

