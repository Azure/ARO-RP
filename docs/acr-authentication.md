# ACR Authentication in CI/E2E Pipelines

## Overview

This document describes how the ARO-RP CI/E2E pipelines authenticate to Azure Container Registries (ACR) without storing long-lived read/write credentials, in compliance with security requirements outlined in ARO-10651 and ARO-9094.

## Background

Previously, PR E2E and release E2E environments depended on `arointsvc.azurecr.io` (MSIT tenant) and required long-lived ACR credentials for cross-tenant access. To comply with Microsoft security requirements of "no passwords anywhere," the team migrated to a new architecture using:

1. **arosvcdev.azurecr.io** - ACR in the E2E tenant for pushing/pulling CI-built images
2. **Service connections with Managed Identities** - Cross-tenant authentication without passwords
3. **Azure CLI authentication via Service Principals** - For E2E test scenarios

## ACR Registry Usage

### arointsvc.azurecr.io (MSIT Tenant)

**Purpose**: Pulling read-only base images (OpenShift release images, MSFT-specific images like autorest)

**Authentication**: 
- Service connection: `arointsvc` (configured in Azure DevOps)
- Uses Azure DevOps `Docker@2` task
- Read-only access, no credentials stored in pipeline

**Usage locations**:
- `.pipelines/ci.yml` - All containerized CI jobs
- `.pipelines/clean-subscription.yml`
- `.pipelines/rp-full-dev-setup.yml`

**Example**:
```yaml
- task: Docker@2
  inputs:
    command: "login"
    containerRegistry: arointsvc  # Service connection name
```

### arosvcdev.azurecr.io (E2E Tenant)

**Purpose**: Pushing and pulling CI-built images (aro, e2e, azext-aro, vpn)

**Authentication Methods**:

#### 1. For Build & Push Operations (Containerized CI Stage)

Uses service connection `ado-pipeline-dev-image-push` via templates:

- **Login**: `.pipelines/templates/template-acr-login.yml`
- **Push**: `.pipelines/templates/template-acr-push.yml`

Both templates use `AzureCLI@2` task with cross-tenant authentication. No passwords stored.

**Example**:
```yaml
- template: ./templates/template-acr-login.yml
  parameters:
    acrFQDN: "arosvcdev.azurecr.io"

- template: ./templates/template-acr-push.yml
  parameters:
    acrFQDN: "arosvcdev.azurecr.io"
    repository: "aro"
    tag: $(TAG)
    pushLatest: true
```

#### 2. For E2E Test Operations

Uses Azure CLI authenticated via Service Principal:

1. Authenticate Azure CLI using `.pipelines/templates/template-az-cli-login.yml`
2. Login to ACR using `az acr login --name arosvcdev`

**Example**:
```yaml
- template: ./templates/template-az-cli-login.yml
  parameters:
    azureDevOpsJSONSPN: $(aro-v4-e2e-devops-spn)

- bash: |
    az acr login --name arosvcdev
    # ... E2E operations
```

## Service Connections

### arointsvc

- **Type**: Azure Container Registry
- **Purpose**: Read-only access to MSIT tenant ACR
- **Configured in**: Azure DevOps project settings

### ado-pipeline-dev-image-push

- **Type**: Azure Service Connection
- **Purpose**: Cross-tenant authentication for pushing to arosvcdev.azurecr.io
- **Configured in**: Azure DevOps project settings
- **Note**: Cannot use simpler Docker@2 push because MSI does not support cross-tenant authentication

## Security Compliance

### ARO-10651 Acceptance Criteria

✅ **No rw credentials for INT or PROD ACRs stored in E2E tenant**
- All authentication uses Service Connections or Service Principal sessions
- No credentials stored in code or configuration files

✅ **No rw credentials for INT or PROD ACRs stored in Variables in ADO**
- Service connections are configured in Azure DevOps project settings
- Service Principal credentials are managed as pipeline variables (aro-v4-e2e-devops-spn) but are short-lived and scoped

✅ **PR E2E deploys RP/gwy using git ref of PR and creates clusters running ARO operator built from that ref**
- Pipeline builds images tagged with PR/commit SHA: `pr-$(PullRequestId)-$(SourceCommitId)` or `master-$(SourceVersion)`
- Images pushed to arosvcdev.azurecr.io with proper tags
- E2E tests use these tagged images

