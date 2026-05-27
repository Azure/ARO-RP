#!/bin/bash -e
######## Bicep-based deployment for shared RP development environment ########
# This script automates the deployment orchestration previously done manually
# via sourcing deploy-shared-env.sh and calling individual functions.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BICEP_DIR="$SCRIPT_DIR/bicep"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
USE_BASIC_IP=false
SKIP_AKS=false
SKIP_MIWI=false
SKIP_POST_DEPLOYMENT=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --use-basic-ip)
      USE_BASIC_IP=true
      shift
      ;;
    --skip-aks)
      SKIP_AKS=true
      shift
      ;;
    --skip-miwi)
      SKIP_MIWI=false
      shift
      ;;
    --skip-post-deployment)
      SKIP_POST_DEPLOYMENT=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --use-basic-ip              Use Basic SKU for Public IP (workaround for VPN issues)"
      echo "  --skip-aks                  Skip AKS deployment"
      echo "  --skip-miwi                 Skip MIWI infrastructure deployment"
      echo "  --skip-post-deployment      Skip post-deployment steps (certs, DNS, VPN config)"
      echo "  -h, --help                  Show this help message"
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      echo "Run '$0 --help' for usage information"
      exit 1
      ;;
  esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validation function
validate_prerequisites() {
    log_info "Validating prerequisites..."

    local errors=0

    # Check required environment variables
    local required_vars=(
        "LOCATION"
        "RESOURCEGROUP"
        "ADMIN_OBJECT_ID"
        "AZURE_FP_CLIENT_ID"
        "AZURE_RP_CLIENT_ID"
        "KEYVAULT_PREFIX"
        "PARENT_DOMAIN_NAME"
        "DATABASE_ACCOUNT_NAME"
        "DOMAIN_NAME"
        "PROXY_HOSTNAME"
        "PULL_SECRET"
        "OIDC_STORAGE_ACCOUNT_NAME"
    )

    for var in "${required_vars[@]}"; do
        if [ -z "${!var:-}" ]; then
            log_error "Required environment variable $var is not set"
            errors=$((errors + 1))
        fi
    done

    # Check required secret files
    local required_files=(
        "secrets/proxy.crt"
        "secrets/proxy-client.crt"
        "secrets/proxy.key"
        "secrets/proxy_id_rsa.pub"
        "secrets/vpn-ca.crt"
    )

    for file in "${required_files[@]}"; do
        if [ ! -f "$REPO_ROOT/$file" ]; then
            log_error "Required file $file not found"
            errors=$((errors + 1))
        fi
    done

    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        log_error "Azure CLI (az) not found. Please install it first."
        errors=$((errors + 1))
    fi

    # Check logged in to Azure
    if ! az account show &> /dev/null; then
        log_error "Not logged in to Azure. Run 'az login' first."
        errors=$((errors + 1))
    fi

    if [ $errors -gt 0 ]; then
        log_error "Found $errors prerequisite error(s). Please fix them and try again."
        log_info "Hint: Make sure you've sourced your env file with '. ./env' or 'source ./secrets/env'"
        exit 1
    fi

    log_success "Prerequisites validation passed"
}

# Display current configuration
display_config() {
    echo ""
    log_info "Deployment Configuration:"
    echo "  Resource Group:        $RESOURCEGROUP"
    echo "  Location:              $LOCATION"
    echo "  Database Account:      $DATABASE_ACCOUNT_NAME"
    echo "  Key Vault Prefix:      $KEYVAULT_PREFIX"
    echo "  Parent Domain:         $PARENT_DOMAIN_NAME"
    echo "  Domain Name:           $DOMAIN_NAME"
    echo "  Proxy Hostname:        $PROXY_HOSTNAME"
    echo "  OIDC Storage Account:  $OIDC_STORAGE_ACCOUNT_NAME"
    echo ""
    echo "  Deploy AKS:            $([ "$SKIP_AKS" = true ] && echo "No" || echo "Yes")"
    echo "  Deploy MIWI:           $([ "$SKIP_MIWI" = true ] && echo "No" || echo "Yes")"
    echo "  Use Basic Public IP:   $([ "$USE_BASIC_IP" = true ] && echo "Yes" || echo "No")"
    echo ""
}

# Create resource group
create_resource_group() {
    log_info "Creating resource group $RESOURCEGROUP in $LOCATION..."

    if az group show -n "$RESOURCEGROUP" &> /dev/null; then
        log_warning "Resource group $RESOURCEGROUP already exists, skipping creation"
    else
        az group create -g "$RESOURCEGROUP" -l "$LOCATION" --tags persist=true > /dev/null
        log_success "Resource group created"
    fi
}

