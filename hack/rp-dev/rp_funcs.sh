#!/bin/bash -e
######## Helper file with the functions to fully deploy RP dev either locally or using Azure DevOps Pipelines ########

source hack/devtools/rp_dev_helper.sh

setup_rp_config() {
  err_str="Usage $0 <AZURE_PREFIX> <GIT_COMMIT> <LOCATION>. Please try again"
  local azure_prefix=${1?$err_str}
  local git_commit=${2?$err_str}
  local location=${3?$err_str}

  # TODO - check if needed
  export USER=dummy
  # Export the SECRET_SA_ACCOUNT_NAME environment variable and run make secrets
  readonly subscription_id="fe16a035-e540-4ab7-80d9-373fa9a3d6ae"
  az account set -s $subscription_id
  log "Using Azure subscription $subscription_id"
  SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
  verify_downloading_secrets

  export LOCATION=$location
  # Source environment variables from the secrets file
  source secrets/env
  # log "Success step 1 ✅ - Directory '$secrets_dir' has been created with files - ${secrets_files[@]}"
  log "Success step 1 ✅ - Directory '$secrets_dir' has been created."

  export AZURE_PREFIX="$azure_prefix" NO_CACHE="true" ARO_INSTALL_VIA_HIVE="true" ARO_ADOPT_BY_HIVE="true" DATABASE_NAME="ARO"
  azure_resource_name=${azure_prefix}-aro-$location
  azure_resource_name="${azure_resource_name:0:20}"
  export RESOURCEGROUP="$azure_resource_name" DATABASE_ACCOUNT_NAME="$azure_resource_name" KEYVAULT_PREFIX="$azure_resource_name"

  export ARO_IMAGE="${azure_prefix}aro.azurecr.io/aro:${git_commit}"

  # Run the make command to generate dev-config.yaml
  make dev-config.yaml

  # Check if the dev-config.yaml file exists
  if [ ! -f "dev-config.yaml" ]; then
    abort "File dev-config.yaml does not exist."
  fi
  log "Success step 2 ✅ - Config file dev-config.yaml has been created"
}

