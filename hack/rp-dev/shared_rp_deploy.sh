#!/bin/bash -e
######## Helper file to automate the Shared RP Development Environment creation ########
# Automate https://github.com/Azure/ARO-RP/blob/master/docs/prepare-a-shared-rp-development-environment.md

main () {
    source hack/devtools/rp_dev_helper.sh && source hack/rp-dev/shared_rp_funcs.sh
    log "##### Make sure to be logged in to Azure prior to running this script ####"
    log "#### Running Shared RP Automation in Container ####"
    local prefix=${SHARED_RP_PREFIX}
    local parent_domain_resourcegroup="global-infra-${prefix}" # usually dns
    export AAD_PREFIX="aro-v4-${prefix}"
    export RBAC_DEV_DEPLOYMENT_NAME="$AAD_PREFIX-rbac-development"
    export RESOURCEGROUP_PREFIX="prefix" # usually v4
    # clean_aad_applications
    # clean_resource_groups

    prerequisites "${SECRET_SA_ACCOUNT_NAME}" "${prefix}" "${LOCATION}" "${parent_domain_resourcegroup}"
    # Should we use e2esecretstorage or  rharosecretsdev?
    aad_applications "${prefix}" "${LOCATION}"
    certificates
    # certificate_rotation # Not sure whether it is needed
    proxy_domain_name_label="myproxy"
    # proxy hostname will be of the form vm0.$PROXY_DOMAIN_NAME_LABEL.$LOCATION.cloudapp.azure.com.
    env_file "${SECRET_SA_ACCOUNT_NAME}" "${parent_domain_resourcegroup}" "${RESOURCEGROUP_PREFIX}" "${proxy_domain_name_label}"
    deploy_shared_rp
    ls secrets/*
}

main "$@"
