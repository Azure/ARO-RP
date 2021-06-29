SERVICE_GROUP_ROOT=$BUILD_SOURCESDIRECTORY/ARO.Pipelines/ev2/Logging/Kusto-Bootstrapper/ServiceGroupRoot
EV2_BIN=$SERVICE_GROUP_ROOT/bin

mkdir $OB_OUTPUTDIRECTORY

cd $EV2_BIN
tar -rvf $EV2_BIN/kusto-bootstrapper-resources.tar kusto-bootstrapper.sh
rm kusto-bootstrapper.sh

cp -r $SERVICE_GROUP_ROOT $OB_OUTPUTDIRECTORY/ServiceGroupRoot/
