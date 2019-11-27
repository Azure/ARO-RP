#!/bin/bash -e

FILES=$(gofmt -s -l pkg cmd)

if [ -n "$FILES" ]; then
    echo You have go format errors in the below files, please run "gofmt -s -w pkg cmd"
    echo $FILES
    exit 1
fi

FILES=$(goimports -e -l -local=github.com/jim-minter/rp pkg cmd)

if [ -n "$FILES" ]; then
    echo You have go import errors in the below files, please run "goimports -e -w -local=github.com/jim-minter/rp pkg cmd"
    echo $FILES
    exit 1
fi

FILES=$(find -name '*:*')

if [ -n "$FILES" ]; then
    echo The following filenames contain :, please rename them for Windows users
    echo $FILES
    exit 1
fi
