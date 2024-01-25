#!/bin/bash

set -xe

# check if we can build and have built a valid FIPS-compatible binary
res=$(go run github.com/acardace/fips-detect@v0.0.0-20230309083406-7157dae5bafd ${1} -j)

binary=$(echo $res | go run ./hack/jq -r '.goBinaryFips.value')
lib=$(echo $res | go run ./hack/jq -r '.cryptoLibFips.value')

if [[ $binary == "false" ]]; then
	echo "binary is not FIPS compatible"
	exit 1
fi

if [[ $lib == "false" ]]; then
	echo "lib is not FIPS compatible"
	exit 1
fi

go tool nm $1 | grep FIPS
