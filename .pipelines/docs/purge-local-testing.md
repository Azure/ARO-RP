# Local Testing Guide for Purge Pipeline

This guide shows how to test the purge logic locally before deploying to Azure DevOps.

## Prerequisites

- Go 1.25+ installed
- Azure CLI installed and authenticated
- Access to the target Azure subscription
- Service principal credentials (or use your own Azure CLI credentials)

## Method 1: Using Service Principal (Recommended)

This method simulates exactly how the pipeline will run in ADO.

### Step 1: Set Up Service Principal Credentials

If you already have a service principal for the new tenant:

```bash
# Export the credentials
export AZURE_CLIENT_ID="<your-client-id>"
export AZURE_CLIENT_SECRET="<your-client-secret>"
export AZURE_TENANT_ID="<your-tenant-id>"
export AZURE_SUBSCRIPTION_ID="<new-tenant-subscription-id>"
```

Or create a temporary service principal:

```bash
# Login to the new tenant
az login --tenant <NEW_TENANT_ID>

# Create a service principal (Reader role is sufficient for dry-run testing)
SP_OUTPUT=$(az ad sp create-for-rbac \
  --name "aro-purge-test-sp" \
  --role Reader \
  --scopes /subscriptions/<SUBSCRIPTION_ID>)

# Export the credentials
export AZURE_CLIENT_ID=$(echo $SP_OUTPUT | jq -r '.appId')
export AZURE_CLIENT_SECRET=$(echo $SP_OUTPUT | jq -r '.password')
export AZURE_TENANT_ID=$(echo $SP_OUTPUT | jq -r '.tenant')
export AZURE_SUBSCRIPTION_ID="<SUBSCRIPTION_ID>"
```

### Step 2: Configure Purge Settings

Set the same environment variables the pipeline uses:

```bash
# Time-to-live: resources older than this will be purged
export AZURE_PURGE_TTL="48h"

# Tag name containing the creation timestamp
export AZURE_PURGE_CREATED_TAG="createdAt"

# Comma-separated prefixes of resource groups to consider
export AZURE_PURGE_RESOURCEGROUP_PREFIXES="aro-,test-,dev-"
```

### Step 3: Build the Tool

```bash
# From the repository root
go build -o ./clean ./hack/clean
```

### Step 4: Run in Dry-Run Mode (Safe)

```bash
# Dry run (default) - shows what would be deleted without actually deleting
./clean -dryRun=true
```

Example output:
```
INFO[0000] Starting the resource cleaner, DryRun: true
INFO[0001] Group aro-test-cluster-20240101 is still less than TTL. SKIP.
INFO[0001] Group aro-old-cluster-20231201 would be DELETED
INFO[0001] Group test-rg-123 does not have createdAt tag. SKIP.
INFO[0002] Group prod-cluster is to persist. SKIP.
```

### Step 5: Run in Production Mode (Dangerous - Actually Deletes)

**⚠️ WARNING: This will actually delete resources!**

Only do this after verifying dry-run output and ensuring you have Contributor role:

```bash
# Update SP to have Contributor role if needed
az role assignment create \
  --assignee $AZURE_CLIENT_ID \
  --role Contributor \
  --scope /subscriptions/$AZURE_SUBSCRIPTION_ID

# Run in production mode
./clean -dryRun=false
```

## Method 2: Use Azure CLI to Create Temporary Service Principal Credentials

For quick testing, you can use Azure CLI to create a temporary service principal
and export the exact environment variables the clean tool requires:

