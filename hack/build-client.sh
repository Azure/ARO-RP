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

  sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/"$FOLDER"/"$API_VERSION"/redhatopenshift.json >> .sha256sum
}

function generate_golang() {
  local AUTOREST_IMAGE=$1
  local API_VERSION=$2
  local FOLDER=$3

  # Generating Track 1 SDK. Needs work to migrate to Track 2.
  docker run -d -t \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
    -v $PWD/swagger:/swagger:z \
    "${AUTOREST_IMAGE}" \
    --go \
    --use=@autorest/go@4.0.0-preview.45 \
    --use=@autorest/modelerfour@~4.26.0 \
    --version=~3.6.3 \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --openapi-type=data-plane \
    --namespace=redhatopenshift \
    --verbose \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift

  docker run -d -t \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
    --entrypoint sed \
    "${AUTOREST_IMAGE}" \
    --in-place \
    --expression='s|azure/aro-rp|Azure/ARO-RP|g' \
    "/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/${API_VERSION}/redhatopenshift/models.go" \
    "/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/${API_VERSION}/redhatopenshift/redhatopenshiftapi/interfaces.go"

  go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP pkg/client
}

function generate_python() {
  local AUTOREST_IMAGE=$1
  local API_VERSION=$2
  local FOLDER=$3

  # Generating Track 2 SDK
  docker run \
    --platform=linux/amd64 \
    --rm \
    -v $PWD/python/client:/python/client:z \
    -v $PWD/swagger:/swagger:z \
    "${AUTOREST_IMAGE}" \
    --use=@autorest/python@~5.12.0 \
    --use=@autorest/modelerfour@~4.20.0 \
    --version=~3.6.3 \
    --python \
    --azure-arm \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/python/client

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"/aio
  >python/client/__init__.py
}

if [ -f .sha256sum ]; then
  rm .sha256sum
fi

AUTOREST_IMAGE=$1

for API_VERSION in "${@: 2}"
do
  FOLDER=stable
  if [[ "$API_VERSION" =~ .*preview ]]; then
    FOLDER=preview
  fi

  printf "\nGENERATING $API_VERSION FROM $FOLDER\n"
  printf '%*s\n' "${COLUMNS:-$(tput cols)}" '' | tr ' ' -

  printf "CLEANING OLD API GENERATED FILES...\n"
  clean "$API_VERSION" "$FOLDER"
  printf "COMPLETED SUCCESSFULLY\n"

  printf "GENERATING CHECKSUM...\n"
  checksum "$API_VERSION" "$FOLDER"
  printf "COMPLETED SUCCESSFULLY\n"

  printf "GENERATING GOLANG...\n"
  generate_golang "$AUTOREST_IMAGE" "$API_VERSION" "$FOLDER"
  printf "COMPLETED SUCCESSFULLY\n"

  printf "GENERATING PYTHON...\n"
  generate_python "$AUTOREST_IMAGE" "$API_VERSION" "$FOLDER"
  printf "COMPLETED SUCCESSFULLY\n"
done
