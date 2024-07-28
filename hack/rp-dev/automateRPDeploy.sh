#!/bin/bash -e

echo "Make sure to be logining in to Azure prior to running this script"
source hack/rp-dev/setupRPConfig.sh $AZURE_PREFIX # setup config file
./hack/rp-dev/preRPDeploy.sh
make deploy
echo "Success step 8 ✅ - fully deploy all the resources for ARO RP"
