#!/bin/bash

cd ./portal/v2

if npm audit --omit=dev | grep -q 'found 0 vulnerabilities'; then
  echo "No vulnerabilities found."
  exit 0
else
  echo "Vulnerabilities found, please run `npm audit --omit=dev` locally and fix or record issues and create an ADO card."
  exit 1
fi
