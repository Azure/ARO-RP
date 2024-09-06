#!/bin/bash

declare -A test_cases=(
  ["Azure Red Hat OpenShift Storage Operator Role"]="0000_50_cluster-storage-operator_03_credentials_request_azure.yaml 5b7237c5-45e1-49d6-bc18-a1f62f400748"
  ["Azure Red Hat OpenShift Azure Files Storage Operator Role"]="0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml 0d7aedc0-15fd-4a67-a412-efad370c947e"
  ["Azure Red Hat OpenShift Image Registry Operator Role"]="0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml 8b32b316-c2f5-4ddf-b05b-83dacd2d08b5"
  ["Azure Red Hat OpenShift Network Operator Role"]="0000_50_cluster-network-operator_02-cncc-credentials.yaml be7a6435-15ae-4171-8f30-4a343eff9e8f"
  ["Azure Red Hat OpenShift Cloud Controller Manager Role"]="0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml a1f96423-95ce-4224-ab27-4e3dc72facd4"
  ["Azure Red Hat OpenShift Machine API Operator Role"]="0000_30_machine-api-operator_00_credentials-request.yaml 0358943c-7e01-48ba-8889-02cc51d78637"
  ["Azure Red Hat OpenShift Cluster Ingress Operator Role"]="0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml 0336e1d3-7a87-462b-b6db-342b63f7802c"
)
