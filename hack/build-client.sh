#!/bin/bash -e

function clean() {
  local API_VERSION=$1
  local FOLDER=$2

  rm -rf pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"
  mkdir pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
  mkdir -p python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
}

function checksum() {
  local API_VERSION=$1
  local FOLDER=$2

  sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json >>.sha256sum
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
    --use=@autorest/python@~5.12.0 \
    --use=@autorest/modelerfour@~4.20.0 \
    --modelerfour.lenient-model-deduplication=true \
    --version=~3.6.2 \
    --python \
    --azure-arm \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/python/client

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"/aio
  >python/client/__init__.py
}

if [ -f .sha256sum ]; then
  rm .sha256sum
fi

AUTOREST_IMAGE=$1

for API_VERSION in "${@:2}"; do
  FOLDER=stable
  if [[ "$API_VERSION" =~ .*preview ]]; then
    FOLDER=preview
  fi

  printf "\nGENERATING API v$API_VERSION\n"
  printf "%*s\n" "${COLUMNS:-$(tput cols)}" "" | tr " " -

  printf "CLEANING OLD API GENERATED FILES...\n"
  clean "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"

  printf "GENERATING CHECKSUM...\n"
  checksum "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"

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
