#!/bin/bash -e

echo "##### Make sure to be logining in to Azure prior to running this script ####"
source hack/rp-dev/setupRPConfig.sh $AZURE_PREFIX # setup config file
make pre-deploy-aks # deploy predeployment resources prior to AKS
echo "Success step 3 ✅ - deploy predeployment resources prior to AKS"
./hack/rp-dev/preRPDeploy.sh
make go-verify deploy
echo "Success step 8 ✅ - fully deploy all the resources for ARO RP"
