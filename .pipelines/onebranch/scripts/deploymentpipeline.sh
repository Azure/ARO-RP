set -e

echo "Creating required directories"

mkdir -p $OB_OUTPUTDIRECTORY/ServiceGroupRoot/bin/
mkdir -p $OB_OUTPUTDIRECTORY/ServiceGroupRoot/Parameters/
mkdir -p $OB_OUTPUTDIRECTORY/Shell/

echo "Downloading Crane"

wget -O $OB_OUTPUTDIRECTORY/Shell/crane.tar.gz https://github.com/google/go-containerregistry/releases/download/v0.4.0/go-containerregistry_Linux_x86_64.tar.gz

echo "Extracting Crane binaries"

pushd $OB_OUTPUTDIRECTORY/Shell
tar xzvf crane.tar.gz
rm crane.tar.gz
popd

echo "Copying required files to ob_outputdirectory: ${OB_OUTPUTDIRECTORY}"

tar -rvf ./ARO.Pipelines/ev2/generator/deployment.tar -C "$OB_OUTPUTDIRECTORY/Shell" $(cd $OB_OUTPUTDIRECTORY/Shell; echo *)
tar -rvf ./ARO.Pipelines/ev2/generator/deployment.tar -C "./ARO.Pipelines/RP-Config" $(cd ./RP-Config; echo *)

echo "Copy tar to ob_outputdirectory dir"
cp -r ./ARO.Pipelines/ev2/Deployment/ServiceGroupRoot/ $OB_OUTPUTDIRECTORY/
cp ./ARO.Pipelines/ev2/generator/deployment.tar $OB_OUTPUTDIRECTORY/ServiceGroupRoot/bin/

echo "Listing the contents of dirs for debugging"
ls $OB_OUTPUTDIRECTORY
ls $OB_OUTPUTDIRECTORY/ServiceGroupRoot/
ls $OB_OUTPUTDIRECTORY/ServiceGroupRoot/bin/
