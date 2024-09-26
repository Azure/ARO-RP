#!/bin/bash -e
######## Helper file with the functions to fully deploy RP dev either locally or using Azure DevOps Pipelines ########
# Run usage_full_rp_funcs to get functions' usage help
source hack/devtools/rp_dev_helper.sh

setup_rp_config() {
  err_str="Usage $0 <AZURE_PREFIX> <GIT_COMMIT> <LOCATION>. Please try again"
  local azure_prefix="${1?$err_str}"
  local git_commit="${2?$err_str}"
  local location="${3?$err_str}"

  # TODO - check if needed
  export USER=dummy
  # Export the SECRET_SA_ACCOUNT_NAME environment variable and run make secrets
  local -r subscription_id="fe16a035-e540-4ab7-80d9-373fa9a3d6ae"
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
  azure_resource_name="${azure_prefix}-aro-$location"
  azure_resource_name="${azure_resource_name:0:20}"
  export RESOURCEGROUP="$azure_resource_name" DATABASE_ACCOUNT_NAME="$azure_resource_name" KEYVAULT_PREFIX="$azure_resource_name"
  # TODO chekc if it is needed here
  export ARO_IMAGE="${azure_prefix}aro.azurecr.io/aro:${git_commit}"

  # Generate new dev-config.yaml
  make dev-config.yaml
  log "Success step 2 ✅ - Config file dev-config.yaml has been created"
}

is_full_rp_succeeded() {
  err_str="Usage $0 <AZURE_PREFIX> <RP_RESOURCE_GROUP> <GWY_RESOURCE_GROUP> <GIT_COMMIT>. Please try again"
  local azure_prefix="${1?$err_str}"
  local rp_resource_group="${2?$err_str}"
  local gwy_resource_group="${3?$err_str}"
  local git_commit="${4?$err_str}"

  local rp_resource_group_info="$(az group show --resource-group "${rp_resource_group}" 2>/dev/null)"
  local gwy_resource_group_info="$(az group show --resource-group "${gwy_resource_group}" 2>/dev/null)"

  if [ -z "${rp_resource_group_info}" ] || [ -z "${gwy_resource_group_info}" ] ; then
    log "🔴❌📦 At least one resourceGroup ('${rp_resource_group}' or '${gwy_resource_group}') does not exist. AZURE_PREFIX='$azure_prefix' was not used."
    log "🟢 No full RP resources interference, you may proceed running the automation"
    return
  fi
  log "AZURE_PREFIX='$azure_prefix' has already been used. There could be interference with another developer"
   
  if ! check_vmss "${rp_resource_group}" "rp-vmss-${git_commit}" || ! check_vmss "${gwy_resource_group}" "gateway-vmss-${git_commit}" \
      || ! check_deployment  "${rp_resource_group}" "rp-production-${git_commit}"  ! check_deployment "${gwy_resource_group}" "gateway-production-${git_commit}" ; then
    log "🔴❌📦 At least one VMSS ('rp-vmss-${git_commit}' or 'gateway-vmss-${git_commit}') does not exist/succeeded or its deployment failed."
    log "🟢 No full RP resources interference, you may proceed running the automation"
    return
  fi
  abort "Both of the resourceGroups and VMSSs exist. 🚀⏭️ skip full RP dev automation and change your AZURE_PREFIX='$azure_prefix', since it is already been used."
}

pre_deploy_resources() {
    err_str="Usage $0 <AZURE_PREFIX> <LOCATION> <RESOURCE_GROUP> [SKIP_DEPLOYMENTS]. Please try again"
    local azure_prefix="${1?$err_str}"
    local location="${2?$err_str}"
    local rp_resource_group="${3?$err_str}"
    local skip_deployments="${4:-"true"}"

    # Don't skip deployment creation when SKIP_DEPLOYMENTS was set to "false" 
    if is_boolean "${skip_deployments}" && [ "${skip_deployments}" = false ]; then
      log "'SKIP_DEPLOYMENTS' env var was set to 'false'.❌⏩ Don't skip predeployment."
      make pre-deploy-no-aks
    else
      local num_deployment=0
      resource_groups=("${azure_prefix}-global" "${azure_prefix}-subscription" "${azure_prefix}-gwy-${location}" "${azure_prefix}-gwy-${location}" "$rp_resource_group" "$rp_resource_group")
      deployments=("rp-global-${location}" "rp-production-subscription-${location}" "gateway-production-predeploy" "gateway-production-managed-identity" "rp-production-managed-identity" "rp-production-predeploy-no-aks")
      for i in "${!deployments[@]}"; do
        check_deployment "${resource_groups[i]}" "${deployments[i]}" && num_deployment="$((num_deployment + 1))"
      done
      if [[ ${num_deployment} -lt 6 ]]; then
        log "Deploy predeployment resources prior to AKS. ${num_deployment}/6 deployments have been deployed." 
        make pre-deploy-no-aks
      else
        log "All the 6 deployments exists. ⏩📋 Predeployment was skipped"
      fi
    fi
    log "Success step 3 ✅ - deploy pre-deployment resources prior to AKS"
}

