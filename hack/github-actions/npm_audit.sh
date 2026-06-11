#!/bin/bash

PACKAGE_DIR="${1:-portal/v2}"

cd "./${PACKAGE_DIR}"

npm audit --audit-level high --omit=dev

if [ "$?" -gt 0 ]; then
  echo "Critical/High vulnerabilities found in ${PACKAGE_DIR}, please run \"npm audit --omit=dev\" locally and fix, or record issues and create a Jira card."
  exit 1
fi

echo "No critical/high vulnerabilities found in ${PACKAGE_DIR}."
