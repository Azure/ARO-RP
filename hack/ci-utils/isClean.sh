#!/bin/bash

set -xe
IGNORED_PATH="api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/OpenShiftClusters/examples/"
CHANGES=$(git status --short | grep -v "$IGNORED_PATH" || true)

if [[ -n "$CHANGES" ]]
then
    echo "there are some modified files"
	echo "$CHANGES"
    exit 1
fi