add_hive(){
  err_str="Usage $0 <LOCATION> <RESOURCE_GROUP> <PULL_SECRET> [SKIP_DEPLOYMENTS]. Please try again"
  local location="${1?$err_str}"
  local resource_group="${2?$err_str}"
  local pull_secret="${3?$err_str}"
  local skip_deployments="${4:-"true"}"

  is_boolean "$skip_deployments"

  source hack/devtools/deploy-shared-env.sh
  if $skip_deployments && check_deployment "${resource_group}" dev-vpn; then
    log "⏩📋 VPN deployment was skipped"
  else
    deploy_vpn_for_dedicated_rp 
    log "Success step 4a 🚀 - VPN has been deployed"
  fi

  if $skip_deployments && check_deployment "${resource_group}" aks-development; then
    log "⏩📋 AKS deployment was skipped"
  else
    deploy_aks_dev 
    log "Success step 4b 🚀 - AKS has been deployed"
  fi

  log "Success step 4 ✅ - VPN & AKS have been deployed"

  vpn_configuration
  screen -dmS connect_dev_vpn bash -c "sudo openvpn secrets/vpn-${location}.ovpn; sleep 10; exec bash" # open new socket to run concurrently
  make aks.kubeconfig
  HOME=/usr KUBECONFIG="$(pwd)/aks.kubeconfig" ./hack/hive/hive-dev-install.sh "${pull_secret}" "${skip_deployments}"
  log "Success step 5 ✅ - Hive has been installed"
}

mirror_images() {
  err_str="Usage $0 <AZURE_PREFIX> <USER_PULL_SECRET> <PULL_SECRET> <GIT_COMMIT> [SKIP_DEPLOYMENTS]. Please try again"
  local azure_prefix="${1?$err_str}"
  local user_pull_secret="${2?$err_str}"
  local pull_secret="${3?$err_str}"
  local git_commit="${4?$err_str}"
  local skip_deployments="${5:-"true"}"

  export DST_ACR_NAME="${azure_prefix}aro"
  export SRC_AUTH_QUAY="$(jq -r '.auths."quay.io".auth' <<< "${user_pull_secret}")"
  export SRC_AUTH_REDHAT="$(jq -r '.auths."registry.redhat.io".auth' <<< "${user_pull_secret}")"
  export DST_AUTH="$(echo -n '00000000-0000-0000-0000-000000000000:'"$(az acr login -n "${DST_ACR_NAME}" -ojson --expose-token | jq -r .accessToken)" | base64 -w0)"
  docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo "$DST_AUTH" | base64 -d | cut -d':' -f2)" "${DST_ACR_NAME}.azurecr.io"
  local acr_string="ACR '${DST_ACR_NAME}'"
  log "Success step 6a ✈️ 🏷️ - Login to ${acr_string}"

  local resource_group_global="${azure_prefix}-global"
  if ( is_boolean "${skip_deployments}" && [ "${skip_deployments}" = false ]) || ! check_acr_repos "$resource_group_global" "$git_commit" "$skip_deployments"; then
    make go-verify # add Vendor directory
    go run -tags containers_image_openpgp,exclude_graphdriver_btrfs ./cmd/aro mirror latest
    log "Success step 6b ✈️ 📦 - Mirror OCP images"

    geneva_prefix="distroless/geneva"
    geneva_names=("MdmImage" "MdsdImage")
    for name in "${geneva_names[@]}"; do
      tag="$(get_repo_tag "${name}")"
      repo="${name/Image/}"
      repo="$geneva_prefix${repo,,}"
      import_geneva_image "$repo" "$tag" "${DST_ACR_NAME}"
    done
    log "Success step 6c ✈️ 📦 - Import MDM and MDSD to ${acr_string}"

    if ! check_acr_repo "${DST_ACR_NAME}" "aro" "${skip_deployments}" "${git_commit}"; then
      make publish-image-aro-multistage
      log "Imported 'aro' to ${acr_string} with tag ${git_commit}"
    else
      log "⏭️📦 Skip importing 'aro' to ${acr_string}, since it already exist with tag ${git_commit}."
    fi
    log "Success step 6d ✈️ 📦 - Build and push ARO image to ${acr_string}"

    fluentbit_tag="$(get_repo_tag "FluentbitImage")"
    if ! check_acr_repo "${DST_ACR_NAME}" "fluentbit" "${skip_deployments}" "${fluentbit_tag}"; then
      copy_digest_tag "${pull_secret}" "arointsvc" "${DST_ACR_NAME}" "fluentbit" "${fluentbit_tag}"
      log "Imported 'fluentbit' to ${acr_string}"
    else
      log "⏭️📦 Skip importing 'fluentbit' to '${acr_string}', since it already exist with tag '${fluentbit_tag}'."
    fi
    log "Success step 6e ✈️ 📦 - Copy Fluenbit image to ${acr_string}"
  
  else
    log "⏩📋 Skip mirroring repos to ${acr_string}" 
  fi
  log "Success step 6 ✅ - Mirror repos to ${acr_string}"
}