```bash
# Login with Azure CLI
az login --tenant <TENANT_ID>
az account set --subscription <SUBSCRIPTION_ID>

# Create a temporary service principal scoped to the target subscription
SP_OUTPUT=$(az ad sp create-for-rbac \
  --name "aro-purge-local-test" \
  --role Contributor \
  --scopes "/subscriptions/<SUBSCRIPTION_ID>" \
  --output json)

# Export the credentials expected by the clean tool
export AZURE_CLIENT_ID="$(echo "$SP_OUTPUT" | jq -r '.appId')"
export AZURE_CLIENT_SECRET="$(echo "$SP_OUTPUT" | jq -r '.password')"
export AZURE_TENANT_ID="$(echo "$SP_OUTPUT" | jq -r '.tenant')"
export AZURE_SUBSCRIPTION_ID="<SUBSCRIPTION_ID>"

# Verify required variables are set before running the tool
env | grep '^AZURE_'

# Optional cleanup when finished testing
# az ad sp delete --id "$AZURE_CLIENT_ID"
```

## Method 3: Test Against a Sandbox Subscription

Create a test environment to safely verify the logic:

### Step 1: Create Test Resource Groups

```bash
# Create some test resource groups with appropriate tags
az group create --name "aro-test-old-rg" --location eastus --tags \
  createdAt="2024-01-01T00:00:00.000Z"

az group create --name "aro-test-recent-rg" --location eastus --tags \
  createdAt="$(date -u +"%Y-%m-%dT%H:%M:%S.%NZ")"

az group create --name "aro-test-persist-rg" --location eastus --tags \
  createdAt="2024-01-01T00:00:00.000Z" \
  persist="true"

az group create --name "other-test-rg" --location eastus --tags \
  createdAt="2024-01-01T00:00:00.000Z"
```

### Step 2: Run Dry-Run Test

```bash
export AZURE_PURGE_TTL="48h"
export AZURE_PURGE_CREATED_TAG="createdAt"
export AZURE_PURGE_RESOURCEGROUP_PREFIXES="aro-"

./clean -dryRun=true
```

