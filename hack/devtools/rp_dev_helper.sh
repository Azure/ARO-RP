#!/bin/bash -e
######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########

secret_files=("dev-ca.crt" "dev-client.crt" \
        "portal-client.pem" "firstparty.pem" "localhost.pem" "arm.pem" \
        "cluster-mdsd-self-signed.pem" "gwy-mdm-self-signed.pem" "gwy-mdsd-self-signed.pem" "rp-mdm-self-signed.pem" \
        "rp-mdsd-self-signed.pem" "full_rp_id_rsa" "full_rp_id_rsa.pub" "env" )

verify_downloading_secrets() {
    # Define the expected directory and file names
    expected_dir="secrets"
    all_secret_files=("firstparty.key" "vpn-client.key" "vpn-eastus.ovpn" "localhost.crt" "proxy.crt" \
        "dev-ca.crt" "proxy-client.pem" "vpn-aks-eastus.ovpn" "portal-client.key" "portal-client.pem" \
        "proxy-client.key" "vpn-ca-cert.pem" "rp-metrics-int.pem" "dev-client.crt" "rp-logging-int.pem" \
        "vpn-ca.crt" "shared-cluster.kubeconfig" "vpn-client.pem" "firstparty.pem" "portal-client.crt" \
        "dev-ca.key" "vpn-aks-australiaeast.ovpn" "vpn-client-cert.pem" "vpn-client-key.pem" "dev-client.pem" \
        "proxy.key" "firstparty.crt" "vpn-client.crt" "vpn-aks-westeurope.ovpn" "localhost.key" "vpn-ca.pem" \
        "vpn-australiaeast.ovpn" "cluster-logging-int.pem" "vpn-ca.key" "localhost.pem" "proxy.pem" "dev-client.key" \
        "proxy_id_rsa.pub" "proxy_id_rsa" "vpn-ca-key.pem" "proxy-client.crt" "vpn-westeurope.ovpn" "dev-ca.pem" \
        "arm.crt" "arm.key" "arm.pem" "cluster-mdsd.crt" "cluster-mdsd.key" "cluster-mdsd.pem" "cluster-mdsd-self-signed.pem" \
        "gwy-mdm-self-signed.pem" "gwy-mdsd-self-signed.pem" "rp-mdm-self-signed.pem" "rp-mdsd-self-signed.pem" \
        "full_rp_id_rsa" "full_rp_id_rsa.pub" "env")

    # Validate that the secrets directory and required files exist
    [ -d "$expected_dir" ] || { echo "Directory '$expected_dir' has not been created."; exit 1; }
    #  TODO check if only the below files are required
    for file in "${secret_files[@]}"; do
    [ -f "$expected_dir/$file" ] || { echo "File '$file' does not exist inside the directory '$expected_dir'."; exit 1; }
    done
}

check_jq_installed() {
  if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed."
    echo "Please install jq to proceed. You can install it using the following command:"
    echo "  For Debian/Ubuntu-based systems: sudo apt-get install jq"
    echo "  For Red Hat/CentOS-based systems: sudo yum install jq"
    echo "  For macOS: brew install jq"
    return 1
  fi
  return 0
}

