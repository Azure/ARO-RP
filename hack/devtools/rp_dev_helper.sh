#!/bin/bash -e
######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########

# Function to extract the image tag - (FUNCTION_NAME)
extract_image_tag() {
    # Extract the line containing the return statement
    local return_line=$(grep -A 1 "func $1" "$2" | grep 'return')
    echo "$return_line" | sed 's/.*"\(.*\)@sha256.*/\1/'
}

# Function to copy image using skopeo
# Usage: copy_digest_tag "$PULL_SECRET" "src_acr" "dst_acr" "$(get_digest_tag FluentbitImage)"
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
# Usage: get_digest_tag "FluentbitImage"
get_digest_tag() {
    local IMAGE_NAME=$1
    local IMAGE_TAG=$(extract_image_tag "$IMAGE_NAME" "$VERSION_CONST_FILE")
    echo "$IMAGE_NAME and Tag: $IMAGE_TAG"
    echo "$IMAGE_TAG"
}

# Prepare only the MISE health-monitor image; do not change shared aro mirror behavior.
mirror_mise_otel_image() {
    if [[ $# -ne 1 || ! "$1" =~ ^[a-zA-Z0-9]{5,50}$ ]]; then
        echo "Usage: mirror_mise_otel_image <destination-acr-name>" >&2
        return 1
    fi

    local pullspec
    if ! pullspec=$(sed -n '/^func OTelImage(/,/^}/s/^[[:space:]]*return acrDomain + "\([^"]*\)"[[:space:]]*$/\1/p' "$VERSION_CONST_FILE"); then
        echo "Unable to read the MISE OTEL image from $VERSION_CONST_FILE" >&2
        return 1
    fi
    if [[ ! "$pullspec" =~ ^/oss/otel/opentelemetry-collector-contrib@sha256:[0-9a-f]{64}$ ]]; then
        echo "Expected a single digest-pinned OTelImage reference in $VERSION_CONST_FILE" >&2
        return 1
    fi

    local repository="${pullspec#/}"
    repository="${repository%@*}"
    local digest="${pullspec##*@}"
    # A digest-derived tag keeps the required manifest out of untagged-image retention.
    if ! az acr import --name "$1" --source "mcr.microsoft.com${pullspec}" \
        --image "${repository}:${digest#sha256:}" --force --only-show-errors --output none; then
        echo "Failed to import MISE OTEL into ACR $1" >&2
        return 1
    fi

    local imported_digest
    if ! imported_digest=$(az acr repository show --name "$1" --image "${pullspec#/}" \
        --query digest --output tsv --only-show-errors); then
        echo "Unable to verify the MISE OTEL digest in ACR $1" >&2
        return 1
    fi
    if [[ "$imported_digest" != "$digest" ]]; then
        echo "MISE OTEL digest mismatch in ACR $1: expected $digest, got $imported_digest" >&2
        return 1
    fi
}

# Function to login to ACR using PULL_SECRET
# Usage: acr_login "arointsvc"
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
    echo "${AUTH#*:}" | podman login "$REGISTRY" -u "${AUTH%%:*}" --password-stdin
}