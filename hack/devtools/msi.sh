#!/bin/bash

# This script creates a mock MSI object to use for local development
# We use a service principal and certificate as the mock object

appName="mock-msi-$(openssl rand -base64 9 | tr -dc 'a-zA-Z0-9' | head -c 6)"

result=$(az ad sp create-for-rbac --name $appName --create-cert)

# Extract the client ID and tenant ID from the result
clientID=$(echo $result | jq -r .appId)
tenantID=$(echo $result | jq -r .tenant)

# Extract the certificate information from the result
certFilePath=$(echo $result | jq -r '.fileWithCertAndPrivateKey')
base64EncodedCert=$(base64 -w 0 $certFilePath)
rm $certFilePath

# Print the extracted values
echo "Client ID: $clientID"
echo "Base64 Encoded Certificate: $base64EncodedCert"
echo "Tenant: $tenantID"