#!/bin/bash -e
######## Helper file to automate the full RP dev creation ########

echo "##### Make sure to be logged in to Azure prior to running this script ####"
echo "##### In case of failure when creating Azure reseource, consider running the full-rp-dev-clenup target ####"
echo "#### E.g., AZURE_PREFIX=$AZURE_PREFIX LOCATION=$LOCATION make full-rp-dev-clenup       "
source hack/rp-dev/rp_funcs.sh 
setup_rp_config $AZURE_PREFIX $LOCATION $SKIP_DEPLOYMENTS
pre_deploy_resources
add_hive
mirror_images
prepare_RP_deployment
fully_deploy_resources
