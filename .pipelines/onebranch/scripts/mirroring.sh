SERVICE_GROUP_ROOT=$BUILD_SOURCESDIRECTORY/ARO.Pipelines/ev2/Mirroring/ServiceGroupRoot
EV2_BIN=$SERVICE_GROUP_ROOT/bin
ARO_DIR=$BUILD_SOURCESDIRECTORY/deployer

mkdir $OB_OUTPUTDIRECTORY

cd $ARO_DIR
tar -cvf $EV2_BIN/aro.tar aro
cd $SERVICE_GROUP_ROOT/bin
tar -rf $EV2_BIN/aro.tar mirror.sh
rm mirror.sh

cp -r $SERVICE_GROUP_ROOT $OB_OUTPUTDIRECTORY/ServiceGroupRoot/
