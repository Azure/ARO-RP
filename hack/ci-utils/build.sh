#!/bin/bash

isClean() {
    if [[ ! -z "$(git status -s)" ]]
    then
        echo "there are some modified files"
        git status
        exit 1 
    fi
}
    

set -xe 

make generate
isClean
make build-all
isClean
make unit-test-go
isClean
