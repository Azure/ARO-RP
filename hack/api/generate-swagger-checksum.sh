#!/bin/bash -e



function checksum() {
  local API_VERSION=$1
  local FOLDER=$2
  local found=0
  local spec_path

  for spec_path in \
    "swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters/$FOLDER/$API_VERSION/redhatopenshift.json" \
    "api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/$FOLDER/$API_VERSION/redhatopenshift.json"; do
    if [ -f "$spec_path" ]; then
      sha256sum "$spec_path" >>.sha256sum
      found=1
    fi
  done

  if [ "$found" -ne 1 ]; then
    printf "ERROR: Could not find swagger spec for API version %s\n" "$API_VERSION" >&2
    return 1
  fi
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
