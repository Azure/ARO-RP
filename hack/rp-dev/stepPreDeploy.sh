export AZURE_PREFIX=$1 NO_CACHE=false AZURE_EXTENSION_DEV_SOURCES="$(pwd)/python" ARO_INSTALL_VIA_HIVE=true
export ARO_ADOPT_BY_HIVE=true ARO_SKIP_PKI_TESTS=true DATABASE_NAME=ARO

secretSA=$(grep -A 2 secretSA .pipelines/templates/rp-dev/rp-dev-params.yml | grep 'default:' | awk '{print $2}')
# Export the SECRET_SA_ACCOUNT_NAME environment variable and run make secrets
export SECRET_SA_ACCOUNT_NAME=$secretSA && make secrets

# Define the expected directory and file names
expected_dir="secrets"
files=("env" "dev-ca.crt" "dev-client.crt")

# Validate that the secrets directory and required files exist
[ -d "$expected_dir" ] || { echo "Directory '$expected_dir' has not been created."; exit 1; }
for file in "${files[@]}"; do
  [ -f "$expected_dir/$file" ] || { echo "File '$file' does not exist inside the directory '$expected_dir'."; exit 1; }
done
echo "Success step 1 - Directory '$expected_dir' has been created with files - ${files[@]}"

paramsFile=".pipelines/templates/rp-dev/rp-dev-params.yml"
varFile=".pipelines/templates/rp-dev/rp-dev-vars.yml"
# Export environment variables from parameters
export LOCATION=$(grep -A 2 location $paramsFile| grep 'default:' | awk '{print $2}' | head -n 1)
azure_resource_name=$AZURE_PREFIX-aro-$LOCATION
export RESOURCEGROUP=$azure_resource_name DATABASEACCOUNTNAME=$azure_resource_name KEYVAULTPREFIX=$azure_resource_name
gitCommit=$(git rev-parse --short=7 HEAD)
export AROIMAGE=$AZURE_PREFIXaro.azurecr.io/aro:$gitCommit
# Source environment variables from the secrets file
source secrets/env
# Generate SSH key
# ssh-keygen -t rsa -N "" -f ~/.ssh/id_rsa

# Run the make command to generate dev-config.yaml
make dev-config.yaml

# Check if the dev-config.yaml file exists
[ -f "dev-config.yaml" ] || { echo "File dev-config.yaml does not exist."; exit 1; }
echo "Success step 2 - Config file dev-config.yaml has been created"
