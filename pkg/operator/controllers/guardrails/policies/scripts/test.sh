#!/bin/bash

set -eu

template_src_path="gktemplates-src"
constraint_path="gkconstraints"
constraint_test_path="gkconstraints-test"

echo "opa test:"
for folder in ${template_src_path}/*
do
  echo "opa test ${folder}/*.rego"
  opa test ${folder}/*.rego
done

echo "gator test:"
if [[ ! -d ${constraint_test_path} ]]; then
  mkdir -p "${constraint_test_path}"
fi
for file in ${constraint_path}/*
do
  echo "expand constraints $file"
  filename="$(basename -- ${file})"
  sed 's/{{.Enforcement}}/deny/g' $file > $constraint_test_path/$filename
done


for folder in ${template_src_path}/*
do
  echo "gator verify ${folder}"
  gator verify ${folder}
done
