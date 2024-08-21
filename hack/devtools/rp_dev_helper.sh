#!/bin/bash  

set -o errexit \
       -o nounset \
       -o monitor

declare -r utils=hack/util.sh
if [ -f "$utils" ]; then
    source "$utils"
fi

######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########
secrets_dir="secrets"
secrets_files=("dev-ca.crt" "dev-client.crt" "portal-client.pem" "firstparty.pem" "localhost.pem" "arm.pem"
        "cluster-mdsd-self-signed.pem" "gwy-mdm-self-signed.pem" "gwy-mdsd-self-signed.pem" "rp-mdm-self-signed.pem"
        "rp-mdsd-self-signed.pem" "full_rp_id_rsa" "full_rp_id_rsa.pub" "env" )

    # # Define the expected directory and file names
    # all_secret_files=("firstparty.key" "vpn-client.key" "vpn-eastus.ovpn" "localhost.crt" "proxy.crt"
    #     "dev-ca.crt" "proxy-client.pem" "vpn-aks-eastus.ovpn" "portal-client.key" "portal-client.pem"
    #     "proxy-client.key" "vpn-ca-cert.pem" "rp-metrics-int.pem" "dev-client.crt" "rp-logging-int.pem"
    #     "vpn-ca.crt" "shared-cluster.kubeconfig" "vpn-client.pem" "firstparty.pem" "portal-client.crt"
    #     "dev-ca.key" "vpn-aks-australiaeast.ovpn" "vpn-client-cert.pem" "vpn-client-key.pem" "dev-client.pem"
    #     "proxy.key" "firstparty.crt" "vpn-client.crt" "vpn-aks-westeurope.ovpn" "localhost.key" "vpn-ca.pem"
    #     "vpn-australiaeast.ovpn" "cluster-logging-int.pem" "vpn-ca.key" "localhost.pem" "proxy.pem" "dev-client.key"
    #     "proxy_id_rsa.pub" "proxy_id_rsa" "vpn-ca-key.pem" "proxy-client.crt" "vpn-westeurope.ovpn" "dev-ca.pem"
    #     "arm.crt" "arm.key" "arm.pem" "cluster-mdsd.crt" "cluster-mdsd.key" "cluster-mdsd.pem" "cluster-mdsd-self-signed.pem"
    #     "gwy-mdm-self-signed.pem" "gwy-mdsd-self-signed.pem" "rp-mdm-self-signed.pem" "rp-mdsd-self-signed.pem"
    #     "full_rp_id_rsa" "full_rp_id_rsa.pub" "env")

verify_downloading_secrets() {
    # Validate that the secrets directory and required files exist
    if [ ! -d "$secrets_dir" ]; then
        abort "Directory '$secrets_dir' has not been created."
    fi
    #  TODO check if only the below files are required
    for file in ${secrets_files[@]}; do
        if [ ! -f "$secrets_dir/$file" ]; then
            abort "File '$file' does not exist inside the directory '$secrets_dir'."
        fi
    done
}

check_jq_installed() {
  if ! command -v jq &> /dev/null; then
    abort "Error: jq is not installed.
                Please install jq to proceed. You can install it using the following command:
                For Debian/Ubuntu-based systems: sudo apt-get install jq
                For Red Hat/CentOS-based systems: sudo yum install jq
                For macOS: brew install jq"
  fi
}

is_it_boolean(){
    if [[ $1 != "false" && $1 != "true" ]]; then
        abort "var $1 isn't a boolean"
    fi
}

