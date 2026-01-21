#!/bin/bash -e
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

# Function to login to ACR using PULL_SECRET - (ACR_NAME)
acr_login() {
    local ACR_NAME=${1:-arointsvc}
    local REGISTRY="$ACR_NAME.azurecr.io"
    
    if podman login --get-login "$REGISTRY" &>/dev/null; then
        echo ">> Already logged into $REGISTRY"
        return 0
    fi
    
    if [ -z "$PULL_SECRET" ]; then
        echo ">> PULL_SECRET not set, cannot login to $REGISTRY, please run 'make pull-secrets' and source the env file"
        return 1
    fi
    
    local AUTH=$(echo "$PULL_SECRET" | jq -r '.auths["'$REGISTRY'"].auth' | base64 -d)
    podman login "$REGISTRY" -u "${AUTH%%:*}" -p "${AUTH#*:}"
}

# Example usage
# get_digest_tag "FluentbitImage"
# copy_digest_tag "<PULL_SECRET>" "src_acr_name" "dst_acr_name" "$(get_digest_tag FluentbitImage)"
# acr_login "arointsvc"