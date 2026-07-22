#!/bin/bash

set -xe
IGNORED_PATH=""
CHANGES=$(git status --short)
if [[ -n "$IGNORED_PATH" ]]; then
    CHANGES=$(printf "%s\n" "$CHANGES" | grep -F -v "$IGNORED_PATH" || true)
fi

if [[ -n "$CHANGES" ]]
then
    echo "there are some modified files"
	echo "$CHANGES"
    exit 1
fi


