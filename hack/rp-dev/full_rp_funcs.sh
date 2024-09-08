#!/bin/bash -e
######## Helper file with the functions to fully deploy RP dev either locally or using Azure DevOps Pipelines ########

source hack/devtools/rp_dev_helper.sh

setup_rp_config() {
  # Check if exactly non-empty two arguments are provided
  if [[ $# -ne 2 ]]; then
    echo "Error: $0 <AZURE_PREFIX>. Please try again"
    exit 1
  fi

  # TODO - check if needed
  export USER=dummy
  # Export the SECRET_SA_ACCOUNT_NAME environment variable and run make secrets
  az account set -s fe16a035-e540-4ab7-80d9-373fa9a3d6ae
  export SECRET_SA_ACCOUNT_NAME=rharosecretsdev && make secrets

  source hack/devtools/rp-dev-helper.sh && verify_downloading_secrets

  export LOCATION=eastus
  # Source environment variables from the secrets file
  source secrets/env
  echo -e "Success step 1 ✅ - Directory '$expected_dir' has been created with files - ${secret_files[@]}\n"

  export AZURE_PREFIX=$1 NO_CACHE=true ARO_INSTALL_VIA_HIVE=true ARO_ADOPT_BY_HIVE=true DATABASE_NAME=ARO

  azure_resource_name=${AZURE_PREFIX}-aro-$LOCATION
  # TODO truncate to 20 characters
  export RESOURCEGROUP=$azure_resource_name DATABASE_ACCOUNT_NAME=$azure_resource_name KEYVAULT_PREFIX=$azure_resource_name
  gitCommit=$(git rev-parse --short=7 HEAD)
  export ARO_IMAGE=${AZURE_PREFIX}aro.azurecr.io/aro:$gitCommit

  # Run the make command to generate dev-config.yaml
  make dev-config.yaml

  # Check if the dev-config.yaml file exists
  [ -f "dev-config.yaml" ] || { echo "File dev-config.yaml does not exist."; exit 1; }
  echo -e "Success step 2 ✅ - Config file dev-config.yaml has been created\n"

}

# Function to predeploy 6 deployments in case they are needed
pre_deploy_resources() {
    num_deployment=0
    check_azure_deployment ${AZURE_PREFIX}-global rp-global-${LOCATION} && num_deployment=$(( num_deployment + 1))
    check_azure_deployment ${AZURE_PREFIX}-subscription rp-production-subscription-${LOCATION} && num_deployment=$(( num_deployment + 1))
    for deployment in "gateway-production-predeploy" "gateway-production-managed-identity"; do
        check_azure_deployment ${AZURE_PREFIX}-gwy-${LOCATION} $deployment  && num_deployment=$(( num_deployment + 1))
    done
    for deployment in "rp-production-managed-identity" "rp-production-predeploy-no-aks"; do
        check_azure_deployment ${RESOURCEGROUP} $deployment  && num_deployment=$(( num_deployment + 1))
    done

    if [[ $num_deployment -lt 6 ]]; then
        echo -e "deploy predeployment resources prior to AKS.\n" 
        make pre-deploy-no-aks
    else
        echo -e "All the six deployments exists. ⏩📋 Predeployment was skipped.\n"
    fi
    echo -e "Success step 3 ✅ - deploy pre-deployment resources prior to AKS\n"
}

add_hive(){
  source hack/devtools/deploy-shared-env.sh
  check_azure_deployment ${RESOURCEGROUP} dev-vpn && echo "⏩📋 VPN deployment was skipped" \
  || deploy_vpn_for_dedicated_rp && echo "Success step 4a 🚀 - VPN has been deployed"
  check_azure_deployment ${RESOURCEGROUP} aks-development && echo "⏩📋 VPN deployment was skipped" \
  || deploy_aks_dev && echo "Success step 4b 🚀 - AKS has been deployed"
  echo -e "Success step 4 ✅ - VPN & AKS have been deployed\n"

  vpn_configuration
  screen -dmS connect_dev_vpn bash -c "sudo openvpn secrets/vpn-$LOCATION.ovpn; sleep 10; exec bash" # open new socket to run concurrently
  make aks.kubeconfig
  HOME=/usr ./hack/hive-generate-config.sh # HOME is default to /usr which isn't in PATH
  KUBECONFIG=$(pwd)/aks.kubeconfig ./hack/hive-dev-install.sh
  echo -e "Success step 5 ✅ - Hive has been installed\n"
}

mirror_images() {
  export DST_ACR_NAME=${AZURE_PREFIX}aro
  export SRC_AUTH_QUAY=$(echo $USER_PULL_SECRET | jq -r '.auths."quay.io".auth')
  export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
  export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n $DST_ACR_NAME --expose-token | jq -r .accessToken) | base64 -w0)
  docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "$DST_ACR_NAME.azurecr.io"
  echo "Success step 6a ✈️ 🏷️ - Login to ACR"

  if ! check_acr_repos ${AZURE_PREFIX}-global; then
    make go-verify
    echo "Success step 6b - Add Vendor directory"

    go run -tags containers_image_openpgp,exclude_graphdriver_btrfs ./cmd/aro mirror latest
    echo "Success step 6c ✈️ 📦 - Mirror OCP images"

    mdm_image_tag=$(get_digest_tag "MdmImage")
    mdm_repo=$(echo $mdm_image_tag | sed -n 's|.*/\([^/]*\/[^:]*\):.*|\1|p')
    mdsd_image_tag=$(get_digest_tag "MdsdImage")
    mdsd_repo=$(echo $mdsd_image_tag | sed -n 's|.*/\([^/]*\/[^:]*\):.*|\1|p')
    import_geneva_image $mdm_repo $mdm_image_tag
    import_geneva_image $mdm_repo $mdm_image_tag
    echo "Success step 6d ✈️ 📦 - Import MDM and MDSD to ACR"

    make publish-image-aro-multistage
    echo "Success step 6e ✈️ 📦 - Build, push and import to ARO image to ACR"

    fluentbit_image_tag=$(get_digest_tag "FluentbitImage")
    copy_digest_tag $PULL_SECRET "arointsvc" $DST_ACR_NAME $fluentbit_image_tag
    echo "Success step 6f ✈️ 📦 - Copy Fluenbit image to ACR"
  else
    echo "⏩📋 Skip mirroring repos to ACR $DST_ACR_NAME" 
  fi
  echo -e "Success step 6 ✅ - Mirror repos to ACR $DST_ACR_NAME\n"
}

