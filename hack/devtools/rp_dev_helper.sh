#!/bin/bash  

set -o errexit \
       -o nounset \
       -o monitor

declare -r utils="hack/util.sh"
declare -r provisioning_state_succeeded="Succeeded"
if [ -f "$utils" ]; then
    # shellcheck source=../util.sh  
    source "$utils"
fi
log "Keep in mind that some of the functions here use 'jq' to parse data from Azure API, and may fail on missing jq."
######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########
# Run usage_rp_dev to get functions' usage help

secrets_dir="secrets"
secrets_files=("dev-ca.crt" "dev-client.crt" "portal-client.pem" "firstparty.pem" "localhost.pem" "arm.pem"
        "cluster-mdsd-self-signed.pem" "gwy-mdm-self-signed.pem" "gwy-mdsd-self-signed.pem" "rp-mdm-self-signed.pem"
        "rp-mdsd-self-signed.pem" "full_rp_id_rsa" "full_rp_id_rsa.pub" "env" )

verify_downloading_secrets() {
    if [ ! -d "$secrets_dir" ]; then
        abort "Directory '$secrets_dir' has not been created."
    fi
    #  TODO check if only the below files are required
    # shellcheck disable=SC2068
    for file in ${secrets_files[@]}; do
        if [ ! -f "$secrets_dir/$file" ]; then
            abort "File '$file' does not exist inside the directory '$secrets_dir'."
        fi
    done
}

is_boolean(){
    if [[ $1 != false && $1 != true ]]; then
        abort "var $1 isn't a boolean"
    fi
}

check_vmss() {
    err_str="Usage $0 <RESOURCE_GROUP> <VMSS_NAME> <DELETE_VMSS>. Please try again"
    local resource_group="${1?$err_str}"
    local vmss_name="${2?$err_str}"
    local delete_vmss="${3:-"false"}"
    # Check if the VMSS exists
    vmss_info="$(az vmss show --resource-group "${resource_group}" --name "${vmss_name}" 2>/dev/null)"
    if [ -z "${vmss_info}" ]; then
        log "🔴❌🖥️ VMSS '${vmss_name}' in Resource group '$resource_group' does not exist."
        return 1
    fi

    provisioning_state="$( jq -r '.provisioningState' <<< "${vmss_info}")"
    if [[ "${provisioning_state}" == "${provisioning_state_succeeded}" ]]; then
        log "🟢🖥️ VMSS '${vmss_name}' in Resource group '$resource_group' has been provisioned successfully. DELETE_VMSS:${delete_vmss}"
        if is_boolean "${delete_vmss}" &&  [ "${delete_vmss}" = true ]; then
           az vmss delete --resource-group "${resource_group}" --name  "${vmss_name}" --force-deletion
           log "🗑️🖥️ VMSS '${vmss_name}' in Resource group '$resource_group' has been deleted."
        fi
    else
        log "🔴🖥️ VMSS '${vmss_name}' in Resource group '$resource_group' has not been provisioned successfully. Current state: ${provisioning_state}."
        return 1
    fi
}

check_deployment() {
    err_str="Usage $0 <RESOURCE_GROUP> <DEPLOYMENT_NAME>. Please try again"
    local resource_group="${1?$err_str}"
    local deployment_name="${2?$err_str}"

    # Check if the ResourceGroup exists
    resource_group_info="$(az group show --resource-group "${resource_group}" 2>/dev/null)"
    if [ -z "${resource_group_info}" ]; then
        log "🔴❌📦 Resource group '${resource_group}' of deployment '${deployment_name}' does not exist."
        return 1
    fi

    # Check if the deployment exists
    deployment_info="$(az deployment group show --resource-group "${resource_group}" --name "${deployment_name}" 2>/dev/null)"
    if [ -z "${deployment_info}" ]; then
        log "🔴❌📦 Deployment '${deployment_name}' does not exist in resource group '${resource_group}'."
        return 1
    fi
    # Check if the provisioning state mathc provisioning_state_succeeded
    provisioning_state="$(jq -r '.properties.provisioningState' <<< "${deployment_info}")"
    if [[ "${provisioning_state}" == "${provisioning_state_succeeded}" ]]; then
        log "🟢📦 Deployment '${deployment_name}' in resource group '${resource_group}' has been provisioned successfully."
    else
        log "🔴📦 Deployment '${deployment_name}' in resource group '${resource_group}' has not been provisioned successfully. Current state: ${provisioning_state}"
        return 1
    fi
}

