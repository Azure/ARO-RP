#!/bin/bash -e

if [[ ! "$(go build ./pkg/util/ 2>&1)" =~ "no Go files" ]]; then
	echo "no Go files should be placed in pkg/util; use util subpackages"
	exit 1
fi

echo "No Go files found in pkg/util"
