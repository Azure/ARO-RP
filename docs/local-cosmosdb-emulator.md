# Local CosmosDB Emulator for ARO-RP Development

This document describes how to use the Azure Cosmos DB Emulator for local ARO-RP development.

## Overview

The CosmosDB emulator provides a local environment that emulates the Azure Cosmos DB service for development and testing purposes.

## Quick Start

1. Enable the Cosmos DB emulator in your `env` file by uncommenting these lines:
   ```bash
   # In your env file:
   export USE_COSMOS_DB_EMULATOR=true
   export COSMOSDB_EMULATOR_ENDPOINT=localhost:8081
   export COSMOSDB_EMULATOR_KEY=[KEY]
   ```
   
   Then source your environment:
   ```bash
   source env
   ```

2. Start the emulator, initialize the database, and run the RP:
   ```bash
   INIT_COSMOSDB_STRUCTURE=true make runlocal-rp-cosmos
   ```

   This single command will:
   - Start the Cosmos DB emulator container
   - Wait for it to be ready
   - Initialize the database structure (collections, triggers)
   - Run the RP with local Cosmos DB

## Manual Setup

### Starting the Emulator

```bash
make start-cosmos-emulator
```

This starts a Docker/Podman container with:
- 25 partitions (supports multiple databases/collections)
- 4GB memory allocation
- 4 CPU cores
- Ports 8081 and 10250-10255 exposed
- Data persistence enabled

### Initializing Database Structure

```bash
make init-cosmos-structure
```

This creates:
- Database
- Collections
- JavaScript triggers

### Running the RP

```bash
make runlocal-rp-cosmos
```

This runs the RP with `USE_COSMOS_DB_EMULATOR=true` and `RP_MODE=development`.

### Stopping the Emulator

```bash
make stop-cosmos-emulator
```

## Configuration

### Required Environment Variables

When `USE_COSMOS_DB_EMULATOR=true`, these must be set:

| Variable | Description | Default |
|----------|-------------|---------|
| `COSMOSDB_EMULATOR_ENDPOINT` | Emulator endpoint | No default, must be set |
| `COSMOSDB_EMULATOR_KEY` | Authentication key | No default, must be set |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `COSMOSDB_EMULATOR_CONTAINER_NAME` | Container name | `aro-cosmosdb-emulator` |
| `COSMOSDB_EMULATOR_IMAGE` | Docker image | `mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest` |
| `CONTAINER_RUNTIME` | Docker or Podman | Auto-detected |
| `INIT_COSMOSDB_STRUCTURE` | Initialize DB on startup | `false` |

## Accessing the Emulator
Access the web-based data explorer at:
```
https://[COSMOSDB_EMULATOR_ENDPOINT]/_explorer/index.html
```