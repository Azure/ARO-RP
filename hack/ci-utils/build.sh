#!/bin/bash

set -xe
if [[ -z "$1" ]]; then
    go run hack/ci-utils/differenceChecker/main.go make generate
    make build-all
    make unit-test-go
    exit 0
fi

OPTION=$1
case "$OPTION" in
    "generate")
        echo "Running make generate."
        make generate
        exit 0
    ;;
    "build-all")
        echo "Running make build-all."
        make build-all
        exit 0
    ;;
    "unit-test-go")
        echo "Running make unit-test-go."
        make unit-test-go
        exit 0
    ;;
    "validate-fips")
        echo "Running make validate-fips."
        make validate-fips
        exit 0
    ;;
    *)
        echo "No valid input is provided"
        exit 1
esac