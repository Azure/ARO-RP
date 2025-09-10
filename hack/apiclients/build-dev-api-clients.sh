#!/bin/bash -e

function make_folders() {
  local API_VERSION=$1
  local FOLDER=$2

  mkdir -p pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"
  mkdir -p python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
}

function generate_golang() {
  local AUTOREST_IMAGE=$1
  local API_VERSION=$2
  local FOLDER=$3

  # Generating Track 1 Golang SDK
  # Needs work to migrate to Track 2
  docker run \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
    -v $PWD/swagger:/swagger:z \
    "${AUTOREST_IMAGE}" \
    --go \
    --use=@microsoft.azure/autorest.go@~2.1.187 \
    --use=@microsoft.azure/autorest.modeler@~2.3.38 \
    --version=~2.0.4421 \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --namespace=redhatopenshift \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift

  docker run \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
    --entrypoint sed \
    "${AUTOREST_IMAGE}" \
    --in-place \
    --expression='s|azure/aro-rp|Azure/ARO-RP|g' \
    "/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/${API_VERSION}/redhatopenshift/models.go" \
    "/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/${API_VERSION}/redhatopenshift/redhatopenshiftapi/interfaces.go"

  goimports -w -local=github.com/Azure/ARO-RP pkg/client
}

function generate_python() {
  local AUTOREST_IMAGE=$1
  local API_VERSION=$2
  local FOLDER=$3

  # Generating Track 2 Python SDK
  docker run \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/python/client:/python/client:z \
    -v $PWD/swagger:/swagger:z \
    "${AUTOREST_IMAGE}" \
    --use=@autorest/python@~6.19.0 \
    --use=@autorest/modelerfour@~4.27.0 \
    --version=3.10.2 \
    --modelerfour.lenient-model-deduplication=true \
    --python \
    --no-async=true \
    --azure-arm \
    --models-mode=msrest \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/python/client

  >python/client/__init__.py
}

AUTOREST_IMAGE=$1

printf "CLEANING OLD API GENERATED FILES...\n"
# Remove golang clients
rm -rf pkg/client/services/redhatopenshift/mgmt
# Remove Python clients
rm -rf python/client/azure/mgmt/redhatopenshift/v*
printf "[\u2714] SUCCESS\n\n"

for API_VERSION in "${@:2}"; do
  FOLDER=stable
  if [[ "$API_VERSION" =~ .*preview ]]; then
    FOLDER=preview
  fi

  printf "\nGENERATING API v$API_VERSION\n"
  printf "%*s\n" "${COLUMNS:-$(tput cols)}" "" | tr " " -
  make_folders "$API_VERSION" "$FOLDER"

  printf "GENERATING GOLANG SDK...\n"
  generate_golang "$AUTOREST_IMAGE" "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"

  printf "GENERATING PYTHON SDK...\n"
  generate_python "$AUTOREST_IMAGE" "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"
  printf "%*s\n" "${COLUMNS:-$(tput cols)}" "" | tr " " -
  printf "\n"
done

printf "[\u2714] CLIENT GENERATION COMPLETED SUCCESSFULLY\n"
