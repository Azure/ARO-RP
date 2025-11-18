#!/bin/bash
# Initialize Cosmos DB emulator database structure from ARM template
# This script reads pkg/deploy/assets/databases-development.json and creates
# the corresponding database, collections, and triggers in the local emulator.

set -e

EMULATOR_ENDPOINT="${COSMOSDB_EMULATOR_ENDPOINT:-localhost:8081}"
SETUP_SCRIPT="./hack/cosmosdb-emulator/setup.go"
ARM_TEMPLATE="./pkg/deploy/assets/databases-development.json"

echo "Initializing Cosmos DB database structure..."

# Check if the emulator is running
if ! curl -s http://localhost:1234 > /dev/null 2>&1; then
    echo "ERROR: Cosmos DB emulator is not running"
    echo "Please start the emulator first with: make start-cosmos-emulator"
    exit 1
fi

echo "Reading database structure from ${ARM_TEMPLATE}..."

# Set environment variable and run the setup
export USE_COSMOS_DB_EMULATOR=true

# Run the setup
echo "Creating database, collections, and triggers..."
if go run ${SETUP_SCRIPT}; then
    echo ""
    echo "✅ Database structure initialized successfully!"
    echo ""
    echo "You can view the database in the Cosmos DB Explorer at:"
    echo "https://${EMULATOR_ENDPOINT}/_explorer/index.html"
else
    echo ""
    echo "❌ Failed to initialize database structure"
    exit 1
fi
