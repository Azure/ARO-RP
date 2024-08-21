#!/bin/bash -e
######## Helper file to automate the full RP dev creation ########
main() {
    echo "##### Make sure to be logged in to Azure prior to running this script ####"
    echo "##### In case of failure when creating Azure reseource, consider running the rp-full-dev-clenup target ####"
    echo "#### E.g., AZURE_PREFIX=$AZURE_PREFIX LOCATION=eastus make rp-full-dev-clenup       "
    git_commit="$(git rev-parse --short=7 HEAD)"
    source hack/rp-dev/rp_funcs.sh
    location="eastus"
    setup_rp_config $AZURE_PREFIX $git_commit $location
    resource_group=$RESOURCEGROUP
    pre_deploy_resources $AZURE_PREFIX $resource_group $location $SKIP_DEPLOYMENTS
    add_hive $resource_group $location $SKIP_DEPLOYMENTS
    mirror_images $AZURE_PREFIX $USER_PULL_SECRET $SKIP_DEPLOYMENTS
    prepare_RP_deployment $AZURE_PREFIX $git_commit $location $SKIP_DEPLOYMENTS
    fully_deploy_resources $AZURE_PREFIX $resource_group $location $SKIP_DEPLOYMENTS
}

# setup_rp_config <AZURE_PREFIX> <GIT_COMMIT> <LOCATION>
# pre_deploy_resources <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
# add_hive <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
# mirror_images <AZURE_PREFIX> <USER_PULL_SECRET> [SKIP_DEPLOYMENTS]
# prepare_RP_deployment <AZURE_PREFIX> <GIT_COMMIT> <LOCATION> [SKIP_DEPLOYMENTS]
# fully_deploy_resources <AZURE_PREFIX> <RESOURCEGROUP> <LOCATION> [SKIP_DEPLOYMENTS]
