#!/bin/bash -e
######## Helper file to automate the Shared RP Development Environment creation ########
# Automate https://github.com/Azure/ARO-RP/blob/master/docs/prepare-a-shared-rp-development-environment.md

main () {
    echo "##### Make sure to be logged in to Azure prior to running this script ####"
    source hack/rp-dev/shared_rp_funcs.sh
    local secret_sa_account_name=${SECRET_SA_ACCOUNT_NAME}
    local prefix=${SHARED_RP_PREFIX}
    local location=${LOCATION}

    local parent_domain_resourcegroup="global-infra-${prefix}" # usually dns
    source hack/devtools/rp_dev_helper.sh
    echo -e "#### Running Shared RP Automation in Container ####\n"
    prerequisites ${secret_sa_account_name} ${prefix} ${location} ${parent_domain_resourcegroup}

    # Should we use e2esecretstorage or  rharosecretsdev?
    aad_applications ${prefix} ${location}
    certificates
    # certificate_rotation # Not sure whether it is needed
    local azure_prefix=${AZURE_PREFIX}
    resourcegroup_prefix="prefix" # usually v4
    proxy_domain_name_label="proxy_name"
    # proxy hostname will be of the form vm0.$PROXY_DOMAIN_NAME_LABEL.$LOCATION.cloudapp.azure.com.
    env_file ${secret_sa_account_name} ${parent_domain_resourcegroup} ${resourcegroup_prefix} ${proxy_domain_name_label}
    ls secrets/*
    # ls secrets/vpn-*.ovpn
}

main "$@"
# cleanup
# az ad sp delete --id 12d31286-4cf3-40f7-851a-e562c6043f82 && az ad sp delete --id 8df991ab-b3b9-4a5d-a940-0b34e50a8310
# az ad app delete --id 18f90d02-dea2-4495-9db8-3832898ebb11 && az ad app delete --id ef71dbb7-7f74-4723-a6b7-c4d945992d69
