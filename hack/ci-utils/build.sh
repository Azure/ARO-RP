#!/bin/bash

set -xe
OPTION=$2

case "$OPTION" in
    "generate")
        echo "Running make generate."
        make generate
    ;;
    "build-all")
        echo "Running make build-all."
        make build-all
    ;;
    "unit-test-go")
        echo "Running make unit-test-go."
        make unit-test-go
    ;;
    "validate-fips")
        echo "Running make validate-fips."
        make validate-fips
    ;;
    *) go run hack/ci-utils/differenceChecker/main.go make generate
        make build-all
        make unit-test-go
    ;;
esac