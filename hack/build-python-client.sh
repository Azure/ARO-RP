#!/bin/bash -e

function clean() {
  local API_VERSION=$1
  local FOLDER=$2

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
  mkdir -p python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
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
    --azure-arm \
    --models-mode=msrest \
    --license-header=MICROSOFT_APACHE_NO_VERSION \
    --namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
    --input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json \
    --output-folder=/python/client

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"/aio
  >python/client/__init__.py
}

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

  printf "GENERATING PYTHON SDK...\n"
  generate_python "$AUTOREST_IMAGE" "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"
  printf "%*s\n" "${COLUMNS:-$(tput cols)}" "" | tr " " -
  printf "\n"
done

printf "[\u2714] PYTHON CLIENT GENERATION COMPLETED SUCCESSFULLY\n"
