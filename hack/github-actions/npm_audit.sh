#!/bin/bash

cd ./portal/v2

npm audit --audit-level high --omit=dev

if [ "$?" -gt 0 ]; then
  echo "Critical/High Vulnerabilities found, please run \"npm audit --omit=dev\" locally and fix, or record issues and create a Jira card."
  exit 1
fi

echo "No critical/high vulnerabilities found."
