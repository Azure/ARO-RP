#!/bin/bash
set -eo pipefail
[[ -z "${BUILDDROP}" ]] && (echo "BUILDDROP is not set"; exit 1)
cp -a --parents \
   aro \
   $BUILDDROP/
