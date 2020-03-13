#!/bin/bash -e
mkdir -p secrets
cat >secrets/parameters.json <<EOF
{
    "\$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "adminApiCaBundle": {
            "value": "$ADMIN_API_CA_BUNDLE"
        },
        "adminApiClientCertCommonName": {
            "value": "$ADMIN_API_CLIENT_CERT_CN"
        },
        "databaseAccountName": {
            "value": "$COSMOSDB_ACCOUNT"
        },
        "domainName": {
            "value": "$DOMAIN_NAME"
        },
        "fpServicePrincipalId": {
            "value": "$AZURE_FP_CLIENT_ID"
        },
        "keyvaultPrefix": {
            "value": "$KEYVAULT_PREFIX"
        },
        "pullSecret": {
            "value": "$PULL_SECRET"
        },
        "rpImage": {
            "value": "$RP_IMAGE"
        },
        "rpImageAuth": {
            "value": "$RP_IMAGE_AUTH"
        },
        "mdmFrontendUrl": {
            "value": "$MDM_FRONTEND"
        },
        "mdsdConfigVersion": {
            "value": "$MDSD_CONFIG_VERSION"
        },
        "mdsdEnvironment": {
            "value": "$MDSD_ENVIROMENT"
        },
        "rpMode": {
            "value": "$RP_MODE"
        },
        "sshPublicKey": {
            "value": "$SSH_PUBLIC_KEY"
        }
    }
}
EOF