# Check deployment existance and provisioning state
check_deployment() {
    err_str="Usage $0 <ResourceGroup> <DeploymentName>. Please try again"
    local resource_group=${1?$err_str}
    local deployment_name=${2?$err_str}

    # Check if the ResourceGroup exists
    resource_group_info=$(az group show --resource-group "${resource_group}" 2>/dev/null)
    if [ -z "${resource_group_info}" ]; then
        log "Resource group '${resource_group}' does not exist."
        return 1
    fi

    # Check if the deployment exists
    deployment_info=$(az deployment group show --resource-group "${resource_group}" --name "${deployment_name}" 2>/dev/null)
    if [ -z "${deployment_info}" ]; then
        log "Deployment '${deployment_name}' does not exist in resource group '${resource_group}'."
        return 1
    fi
    # Check if the provisioning state is 'Succeeded'
    check_jq_installed
    provisioning_state=$(jq -r '.properties.provisioningState' <<< "${deployment_info}")
    if [[ "${provisioning_state}" == "Succeeded" ]]; then
        log "Deployment '${deployment_name}' in resource group '${resource_group}' has provisioned successfully."
    else
        log "Deployment '${deployment_name}' in resource group '${resource_group}' has not provisioned successfully. Current state: ${provisioning_state}"
        return 1
    fi
}

