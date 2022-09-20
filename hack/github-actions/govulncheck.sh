#!/bin/bash

if GOMEMLIMIT=10GiB govulncheck ./... | grep -q 'No vulnerabilities found'; then
  echo "No vulnerabilities found."
  exit 0
else
  echo "Vulnerabilities found, please run $(govulncheck ./...) locally and fix before merging."
  exit 1
fi