# Function to check deployment existance and provisioning state
check_azure_deployment() {
    # Check if exactly two non-empty arguments are provided
    if [[ $# -ne 2 ]]; then
        echo "Usage $0 <ResourceGroup> <DeploymentName>. Please try again"
        exit 1
    fi
    local resource_group=$1
    local deployment_name=$2

    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if [[ "${SKIP_DEPLOYMENTS}" == "false" ]]; then
        echo "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip deploying '${deployment_name}' in resource group '${resource_group}'"
        return 1
    fi

    # Check if the ResourceGroup exists
    resource_group_info=$(az group show --resource-group "${resource_group}" 2>/dev/null)
    if [ -z "${resource_group_info}" ]; then
        echo "Resource group '${resource_group}' does not exist."
        return 1
    fi

    # Check if the deployment exists
    deployment_info=$(az deployment group show --resource-group "${resource_group}" --name "${deployment_name}" 2>/dev/null)
    if [ -z "${deployment_info}" ]; then
        echo "Deployment '${deployment_name}' does not exist in resource group '${resource_group}'."
        return 1
    fi

    # Check if jq is installed
    if ! check_jq_installed; then
        exit 1
    fi

    # Extract the provisioning state using jq
    provisioning_state=$(echo "${deployment_info}" | jq -r '.properties.provisioningState')
    # Check if the provisioning state is 'Succeeded'
    if [[ "${provisioning_state}" == "Succeeded" ]]; then
        echo "Deployment '${deployment_name}' in resource group '${resource_group}' has succeeded."
        return 0
    else
        echo "Deployment '${deployment_name}' in resource group '${resource_group}' has not succeeded. Current state: ${provisioning_state}"
        return 1
    fi
}

# Function to extract the image tag
extract_image_tag() {
    # Check if exactly two arguments are provided
    if [[ $# -ne 2 ]]; then        
        echo "Usage $0 <FUNCTION_NAME> <FILE_TO_EXTRACT>. Please try again"
        return 1
    fi
    for arg in "$1" "$2"; do
        if  [[ -z "$arg" ]];  then
            echo "Error: an empty var. Please try again"
            return 1
        fi
    done

    local return_line=$(grep -A 2 "func $1" "$2" | grep 'return')
    echo "$return_line" | sed 's/.*"\(.*\)@sha256.*/\1/'
}

VERSION_CONST_FILE="pkg/util/version/const.go"
# Function to get image name and tag
get_digest_tag() {
    if [[ $# -ne 1 ]]; then        
        echo "Usage $0 <IMAGE_NAME>. Please try again"
        exit 1
    fi
    local IMAGE_NAME=$1
    local IMAGE_TAG=$(extract_image_tag "$IMAGE_NAME" "$VERSION_CONST_FILE")
    echo $IMAGE_TAG
}

# Function to copy image using skopeo
copy_digest_tag() {
    # Check if exactly non-empty four arguments are provided
    if [[ $# -ne 4 ]]; then        
        echo "Usage $0 <PULL_SECRET> <SRC_ACR_NAME> <DST_ACR_NAME> <IMAGE_TAG>. Please try again"
        exit 1
    fi
    for arg in "$1" "$2" "$3" "$4"; do
        if  [[ -z "$arg" ]];  then
            echo "Error: an empty var. Please try again"
            exit 1
        fi
    done
    echo "INFO: Copy image from one ACR to another ..."
    local PULL_SECRET=$1
    local SRC_ACR_NAME=$2
    local DST_ACR_NAME=$3
    local IMAGE_TAG=$4
    
    SRC_AUTH=$(echo "$PULL_SECRET" | jq -r '.auths["'$SRC_ACR_NAME'.azurecr.io"].auth' | base64 -d)
    DST_TOKEN=$(az acr login -n "$DST_ACR_NAME" --expose-token | jq -r .accessToken)
    
    skopeo copy \
        --src-creds "$SRC_AUTH" \
        --dest-creds "00000000-0000-0000-0000-000000000000:$DST_TOKEN" \
        "docker://$SRC_ACR_NAME.azurecr.io$IMAGE_TAG" \
        "docker://$DST_ACR_NAME.azurecr.io$IMAGE_TAG"
}

check_acr_repo() {
    # Check if exactly two non-empty arguments are provided
    if [[ $# -ne 2 ]]; then
        echo "Usage: $0 <ACR_Name> <Repository>"
        exit 1
    fi

    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if [[ "${SKIP_DEPLOYMENTS}" == "false" ]]; then
        echo "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip ACR repo mirroring for repository $2."
        return 1
    fi

    # Get the repository tag
    repo_tag=$(az acr repository show-tags --name "$1" --repository "$2" -o tsv | tr '\n' ' ')

    # Check if the repository tag is empty
    if [[ -n "$repo_tag" ]]; then
        echo "Repository '$2' in ACR '$1' exists with tag '${repo_tag}'."
        return 0
    else
        echo "Repository '$2' doesn't exist in ACR '$1'."
        return 1
    fi
}

# "openshift-release-dev/ocp-release" "openshift-release-dev/ocp-v4.0-art-dev" were excluded as they don't include an image/tag
acr_repos=("azure-cli" "rhel8/support-tools" "rhel9/support-tools" "openshift4/ose-tools-rhel8" "ubi8/ubi-minimal" "ubi9/ubi-minimal" \
        "ubi8/nodejs-18" "ubi8/go-toolset" "app-sre/managed-upgrade-operator" "app-sre/hive" "distroless/genevamdm" "distroless/genevamdsd" \
        "aro" "fluentbit")

check_acr_repos() {
    # Check if exactly one non-empty argument is provided
    if [[ $# -ne 1 ]]; then
        echo "Usage: $0 <ResourceGroup>. Please try again"
        exit 1
    fi

    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false"    
    if [[ "${SKIP_DEPLOYMENTS}" == "false" ]]; then
        echo "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip acr repo mirroring in ResourceGroup $1"
        return 1
    fi
   
    # Check if jq is installed
    if ! check_jq_installed; then
        exit 1
    fi
     # Get the first Azure Container Registry (ACR) name under ResourceGroup
    acr_name=$(az acr list --resource-group $1 | jq -r '.[0].name')
    if [[ -z "$acr_name" ]]; then
        echo "Error: There are no Azure Container Registries under ResourceGroup '$1'."
        return 1
    fi
    
    # Flag to track if any repos need to be imported
    missing_repos=false
    for repo in "${acr_repos[@]}"; do
        if ! check_acr_repo "$acr_name" "$repo"; then
            missing_repos=true
        fi
    done

    $missing_repos && echo -e "Some repositories are missing and need to be imported.\n" && return 1 \
    || echo -e "All repositories exist in ACR '$acr_name'.\n" && return 0
}

# Function to import a Geneva image only when it isn't exist
import_geneva_image() {
    # Check if exactly two non-empty arguments are provided
    if [[ $# -ne 2 ]]; then
        echo "Usage: $0 <Repository> <Tag>"
        exit 1
    fi

    if ! check_acr_repo $DST_ACR_NAME $1 ;then
        az acr import --name $DST_ACR_NAME.azurecr.io$2 --source linuxgeneva-microsoft.azurecr.io$2
        echo "Imported $1 to ACR $DST_ACR_NAME"
    else
        echo "⏭️📦 Skip importing $1 to ACR $DST_ACR_NAME, since it already exist. Can not run twice"
    fi
}

# Function to check certificars existance, their enablement and expiration date
check_keyvault_certificate() {
    # Check if exactly non-empty two arguments are provided
    if [[ $# -ne 2 ]]; then
        echo "Usage $0 <KeyVault> <Certificate>. Please try again"
        exit 1
    fi

    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if [[ "${SKIP_DEPLOYMENTS}" == "false" ]]; then
        echo "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip keyvault's certificate import"
        return 1
    fi

    # Check if the Key Vault exists
    if ! az keyvault show --name "$1" >/dev/null 2>&1; then
        echo "Error: Key Vault '$vault_name' does not exist."
        exit 1
    fi

    certificate_info=$(az keyvault certificate show --vault-name "$1" --name "$2" 2>/dev/null)
    if [ -z "${certificate_info}" ]; then
        echo "Certificate '$2' in Key Vault '$1' does not exist."
        return 1
    fi

    # Check if jq is installed
    if ! check_jq_installed; then
        exit 1
    fi

    if [[ "$(echo "$certificate_info" | jq -r '.attributes.enabled')" == "true" ]]; then
        echo "Certificate '$2' in Key Vault '$1' exists and is enabled with expiration date '$(echo "$certificate_info" | jq -r '.attributes.expires')'."
    else
        echo "Certificate '$2' in Key Vault '$1' exists but is not enabled."
        exit 1
    fi
}

# Function to import 10 certificates in case they are needed
check_and_import_certificates (){
    # Check if exactly one non-empty argument is provided
    if [[ -z "${KEYVAULT_PREFIX}" ]]; then
        echo "Error: KEYVAULT_PREFIX environment variable is not set."
        exit 1
    fi

    cert="rp-mdm"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}.\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/rp-mdm-self-signed.pem >/dev/null

    cert="rp-mdsd"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/rp-mdsd-self-signed.pem >/dev/null

    cert="cluster-mdsd"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/cluster-mdsd-self-signed.pem >/dev/null

    cert="dev-arm"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/arm.pem >/dev/null

    cert="rp-firstparty"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/firstparty.pem >/dev/null

    cert="rp-server"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-svc ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-svc" \
        --name ${cert} \
        --file secrets/localhost.pem >/dev/null
 
    cert="gwy-mdm"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-gwy ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-gwy" \
        --name ${cert} \
        --file secrets/gwy-mdm-self-signed.pem >/dev/null

    cert="gwy-mdsd"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-gwy ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-gwy" \
        --name ${cert} \
        --file secrets/gwy-mdsd-self-signed.pem >/dev/null

    cert="portal-server"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-por ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-por" \
        --name ${cert} \
        --file secrets/localhost.pem >/dev/null

    cert="portal-server"
    check_keyvault_certificate ${KEYVAULT_PREFIX}-por ${cert} && echo -e "⏭️🔑💼 Skip import for certificate ${cert}\n" \
    || echo -e "Import certificate ${cert}\n" && az keyvault certificate import \
        --vault-name "${KEYVAULT_PREFIX}-por" \
        --name ${cert} \
        --file secrets/portal-client.pem >/dev/null
}

# Function to cleanup all the created resources from the full aro RP dev
clean_rp_dev_env() {
    echo "########## Deleting Dev Env in $LOCATION ##########"

    # Validate number of arguments
    if [ $# -gt 2 ]; then
        echo "Usage: $0 [LIST_RESOURCE_GROUPS] [LIST_KEYVAULTS]. Please try again"
        exit 1
    fi
    if [[ $# -ne 2 ]]; then
        echo "Info: Two input arguments were required. Checking two env vars for default values"
        # Check if AZURE_PREFIX environment variable are set
        if [[ -z "${AZURE_PREFIX}" ]]; then
            echo "Error: AZURE_PREFIX environment variable is not set."
            exit 1
        fi
        echo "Info: AZURE_PREFIX=${AZURE_PREFIX}"
        # Check if LOCATION environment variable are set
        if [[ -z "$LOCATION" ]]; then
            echo "Error: LOCATION environment variable is not set."
            exit 1
        fi
        echo "Info: LOCATION=$LOCATION"
    fi
    
    # Convert input strings to arrays
    eval "rgs=($1)"
    eval "kvs=($2)"

   if [[ ${#rgs[@]} -eq 0 ]]; then
        rg_suffixes=("global" "subscription" "gwy-$LOCATION" "aro-$LOCATION")
        rgs=()
        for suffix in "${rg_suffixes[@]}"; do
            rgs+=("${AZURE_PREFIX}-$suffix")
        done
        echo "No resource groups were provided. Use default values for list: ${rgs[*]}"
    fi

     for rg in "${rgs[@]}"; do
        echo "########## Delete Resource Group $rg in $LOCATION ##########"
        az group delete --resource-group "$rg" -y
    done

    if [[ ${#kvs[@]} -eq 0 ]]; then
        kv_suffixes=("gwy" "por" "svc" "cls")
        kvs=()
        for suffix in "${kv_suffixes[@]}"; do
            kvs+=("${AZURE_PREFIX}-aro-$LOCATION-$suffix")
        done
        echo "No KeyVaults were provided. Use default values for list: ${kvs[*]}"
    fi

    for kv in "${kvs[@]}"; do
        echo "########## Delete KeyVault $kv in $LOCATION ##########"
        az keyvault purge --name "$kv" # add --no-wait to stop waiting
    done
}

# Example usage
# check_azure_deployment  "<ResourceGroup>"" "<DeploymentName>""
# check_jq_installed
# extract_image_tag "<FUNCTION_NAME>" "<FILE_TO_EXTRACT>"
# get_digest_tag "FluentbitImage"
# copy_digest_tag "<PULL_SECRET>" "src_acr_name" "dst_acr_name" "$(get_digest_tag FluentbitImage)"
# check_acr_repo <ACR_Name> <Repository>
# check_acr_repos <ResourceGroup>
# import_geneva_image <Repository> <Tag>
# check_keyvault_certificate "<KeyVault>" "<Certificate>"
# check_and_import_certificates
# clean_rp_dev_env "rg-1 rg-2 rg-3 rg-4" "kv-1 kv-2 kv-3 kv-4" 