#!/bin/bash

set -eu

template_src_path="gktemplates-src"

echo "opa test:"
for folder in ${template_src_path}/*
do
  echo "opa test ${folder}/*.rego"
  opa test ${folder}/*.rego
done

echo "gator verify:"
for folder in ${template_src_path}/*
do
  echo "gator verify ${folder}"
  gator verify ${folder}
done
