#!/bin/bash

set -xe

# check if we can build and have built a valid FIPS-compatible binary
res=$(fips-detect ${1} -j)

binary=$(echo $res | gojq -r '.goBinaryFips.value')
lib=$(echo $res | gojq -r '.cryptoLibFips.value')

if [[ $binary == "false" ]]; then
	echo "binary is not FIPS compatible"
	exit 1
fi

tool=$(go tool nm ${1} | grep FIPS)
echo $tool
