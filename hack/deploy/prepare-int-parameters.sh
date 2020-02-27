#!/bin/bash -e
mkdir -p secrets
cat >secrets/parameters.json <<EOF
{
    "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "clusterMdmMetricNamespace": {
            "value": "$CLUSTER_MDM_NAMESPACE"
        },
        "clusterMdmMonitoringAccount": {
            "value": "$CLUSTER_MDM_ACCOUNT"
        },
        "databaseAccountName": {
            "value": "$COSMOSDB_ACCOUNT"
        },
        "domainName": {
            "value": "$DOMAIN_NAME.$PARENT_DOMAIN_NAME"
        },
        "extraCosmosDBIPs": {
            "value": ""
        },
        "extraKeyvaultAccessPolicies": {
            "value": []
        },
        "fpServicePrincipalId": {
            "value": "$AZURE_FP_CLIENT_ID"
        },
        "keyvaultPrefix": {
            "value": "$KEYVAULT_PREFIX"
        },
        "pullSecret": {
            "value": "SET-LATER"
        },
        "rpImage": {
            "value": "$RP_IMAGE"
        },
        "rpImageAuth": {
            "value": "$RP_IMAGE_AUTH"
        },
        "rpMdmFrontendUrl": {
            "value": "$MDM_FRONTEND"
        },
        "rpMdmMetricNamespace": {
            "value": "$RP_MDM_NAMESPACE"
        },
        "rpMdmMonitoringAccount": {
            "value": "$RP_MDM_ACCOUNT"
        },
        "rpMdsdAccount": {
            "value": "$MDSD_ACCOUNT"
        },
        "rpMdsdConfigVersion": {
            "value": "$MDSD_CONFIG_VERSION"
        },
        "rpMdsdEnvironment": {
            "value": "$MDSD_ENVIROMENT"
        },
        "rpMdsdNamespace": {
            "value": "$MDSD_NAMESPACE"
        },
        "rpMode": {
            "value": "$RP_MODE"
        },
        "rpServicePrincipalId": {
            "value": "SET-LATER"
        },
        "sshPublicKey": {
            "value": "SET-LATER"
        },
        "vmssCount": {
            "value": 3
        },
        "vmssDomainNameLabel": {
            "value": "$VMSS_DOMAIN_NAME_LABEL"
        },
        "vmssName": {
            "value": "$VMSS_NAME"
        }
    }
}
EOF