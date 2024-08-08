#!/bin/bash -e
######## Helper file to run full RP dev either locally or using Azure DevOps Pipelines ########

# Function to extract the image tag
extract_image_tag() {
    # Check if exactly two arguments are provided
    if [[ $# -ne 2 ]]; then        
        echo "Error: $0 <FUNCTION_NAME> <FILE_TO_EXTRACT>. Please try again"
        exit 1
    fi
    for arg in "$1" "$2"; do
        if  [[ -z "$arg" ]];  then
            echo "Error: an empty var. Please try again"
            exit 1
        fi
    done

    
    local return_line=$(grep -A 1 "func $1" "$2" | grep 'return')
    echo "$return_line" | sed 's/.*"\(.*\)@sha256.*/\1/'
}

# Function to copy image using skopeo
copy_digest_tag() {
    # Check if exactly non-empty four arguments are provided
    if [[ $# -ne 4 ]]; then        
        echo "Error: $0 <PULL_SECRET> <SRC_ACR_NAME> <DST_ACR_NAME> <IMAGE_TAG>. Please try again"
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

VERSION_CONST_FILE="pkg/util/version/const.go"
# Function to get image name and tag
get_digest_tag() {
    if [[ $# -ne 1 ]]; then        
        echo "Error: $0 <IMAGE_NAME>. Please try again"
        exit 1
    fi
    local IMAGE_NAME=$1
    local IMAGE_TAG=$(extract_image_tag "$IMAGE_NAME" "$VERSION_CONST_FILE")
    echo $IMAGE_TAG
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
# get_digest_tag "FluentbitImage"
# copy_digest_tag "<PULL_SECRET>" "src_acr_name" "dst_acr_name" "$(get_digest_tag FluentbitImage)"
# clean_rp_dev_env "zzz-global zzz-subscription zzz-gwy-eastus zzz-aro-eastus" "zzz-aro-eastus-gwy zzz-aro-eastus-por zzz-aro-eastus-svc zzz-aro-eastus-cls" 