#!/bin/bash

set -xe 

go run hack/ci-utils/differenceChecker/main.go make generate
make build-all
make unit-test-go
