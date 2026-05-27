#!/bin/bash
# Test script for the purge pipeline logic
# This script helps you test the purge logic locally before deploying to ADO

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ARO Purge Pipeline Test Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Check if clean binary exists
if [ ! -f "./clean" ]; then
    echo -e "${YELLOW}Building the clean tool...${NC}"
    cd "$(dirname "$0")/../.."
    go build -o ./hack/clean/clean ./hack/clean
    cd ./hack/clean
    echo -e "${GREEN}✓ Clean tool built successfully${NC}"
else
    echo -e "${GREEN}✓ Clean tool already exists${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Step 1: Azure Credentials${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if environment variables are already set
if [ -n "$AZURE_CLIENT_ID" ] && [ -n "$AZURE_CLIENT_SECRET" ] && [ -n "$AZURE_TENANT_ID" ] && [ -n "$AZURE_SUBSCRIPTION_ID" ]; then
    echo -e "${GREEN}✓ Azure credentials already set in environment${NC}"
    echo "  Client ID: $AZURE_CLIENT_ID"
    echo "  Tenant ID: $AZURE_TENANT_ID"
    echo "  Subscription ID: $AZURE_SUBSCRIPTION_ID"
    echo ""
    read -p "Do you want to use these credentials? (y/n): " use_existing
    if [ "$use_existing" != "y" ]; then
        unset AZURE_CLIENT_ID AZURE_CLIENT_SECRET AZURE_TENANT_ID AZURE_SUBSCRIPTION_ID
    fi
fi

# If not set, guide user to set them
if [ -z "$AZURE_CLIENT_ID" ]; then
    echo -e "${YELLOW}You need to provide Azure credentials.${NC}"
    echo ""
    echo "Option 1: Use existing Service Principal"
    echo "  Set these environment variables:"
    echo "    export AZURE_CLIENT_ID=\"<client-id>\""
    echo "    export AZURE_CLIENT_SECRET=\"<client-secret>\""
    echo "    export AZURE_TENANT_ID=\"<tenant-id>\""
    echo "    export AZURE_SUBSCRIPTION_ID=\"<subscription-id>\""
    echo ""
    echo "Option 2: Create a new test Service Principal"
    echo "  Run: az ad sp create-for-rbac --name \"aro-purge-test-sp\" --role Reader --scopes /subscriptions/<SUBSCRIPTION_ID>"
    echo ""
    echo -e "${RED}Please set the credentials and run this script again.${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Step 2: Purge Configuration${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Set default purge configuration
if [ -z "$AZURE_PURGE_TTL" ]; then
    export AZURE_PURGE_TTL="48h"
    echo -e "${YELLOW}Setting default TTL: $AZURE_PURGE_TTL${NC}"
else
    echo -e "${GREEN}✓ TTL already set: $AZURE_PURGE_TTL${NC}"
fi

if [ -z "$AZURE_PURGE_CREATED_TAG" ]; then
    export AZURE_PURGE_CREATED_TAG="createdAt"
    echo -e "${YELLOW}Setting default created tag: $AZURE_PURGE_CREATED_TAG${NC}"
else
    echo -e "${GREEN}✓ Created tag already set: $AZURE_PURGE_CREATED_TAG${NC}"
fi

if [ -z "$AZURE_PURGE_RESOURCEGROUP_PREFIXES" ]; then
    export AZURE_PURGE_RESOURCEGROUP_PREFIXES="aro-,test-,dev-"
    echo -e "${YELLOW}Setting default prefixes: $AZURE_PURGE_RESOURCEGROUP_PREFIXES${NC}"
else
    echo -e "${GREEN}✓ Prefixes already set: $AZURE_PURGE_RESOURCEGROUP_PREFIXES${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Step 3: View Current Resource Groups${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Azure CLI is available
