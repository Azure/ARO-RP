# Purge New Tenant Pipeline Setup Guide

This document describes how to set up the purge pipeline for the new tenant subscription in Azure DevOps.

## Overview

The `purge-new-tenant.yml` pipeline removes old resources from the new tenant subscription on a scheduled basis. It runs daily at 2 AM UTC and purges resources older than a configurable TTL (time-to-live) period.

## Prerequisites

- Access to the new tenant Azure subscription
- Access to Azure DevOps with permissions to create pipelines and variable groups
- Access to Azure Key Vault for storing credentials
- Permissions to create service principals in the new tenant

## Setup Steps

### 1. Create Service Principal

Create a service principal for the purge pipeline to authenticate with Azure:

```bash
az login --tenant <NEW_TENANT_ID>

# Create the service principal
az ad sp create-for-rbac \
  --name "aro-new-tenant-purge-sp" \
  --role Contributor \
  --scopes /subscriptions/<NEW_TENANT_SUBSCRIPTION_ID>
```

Save the output containing:
- `appId` (will be used as `clientId`)
- `password` (will be used as `clientSecret`)
- `tenant` (tenant ID)

### 2. Assign Contributor Role

If not already assigned in step 1, ensure the service principal has Contributor role:

```bash
az role assignment create \
  --assignee <SERVICE_PRINCIPAL_APP_ID> \
  --role Contributor \
  --scope /subscriptions/<NEW_TENANT_SUBSCRIPTION_ID>
```

### 3. Grant Microsoft Graph API Permissions

The purge pipeline cleans up orphaned service principals created during e2e tests. To enable this functionality, grant Microsoft Graph API permissions to the service principal:

**Required permissions:**
- `Application.Read.All` - Required to list applications and service principals
- `Application.ReadWrite.All` - Required to delete orphaned applications and service principals

**To grant permissions via Azure Portal:**

1. Navigate to: **Azure Active Directory** > **App registrations**
2. Find the app registration for `aro-new-tenant-purge-sp` (you may need to switch to "All applications" tab)
3. Click on **API permissions** in the left sidebar
4. Click **Add a permission** > **Microsoft Graph** > **Application permissions**
5. Search for and add: `Application.ReadWrite.All`
6. Click **Add permissions**
7. Click **Grant admin consent for [Your Tenant]** (requires admin privileges)

**To grant permissions via Azure CLI:**

```bash
# Get the service principal object ID
SP_OBJECT_ID=$(az ad sp list --display-name "aro-new-tenant-purge-sp" --query '[0].id' -o tsv)

# Get Microsoft Graph service principal ID (this is a well-known constant)
GRAPH_SP_ID=$(az ad sp list --display-name "Microsoft Graph" --query '[0].id' -o tsv)

# Application.ReadWrite.All app role ID (well-known constant)
APP_ROLE_ID="1bfefb4e-e0b5-418b-a88f-73c46d2cc8e9"

# Grant the permission
az rest --method POST \
  --uri "https://graph.microsoft.com/v1.0/servicePrincipals/$SP_OBJECT_ID/appRoleAssignments" \
  --headers "Content-Type=application/json" \
  --body "{\"principalId\":\"$SP_OBJECT_ID\",\"resourceId\":\"$GRAPH_SP_ID\",\"appRoleId\":\"$APP_ROLE_ID\"}"
```

**Note:** Without these Graph API permissions, the pipeline will still clean up resource groups successfully, but it will not be able to remove orphaned service principals. The pipeline logs will show errors when attempting service principal cleanup if permissions are missing.

### 4. Create Client Secret Credentials JSON

Create a JSON file with the service principal credentials:

```json
{
  "clientId": "<APP_ID_FROM_STEP_1>",
  "clientSecret": "<PASSWORD_FROM_STEP_1>",
  "tenantId": "<TENANT_ID>",
  "subscriptionId": "<NEW_TENANT_SUBSCRIPTION_ID>"
}
```

