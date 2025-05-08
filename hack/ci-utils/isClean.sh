#!/bin/bash

set -xe
if [[ ! -z "$(git status -s)" ]]
then
    echo "there are some modified files"
    git status
    exit 1 
fi


