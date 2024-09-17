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
    export AAD_PREFIX="aro-v4-${prefix}"
    # clean_aad_applications
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
}

main "$@"
# cleanup