## Variables Used

### Build Variables
- `RP_IMAGE_ACR`: Set to `arointsvc` (for backwards compatibility with Makefile)
- `REGISTRY`: Set to `arointsvc.azurecr.io`
- `BUILDER_REGISTRY`: Set to `arointsvc.azurecr.io`
- `LOCAL_ARO_RP_IMAGE`: Set to `arosvcdev.azurecr.io/aro`
- `LOCAL_ARO_AZEXT_IMAGE`: Set to `arosvcdev.azurecr.io/azext-aro`
- `LOCAL_VPN_IMAGE`: Set to `arosvcdev.azurecr.io/vpn`
- `LOCAL_E2E_IMAGE`: Set to `arosvcdev.azurecr.io/e2e`
- `ARO_IMAGE`: Set to `arosvcdev.azurecr.io/aro:$(TAG)`
- `TAG`: Dynamic tag based on PR or commit

### Secret Variables
- `aro-v4-e2e-devops-spn`: Service Principal JSON for Azure CLI authentication (managed as pipeline variable)
- `SECRET_SA_ACCOUNT_NAME`: Storage account name for secrets
- `AZURE_SUBSCRIPTION_ID`: Target subscription for E2E

## Pipeline Flow

### 1. Set Tag Stage
Determines image tag based on build type (PR vs master)

### 2. Containerized CI Stage
Three parallel jobs:
- **Build and Push Az ARO Extension**: Builds and pushes azext-aro image
- **Build and Test RP and Portal**: Builds and pushes aro image
- **Build and Push E2E Image**: Builds and pushes e2e image

All jobs:
1. Login to arointsvc (Docker@2 with service connection)
2. Login to arosvcdev (template-acr-login.yml)
3. Build images
4. Push to arosvcdev (template-acr-push.yml)

### 3. E2E Stage
Two parallel jobs (CSP and MIWI):
1. Install Docker and Docker Compose
2. Login to arointsvc (Docker@2)
3. Login to Azure CLI (Service Principal)
4. Login to arosvcdev (`az acr login`)
5. Fetch secrets from storage account
6. Get AKS kubeconfig
7. Run E2E tests using docker compose
8. Cleanup

## Migration from Old Method

### Deprecated Template

`.pipelines/templates/template-push-images-to-acr.yml` - This template supports both:
- Modern method: `az acr login` (when acrCredentialsJSON is empty)
- Legacy method: Username/password login (when acrCredentialsJSON is provided)

**Status**: Not currently used in any active pipeline. Kept for reference but can be removed.

## Troubleshooting

### ACR Login Failures

If you encounter ACR login failures:

1. **Check service connection**: Ensure `arointsvc` and `ado-pipeline-dev-image-push` service connections are properly configured in Azure DevOps
2. **Check Azure CLI authentication**: For E2E jobs, verify that `template-az-cli-login.yml` succeeds before `az acr login`
3. **Check token expiration**: Service Principal tokens may expire; verify `aro-v4-e2e-devops-spn` is current

### Cross-tenant Authentication Issues

The current implementation uses `AzureCLI@2` task because MSI does not support cross-tenant authentication. If you encounter cross-tenant errors:

1. Verify the service connection uses Service Principal, not Managed Identity
2. Ensure proper permissions are granted across tenants

## References

- **Jira Tickets**:
  - ARO-10651: ACR for pushing/pulling images
  - ARO-9094: Create new E2E specific ACR (Epic - Closed)
  - ARO-8265: Migrate PR E2E out of MSIT
  
- **Related PRs**:
  - Initial arosvcdev migration work (see git history for "arosvcdev" commits)
  - ACR credential leak fix: ARO-24815

- **Azure DevOps**:
  - Pipeline: `.pipelines/ci.yml`
  - Service connections configured in project settings

## SME

- **Primary**: Brendan Bergen
- **Related work**:
  - Brian Ragazzi (current assignee of ARO-10651)
  - Shubhada Sanjay Paithankar (current assignee of parent epic ARO-10296)