if command -v az &> /dev/null; then
    echo -e "${YELLOW}Fetching resource groups from subscription...${NC}"
    echo ""

    # Try to list resource groups
    if az group list --subscription "$AZURE_SUBSCRIPTION_ID" -o table 2>/dev/null; then
        echo ""
        echo -e "${GREEN}✓ Successfully connected to Azure${NC}"

        # Show filtered resource groups
        echo ""
        echo -e "${YELLOW}Resource groups matching prefixes ($AZURE_PURGE_RESOURCEGROUP_PREFIXES):${NC}"
        IFS=',' read -ra PREFIXES <<< "$AZURE_PURGE_RESOURCEGROUP_PREFIXES"
        for prefix in "${PREFIXES[@]}"; do
            az group list --subscription "$AZURE_SUBSCRIPTION_ID" \
                --query "[?starts_with(name, '$prefix')].{Name:name, CreatedAt:tags.createdAt, Persist:tags.persist, Location:location}" \
                -o table 2>/dev/null || true
        done
    else
        echo -e "${YELLOW}Could not list resource groups. You may need to authenticate with Azure CLI:${NC}"
        echo "  az login --tenant $AZURE_TENANT_ID"
        echo ""
        echo -e "${YELLOW}Note: The clean tool will use service principal credentials, not Azure CLI.${NC}"
    fi
else
    echo -e "${YELLOW}Azure CLI not found. Skipping resource group listing.${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Step 4: Test Options${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo "What would you like to do?"
echo ""
echo "1. DRY RUN - Show what would be deleted (SAFE, recommended first)"
echo "2. PRODUCTION RUN - Actually delete resources (DANGEROUS)"
echo "3. Create test resource groups first"
echo "4. Show current configuration"
echo "5. Exit"
echo ""
read -p "Enter your choice (1-5): " choice

