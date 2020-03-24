#!/bin/bash
set -eo pipefail
[[ -z "${BUILDDROP}" ]] && (echo "BUILDDROP is not set"; exit 1)
[[ -z "${RELEASE_TAG}" ]] && (echo "RELEASE_TAG is not set"; exit 1)
[[ -z "${VERSION}" ]] && (echo "VERSION is not set"; exit 1)
cp -a --parents \
   aro \
   $BUILDDROP/
