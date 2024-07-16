#!/bin/bash -e

source ./hack/devtools/deploy-shared-env.sh
deploy_vpn_for_dedicated_rp
deploy_aks_dev
echo "Success step 4 - VPN and AKS have been deployed"

vpn_configuration
sudo openvpn secrets/vpn-$location.ovpn &
make aks.kubeconfig
./hack/hive-generate-config.sh
KUBECONFIG=$(pwd)/aks.kubeconfig ./hack/hive-dev-install.sh
echo "Success step 5 - VPN and AKS have been deployed"

export SRC_AUTH_QUAY=$(echo $USER_PULL_SECRET | jq -r '.auths."quay.io".auth')
export SRC_AUTH_REDHAT=$(echo $USER_PULL_SECRET | jq -r '.auths."registry.redhat.io".auth')
export DST_AUTH=$(echo -n '00000000-0000-0000-0000-000000000000:'$(az acr login -n $DST_ACR_NAME --expose-token | jq -r .accessToken) | base64 -w0)
docker login -u 00000000-0000-0000-0000-000000000000 -p "$(echo $DST_AUTH | base64 -d | cut -d':' -f2)" "$DST_ACR_NAME.azurecr.io"
echo "Success step 6a - Login to ACR"

go run ./cmd/aro mirror latest
echo "Success step 6b - Mirror OCP images"

mdm_image_tag=$(get_digest_tag "MdmImage")
mdsd_image_tag=$(get_digest_tag "MdsdImage")
az acr import --name $DST_ACR_NAME.azurecr.io/${mdm_image_tag} --source linuxgeneva-microsoft.azurecr.io/${mdm_image_tag}
az acr import --name $DST_ACR_NAME.azurecr.io/${mdsd_image_tag} --source linuxgeneva-microsoft.azurecr.io/${mdsd_image_tag}
echo "Success step 6c - Import MDM and MDSD to ACR"

make publish-image-aro-multistage
echo "Success step 6d - Build, push and import to ARO image to ACR"

source ./hack/rp-dev-helper.sh
fluentbit_image_tag=$(get_digest_tag "FluentbitImage")
copy_digest_tag $PULL_SECRET, "arointsvc", $DST_ACR_NAME, $fluentbit_image_tag
echo "Success step 6e - Copy Fluenbit image to ACR"

export PARENT_DOMAIN_NAME=osadev.cloud
export PARENT_DOMAIN_RESOURCEGROUP=dns
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
echo "Success step 7a - Update DNS Child Domains and certificates in keyvault"

gitCommit=$(git rev-parse --short=7 HEAD)
az vmss delete -g $(AZURE_PREFIX)-aro-$LOCATION --name rp-vmss-$gitCommit  --force-deletion
az vmss delete -g $(AZURE_PREFIX)-gwy-$LOCATION --name gateway-vmss-$gitCommit  --force-deletion
echo "Success step 7b - Remove RP VMSS and GWY VMSS"
