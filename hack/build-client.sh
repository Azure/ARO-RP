#!/bin/bash

function clean() {
  local API_VERSION=$1

  rm .sha256sum

  rm -rf pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"
  mkdir pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"

  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
  mkdir -p python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"
}

function checksum() {
  sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$1"/redhatopenshift.json >> .sha256sum
}

function generate_golang() {
  local API_VERSION=$1

  sudo docker run \
		--rm \
		-v ${PWD}/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
		-v ${PWD}/swagger:/swagger:z \
		azuresdk/autorest \
		--go \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=redhatopenshift \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$API_VERSION"/redhatopenshift.json \
		--output-folder=/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift

  sudo chown -R $(id -un):$(id -gn) pkg/client
  sed -i -e 's|azure/aro-rp|Azure/ARO-RP|g' pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift/models.go pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift/redhatopenshiftapi/interfaces.go
  go run ./vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP pkg/client
}

function generate_python() {
  local API_VERSION=$1

  sudo docker run \
		--rm \
		-v ${PWD}/python/client:/python/client:z \
		-v ${PWD}/swagger:/swagger:z \
		azuresdk/autorest \
		--use=@microsoft.azure/autorest.python@4.0.70 \
		--python \
		--azure-arm \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$API_VERSION"/redhatopenshift.json \
		--output-folder=/python/client

  sudo chown -R $(id -un):$(id -gn) python/client
  rm -rf python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"/aio
  >python/client/__init__.py
}


for API in "$@"
do
  clean "${API}"
  checksum "${API}"
  generate_golang "${API}"
  generate_python "${API}"
done