Expected behavior:
- ✅ `aro-test-old-rg` - Should be marked for deletion (has prefix, old, no persist tag)
- ❌ `aro-test-recent-rg` - Should be skipped (created recently)
- ❌ `aro-test-persist-rg` - Should be skipped (has persist=true)
- ❌ `other-test-rg` - Should be skipped (doesn't match prefix)

### Step 3: Verify and Clean Up Test Resources

```bash
# If everything looks good, you can test actual deletion on these test resources
./clean -dryRun=false

# Or manually clean up
az group delete --name "aro-test-old-rg" --yes --no-wait
az group delete --name "aro-test-recent-rg" --yes --no-wait
az group delete --name "aro-test-persist-rg" --yes --no-wait
az group delete --name "other-test-rg" --yes --no-wait
```

## Debugging and Logging

### Increase Log Verbosity

The tool uses logrus for logging. You can see more detailed output by checking the code, but the default INFO level should show all decisions.

### Check What Resources Exist

Before running the purge tool:

```bash
# List all resource groups in the subscription
az group list --subscription $AZURE_SUBSCRIPTION_ID -o table

# List with tags
az group list --subscription $AZURE_SUBSCRIPTION_ID \
  --query "[].{Name:name, CreatedAt:tags.createdAt, Persist:tags.persist}" \
  -o table

# Filter by prefix
az group list --subscription $AZURE_SUBSCRIPTION_ID \
  --query "[?starts_with(name, 'aro-')]" -o table
```

### Verify Tag Values

Check if tags are set correctly:

```bash
az group show --name <RESOURCE_GROUP_NAME> --query tags
```

### Calculate Age Manually

```bash
# Get creation time
CREATED_AT=$(az group show --name <RG_NAME> --query "tags.createdAt" -o tsv)

# Show creation date
echo "Created at: $CREATED_AT"

# Current time
echo "Current time: $(date -u +"%Y-%m-%dT%H:%M:%S.%NZ")"

# Calculate hours since creation (requires GNU date)
CREATED_EPOCH=$(date -d "$CREATED_AT" +%s)
NOW_EPOCH=$(date +%s)
HOURS_OLD=$(( ($NOW_EPOCH - $CREATED_EPOCH) / 3600 ))
echo "Resource is $HOURS_OLD hours old"
```

## Testing Different Scenarios

### Scenario 1: Test TTL Threshold

```bash
# Test with 1 hour TTL (very aggressive)
export AZURE_PURGE_TTL="1h"
./clean -dryRun=true

# Test with 7 days TTL (very conservative)
export AZURE_PURGE_TTL="168h"
./clean -dryRun=true
```

### Scenario 2: Test Different Prefixes

```bash
# Test with multiple prefixes
export AZURE_PURGE_RESOURCEGROUP_PREFIXES="aro-,test-,dev-,tmp-"
./clean -dryRun=true

# Test with no prefix filter (considers all groups)
unset AZURE_PURGE_RESOURCEGROUP_PREFIXES
./clean -dryRun=true
```

### Scenario 3: Test Different Tag Names

```bash
# If using a different tag name
export AZURE_PURGE_CREATED_TAG="creationTime"
./clean -dryRun=true
```

## Common Issues and Solutions

### Issue: "cannot ValidateVars: missing environment variable"

**Solution:** Ensure all required variables are set:
```bash
echo "AZURE_CLIENT_ID: $AZURE_CLIENT_ID"
echo "AZURE_CLIENT_SECRET: ${AZURE_CLIENT_SECRET:0:4}***"
echo "AZURE_TENANT_ID: $AZURE_TENANT_ID"
echo "AZURE_SUBSCRIPTION_ID: $AZURE_SUBSCRIPTION_ID"
```

### Issue: "authentication failed"

**Solution:** 
- Verify service principal credentials are correct
- Check if service principal has not expired
- Ensure you're using the correct tenant ID

### Issue: "No resources are being marked for deletion"

**Solution:**
- Verify resource groups have the `createdAt` tag (or your configured tag name)
- Check the tag value is in RFC3339 format: `2024-01-01T00:00:00.000Z`
- Ensure TTL has actually expired
- Verify prefix matching if configured

### Issue: Tool crashes or panics

**Solution:**
- Check the implementation in `hack/clean/clean.go`
- Verify Go dependencies are up to date: `make go-tidy`
- Rebuild the tool: `go build -o ./clean ./hack/clean`

## Environment Variables Reference

| Variable | Required | Default | Example | Description |
|----------|----------|---------|---------|-------------|
| `AZURE_CLIENT_ID` | Yes | - | `a1b2c3...` | Service principal application ID |
| `AZURE_CLIENT_SECRET` | Yes | - | `secret123` | Service principal password |
| `AZURE_TENANT_ID` | Yes | - | `d4e5f6...` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Yes | - | `g7h8i9...` | Target subscription ID |
| `AZURE_PURGE_TTL` | No | `48h` | `72h`, `168h` | Time-to-live duration |
| `AZURE_PURGE_CREATED_TAG` | No | `createdAt` | `creationTime` | Tag name for creation timestamp |
| `AZURE_PURGE_RESOURCEGROUP_PREFIXES` | No | `` | `aro-,test-` | Comma-separated prefixes |

## Safety Checklist

Before running in production mode (`-dryRun=false`):

- [ ] Ran in dry-run mode and reviewed output
- [ ] Verified only expected resources are marked for deletion
- [ ] Confirmed critical resources have `persist=true` tag
- [ ] Checked hardcoded denylist in `hack/clean/clean.go` includes production RGs
- [ ] Have backup/restore plan if something goes wrong
- [ ] Tested with a sandbox subscription first
- [ ] Service principal has only necessary permissions
- [ ] Coordinated with team (if affecting shared resources)

## Next Steps

After successful local testing:
1. Follow the ADO setup guide in `purge-new-tenant-setup.md`
2. Run the pipeline in ADO with dry-run first
3. Monitor the scheduled runs
4. Set up alerts for pipeline failures