# Extract the image tag
extract_image_tag() {
    err_str="Usage $0 <FUNCTION_NAME> <FILE_TO_EXTRACT>. Please try again"
    local -n function_name=${1?$err_str}
    local -n file_to_extract=${2?$err_str}
    for arg in "$function_name" "$file_to_extract"; do
        if  [[ -z "$arg" ]];  then
            abort "Error: an empty argument, \"$arg\". Please try again"
        fi
    done

    local return_line
    return_line="$(grep -A 2 "func $function_name" "$file_to_extract" | grep 'return')"
    imageName=$(tr '[:upper:]' '[:lower:]' <<< ${$1%Image})
    tagDigest=${return_line#*/$baseName}
    result="/$imageName${tagDigest%%@*}"
    TODO
    echo $result
}

VERSION_CONST_FILE="pkg/util/version/const.go"
# Get image name and tag
get_digest_tag() {
    err_str="Usage $0 <IMAGE_NAME>. Please try again"
    local -n image_name=${1?$err_str}
    local image_tag
    image_tag="$(extract_image_tag "$image_name" "$VERSION_CONST_FILE")"
    echo "$image_tag"
}

# Copy image using skopeo
copy_digest_tag() {
    # Check if exactly non-empty four arguments are provided
    err_str="Usage $0 <PULL_SECRET> <SRC_ACR_NAME> <DST_ACR_NAME> <IMAGE_TAG>. Please try again"
    local -n pull_secret=${1?$err_str}
    local -n src_acr_name=${2?$err_str}
    local -n dst_acr_name=${3?$err_str}
    local -n image_tag=${4?$err_str}
    
    for arg in "$1" "$2" "$3" "$4"; do
        if  [[ -z "$arg" ]];  then
            abort "Error: an empty argument, \"$arg\". Please try again"
        fi
    done
    log "INFO: Copy image from one ACR to another ..."
   
    src_auth="$(jq -r '.auths["'$src_acr_name'.azurecr.io"].auth' <<< $pull_secret | base64 -d)"
    dst_token="$(az acr login -n $dst_acr_name --expose-token | jq -r .accessToken)"
    
    skopeo copy \
        --src-creds "$src_auth" \
        --dest-creds "00000000-0000-0000-0000-000000000000:$dst_token" \
        "docker://$src_acr_name.azurecr.io$image_tag" \
        "docker://$dst_acr_name.azurecr.io$image_tag"
}

# Check if a repo exist in the ACR
check_acr_repo() {
    err_str="Usage: $0 <ACR_Name> <Repository> [SKIP_DEPLOYMENTS]. Please try again"
    local -n acr_name=${1?$err_str}
    local -n repository=${2?$err_str}
    local -n skip_deployments=${3?$err_str}
CHECK ITTT
    if  [[ -z "$3" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if $skip_deployments; then
        abort "'SKIP_DEPLOYMENTS' env var is set to $skip_deployments. Don't skip ACR $acr_name repo mirroring for repository $repository."
    fi

    # Get the repository tag
    repo_tag=$(az acr repository show-tags --name "$acr_name" --repository "$repository" -o tsv | tr '' ' ')

    # Check if the repository tag is empty
    if [[ -n "$repo_tag" ]]; then
        log "Repository '$repository' in ACR '$acr_name' exists with tag '${repo_tag}'."
        return 0
    fi
    log "Repository '$repository' doesn't exist in ACR '$acr_name'."
    return 1
}

# "openshift-release-dev/ocp-release" "openshift-release-dev/ocp-v4.0-art-dev" were excluded as they don't include an image/tag
acr_repos=("azure-cli" "rhel8/support-tools" "rhel9/support-tools" "openshift4/ose-tools-rhel8" "ubi8/ubi-minimal" "ubi9/ubi-minimal" \
        "ubi8/nodejs-18" "ubi8/go-toolset" "app-sre/managed-upgrade-operator" "app-sre/hive" "distroless/genevamdm" "distroless/genevamdsd" \
        "aro" "fluentbit")

# Check if all the required repos exist in the ACR and list the missing ones
check_acr_repos() {
    err_str="Usage: $0 <ResourceGroup> [SKIP_DEPLOYMENTS]. Please try again"
    local -n resource_group=${1?$err_str}
    local -n skip_deployments=${2?$err_str}
    if  [[ -z "$2" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if $skip_deployments; then
        abort "'SKIP_DEPLOYMENTS' env var is set to $skip_deployments. Don't skip acr repo mirroring in ResourceGroup $resource_group"
    fi
   
    check_jq_installed
     # Get the first Azure Container Registry (ACR) name under ResourceGroup
    acr_name="$(az acr list --resource-group $resource_group | jq -r '.[0].name')"
    if [[ -z "$acr_name" ]]; then
        abort "Error: There are no Azure Container Registries under ResourceGroup '$resource_group'."
    fi
    
    # Do all the needed repos already imported
    local -a missing_repos_names
    for repo in ${acr_repos[@]}; do
        if ! check_acr_repo "$acr_name" "$repo" $skip_deployments; then
            missing_repos_names=("$repo")
        fi
    done
    if [ -z "${missing_repos_names}" ]; then
        log "All repositories exist in ACR '$acr_name'."
        return 0
    fi
    log "Some repositories are missing and need to be imported. Repositories: ${missing_repos_names[@]}"
    return 1
}

# Import a Geneva image only when it isn't exist
import_geneva_image() {
    err_str="Usage: $0 <Repository> <Tag> <DST_ACR_NAME>. Please try again"
    local -n repository=${1?$err_str}
    local -n tag=${2?$err_str}
    local -n dst_acr_name=${3?$err_str}

    if ! check_acr_repo $dst_acr_name $repository $skip_deployments ;then
        az acr import --name $dst_acr_name.azurecr.io$tag --source linuxgeneva-microsoft.azurecr.io$tag
        log "Imported $repository to ACR $dst_acr_name"
    else
        log "⏭️📦 Skip importing $repository to ACR $dst_acr_name, since it already exist. Import can not run twice"
    fi
}

# Check certificars existance, their enablement and expiration date
check_keyvault_certificate() {
    err_str="Usage $0 <KeyVault> <Certificate> [SKIP_DEPLOYMENTS]. Please try again"
    local -n key_vault=${1?$err_str}
    local -n certificate=${1?$err_str}
    local -n skip_deployments=${3?$err_str}
    if  [[ -z "$3" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if $skip_deployments; then
        abort "'SKIP_DEPLOYMENTS' env var is set to $skip_deployments. Don't skip keyvault's certificate $certificate import"
    fi
    
    # Check if the Key Vault exists
    if ! az keyvault show --name "$key_vault" >/dev/null 2>&1; then
        abort "Error: Key Vault '$key_vault' does not exist."
    fi

    certificate_info="$(az keyvault certificate show --vault-name "$key_vault" --name "$certificate" 2>/dev/null)"
    if [ -z "${certificate_info}" ]; then
        log "Certificate '$certificate' in Key Vault '$key_vault' does not exist."
        return 1
    fi

    check_jq_installed

    local -r attributes_enabled="$( jq -r '.attributes.enabled' <<< $certificate_info)"
    local -r attributes_expires="$( jq -r '.attributes.expires' <<< $certificate_info)"
    if $attributes_enabled; then 
        log "Certificate '$certificate' in Key Vault '$key_vault' exists and is enabled with expiration date '$attributes_expires'."
    else
        abort "Certificate '$certificate' in Key Vault '$key_vault' exists but is not enabled."
    fi
}

# Import 10 certificates in case they are needed
skip_and_import_certificates(){
    err_str="Usage $0 <Certificates> <KEYVAULT> <SECRET_FILES> [SKIP_DEPLOYMENTS] . Please try again"
    local -n certificates=${1?$err_str}
    local -n keyVault=${2?$err_str}
    local -n secret_files=${3?$err_str}
    local -n skip_deployments=${4?$err_str}
    if  [[ -z "$4" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if ! $skip_deployments; then
        abort "'SKIP_DEPLOYMENTS' env var is set to $skip_deployments. Don't skip certs import to keyVault $keyVault"
    fi

    for i in ${!certificates[@]}; do
        if check_keyvault_certificate "${keyVault}" "${certificates[i]}"; then
            log "⏭️🔑💼 Skip import for certificate ${certificates[i]}"
        else
            log "Import certificate ${certificates[i]}"
            az keyvault certificate import \
                --vault-name "${keyVault}" \
                --name "${certificates[i]}" \
                --file "${secret_files[i]}" >/dev/null
        fi
    done
}

# Import 10 certs if possible
check_and_import_certificates (){
    err_str="Usage $0 <KEYVAULT_PREFIX> [SKIP_DEPLOYMENTS]. Please try again"
    local -n keyvault_prefix=${1?$err_str}   
    local -n skip_deployments=${2?$err_str}
    if  [[ -z "$2" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if ! $skip_deployments; then
        abort "'SKIP_DEPLOYMENTS' env var is set to $skip_deployments. Don't skip certs import"
    fi

    local files
    local certs
    local key_vault
    key_vault=${keyvault_prefix"-svc"}
    log "Check import certificates for the service keyVault ${key_vault}"
    certs=("rp-mdm" "rp-mdsd" "cluster-mdsd" "dev-arm" "rp-firstparty" "rp-server")
    files=("secrets/rp-mdm-self-signed.pem" "secrets/rp-mdsd-self-signed.pem" "secrets/cluster-mdsd-self-signed.pem" "secrets/arm.pem" "secrets/firstparty.pem" "secrets/localhost.pem")
    skip_and_import_certificates $certs ${key_vault} $files
    
    key_vault=${keyvault_prefix"-gwy"}
    log "Check import certificates for the gateway keyVault ${key_vault}}"
    certs=("gwy-mdm" "gwy-mdsd")
    files=("secrets/gwy-mdm-self-signed.pem" "secrets/gwy-mdsd-self-signed.pem")
    skip_and_import_certificates $certs ${key_vault} $files

    key_vault=${keyvault_prefix"-por"}
    log "Check import certificates for the portal keyVault ${key_vault}"
    certs=("portal-server" "portal-client")
    files=("secrets/localhost.pem" "secrets/portal-client.pem")
    skip_and_import_certificates $certs ${key_vault} $files
}

# Check VMSS existance and its provisioning state
check_vmss() {
    err_str="Usage $0 <ResourceGroup> <VMSS_NAME>. Please try again"
    local resource_group=${1?$err_str}
    local vmss_name=${2?$err_str}

    # Check if the VMSS exists
    vmss_info="$(az vmss show --resource-group ${resource_group} --name ${vmss_name} 2>/dev/null)"
    if [ -z "${vmss_info}" ]; then
        log "VMSS '${vmss_name}' in Resource group '$resource_group' does not exist."
        return 1
    fi

    check_jq_installed

    provisioning_state="$( jq -r '.properties.provisioningState' <<< "${vmss_info}")"
    if [[ "${provisioning_state}" == "Succeeded" ]]; then
        log "VMSS '${vmss_name}' in Resource group '$resource_group' has provisioned successfully."
    else
        log "VMSS '${vmss_name}' in Resource group '$resource_group' has not provisioned successfully. Current state: ${provisioning_state}."
        return 1
    fi
}

# Cleanup all the created resources from the full aro RP dev
clean_rp_dev_env() {
    err_str="Usage $0 [LOCATION] [LIST_RESOURCE_GROUPS] [LIST_KEYVAULTS]. Please try again"
    local -n location=${1?$err_str}
    local -n list_resource_groups=${2?$err_str}
    local -n list_keyvaults=${3?$err_str}
    log "########## Deleting Dev Env in $LOCATION ##########"

    if [[ $# -ne 3 ]]; then
        log "Info: Three input arguments were required. Checking two env vars for default values"
        # Check if AZURE_PREFIX environment variable is set
        if [[ -z "${AZURE_PREFIX}" ]]; then
            abort "Error: AZURE_PREFIX environment variable was not set."
        fi
        log "Info: AZURE_PREFIX=${AZURE_PREFIX}"
        # Check if LOCATION environment variable is set
        if [[ -z "$LOCATION" ]]; then
            abort "Error: LOCATION environment variable was not set."
        fi
        log "Info: LOCATION=$LOCATION"
    fi
    
    # Convert input strings to arrays
    eval "rgs=($list_resource_groups)"
    eval "kvs=($list_keyvaults)"

   if [[ ${#rgs[@]} -eq 0 ]]; then
        rg_suffixes=("global" "subscription" "gwy-$LOCATION" "aro-$LOCATION")
        for suffix in ${rg_suffixes[@]}; do
            rgs+=("${AZURE_PREFIX}-$suffix")
        done
        log "No resource groups were provided. Use default values for list: ${rgs[*]}"
    fi

     for rg in ${rgs[@]}; do
        log "########## Delete Resource Group $rg in $LOCATION ##########"
        az group delete --resource-group "$rg" -y
    done

    if [[ ${#kvs[@]} -eq 0 ]]; then
        kv_suffixes=("gwy" "por" "svc" "cls")
        for suffix in ${kv_suffixes[@]}; do
            kvs+=("${AZURE_PREFIX}-aro-$LOCATION-$suffix")
        done
        log "No KeyVaults were provided. Use default values for list: ${kvs[*]}"
    fi

    for kv in ${kvs[@]}; do
        log "########## Delete KeyVault $kv in $LOCATION ##########"
        az keyvault purge --name "$kv" # add --no-wait to stop waiting
    done
}

# Example usage
# check_deployment  "<ResourceGroup>" "<DeploymentName>"
# check_jq_installed
# extract_image_tag "<FUNCTION_NAME>" "<FILE_TO_EXTRACT>"
# get_digest_tag "FluentbitImage"
# copy_digest_tag "<PULL_SECRET>" "src_acr_name" "dst_acr_name" "$(get_digest_tag FluentbitImage)"
# check_acr_repo <ACR_Name> <Repository> [SKIP_DEPLOYMENTS]
# check_acr_repos <ResourceGroup> [SKIP_DEPLOYMENTS]
# import_geneva_image <Repository> <Tag> <DST_ACR_NAME>
# check_keyvault_certificate "<KeyVault>" "<Certificate>" [SKIP_DEPLOYMENTS]
# skip_and_import_certificates <Certificates> <KEYVAULT> <SECRET_FILES> [SKIP_DEPLOYMENTS]
# check_and_import_certificates <KEYVAULT_PREFIX> [SKIP_DEPLOYMENTS]
# clean_rp_dev_env "rg-1 rg-2 rg-3 rg-4" "kv-1 kv-2 kv-3 kv-4" 