Base64 encode this JSON:

```bash
cat credentials.json | base64 -w 0
```

### 5. Store Credentials in Key Vault

Store the base64-encoded credentials in Azure Key Vault:

```bash
az keyvault secret set \
  --vault-name <YOUR_KEYVAULT_NAME> \
  --name aro-new-tenant-purge-spn \
  --value "<BASE64_ENCODED_CREDENTIALS>"
```

### 6. Create Variable Group in Azure DevOps

1. Navigate to Azure DevOps → Pipelines → Library
2. Click "+ Variable group"
3. Name it: `aro-new-tenant-purge`
4. Add the following variables:

| Variable Name | Value | Secret? | Description |
|--------------|-------|---------|-------------|
| `aro-new-tenant-purge-spn` | Link to Key Vault secret | Yes | Base64-encoded service principal credentials |
| `newTenantSubscriptionId` | `<SUBSCRIPTION_ID>` | No | Azure subscription ID for new tenant |
| `newTenantPurgeCreatedTag` | `createdAt` | No | Tag name used to identify resource creation time |
| `newTenantResourceGroupDeletePrefixes` | `aro-,test-,dev-` | No | Comma-separated prefixes of resource groups to consider for deletion |
| `newTenantPurgeTTL` | `48h` | No | Time-to-live duration (e.g., 48h, 72h, 168h) |

**Note:** For the `aro-new-tenant-purge-spn` variable:
- Click "Add" → Select "Link secrets from an Azure key vault as variables"
- Select your Key Vault and choose the `aro-new-tenant-purge-spn` secret

### 7. Create Pipeline in Azure DevOps

1. Navigate to Azure DevOps → Pipelines → Pipelines
2. Click "New pipeline"
3. Select "Azure Repos Git" (or your repository source)
4. Select the ARO-RP repository
5. Select "Existing Azure Pipelines YAML file"
6. Choose the path: `/.pipelines/purge-new-tenant.yml`
7. Click "Continue"
8. Review the pipeline and click "Save"

### 8. Link Variable Group to Pipeline

1. Edit the newly created pipeline
2. Click on the three dots (•••) → "Triggers"
3. Go to "Variables" tab
4. Click "Variable groups"
5. Link the `aro-new-tenant-purge` variable group
6. Save

### 9. Configure Scheduled Trigger

The pipeline is already configured with a cron schedule in the YAML file:
- Runs daily at 2 AM UTC
- Runs on the `master` branch
- `always: true` ensures it runs even if there are no code changes

To modify the schedule, edit the `schedules` section in `purge-new-tenant.yml`.

### 10. Set Pipeline Permissions

Ensure the pipeline has the necessary permissions:

1. Go to pipeline → Edit → More actions → Security
2. Verify the pipeline has access to:
   - The repository
   - The variable group `aro-new-tenant-purge`
   - The service connection to the container registry (`arointsvc`)

## Testing the Pipeline

### Dry Run Test

Before running the pipeline in production mode, test with dry run enabled:

1. Go to the pipeline in Azure DevOps
2. Click "Run pipeline"
3. Set `dryRun` parameter to `true`
4. Click "Run"
5. Review the logs to see what resources would be deleted without actually deleting them

### Production Run

Once you've verified the dry run results:

1. Click "Run pipeline"
2. Set `dryRun` parameter to `false`
3. Click "Run"
4. Monitor the execution and verify resources are cleaned up as expected

## Pipeline Behavior

### Resource Selection Criteria

The pipeline will delete a resource group if ALL of the following conditions are met:

1. **Prefix Match** (if configured): Resource group name starts with one of the prefixes in `newTenantResourceGroupDeletePrefixes`
2. **No Persist Tag**: Resource group does NOT have a `persist=true` tag
3. **Has Creation Tag**: Resource group has the `createdAt` tag (or the tag specified in `newTenantPurgeCreatedTag`)
4. **TTL Exceeded**: Time since the `createdAt` timestamp exceeds the `newTenantPurgeTTL` duration
5. **Not Denylisted**: Resource group is not in the hardcoded denylist in `hack/clean/clean.go`

