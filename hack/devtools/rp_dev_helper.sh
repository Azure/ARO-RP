#!/bin/bash  

set -o errexit \
       -o nounset \
       -o monitor

declare -r utils="hack/util.sh"
if [ -f "$utils" ]; then
    source "$utils"
fi

######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########

# Function to extract the image tag - (FUNCTION_NAME)
extract_image_tag() {
    # Extract the line containing the return statement
    local return_line=$(grep -A 1 "func $1" "$2" | grep 'return')
    echo "$return_line" | sed 's/.*"\(.*\)@sha256.*/\1/'
}

# Function to copy image using skopeo -(PULL_SECRET, SRC_ACR_NAME, DST_ACR_NAME, IMAGE_TAG)
copy_digest_tag() {
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

VERSION_CONST_FILE="pkg/util/version/const.go"
# Function to get image name and tag
get_digest_tag() {
    local IMAGE_NAME=$1
    local IMAGE_TAG=$(extract_image_tag "$IMAGE_NAME" "$VERSION_CONST_FILE")
    echo "$IMAGE_NAME and Tag: $IMAGE_TAG"
    echo "$IMAGE_TAG"
}

check_deployment() {
    err_str="Usage $0 <RESOURCE_GROUP> <DEPLOYMENT_NAME>. Please try again"
    local resource_group=${1?$err_str}
    local deployment_name=${2?$err_str}

    # Check if the ResourceGroup exists
    resource_group_info=$(az group show --resource-group "${resource_group}" 2>/dev/null)
    if [ -z "${resource_group_info}" ]; then
        log "🔴❌📦 Resource group '${resource_group}' does not exist."
        return 1
    fi

    # Check if the deployment exists
    deployment_info=$(az deployment group show --resource-group "${resource_group}" --name "${deployment_name}" 2>/dev/null)
    if [ -z "${deployment_info}" ]; then
        log "🔴❌📦 Deployment '${deployment_name}' does not exist in resource group '${resource_group}'."
        return 1
    fi
    # Check if the provisioning state is 'Succeeded'
    # check_jq_installed - Might not needed
    provisioning_state=$(jq -r '.properties.provisioningState' <<< "${deployment_info}")
    if [[ "${provisioning_state}" == "Succeeded" ]]; then
        log "🟢📦 Deployment '${deployment_name}' in resource group '${resource_group}' has been provisioned successfully."
    else
        log "🔴📦 Deployment '${deployment_name}' in resource group '${resource_group}' has not been provisioned successfully. Current state: ${provisioning_state}"
        return 1
    fi
}

# Example usage
# get_digest_tag "FluentbitImage"
# copy_digest_tag "<PULL_SECRET>" "src_acr_name" "dst_acr_name" "$(get_digest_tag FluentbitImage)"
