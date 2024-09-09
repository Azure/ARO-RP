#!/usr/bin/env bash

set -e

OC=$1
confirmed_version=4.14.35
version=$(curl -s "https://api.openshift.com/api/upgrades_info/v1/graph?channel=fast-4.16" | jq -r ".nodes[].version" | sort -V --rev | head -n1)

"$OC" adm release extract --credentials-requests --to "/tmp/credreqs/$confirmed_version" "quay.io/openshift-release-dev/ocp-release:$confirmed_version-x86_64"
echo "Extracted $confirmed_version"
"$OC" adm release extract --credentials-requests --to "/tmp/credreqs/$version" "quay.io/openshift-release-dev/ocp-release:$version-x86_64"
echo "Extracted $version"

declare -A test_cases=(
  ["0000_50_cluster-storage-operator_03_credentials_request_azure.yaml"]="5b7237c5-45e1-49d6-bc18-a1f62f400748"
  ["0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml"]="0d7aedc0-15fd-4a67-a412-efad370c947e"
  ["0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml"]="8b32b316-c2f5-4ddf-b05b-83dacd2d08b5"
  ["0000_50_cluster-network-operator_02-cncc-credentials.yaml"]="be7a6435-15ae-4171-8f30-4a343eff9e8f"
  ["0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml"]="a1f96423-95ce-4224-ab27-4e3dc72facd4"
  ["0000_30_machine-api-operator_00_credentials-request.yaml"]="0358943c-7e01-48ba-8889-02cc51d78637"
  ["0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml"]="0336e1d3-7a87-462b-b6db-342b63f7802c"
)

for role in "${!test_cases[@]}"; do
  echo "Testing $role"
  diff=$(comm -3 \
    <(yq '.[] | select(.providerSpec.kind == "AzureProviderSpec") | .providerSpec.permissions[]' "/tmp/credreqs/$confirmed_version/$role" | sort) \
    <(yq '.[] | select(.providerSpec.kind == "AzureProviderSpec") | .providerSpec.permissions[]' "/tmp/credreqs/$version/$role" | sort))
  if [[ -z $diff ]]; then
    echo "No changes in permissions for $role"
    continue
  fi

  missing=$(comm -23 \
    <(yq '.[] | select(.providerSpec.kind == "AzureProviderSpec") | .providerSpec.permissions[]' "/tmp/credreqs/$version/$role" | sort) \
    <(az role definition list -n "${test_cases[$role]}" | jq -r ".[].permissions[].actions[]"| sort))

  if [[ -n $missing ]]; then
    echo "Missing permissions for $role:"
    for permission in $missing; do
      echo "  $permission"
    done
    exit 1
  fi

  missing=$(comm -23 \
    <(yq '.[] | select(.providerSpec.kind == "AzureProviderSpec") | .providerSpec.dataPermissions[]' "/tmp/credreqs/$version/$role" | sort) \
    <(az role definition list -n "${test_cases[$role]}" | jq -r ".[].permissions[].dataActions[]"| sort))

  if [[ -n $missing ]]; then
    echo "Missing data permissions for $role:"
    for permission in $missing; do
      echo "  $permission"
    done
    exit 1
  fi
done


