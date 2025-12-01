#!/bin/bash -e

function checksum() {
  local API_VERSION=$1
  local FOLDER=$2

  sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/"$FOLDER"/"$API_VERSION"/redhatopenshift.json >>.sha256sum
}

if [ -f .sha256sum ]; then
  rm .sha256sum
fi

for API_VERSION in "${@:1}"; do
  FOLDER=stable
  if [[ "$API_VERSION" =~ .*preview ]]; then
    FOLDER=preview
  fi

  printf "GENERATING CHECKSUM...\n"
  checksum "$API_VERSION" "$FOLDER"
  printf "[\u2714] SUCCESS\n\n"

done

printf "[\u2714] CHECKSUM GENERATION COMPLETED SUCCESSFULLY\n"
