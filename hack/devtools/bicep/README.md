# Bicep-Based Shared Environment Deployment

This directory contains Bicep infrastructure-as-code for automating the deployment orchestration of ARO RP shared development environments.

## Overview

The shared environment deployment has been automated using Azure Bicep to replace the manual bash function orchestration previously required. This provides:

- **Declarative infrastructure**: All resources defined in `shared-env-main.bicep`
- **Dependency management**: Bicep automatically handles deployment order
- **Idempotency**: Safe to re-run deployments
- **Better error handling**: Azure deployment validation and rollback

## Files

- `shared-env-main.bicep` - Main orchestration template that deploys all infrastructure modules
- `../deploy-shared-env-bicep.sh` - Wrapper script that prepares parameters and invokes Bicep deployment
- `../deploy-shared-env.sh` - Legacy helper functions (still sourced for post-deployment steps)

## Architecture

The Bicep template orchestrates the following ARM template deployments in order:

1. **rp-development-predeploy** - Key Vaults and initial RBAC
2. **rp-development** - CosmosDB, VNet, DNS zones, NSGs
3. **rp-managed-identity** - Managed identity for AKS/Hive
4. **env-development** - Proxy VM and VPN gateway
5. **aks-development** (optional) - AKS cluster for Hive
6. **rp-development-miwi** (optional) - OIDC storage account and workload identity infrastructure

Post-deployment steps (handled by wrapper script):
- Enable static website on OIDC storage account
- Import certificates to Key Vault
- Update parent DNS zone with NS records
- Generate VPN client configuration

## Usage

### Prerequisites

1. Source your environment file:
   ```bash
   . ./env
   ```

2. Ensure required environment variables are set:
   - `LOCATION` - Azure region (e.g., eastus)
   - `RESOURCEGROUP` - Resource group name
   - `ADMIN_OBJECT_ID` - AAD object ID for Key Vault access
   - `AZURE_FP_CLIENT_ID` - First Party service principal
   - `AZURE_RP_CLIENT_ID` - RP service principal
   - `KEYVAULT_PREFIX` - Prefix for Key Vault names
   - `PARENT_DOMAIN_NAME` - Parent DNS zone
   - `DATABASE_ACCOUNT_NAME` - CosmosDB account name
   - `DOMAIN_NAME` - Full DNS domain name
   - `PROXY_HOSTNAME` - Proxy VM hostname
   - `PULL_SECRET` - ARO pull secret (JSON)
   - `OIDC_STORAGE_ACCOUNT_NAME` - Storage account for OIDC

3. Ensure required secret files exist:
   - `secrets/proxy.crt`
   - `secrets/proxy-client.crt`
   - `secrets/proxy.key`
   - `secrets/proxy_id_rsa.pub`
   - `secrets/vpn-ca.crt`

### Basic Deployment

```bash
./hack/devtools/deploy-shared-env-bicep.sh
```

This will:
1. Validate prerequisites
2. Create the resource group (if needed)
3. Deploy all infrastructure using Bicep
4. Configure OIDC storage account
5. Import certificates to Key Vault
6. Update DNS zone
7. Generate VPN configuration

Deployment typically takes 30-45 minutes.

### Options

**Skip AKS deployment:**
```bash
./hack/devtools/deploy-shared-env-bicep.sh --skip-aks
```

**Skip MIWI infrastructure:**
```bash
./hack/devtools/deploy-shared-env-bicep.sh --skip-miwi
```

**Use Basic SKU for Public IP** (workaround for VPN gateway issues):
```bash
./hack/devtools/deploy-shared-env-bicep.sh --use-basic-ip
```

**Skip post-deployment steps:**
```bash
./hack/devtools/deploy-shared-env-bicep.sh --skip-post-deployment
```

Combine multiple options:
```bash
./hack/devtools/deploy-shared-env-bicep.sh --skip-aks --use-basic-ip
```

### After Deployment

1. Get the AKS kubeconfig (if AKS was deployed):
   ```bash
   make aks.kubeconfig
   mv aks.kubeconfig secrets/
   make secrets-update
   ```

2. Install Hive on AKS (if needed):
   See [docs/hive.md](../../../docs/hive.md)

3. Connect to VPN:
   ```bash
   sudo openvpn secrets/vpn-$LOCATION.ovpn
   ```

## Comparison with Legacy Approach

### Legacy (Manual)

Required calling bash functions in specific order:
```bash
. ./hack/devtools/deploy-shared-env.sh
create_infra_rg
deploy_rp_dev_predeploy
deploy_rp_dev
deploy_rp_managed_identity
deploy_env_dev
deploy_aks_dev
deploy_miwi_infra_dev
import_certs_secrets
update_parent_domain_dns_zone
vpn_configuration
```

### Bicep (Automated)

Single command:
```bash
./hack/devtools/deploy-shared-env-bicep.sh
```

Benefits:
- **Less error-prone**: No risk of calling functions out of order
- **Rollback support**: Azure tracks deployment state
- **Parallel execution**: Bicep deploys independent resources concurrently
- **Validation**: Pre-flight checks catch issues before deployment
- **Repeatable**: Idempotent deployments

## Troubleshooting

### VirtualNetworkGatewayCannotUseStandardPublicIP

Use the `--use-basic-ip` flag:
```bash
./hack/devtools/deploy-shared-env-bicep.sh --use-basic-ip
```

### Missing Environment Variables

The script validates all required variables before deployment. If validation fails, check:
1. Your `env` file is sourced: `. ./env`
2. All required variables are set in your environment
3. The `secrets/` directory contains required certificate files

### Deployment Failures

Check the deployment in Azure Portal:
```bash
az deployment group show -g $RESOURCEGROUP -n shared-env-<timestamp>
```

View detailed error messages:
```bash
az deployment group show -g $RESOURCEGROUP -n shared-env-<timestamp> --query properties.error
```

## Development

### Linting

Lint the Bicep file:
```bash
az bicep lint --file hack/devtools/bicep/shared-env-main.bicep
```

### What-If Analysis

Preview changes before deployment:
```bash
az deployment group what-if \
  -g $RESOURCEGROUP \
  -f hack/devtools/bicep/shared-env-main.bicep \
  --parameters @parameters.json
```

## Related Documentation

- [Prepare a shared RP development environment](../../../docs/prepare-a-shared-rp-development-environment.md)
- [Hive installation](../../../docs/hive.md)
- [Legacy deployment helper](../deploy-shared-env.sh)