prepare_RP_deployment() {
  err_str="Usage $0 <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> [SKIP_DEPLOYMENTS]. Please try again"
  local azure_prefix="${1?$err_str}"
  local git_commit="${2?$err_str}"
  local location="${3?$err_str}"
  local skip_deployments="${4:-"true"}"
  
  # TODO maybe it should be exported
  parent_domain_name="osadev.cloud"
  global_resourcegroup="${azure_prefix}-global"

  for DOMAIN_NAME in ${azure_prefix}-clusters.$parent_domain_name ${azure_prefix}-rp.$parent_domain_name; do
    child_domain_prefix="$(cut -d. -f1 <<<"$DOMAIN_NAME")"
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
  
  if is_boolean "$skip_deployments" && check_and_import_certificates "$KEYVAULT_PREFIX" "$skip_deployments"; then
    log "Success step 7b ✅ - Update certificates in keyvault"
  fi

  if ! check_vmss "${azure_prefix}-aro-${location}" "rp-vmss-${git_commit}" true \
    &&  ! check_vmss "${azure_prefix}-gwy-${location}" "gateway-vmss-${git_commit}" true; then
    log "Success step 7c ✅ - Remove RP VMSS and GWY VMSS"
  fi
  log "Success step 7 ✅ - Final preperation before deploying RP"
}

fully_deploy_resources() {
    err_str="Usage $0 <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> <RESOURCE_GROUP> [SKIP_DEPLOYMENTS]. Please try again"
    local azure_prefix="${1?$err_str}"
    local git_commit="${2?$err_str}"
    local location="${3?$err_str}"
    local rp_resource_group="${4?$err_str}"
    local skip_deployments="${5:-"true"}"

    # Don't skip deployment creation when SKIP_DEPLOYMENTS was set to "false" 
    if is_boolean "${skip_deployments}" && [ "${skip_deployments}" = false ]; then
      log "'SKIP_DEPLOYMENTS' env var was set to 'false'.❌⏩ Don't skip full deployments of RP and GYW."
      FULL_RP_DEV=true make go-verify deploy
    else
      local num_deployment=0
      check_deployment "${azure_prefix}-global" "rp-global-${location}" && num_deployment="$((num_deployment + 1))"
      check_deployment "${azure_prefix}-subscription" "rp-production-subscription-${location}" && num_deployment="$((num_deployment + 1))"
      local gwy_resource_group="${azure_prefix}-gwy-${location}"
      for deployment in "gateway-production-predeploy" "gateway-production-managed-identity" "gateway-production-${git_commit}"; do
        check_deployment "${gwy_resource_group}" "$deployment" && num_deployment="$((num_deployment + 1))"
      done
      for deployment in "rp-production-managed-identity" "rp-production-predeploy" "rpServiceKeyvaultDynamic" "dev-vpn" "aks-development" "rp-production-${git_commit}"; do
        check_deployment "${rp_resource_group}" "$deployment" && num_deployment="$((num_deployment + 1))"
      done

      if [[ ${num_deployment} -lt 11 ]]; then
        log "Fully deploy RP and GYW deployments. ${num_deployment}/11 deployments have been deployed."
        FULL_RP_DEV=true make go-verify deploy
      else
        log "All the 11 deployments exists. ⏩📋 Full deployments of RP and GYW was skipped."
      fi
    fi

    # Verify VMSS have been provisioned successfully
    check_vmss "${rp_resource_group}" "rp-vmss-${git_commit}"
    check_vmss "${gwy_resource_group}" "gateway-vmss-${git_commit}"
    log "Success step 8 ✅ - fully deploy all the resources for ARO RP and GWY VMSSs"
}

