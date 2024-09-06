#!/bin/bash

version=4.15.30

declare -A test_cases=(
  ["Azure Red Hat OpenShift Storage Operator Role"]="0000_50_cluster-storage-operator_03_credentials_request_azure.yaml"
  ["Azure Red Hat OpenShift Azure Files Storage Operator Role"]="0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml"
  ["Azure Red Hat OpenShift Image Registry Operator Role"]="0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml"
  ["Azure Red Hat OpenShift Network Operator Role"]="0000_50_cluster-network-operator_02-cncc-credentials.yaml"
  ["Azure Red Hat OpenShift Cloud Controller Manager Role"]="0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml"
  ["Azure Red Hat OpenShift Machine API Operator Role"]="0000_30_machine-api-operator_00_credentials-request.yaml"
  ["Azure Red Hat OpenShift Cluster Ingress Operator Role"]="0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml"
)

for role in "${!test_cases[@]}"; do
  echo "Testing $role"
  comm -23 \
    <(yq ".spec.providerSpec.permissions[]" "/tmp/credreqs/$version/${test_cases[$role]}" | sort) \
    <(az role definition list -n $role | jq -r ".[].permissions[].actions[]"| sort)
done
