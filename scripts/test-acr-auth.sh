#!/bin/bash
# Test ACR authentication locally using Azure CLI
# Requires: Azure CLI, appropriate Azure credentials
#
# SECURITY NOTE: This script uses --expose-token for local testing only.
# Do not share or log the access tokens in production environments.

set -e

echo "Testing ACR Authentication"
echo "=========================="
echo ""

# Check if logged in to Azure
if ! az account show &>/dev/null; then
    echo "❌ Not logged in to Azure. Run: az login"
    exit 1
fi

SUBSCRIPTION=$(az account show --query id -o tsv)
echo "✅ Logged in to Azure"
echo "   Subscription: $SUBSCRIPTION"
echo ""

# Test arosvcdev ACR access
echo "Testing arosvcdev.azurecr.io access..."
echo "--------------------------------------"

if az acr show --name arosvcdev &>/dev/null; then
    echo "✅ Can access arosvcdev ACR"

    echo "   Testing login..."
    if az acr login --name arosvcdev --expose-token &>/dev/null; then
        echo "✅ Successfully authenticated to arosvcdev.azurecr.io"
    else
        echo "⚠️  Cannot login to arosvcdev (may need AcrPush/AcrPull permissions)"
    fi
else
    echo "⚠️  Cannot access arosvcdev ACR (may not exist in this subscription or no permissions)"
fi

echo ""

# Test arointsvc ACR access (read-only expected)
echo "Testing arointsvc.azurecr.io access..."
echo "--------------------------------------"

if az acr show --name arointsvc &>/dev/null; then
    echo "✅ Can access arointsvc ACR"

    echo "   Testing login..."
    if az acr login --name arointsvc --expose-token &>/dev/null; then
        echo "✅ Successfully authenticated to arointsvc.azurecr.io"
    else
        echo "⚠️  Cannot login to arointsvc (expected if in different tenant)"
    fi
else
    echo "⚠️  Cannot access arointsvc ACR (expected if in different subscription/tenant)"
fi

echo ""
echo "=========================="
echo "Note: Service connection authentication cannot be tested locally."
echo "This script only tests Azure CLI-based authentication."
echo ""
echo "To test the full pipeline:"
echo "  1. Push changes to a PR branch"
echo "  2. Monitor the Azure DevOps pipeline run"
echo "  3. Check for authentication errors in pipeline logs"
