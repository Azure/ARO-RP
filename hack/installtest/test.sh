#!/bin/bash

usage() {
	echo -e "usage: ${0} <create|delete>"
	exit 1
}

CLUSTER=${CLUSTER:-loadtest}
RESOURCEGROUP=${RESOURCEGROUP:-loadtest}
CONCURRENCY=${CONCURRENCY:-5}
BASEDIR=$(dirname "$0")

case "$1" in
  create)
    COMMAND="$BASEDIR/create.sh"
    if [ -z $VERSION ]; then
      echo "env VERSION required"
      exit 1
    fi
    ;;
  delete)
    COMMAND="$BASEDIR/delete.sh"
    ;;
  *)
    usage
    exit 1
    ;;
esac

regions=(
"australiaeast"
"australiacentral"
"australiacentral2"
"australiasoutheast"
"brazilsouth"
"brazilsoutheast"
"canadacentral"
"canadaeast"
"centralindia"
"centralus"
"eastus"
"eastus2"
"eastasia"
"francecentral"
"germanywestcentral"
"italynorth"
"japaneast"
"japanwest"
"koreacentral"
"northcentralus"
"northeurope"
"norwayeast"
"norwaywest"
"qatarcentral"
"southcentralus"
"southafricanorth"
"southeastasia"
"southindia"
"swedencentral"
"switzerlandnorth"
"switzerlandwest"
"taiwannorth"
"uaenorth"
"uksouth"
"ukwest"
"westeurope"
)
PS3="Select your Region please: "
select region in "${regions[@]}" Quit
do
    LOCATION=$region
    break;
done

tmux start-server
tmux new-session -d -n $LOCATION -s $LOCATION
tmux select-pane -T 1

for (( i=1; i<$CONCURRENCY; i++ ))
do
  tmux split-window -h
  tmux select-pane -T $i
done
tmux select-layout even-horizontal

for (( i=0; i<$CONCURRENCY; i++ ))
do
  tmux send-keys -t $i "LOCATION=$LOCATION CLUSTER=$CLUSTER-$LOCATION-$i RESOURCEGROUP=$RESOURCEGROUP-$LOCATION-$i VERSION=$VERSION $COMMAND" Enter
done

tmux attach-session -t $LOCATION