#!/bin/bash

set -eu

template_src_path="gktemplates-src"

for folder in ${template_src_path}/*
do
  echo "opa test ${folder}/src.rego"
  opa test ${folder}/src.rego
done
