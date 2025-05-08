# Prepare the CI environment

Follow these steps to build a shared CI environment.

## Prerequisites

1. You will need `Contributor` and `User Access Administrator` roles on your
   Azure subscription.

1. Set the az account
   ```bash
   az account set -n "<your-azure-subscription>"
   ```

1. You will need to set the proper resource group for global infrastructure
   ```bash
   export GLOBAL_RESOURCEGROUP=global-infra
   ```

1. Run the following shell script to configure and deploy the CI components.

   ```bash
   ./hack/devtools/deploy-ci-env.sh
   
   ```