# Function to predeploy 6 deployments in case they are needed
pre_deploy_resources() {
    err_str="Usage $0 <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]. Please try again"
    local azure_prefix=${1?$err_str}
    local rp_resource_group=${2?$err_str}
    local location=${3?$err_str}
    local skip_deployments=${4?$err_str}
    if  [[ -z "$4" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if ! $skip_deployments; then
        log "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip predeployment."
        make pre-deploy-no-aks
    fi
  
    num_deployment=0
    resource_groups=("${azure_prefix}-global" "${azure_prefix}-subscription" "${azure_prefix}-gwy-${location}" "${azure_prefix}-gwy-${location}" "$rp_resource_group" "$rp_resource_group")
    deployments=("rp-global-${location}" "rp-production-subscription-${location}" "gateway-production-predeploy" "gateway-production-managed-identity" "rp-production-managed-identity" "rp-production-predeploy-no-aks")
    for i in "${!deployments[@]}"; do
      check_deployment ${resource_groups[i]} ${deployments[i]} && num_deployment=$(( num_deployment + 1))
    done
    # resource_group_global="${azure_prefix}-global"
    # resource_group_sub="${azure_prefix}-subscription"
    # resource_group_gwy="${azure_prefix}-subscription"
    # deployments=(rp-global-${location})
    
    # local_deployment="rp-global-${location}"
    # check_deployment resource_group_global local_deployment && num_deployment=$(( num_deployment + 1))
    # local_deployment="rp-production-subscription-${location}"
    # check_deployment resource_group_sub local_deployment && num_deployment=$(( num_deployment + 1))
    # for deployment in "gateway-production-predeploy" "gateway-production-managed-identity"; do
    #     check_deployment resource_group_gwy deployment  && num_deployment=$(( num_deployment + 1))
    # done
    # for deployment in "rp-production-managed-identity" "rp-production-predeploy-no-aks"; do
    #     check_deployment resource_group deployment  && num_deployment=$(( num_deployment + 1))
    # done

    if [[ $num_deployment -lt 6 ]]; then
        log "deploy predeployment resources prior to AKS." 
        make pre-deploy-no-aks
    else
        log "All the 6 deployments exists. ⏩📋 Predeployment was skipped"
    fi
    log "Success step 3 ✅ - deploy pre-deployment resources prior to AKS"
}

add_hive(){
  err_str="Usage $0 <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]. Please try again"
  local resource_group=${1?$err_str}
  local location=${2?$err_str}
  local skip_deployments=${3?$err_str}
  
  if  [[ -z "$skip_deployments" ]];  then
    log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
    skip_deployments=true
  fi
  is_it_boolean $skip_deployments  

  source hack/devtools/deploy-shared-env.sh
  if $skip_deployments && check_deployment ${resource_group} dev-vpn; then
    log "⏩📋 VPN deployment was skipped"
  else
    deploy_vpn_for_dedicated_rp 
    log "Success step 4a 🚀 - VPN has been deployed"
  fi

  if $skip_deployments && check_deployment ${resource_group} aks-development; then
    log "⏩📋 AKS deployment was skipped"
  else
    deploy_aks_dev 
    log "Success step 4b 🚀 - AKS has been deployed"
  fi

  log "Success step 4 ✅ - VPN & AKS have been deployed"

  vpn_configuration
  screen -dmS connect_dev_vpn bash -c "sudo openvpn secrets/vpn-${location}.ovpn; sleep 10; exec bash" # open new socket to run concurrently
  make aks.kubeconfig
  HOME=/usr ./hack/hive-generate-config.sh # HOME is default to /usr which isn't in PATH
  KUBECONFIG=$(pwd)/aks.kubeconfig ./hack/hive-dev-install.sh
  log "Success step 5 ✅ - Hive has been installed"
}

mirror_images() {
  err_str="Usage $0 <AZURE_PREFIX> <USER_PULL_SECRET> [SKIP_DEPLOYMENTS]. Please try again"
  local azure_prefix=${1?$err_str}
  local user_pull_secret=${3?$err_str}
  local skip_deployments=${3?$err_str}

  dst_acr_name=${AZURE_PREFIX}aro
  export SRC_AUTH_QUAY=$(jq -r '.auths."quay.io".auth' <<< $user_pull_secret)
  export SRC_AUTH_REDHAT=$(jq -r '.auths."registry.redhat.io".auth' <<< $user_pull_secret)
  export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n $dst_acr_name --expose-token | jq -r .accessToken) | base64 -w0)
  docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "$dst_acr_name.azurecr.io"
  log "Success step 6a ✈️ 🏷️ - Login to ACR"

  if  [[ -z "$skip_deployments" ]];  then
    log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
    skip_deployments=true
  fi
  is_it_boolean $skip_deployments  

  if ! $skip_deployments || ! check_acr_repos ${AZURE_PREFIX}-global $skip_deployments; then
    make go-verify
    log "Success step 6b - Add Vendor directory"

    go run -tags containers_image_openpgp,exclude_graphdriver_btrfs ./cmd/aro mirror latest
    log "Success step 6c ✈️ 📦 - Mirror OCP images"

    mdm_image_tag=$(get_digest_tag "MdmImage")
    mdm_repo=$(sed -n 's|.*/\([^/]*\/[^:]*\):.*|\1|p' <<< $mdm_image_tag)
    mdsd_image_tag=$(get_digest_tag "MdsdImage")
    mdsd_repo=$(sed -n 's|.*/\([^/]*\/[^:]*\):.*|\1|p' <<< $mdsd_image_tag)
    import_geneva_image $mdm_repo $mdm_image_tag $dst_acr_name
    import_geneva_image $mdm_repo $mdm_image_tag
    log "Success step 6d ✈️ 📦 - Import MDM and MDSD to ACR"

    make publish-image-aro-multistage
    log "Success step 6e ✈️ 📦 - Build, push and import to ARO image to ACR"

    fluentbit_image_tag=$(get_digest_tag "FluentbitImage")
    copy_digest_tag $PULL_SECRET "arointsvc" $dst_acr_name $fluentbit_image_tag
    log "Success step 6f ✈️ 📦 - Copy Fluenbit image to ACR"
  else
    log "⏩📋 Skip mirroring repos to ACR $dst_acr_name" 
  fi
  log "Success step 6 ✅ - Mirror repos to ACR $dst_acr_name"
}