case $choice in
    1)
        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}Running DRY RUN (safe mode)${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo -e "${YELLOW}Configuration:${NC}"
        echo "  TTL: $AZURE_PURGE_TTL"
        echo "  Created Tag: $AZURE_PURGE_CREATED_TAG"
        echo "  Prefixes: $AZURE_PURGE_RESOURCEGROUP_PREFIXES"
        echo "  Subscription: $AZURE_SUBSCRIPTION_ID"
        echo ""
        echo -e "${BLUE}Running: ./clean -dryRun=true${NC}"
        echo ""
        ./clean -dryRun=true
        echo ""
        echo -e "${GREEN}✓ Dry run complete!${NC}"
        echo -e "${YELLOW}No resources were actually deleted.${NC}"
        ;;
    2)
        echo ""
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}WARNING: PRODUCTION MODE${NC}"
        echo -e "${RED}========================================${NC}"
        echo ""
        echo -e "${RED}This will ACTUALLY DELETE resources!${NC}"
        echo ""
        echo "Configuration:"
        echo "  TTL: $AZURE_PURGE_TTL"
        echo "  Created Tag: $AZURE_PURGE_CREATED_TAG"
        echo "  Prefixes: $AZURE_PURGE_RESOURCEGROUP_PREFIXES"
        echo "  Subscription: $AZURE_SUBSCRIPTION_ID"
        echo ""
        echo -e "${YELLOW}It is STRONGLY recommended to run dry-run first (option 1)${NC}"
        echo ""
        read -p "Are you ABSOLUTELY SURE you want to proceed? (type 'DELETE' to confirm): " confirm
        if [ "$confirm" = "DELETE" ]; then
            echo ""
            echo -e "${RED}Running: ./clean -dryRun=false${NC}"
            echo ""
            ./clean -dryRun=false
            echo ""
            echo -e "${GREEN}✓ Production run complete!${NC}"
        else
            echo -e "${YELLOW}Cancelled. Good choice!${NC}"
        fi
        ;;
    3)
        echo ""
        echo -e "${BLUE}========================================${NC}"
        echo -e "${BLUE}Create Test Resource Groups${NC}"
        echo -e "${BLUE}========================================${NC}"
        echo ""

        if ! command -v az &> /dev/null; then
            echo -e "${RED}Azure CLI is required to create test resource groups.${NC}"
            exit 1
        fi

        read -p "Enter Azure location (e.g., eastus): " location
        if [ -z "$location" ]; then
            location="eastus"
        fi

        echo ""
        echo -e "${YELLOW}Creating test resource groups in $location...${NC}"
        echo ""

        # Create old resource group (should be purged)
        OLD_DATE="2024-01-01T00:00:00.000Z"
        echo "1. Creating 'aro-test-old-rg' (old, should be purged)..."
        az group create --name "aro-test-old-rg" --location "$location" \
            --subscription "$AZURE_SUBSCRIPTION_ID" \
            --tags createdAt="$OLD_DATE"

        # Create recent resource group (should be kept)
        RECENT_DATE=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
        echo "2. Creating 'aro-test-recent-rg' (recent, should be kept)..."
        az group create --name "aro-test-recent-rg" --location "$location" \
            --subscription "$AZURE_SUBSCRIPTION_ID" \
            --tags createdAt="$RECENT_DATE"

        # Create old resource group with persist tag (should be kept)
        echo "3. Creating 'aro-test-persist-rg' (old but persist=true, should be kept)..."
        az group create --name "aro-test-persist-rg" --location "$location" \
            --subscription "$AZURE_SUBSCRIPTION_ID" \
            --tags createdAt="$OLD_DATE" persist="true"

        # Create old resource group without prefix (should be kept)
        echo "4. Creating 'other-test-old-rg' (old but wrong prefix, should be kept)..."
        az group create --name "other-test-old-rg" --location "$location" \
            --subscription "$AZURE_SUBSCRIPTION_ID" \
            --tags createdAt="$OLD_DATE"

        echo ""
        echo -e "${GREEN}✓ Test resource groups created!${NC}"
        echo ""
        echo "Expected behavior when you run dry-run:"
        echo -e "${GREEN}  ✓ aro-test-old-rg - SHOULD BE PURGED${NC}"
        echo -e "${RED}  ✗ aro-test-recent-rg - should be KEPT (too new)${NC}"
        echo -e "${RED}  ✗ aro-test-persist-rg - should be KEPT (persist tag)${NC}"
        echo -e "${RED}  ✗ other-test-old-rg - should be KEPT (wrong prefix)${NC}"
        echo ""
        echo "Run this script again and choose option 1 to test!"
        ;;
    4)
        echo ""
        echo -e "${BLUE}========================================${NC}"
        echo -e "${BLUE}Current Configuration${NC}"
        echo -e "${BLUE}========================================${NC}"
        echo ""
        echo "Azure Credentials:"
        echo "  AZURE_CLIENT_ID: $AZURE_CLIENT_ID"
        echo "  AZURE_CLIENT_SECRET: ${AZURE_CLIENT_SECRET:0:4}***"
        echo "  AZURE_TENANT_ID: $AZURE_TENANT_ID"
        echo "  AZURE_SUBSCRIPTION_ID: $AZURE_SUBSCRIPTION_ID"
        echo ""
        echo "Purge Settings:"
        echo "  AZURE_PURGE_TTL: $AZURE_PURGE_TTL"
        echo "  AZURE_PURGE_CREATED_TAG: $AZURE_PURGE_CREATED_TAG"
        echo "  AZURE_PURGE_RESOURCEGROUP_PREFIXES: $AZURE_PURGE_RESOURCEGROUP_PREFIXES"
        echo ""
        ;;
    5)
        echo ""
        echo -e "${BLUE}Exiting. No changes made.${NC}"
        exit 0
        ;;
    *)
        echo ""
        echo -e "${RED}Invalid choice. Exiting.${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Next Steps${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "To run the test again with different settings:"
echo ""
echo "  # Adjust TTL"
echo "  export AZURE_PURGE_TTL=\"72h\""
echo ""
echo "  # Adjust prefixes"
echo "  export AZURE_PURGE_RESOURCEGROUP_PREFIXES=\"aro-,test-\""
echo ""
echo "  # Run this script again"
echo "  ./test-purge.sh"
echo ""
echo "To clean up test resource groups:"
echo "  az group delete --name aro-test-old-rg --yes --no-wait"
echo "  az group delete --name aro-test-recent-rg --yes --no-wait"
echo "  az group delete --name aro-test-persist-rg --yes --no-wait"
echo "  az group delete --name other-test-old-rg --yes --no-wait"
echo ""