# Deploy infrastructure using Bicep
deploy_infrastructure() {
    log_info "Deploying infrastructure using Bicep orchestration..."

    # Resolve service principal IDs
    local FP_SP_ID=$(az ad sp list --filter "appId eq '$AZURE_FP_CLIENT_ID'" --query '[].id' -o tsv)
    local RP_SP_ID=$(az ad sp list --filter "appId eq '$AZURE_RP_CLIENT_ID'" --query '[].id' -o tsv)
    local DEVOPS_SP_ID=""

    if [ -n "${AZURE_DEVOPS_ID:-}" ]; then
        DEVOPS_SP_ID=$(az ad sp list --filter "appId eq '$AZURE_DEVOPS_ID'" --query '[].id' -o tsv)
    fi

    # Extract proxy domain label
    local PROXY_DOMAIN_LABEL=$(cut -d. -f2 <<<"$PROXY_HOSTNAME")

    # Extract proxy image auth from pull secret
    local PROXY_IMAGE_AUTH=$(jq -r '.auths["arointsvc.azurecr.io"].auth' <<<"$PULL_SECRET")

    # Prepare base64 encoded secrets
    # Note: Use -w0 for Linux, -b0 for macOS
    local BASE64_FLAG="-w0"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        BASE64_FLAG="-b0"
    fi

    local PROXY_CERT=$(base64 $BASE64_FLAG < "$REPO_ROOT/secrets/proxy.crt")
    local PROXY_CLIENT_CERT=$(base64 $BASE64_FLAG < "$REPO_ROOT/secrets/proxy-client.crt")
    local PROXY_KEY=$(base64 $BASE64_FLAG < "$REPO_ROOT/secrets/proxy.key")
    local VPN_CA_CERT=$(base64 $BASE64_FLAG < "$REPO_ROOT/secrets/vpn-ca.crt")
    local SSH_PUBLIC_KEY=$(<"$REPO_ROOT/secrets/proxy_id_rsa.pub")

    # Build parameters
    local DEPLOY_AKS=$([ "$SKIP_AKS" = true ] && echo "false" || echo "true")
    local DEPLOY_MIWI=$([ "$SKIP_MIWI" = true ] && echo "false" || echo "true")
    local USE_BASIC=$([ "$USE_BASIC_IP" = true ] && echo "true" || echo "false")

    log_info "Starting Bicep deployment (this may take 30-45 minutes)..."

    az deployment group create \
        -g "$RESOURCEGROUP" \
        -n "shared-env-$(date +%Y%m%d-%H%M%S)" \
        -f "$BICEP_DIR/shared-env-main.bicep" \
        --parameters \
            adminObjectId="$ADMIN_OBJECT_ID" \
            fpServicePrincipalId="$FP_SP_ID" \
            rpServicePrincipalId="$RP_SP_ID" \
            keyvaultPrefix="$KEYVAULT_PREFIX" \
            clusterParentDomainName="$PARENT_DOMAIN_NAME" \
            databaseAccountName="$DATABASE_ACCOUNT_NAME" \
            domainName="$DOMAIN_NAME" \
            oidcStorageAccountName="$OIDC_STORAGE_ACCOUNT_NAME" \
            proxyDomainNameLabel="$PROXY_DOMAIN_LABEL" \
            sshPublicKey="$SSH_PUBLIC_KEY" \
            proxyCert="$PROXY_CERT" \
            proxyClientCert="$PROXY_CLIENT_CERT" \
            proxyKey="$PROXY_KEY" \
            vpnCACertificate="$VPN_CA_CERT" \
            proxyImageAuth="$PROXY_IMAGE_AUTH" \
            globalDevopsServicePrincipalId="$DEVOPS_SP_ID" \
            deployAks="$DEPLOY_AKS" \
            deployMiwi="$DEPLOY_MIWI" \
            useBasicPublicIp="$USE_BASIC"

    log_success "Infrastructure deployment completed"
}

# Post-deployment steps
run_post_deployment() {
    if [ "$SKIP_POST_DEPLOYMENT" = true ]; then
        log_warning "Skipping post-deployment steps as requested"
        return
    fi

    log_info "Running post-deployment configuration steps..."

    # Source the original script to get helper functions
    source "$SCRIPT_DIR/deploy-shared-env.sh"

    # Enable static website on OIDC storage account
    if [ "$SKIP_MIWI" = false ]; then
        log_info "Enabling static website for OIDC storage account..."
        az storage blob service-properties update \
            --static-website true \
            --account-name "${OIDC_STORAGE_ACCOUNT_NAME}" \
            --auth-mode login > /dev/null
        log_success "Static website enabled"
    fi

    # Import certificates and secrets to Key Vault
    log_info "Importing certificates and secrets to Key Vault..."
    import_certs_secrets
    log_success "Certificates and secrets imported"

    # Update parent domain DNS zone
    log_info "Updating parent domain DNS zone..."
    update_parent_domain_dns_zone
    log_success "DNS zone updated"

    # Configure VPN
    log_info "Configuring VPN client..."
    vpn_configuration
    log_success "VPN configuration completed"
}

# Display next steps
display_next_steps() {
    echo ""
    log_success "============================================"
    log_success "Shared environment deployment completed!"
    log_success "============================================"
    echo ""
    log_info "Next steps:"
    echo "  1. Get the AKS kubeconfig (if AKS was deployed):"
    echo "     make aks.kubeconfig"
    echo "     mv aks.kubeconfig secrets/"
    echo "     make secrets-update"
    echo ""
    echo "  2. Install Hive on AKS (if needed):"
    echo "     See: docs/hive.md"
    echo ""
    echo "  3. Upload secrets to storage account:"
    echo "     make secrets-update"
    echo ""
    echo "  4. Connect to VPN:"
    echo "     sudo openvpn secrets/vpn-$LOCATION.ovpn"
    echo ""

    if [ "$SKIP_POST_DEPLOYMENT" = true ]; then
        log_warning "Post-deployment steps were skipped. You will need to run them manually:"
        echo "  source ./hack/devtools/deploy-shared-env.sh"
        echo "  import_certs_secrets"
        echo "  update_parent_domain_dns_zone"
        echo "  vpn_configuration"
        echo ""
    fi
}

# Main execution
main() {
    echo ""
    log_info "============================================"
    log_info "ARO-RP Shared Environment Deployment (Bicep)"
    log_info "============================================"
    echo ""

    validate_prerequisites
    display_config
    create_resource_group
    deploy_infrastructure
    run_post_deployment
    display_next_steps
}

# Run main function
main