extract_image_tag() {
    err_str="Usage $0 <IMAGE_NAME> <FILE>. Please try again"
    local image_name="${1?$err_str}"
    local file="${2?$err_str}"

    local return_line
    return_line="$(awk "/func $image_name/ {flag=1; next} /return/ && flag {print; exit}" "$file")"
    return_line="${return_line#*:}"
    echo "${return_line%@*}"
}

get_repo_tag() {
    declare -r version_const_file="pkg/util/version/const.go"
    err_str="Usage $0 <IMAGE_NAME>. Please try again"
    local image_name="${1?$err_str}"
    extract_image_tag "${image_name}" "${version_const_file}"
}

copy_digest_tag() {
    err_str="Usage $0 <PULL_SECRET> <SRC_ACR_NAME> <DST_ACR_NAME> <REPOSITORY> <TAG>. Please try again"
    local pull_secret="${1?$err_str}"
    local src_acr_name="${2?$err_str}"
    local dst_acr_name="${3?$err_str}"
    local repository="${4?$err_str}"
    local tag="${5?$err_str}"
    
    log "INFO: Copy image from one ACR to another ..."
   
    src_auth="$(jq -r '.auths["'"$src_acr_name"'.azurecr.io"].auth' <<< $pull_secret | base64 -d)"
    dst_token="$(az acr login -n "${dst_acr_name}" --expose-token | jq -r .accessToken)"
    
    skopeo copy \
        --src-creds "$src_auth" \
        --dest-creds "00000000-0000-0000-0000-000000000000:$dst_token" \
        "docker://$src_acr_name.azurecr.io/${repository}:${tag}" \
        "docker://${dst_acr_name}.azurecr.io/${repository}:${tag}"
}

check_acr_repo() {
    err_str="Usage: $0 <ACR_NAME> <REPOSITORY> [SKIP_DEPLOYMENTS] [TAG]. Please try again"
    local acr_name="${1?$err_str}"
    local repository="${2?$err_str}"
    local skip_deployments="${3:-"true"}"
    local tag="${4:-"no-tag"}"

   # Don't skip deployment creation when skip_deployments was set to 'false'
    if is_boolean "$skip_deployments" && [ "${skip_deployments}" = false ]; then
        log "'skip_deployments' was set to 'false'. ❌⏩ Don't skip ACR '$acr_name' repo mirroring for repository '${repository}'."
        return 1
    fi

    # Check if the repository tag is not empty and if it matches an optional tag
    repo_tag="$(az acr repository show-tags --name "$acr_name" --repository "${repository}" -o tsv | awk '{printf "%s%s", sep, $0; sep=","} END {print ""}')"
    if [[ -n "$repo_tag" ]]; then
        if [[ "${tag}" != "no-tag" && "${tag}" != "${repo_tag}" ]] ; then
            log "🔴✈️ Repository '${repository}' in ACR '$acr_name' exists with different tag/s '${repo_tag}'. Expected tag: '${tag}'."
            return 1
        fi
        log "🟢✈️ Repository '${repository}' in ACR '$acr_name' exists with tag '${repo_tag}'."
        return 0
    fi
    log "🔴❌✈️ Repository '${repository}' doesn't exist in ACR '$acr_name'."
    return 1
}