### Protected Resource Groups

The following resource groups are protected and will never be deleted (hardcoded in `hack/clean/clean.go`):
- `v4-eastus`, `v4-australiasoutheast`, `v4-westeurope`
- `v4-eastus-aks1`, `v4-australiasoutheast-aks1`, `v4-westeurope-aks1`
- `management-westeurope`, `management-eastus`, `management-australiasoutheast`
- `images`, `secrets`, `dns`

### Tagging Resources for Purge Control

To control purge behavior, tag your resource groups:

**To allow purging** (after TTL expires):
```bash
az group update --name <RG_NAME> --tags createdAt=$(date -u +"%Y-%m-%dT%H:%M:%S.%NZ")
```

**To prevent purging**:
```bash
az group update --name <RG_NAME> --tags persist=true
```

## Monitoring and Troubleshooting

### Viewing Pipeline Runs

- Navigate to Pipelines → Select the purge pipeline → Runs
- Review logs for each run to see:
  - Which resource groups were evaluated
  - Which resource groups were deleted (or would be deleted in dry run mode)
  - Any errors encountered

### Common Issues

**Issue: Service principal authentication fails**
- Verify the credentials in Key Vault are correct and base64-encoded properly
- Ensure the service principal has not expired
- Check that the Contributor role assignment is still active

**Issue: No resources are being deleted**
- Verify resources have the correct `createdAt` tag
- Check that the TTL has actually expired
- Ensure resources don't have `persist=true` tag
- Verify resource group name matches the configured prefixes

**Issue: Pipeline doesn't run on schedule**
- Check the pipeline has been saved with the schedule configuration
- Verify the branch specified in the schedule exists
- Ensure the repository has not disabled scheduled runs

## Environment Variables Reference

The pipeline passes the following environment variables to the `hack/clean` tool:

- `AZURE_CLIENT_ID`: From service principal credentials
- `AZURE_CLIENT_SECRET`: From service principal credentials
- `AZURE_TENANT_ID`: From service principal credentials
- `AZURE_SUBSCRIPTION_ID`: From `newTenantSubscriptionId` variable
- `AZURE_PURGE_TTL`: From `newTenantPurgeTTL` variable
- `AZURE_PURGE_CREATED_TAG`: From `newTenantPurgeCreatedTag` variable
- `AZURE_PURGE_RESOURCEGROUP_PREFIXES`: From `newTenantResourceGroupDeletePrefixes` variable

## Security Considerations

1. **Credentials**: Store all credentials in Azure Key Vault, never commit them to code
2. **Least Privilege**: Consider using a more restricted role than Contributor if possible
3. **Audit Logs**: Monitor Azure Activity Logs for resource deletions
4. **Dry Run First**: Always test with dry run before production runs
5. **Review Regularly**: Periodically review what's being deleted to ensure expected behavior

## Maintenance

### Updating TTL

To change the purge TTL:
1. Edit the `newTenantPurgeTTL` variable in the `aro-new-tenant-purge` variable group
2. Valid formats: `48h`, `72h`, `168h`, etc. Day format (`7d`) is NOT supported.

### Updating Resource Group Prefixes

To change which resource groups are considered:
1. Edit the `newTenantResourceGroupDeletePrefixes` variable
2. Use comma-separated values: `prefix1,prefix2,prefix3`

### Rotating Service Principal Credentials

1. Create new client secret in Azure AD
2. Update credentials JSON and base64 encode
3. Update the Key Vault secret `aro-new-tenant-purge-spn`
4. Pipeline will use new credentials on next run

## Support

For issues or questions about the purge pipeline:
- Review pipeline logs in Azure DevOps
- Check the implementation in `hack/clean/clean.go`
- Consult the team's Slack channel or create a Jira ticket
