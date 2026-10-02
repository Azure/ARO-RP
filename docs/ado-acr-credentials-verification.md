# Azure DevOps ACR Credentials Verification Checklist

## Purpose

This checklist verifies that no long-lived ACR read/write credentials are stored in Azure DevOps, in compliance with ARO-10651 acceptance criteria.

## Verification Steps

### 1. Check Variable Groups

Navigate to Azure DevOps → Pipelines → Library → Variable Groups

**Action Items**:
- [ ] Review all variable groups associated with the ARO-RP project
- [ ] Verify no variable groups contain ACR credentials for:
  - `arointsvc.azurecr.io`
  - `arosvcdev.azurecr.io`
  - Any other ACR registries in MSIT tenant
- [ ] Document any ACR-related variables found

**Expected Result**: No ACR username/password variables in any variable groups

**Variables that SHOULD exist** (these are acceptable):
- `aro-v4-e2e-devops-spn`: Service Principal JSON (short-lived, scoped)
- `SECRET_SA_ACCOUNT_NAME`: Storage account name
- `AZURE_SUBSCRIPTION_ID`: Subscription ID

**Variables that should NOT exist**:
- Any variable containing "ACR_USERNAME", "ACR_PASSWORD", "acrCredentials", etc.
- Any variable containing Docker registry credentials
- Any base64-encoded credential JSON for ACR

### 2. Check Pipeline Variables

For each pipeline in `.pipelines/`:
- [ ] ci.yml
- [ ] deploy-dev-env.yml
- [ ] clean-subscription.yml
- [ ] rp-full-dev-setup.yml

**Action Items**:
- [ ] Review pipeline-specific variables in Azure DevOps UI
- [ ] Verify no ACR credentials are defined as pipeline variables
- [ ] Check pipeline YAML files for any hardcoded credentials (should be none)

**Expected Result**: No ACR credentials in pipeline variables

### 3. Check Service Connections

Navigate to Azure DevOps → Project Settings → Service Connections

**Service connections that SHOULD exist**:

#### arointsvc
- [ ] Type: Docker Registry
- [ ] Registry: arointsvc.azurecr.io
- [ ] Authentication: Service Principal or Managed Identity (NOT username/password)
- [ ] Usage: Read-only access for pulling base images
- [ ] Used in: ci.yml, clean-subscription.yml, rp-full-dev-setup.yml

#### ado-pipeline-dev-image-push
- [ ] Type: Azure Resource Manager
- [ ] Authentication: Service Principal
- [ ] Purpose: Cross-tenant authentication for arosvcdev.azurecr.io
- [ ] Usage: Push/pull access for CI-built images
- [ ] Used in: template-acr-login.yml, template-acr-push.yml

**Service connections that should NOT exist**:
- [ ] No service connections using username/password for ACR
- [ ] No service connections with "arointsvc" credentials stored as password

**Action Items**:
- [ ] Verify both service connections exist
- [ ] Verify authentication method is Service Principal, not credentials
- [ ] Verify no deprecated service connections exist for ACR

### 4. Check Secrets

Navigate to Azure DevOps → Pipelines → Library → Secure files

**Action Items**:
- [ ] Verify no secure files contain ACR credentials
- [ ] Verify no files named like "acr-credentials.json" or similar

**Expected Result**: No ACR credential files

### 5. Check Pipeline Runs

Review recent pipeline run logs:

**Action Items**:
- [ ] Check for any usage of deprecated `acrCredentialsJSON` parameter
- [ ] Verify all ACR logins use service connections or Azure CLI
- [ ] Look for any "WARNING: Using username/password" messages

**Expected Result**: All ACR authentication via service connections or `az acr login`

### 6. Check Environment-Specific Settings

If using Azure DevOps Environments:

Navigate to Azure DevOps → Pipelines → Environments

**Action Items**:
- [ ] Review each environment's variables and secrets
- [ ] Verify no ACR credentials stored in environment variables

**Expected Result**: No ACR credentials in environment variables

## Compliance Verification

After completing all steps above, verify the following ARO-10651 acceptance criteria:

### ✅ Acceptance Criteria 1
**There are no rw credentials allowing to push to INT or PROD ACRs stored in the E2E tenant**

Verification:
- [ ] No Azure Key Vaults in E2E tenant contain ACR credentials
- [ ] No storage accounts in E2E tenant contain ACR credential files
- [ ] No Azure DevOps variable groups contain ACR credentials

### ✅ Acceptance Criteria 2
**There are no rw credentials allowing to push to INT or PROD ACRs stored in Variables in ADO**

Verification:
- [ ] No pipeline variables contain ACR credentials
- [ ] No variable groups contain ACR credentials
- [ ] No secure files contain ACR credentials
- [ ] Service connections use Service Principal, not passwords

### ✅ Acceptance Criteria 3
**PR E2E deploys RP/gwy using the git ref of the PR and creates clusters running the ARO operator built from that same ref**

Verification:
- [ ] Pipeline builds images with tags: `pr-$(PullRequestId)-$(SourceCommitId)`
- [ ] Images are pushed to `arosvcdev.azurecr.io`
- [ ] E2E tests use these tagged images
- [ ] No hardcoded image versions in E2E tests

## Remediation Steps (if credentials found)

If any ACR credentials are found during verification:

1. **Document the finding**:
   - Variable/file name
   - Location (variable group, pipeline, etc.)
   - Credential type (username/password, token, etc.)
   - Last modified date and by whom

2. **Verify if still in use**:
   - Check recent pipeline runs
   - Search pipeline YAML files for references
   - Confirm with team if intentionally removed or orphaned

3. **Remove the credential**:
   - Delete from Azure DevOps
   - Rotate the credential if it was actively used
   - Update any documentation

4. **Verify removal**:
   - Re-run verification steps
   - Test pipeline runs to ensure no breakage

## References

- ARO-10651: ACR for pushing/pulling images
- ARO-9094: Create new E2E specific ACR (Epic)
- [ACR Authentication Documentation](./acr-authentication.md)