usage_full_rp_funcs() {
    cat <<EOF
######## Helper functions for Full RP dev automation ########
Usage: $0 <function_name> [arguments]

Available functions:
  setup_rp_config         - Setup dev-config.yaml file for deploying the RP and GWY VMSSs
  is_full_rp_succeeded    - Get fast feedback whether the automation is needed. Both of the resource groups and VMSSs exist and succeeded (respectively), then skip automation.
  pre_deploy_resources    - Deploy 6 deployments in case they are needed
  add_hive                - Add VPN configuration, install AKS and then install Hive
  mirror_images           - Login and copy many repos (e.g., mdm, mdsd, fluentbit and aro) to your ACR 
  prepare_RP_deployment   - Create DNS records, add certs and remove old VMSSs
  fully_deploy_resources  - Fully deploy 12 deployments in case they are needed

Examples:
  $0 setup_rp_config zzz bfc8993 eastus
  $0 is_full_rp_succeeded zzz zzz-aro-eastus zzz-gwy-eastus bfc8993
  $0 pre_deploy_resources zzz eastus zzz-aro-eastus true
  $0 add_hive eastus zzz-aro-eastus "'{"auths":{"...":{"auth":"..."}}'}" true
  $0 mirror_images zzz "'{"auths":{"...":{"auth":"..."}}'}" true
  $0 prepare_RP_deployment zzz bfc8993 eastus true
  $0 fully_deploy_resources zzz bfc8993 eastus zzz-aro-eastus true

To get detailed usage for a specific function, run:
  $0 usage_full_rp_funcs <function_name>
EOF

    local fun_name="${1-"missing-fun_name"}"
    # Specific function usage
    case "${fun_name}" in
        setup_rp_config)
            echo "Usage: $0 setup_rp_config <AZURE_PREFIX> <GIT_COMMIT> <LOCATION>"
            ;;
        is_full_rp_succeeded)
            echo "Usage: $0 pre_deploy_resources <AZURE_PREFIX> <RP_RESOURCE_GROUP> <GWY_RESOURCE_GROUP> <GIT_COMMIT>"
            ;;
        pre_deploy_resources)
            echo "Usage: $0 pre_deploy_resources <AZURE_PREFIX> <LOCATION> <RESOURCE_GROUP> [SKIP_DEPLOYMENTS]"
            ;;
        add_hive)
            echo "Usage: $0 add_hive <LOCATION> <RESOURCE_GROUP> <PULL_SECRET> [SKIP_DEPLOYMENTS]"
            ;;
        mirror_images)
            echo "Usage: $0 mirror_images <AZURE_PREFIX> <USER_PULL_SECRET> [SKIP_DEPLOYMENTS]"
            ;;
        prepare_RP_deployment)
            echo "Usage: $0 prepare_RP_deployment <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> [SKIP_DEPLOYMENTS]"
            ;;
        fully_deploy_resources)
            echo "Usage: $0 fully_deploy_resources <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> <RESOURCE_GROUP> [SKIP_DEPLOYMENTS]"
            ;;
        *)
            # If no specific function is passed, or an invalid function is passed.
            echo "Specify a valid function name to get more details."
            ;;
    esac
}
