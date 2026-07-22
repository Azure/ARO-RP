#!/bin/bash

set -xe
IGNORED_PATH=""
CHANGES=$(git status --short | grep -F -v "$IGNORED_PATH" || true)

if [[ -n "$CHANGES" ]]
then
    echo "there are some modified files"
	echo "$CHANGES"
    exit 1
fi