# "openshift-release-dev/ocp-release" "openshift-release-dev/ocp-v4.0-art-dev" were excluded as they don't include an image/tag
acr_repos=("azure-cli" "rhel8/support-tools" "rhel9/support-tools" "openshift4/ose-tools-rhel8" "ubi8/ubi-minimal" "ubi9/ubi-minimal"
        "ubi8/nodejs-18" "ubi8/go-toolset" "app-sre/managed-upgrade-operator" "app-sre/hive" "distroless/genevamdm" "distroless/genevamdsd"
        "aro" "fluentbit")

check_acr_repos() {
    err_str="Usage: $0 <RESOURCE_GROUP> [GIT_COMMIT] [SKIP_DEPLOYMENTS]. Please try again"
    local resource_group="${1?$err_str}"
    local git_commit="${2:-"no-commit"}"
    local skip_deployments="${3:-"true"}"

    # Don't skip deployment creation when skip_deployments was set to 'false'
    if is_boolean "$skip_deployments" && [ "${skip_deployments}" = false ]; then
        log "'skip_deployments' was set to 'false'. ❌⏩ Don't skip acr repo mirroring in ResourceGroup $resource_group"
        return 1
    fi
   
    # Get the first Azure Container Registry (ACR) name under ResourceGroup
    acr_name="$(az acr list --resource-group $resource_group | jq -r '.[0].name')"
    if [[ -z "$acr_name" ]]; then
        abort "🔴❌✈️ Error: There are no Azure Container Registries under ResourceGroup '$resource_group'."
    fi
    
    # Do all the needed repos already imported
    local -a missing_repos_names=()
    # shellcheck disable=SC2068
    for repo in ${acr_repos[@]}; do
        if [[ ${repo} == "aro" && $git_commit != "no-commit" ]]; then
            if ! check_acr_repo "$acr_name" "$repo" "$skip_deployments" "$git_commit"; then
                missing_repos_names+=("$repo")
            fi
        else
            if ! check_acr_repo "$acr_name" "$repo" "$skip_deployments"; then
                missing_repos_names+=("$repo")
            fi
        fi
    done
    if [ ${#missing_repos_names[@]} -eq 0 ]; then
        log "🟢✈️ All repositories exist in ACR '$acr_name'."
        return 0
    fi
    log "🔴✈️ Some repositories are missing and need to be imported: ${missing_repos_names[*]}."
    return 1
}

import_geneva_image() {
    err_str="Usage: $0 <REPOSITORY> <Tag> <DST_ACR_NAME>. Please try again"
    local repository="${1?$err_str}"
    local tag="${2?$err_str}"
    local dst_acr_name="${3?$err_str}"
    if ! check_acr_repo "${dst_acr_name}" "${repository}" "true" "${tag}" ;then
        az acr import --name "${dst_acr_name}.azurecr.io/${repository}:${tag}" --source "linuxgeneva-microsoft.azurecr.io/${repository}:${tag}"
        log "🟢✈️ Imported repository '${repository}' to ACR '${dst_acr_name}'"
    else
        log "⏭️📦 Skip importing repository '${repository}' to ACR '${dst_acr_name}', since it already exist with tag ${tag}. Import can not run twice"
    fi
}

check_keyvault_certificate() {
    err_str="Usage $0 <KEYVAULT> <CERTIFICATE> [SKIP_DEPLOYMENTS]. Please try again"
    local key_vault="${1?$err_str}"
    local certificate="${2?$err_str}"
    local skip_deployments="${3:-"true"}"

    # Don't skip deployment creation when skip_deployments was set to 'false'
    if is_boolean "$skip_deployments" && [ "${skip_deployments}" = false ]; then
        abort "'skip_deployments' was set to 'false'. ❌⏩ Don't skip keyvault's certificate $certificate import"
    fi
    
    # Check if the Key Vault exists
    if ! az keyvault show --name "$key_vault" >/dev/null 2>&1; then
        abort "🔴❌💼 Error: Key Vault '$key_vault' does not exist."
    fi

    certificate_info="$(az keyvault certificate show --vault-name "$key_vault" --name "$certificate" 2>/dev/null)"
    if [ -z "${certificate_info}" ]; then
        log "🔴❌💼 Certificate '$certificate' in Key Vault '$key_vault' does not exist."
        return 1
    fi

    local -r attributes_enabled="$( jq -r '.attributes.enabled' <<< "$certificate_info")"
    local -r attributes_expires="$( jq -r '.attributes.expires' <<< "$certificate_info")"
    # we don't validate the thumbprint for exact match
    if $attributes_enabled; then 
        log "🟢💼 Certificate '$certificate' in Key Vault '$key_vault' exists and is enabled with expiration date '$attributes_expires'."
    else
        abort "🔴💼 Certificate '$certificate' in Key Vault '$key_vault' exists but is not enabled."
    fi
}

skip_and_import_certificates(){
    err_str="Usage $0 <KEYVAULT>  <CERTIFICATEs...> <SECRET_FILES...> [SKIP_DEPLOYMENTS]. Please try again"
    local keyVault="${1?$err_str}"

    # Extract the certificates and secret_files arraies based on half the remaining arguments
    local num_certs="$((($# -1) / 2))"
    local -a certificates=("${@:2:$num_certs}")
    local -a secret_files=("${@:$((num_certs + 2)):$num_certs}")
    # Optional flag (last argument)
    local skip_deployments="${*: -1}"

    # Don't skip deployment creation when skip_deployments was set to "false" 
    if is_boolean "$skip_deployments" && [ "${skip_deployments}" = false ]; then
        abort "'skip_deployments' was set to 'false'. ❌⏩ Don't skip certs import to keyVault $keyVault"
    fi

    # shellcheck disable=SC2068
    for i in ${!certificates[*]}; do
         if check_keyvault_certificate "${keyVault}" "${certificates[i]}" "$skip_deployments"; then
            log "⏭️🔑💼 Skip import for certificate ${certificates[i]}"
        else
            log "🔐📥 Importing certificate ${certificates[i]}"
            az keyvault certificate import \
                --vault-name "${keyVault}" \
                --name "${certificates[i]}" \
                --file "${secret_files[i]}" >/dev/null
        fi
    done
}

check_and_import_certificates (){
    err_str="Usage $0 <KEYVAULT_PREFIX> [SKIP_DEPLOYMENTS]. Please try again"
    local keyvault_prefix="${1?$err_str}"
    local skip_deployments="${2:-"true"}"

    # Don't skip deployment creation when skip_deployments was set to "false" 
    if is_boolean "$skip_deployments" && [ "${skip_deployments}" = false ]; then
        abort "'skip_deployments' was set to 'false'. ❌⏩ Don't skip certs import"
    fi

    local files
    local certs
    local key_vault
    key_vault="${keyvault_prefix}-svc"
    log "Check import certificates for the service keyVault ${key_vault}"
    certs=("rp-mdm" "rp-mdsd" "cluster-mdsd" "dev-arm" "rp-firstparty" "rp-server")
    files=("secrets/rp-mdm-self-signed.pem" "secrets/rp-mdsd-self-signed.pem" "secrets/cluster-mdsd-self-signed.pem" "secrets/arm.pem" "secrets/firstparty.pem" "secrets/localhost.pem")
    skip_and_import_certificates "${key_vault}" "${certs[@]}" "${files[@]}" "$skip_deployments"
    
    key_vault="${keyvault_prefix}-gwy"
    log "Check import certificates for the gateway keyVault ${key_vault}"
    certs=("gwy-mdm" "gwy-mdsd")
    files=("secrets/gwy-mdm-self-signed.pem" "secrets/gwy-mdsd-self-signed.pem")
    skip_and_import_certificates "${key_vault}" "${certs[@]}" "${files[@]}" "$skip_deployments"

    key_vault="${keyvault_prefix}-por"
    log "Check import certificates for the portal keyVault ${key_vault}"
    certs=("portal-server" "portal-client")
    files=("secrets/localhost.pem" "secrets/portal-client.pem")
    skip_and_import_certificates "${key_vault}" "${certs[@]}" "${files[@]}" "$skip_deployments"
}

clean_rp_dev_env() {
    err_str="Usage $0 <LOCATION> [LIST_RESOURCE_GROUPS] [LIST_KEYVAULTS]. Please try again"
    local location="${1?$err_str}"
    local -a list_resource_groups=("${2:-}")
    local -a list_keyvaults=("${3:-}")
    log "########## Deleting Dev Env in $location ##########"

    if [[ $# -lt 2 ]]; then
        log "Info: One input argument was required. Checking AZURE_PREFIX env var for default values"
        if [[ -z "${AZURE_PREFIX}" ]]; then
            abort "Error: AZURE_PREFIX environment variable was not set."
        fi
        log "Info: AZURE_PREFIX=${AZURE_PREFIX}"
    fi

    # Convert input strings to arrays
    eval "rgs=($list_resource_groups)"
    eval "kvs=($list_keyvaults)"

   if [[ ${#rgs[@]} -eq 0 ]]; then
        rg_suffixes=("global" "subscription" "gwy-$location" "aro-$location")
         # shellcheck disable=2068
        for suffix in ${rg_suffixes[@]}; do
            rgs+=("${AZURE_PREFIX}-$suffix")
        done
        log "No resource groups were provided. Use default values for list: ${rgs[*]}"
    fi

     # shellcheck disable=SC2068
     for rg in ${rgs[@]}; do
        log "########## Delete Resource Group $rg in $location ##########"
        az group delete --resource-group "$rg" -y || true
    done

    if [[ ${#kvs[@]} -eq 0 ]]; then
        kv_suffixes=("gwy" "por" "svc" "cls")
        # shellcheck disable=2068
        for suffix in ${kv_suffixes[@]}; do
            kvs+=("${AZURE_PREFIX}-aro-$location-$suffix")
        done
        log "No KeyVaults were provided. Use default values for list: ${kvs[*]}"
    fi

    # shellcheck disable=SC2068
    for kv in ${kvs[@]}; do
        log "########## Delete KeyVault $kv in $location ##########"
        az keyvault purge --name "$kv" || true  # add --no-wait to stop waiting
    done
}

usage_rp_dev() {
    cat <<EOF
######## Helper functions for Full RP dev automation ########
Usage: $0 <function_name> [arguments]

Available functions:
  verify_downloading_secrets    - Download the secrets storage account and validates that the secrets directory and required files exist
  is_boolean                 - Check if the input value is true or false
  check_deployment              - Check deployment DEPLOYMENT_NAME existance in resource group RESOURCE_GROUP and provisioning state
  check_vmss                    - Check VMSS existance and its provisioning state
  extract_image_tag             - Extract the image tag
  get_repo_tag                  - Get image name and tag
  copy_digest_tag               - Copy image from ACR to another using Skopeo
  check_acr_repo                - Check the repo REPOSITORY existance in the ACR and if it matches an optional tag TAG
  check_acr_repos               - Check if all the required repos exist in the ACR and list the missing ones
  import_geneva_image           - Import a Geneva image of repo REPOSITORY only when it is missing
  check_keyvault_certificate    - Check certificate CERTIFICATE existance in keyVault KEYVAULT, enablement and expiration date
  skip_and_import_certificates  - Import array of certificates <CERTIFICATE...> using array of secret files <SECRET_FILE...> to keyVault KEYVAULT
  check_and_import_certificates - Import the certificates if possible based on prefix KEYVAULT_PREFIX
  clean_rp_dev_env              - Cleanup all the created resources from the full RP dev (4 resourceGroups and 4 KeyVaults) based on input or defualt values.

Examples:
  $0 verify_downloading_secrets
  $0 is_boolean
  $0 check_deployment xxx-aro-eastus aks-development
  $0 check_vmss xxx-aro-eastus rp-vmss-bfc8993 true
  $0 extract_image_tag pkg/util/version/const.go
  $0 get_repo_tag FluentbitImage
  $0 copy_digest_tag PULL_SECRET arointsvc xxxaro fluentbit 1.9.10-cm20240628
  $0 check_acr_repo xxxaro fluentbit true 1.9.10-cm20240628
  $0 check_acr_repos xxxaro-eastus-global bfc8993 true
  $0 import_geneva_image fluentbit 1.9.10-cm20240628 xxxaro
  $0 check_keyvault_certificate xxx-aro-eastus-svc rp-mdm true
  $0 skip_and_import_certificates xxx-aro-eastus-svc rp-mdm rp-mdsd secrets/rp-mdm-self-signed.pem secrets/rp-mdsd-self-signed.pem true
  $0 check_and_import_certificates xxx-aro-eastus-svc true
  $0 clean_rp_dev_env eastus "xxx-global xxx-subscription" "xxx-aro-eastus-gwy xxx-aro-eastus-por"

To get detailed usage for a specific function, run:
  $0 usage_rp_dev <function_name>
EOF
    local fun_name="${1-"missing-fun_name"}"
    # Specific function usage
    case "${fun_name}" in
        verify_downloading_secrets)
            echo "Usage: $0 verify_downloading_secrets"
            ;;
        is_boolean)
            echo "Usage: $0 is_boolean"
            ;;
        check_deployment)
            echo "Usage: $0 check_deployment <RESOURCE_GROUP> <DEPLOYMENT_NAME>"
            ;;
        check_vmss)
            echo "Usage: $0 check_vmss <RESOURCE_GROUP> <VMSS_NAME> <DELETE_VMSS>"
            ;;
        extract_image_tag)
            echo "Usage: $0 extract_image_tag <IMAGE_NAME> <FILE>"
            ;;
        get_repo_tag)
            echo "Usage: $0 get_repo_tag <IMAGE_NAME>"
            ;;
        copy_digest_tag)
            echo "Usage: $0 copy_digest_tag <PULL_SECRET> <SRC_ACR_NAME> <DST_ACR_NAME> <REPOSITORY> <TAG>"
            ;;
        check_acr_repo)
            echo "Usage: $0 check_acr_repo <ACR_NAME> <REPOSITORY> [SKIP_DEPLOYMENTS] [TAG]"
            ;;
        check_acr_repos)
            echo "Usage: $0 check_acr_repos <RESOURCE_GROUP> [GIT_COMMIT] [SKIP_DEPLOYMENTS]"
            ;;
        import_geneva_image)
            echo "Usage: $0 import_geneva_image <REPOSITORY> <Tag> <DST_ACR_NAME>"
            ;;
        check_keyvault_certificate)
            echo "Usage: $0 check_keyvault_certificate <KEYVAULT> <CERTIFICATE> [SKIP_DEPLOYMENTS]"
            ;;
        skip_and_import_certificates)
            echo "Usage: $0 skip_and_import_certificates <KEYVAULT>  <CERTIFICATE...> <SECRET_FILE...> [SKIP_DEPLOYMENTS]"
            ;;
        check_and_import_certificates)
            echo "Usage: $0 check_and_import_certificates <KEYVAULT_PREFIX> [SKIP_DEPLOYMENTS]"
            ;;
        clean_rp_dev_env)
            echo "Usage: $0 clean_rp_dev_env <LOCATION> [LIST_RESOURCE_GROUPS] [LIST_KEYVAULTS]"
            ;;
        *)
            # If no specific function is passed, or an invalid function is passed.
            echo "Specify a valid function name to get more details."
            ;;
    esac
}
