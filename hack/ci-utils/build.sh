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
        go run hack/ci-utils/differenceChecker/main.go make generate
        if [[ -z "$(/usr/bin/git status -s)" ]]; then exit 0; else exit 1; fi
    ;;
    "build-all")
        echo "Running make build-all."
        make build-all
        if [[ -z "$(/usr/bin/git status -s)" ]]; then exit 0; else exit 1; fi
    ;;
    "unit-test-go")
        echo "Running make unit-test-go."
        make unit-test-go
        if [[ -z "$(/usr/bin/git status -s)" ]]; then exit 0; else exit 1; fi
    ;;
    "validate-fips")
        echo "Running make validate-fips."
        make validate-fips
        if [[ -z "$(/usr/bin/git status -s)" ]]; then exit 0; else exit 1; fi
    ;;
    *)
        echo "No valid input is provided"
        exit 1
esac