prepare_RP_deployment() {
  err_str="Usage $0 <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> [SKIP_DEPLOYMENTS]. Please try again"
  local azure_prefix=${1?$err_str}
  local git_commit=${2?$err_str}
  local location=${3?$err_str}

  # TODO maybe it should be exported
  parent_domain_name="osadev.cloud"
  global_resourcegroup="${azure_prefix}-global"

  for DOMAIN_NAME in ${azure_prefix}-clusters.$parent_domain_name ${azure_prefix}-rp.$parent_domain_name; do
    child_domain_prefix="$(cut -d. -f1 <<<$DOMAIN_NAME)"
    log "########## Creating NS record to DNS Zone $child_domain_prefix ##########"
    az network dns record-set ns create \
      --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
      --zone "$parent_domain_name" \
      --name "$child_domain_prefix" >/dev/null
    for ns in $(az network dns zone show \
      --resource-group "$global_resourcegroup" \
      --name "$DOMAIN_NAME" \
      --query nameServers -o tsv); do
      az network dns record-set ns add-record \
      --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
      --zone "$parent_domain_name" \
      --record-set-name "$child_domain_prefix" \
      --nsdname "$ns" >/dev/null
    done
  done
  log "Success step 7a ✅ - Update DNS Child Domains"

  if check_and_import_certificates $KEYVAULT_PREFIX $4; then
    log "Success step 7b ✅ - Update certificates in keyvault"
  fi

  rp_vmss_deleted="${az vmss delete --resource-group "${azure_prefix}-aro-$location" --name "rp-vmss-$git_commit" --force-deletion}"
  gwy_vmss_deleted="${az vmss delete --resource-group "${azure_prefix}-gwy-$location" --name "gateway-vmss-$git_commit" --force-deletion}"
  if rp_vmss_deleted && gwy_vmss_deleted; then
    log "Success step 7c ✅ - Remove RP VMSS and GWY VMSS"
  fi
  log "Success step 7 ✅ - Final preperation before deploying RP"
}

# Function to fully deploy 12 deployments in case they are needed
fully_deploy_resources() {
    err_str="Usage $0 <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]. Please try again"
    local azure_prefix=${1?$err_str}
    local resource_group=${2?$err_str}
    local location=${3?$err_str}
    local skip_deployments=${4?$err_str}
    if  [[ -z "$4" ]];  then
        log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
    fi
    is_it_boolean $skip_deployments
    # Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
    if ! $skip_deployments; then
        log "'SKIP_DEPLOYMENTS' env var is set to false. Don't skip full deployments of RP and GYW."
        make go-verify deploy
    fi

    num_deployment=0
    check_deployment ${AZURE_PREFIX}-global rp-global-${LOCATION} && num_deployment=$(( num_deployment + 1))
    check_deployment ${AZURE_PREFIX}-subscription rp-production-subscription-${LOCATION} && num_deployment=$(( num_deployment + 1))

    git_commit="$(git rev-parse --short=7 HEAD)"
    for deployment in "gateway-production-predeploy" "gateway-production-managed-identity" "gateway-production-${git_commit}"; do
        check_deployment ${AZURE_PREFIX}-gwy-${LOCATION} $deployment  && num_deployment=$(( num_deployment + 1))
    done
    for deployment in "rp-production-managed-identity" "rp-production-predeploy" "rpServiceKeyvaultDynamic" "dev-vpn" "aks-development" "rp-production-${git_commit}"; do
        check_deployment ${RESOURCEGROUP} $deployment  && num_deployment=$(( num_deployment + 1))
    done

    if [[ $num_deployment -lt 11 ]]; then
        log "Fully deploy RP and GYW deployments." 
        make go-verify deploy
    else
        log "All the 11 deployments exists. ⏩📋 Full deployments of RP and GYW was skipped."
    fi

    # Verify VMSS have been provisioned successfully
    git_commit="$(git rev-parse --short=7 HEAD)"
    check_vmss ${AZURE_PREFIX}-aro-$LOCATION rp-vmss-$git_commit
    check_vmss ${AZURE_PREFIX}-gwy-$LOCATION gateway-vmss-$git_commit
    log "Success step 8 ✅ - fully deploy all the resources for ARO RP and GWY VMSSs"
}

# Example usage
# setup_rp_config <AZURE_PREFIX> <GIT_COMMIT> <LOCATION>
# pre_deploy_resources <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
# add_hive <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
# mirror_images <AZURE_PREFIX> <USER_PULL_SECRET> [SKIP_DEPLOYMENTS]
# prepare_RP_deployment <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> [SKIP_DEPLOYMENTS]
# fully_deploy_resources <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
