#!/bin/bash -e
######## Helper file to automate the full RP dev creation ########
# Run usage_rp_funcs and usage_rp_dev to get functions' usage help
main() {
    echo "##### Make sure to be logged in to Azure prior to running this script ####"
    echo "##### In case of failure when creating Azure reseource, consider running the clean_rp_dev_env function ####"
    echo "#### E.g., AZURE_PREFIX=$AZURE_PREFIX clean_rp_dev_env $LOCATION ####"
    source hack/rp-dev/full_rp_funcs.sh
    local git_commit="$(git rev-parse --short=7 HEAD)"
    is_full_rp_succeeded $AZURE_PREFIX "${AZURE_PREFIX}-aro-$LOCATION" "${AZURE_PREFIX}-gwy-$LOCATION" $git_commit

    setup_rp_config $AZURE_PREFIX $git_commit $LOCATION
    pre_deploy_resources $AZURE_PREFIX $LOCATION $RESOURCEGROUP $SKIP_DEPLOYMENTS
    add_hive $LOCATION $RESOURCEGROUP $SKIP_DEPLOYMENTS
    mirror_images $AZURE_PREFIX $USER_PULL_SECRET $PULL_SECRET $git_commit $SKIP_DEPLOYMENTS
    prepare_RP_deployment $AZURE_PREFIX $git_commit $LOCATION $SKIP_DEPLOYMENTS
    log "VMSSs suffix is $git_commit"
    fully_deploy_resources $AZURE_PREFIX $git_commit $LOCATION $RESOURCEGROUP $SKIP_DEPLOYMENTS
}

main "$@"