prepare_RP_deployment() {
  export PARENT_DOMAIN_NAME=osadev.cloud
  export GLOBAL_RESOURCEGROUP=${AZURE_PREFIX}-global

  for DOMAIN_NAME in ${AZURE_PREFIX}-clusters.$PARENT_DOMAIN_NAME ${AZURE_PREFIX}-rp.$PARENT_DOMAIN_NAME; do
    CHILD_DOMAIN_PREFIX="$(cut -d. -f1 <<<$DOMAIN_NAME)"
    echo "########## Creating NS record to DNS Zone $CHILD_DOMAIN_PREFIX ##########"
    az network dns record-set ns create \
      --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
      --zone "$PARENT_DOMAIN_NAME" \
      --name "$CHILD_DOMAIN_PREFIX" >/dev/null
    for ns in $(az network dns zone show \
      --resource-group "$GLOBAL_RESOURCEGROUP" \
      --name "$DOMAIN_NAME" \
      --query nameServers -o tsv); do
      az network dns record-set ns add-record \
      --resource-group "$PARENT_DOMAIN_RESOURCEGROUP" \
      --zone "$PARENT_DOMAIN_NAME" \
      --record-set-name "$CHILD_DOMAIN_PREFIX" \
      --nsdname "$ns" >/dev/null
    done
  done
  echo "Success step 7a ✅ - Update DNS Child Domains"

  check_and_import_certificates
  echo "Success step 7b ✅ - Update certificates in keyvault"

  gitCommit=$(git rev-parse --short=7 HEAD)
  az vmss delete --resource-group ${AZURE_PREFIX}-aro-$LOCATION --name rp-vmss-$gitCommit --force-deletion
  az vmss delete --resource-group ${AZURE_PREFIX}-gwy-$LOCATION --name gateway-vmss-$gitCommit --force-deletion
  echo "Success step 7c ✅ - Remove RP VMSS and GWY VMSS"
  echo -e "Success step 7 ✅ - Final preperation before deploying RP\n"
}

# Function to fully deploy 12 deployments in case they are needed
fully_deploy_resources() {
    num_deployment=0
    check_azure_deployment ${AZURE_PREFIX}-global rp-global-${LOCATION} && num_deployment=$(( num_deployment + 1))
    check_azure_deployment ${AZURE_PREFIX}-subscription rp-production-subscription-${LOCATION} && num_deployment=$(( num_deployment + 1))

    gitCommit=$(git rev-parse --short=7 HEAD)
    for deployment in "gateway-production-predeploy" "gateway-production-managed-identity" "gateway-production-${gitCommit}"; do
        check_azure_deployment ${AZURE_PREFIX}-gwy-${LOCATION} $deployment  && num_deployment=$(( num_deployment + 1))
    done
    for deployment in "rp-production-managed-identity" "rp-production-predeploy" "rpServiceKeyvaultDynamic" "dev-vpn" "aks-development" "rp-production-${gitCommit}"; do
        check_azure_deployment ${RESOURCEGROUP} $deployment  && num_deployment=$(( num_deployment + 1))
    done

    if [[ $num_deployment -lt 11 ]]; then
        echo -e "Fully deploy RP and GYW deployments.\n" 
        make go-verify deploy
    else
        echo -e "All the 11 deployments exists. ⏩📋 Full deployments of RP and GYW was skipped.\n"
    fi

    echo "Success step 8 ✅ - fully deploy all the resources for ARO RP and GWY VMSSs"
}

# Example usage
# setup_rp_config <AZURE_PREFIX>
# pre_deploy_resources
# add_hive
# mirror_images
# prepare_RP_deployment
# fully_deploy_resources
