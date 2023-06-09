#!/bin/bash

set -eo pipefail
IFS=$'\n\t'

template_src_path="gktemplates-src"
constraint_path="gkconstraints"
constraint_test_path="gkconstraints-test"
library="${template_src_path}/library/common.rego"

usage() {
  echo "Usage: $0 [policy_folder] [constraint_file]"
  echo "  policy_folder: Optional parameter to specify a specific policy folder to test."
  echo "                 If not provided, all policy folders will be tested."
  echo "  constraint_file: Optional parameter to specify a specific constraint file to test."
  echo "                 If not provided, all constraint files will be tested."
  echo "                 This parameter is mandatory when policy_folder is specified."
  exit 1
}

expand_constraint() {
  local file=$1
  echo "expand constraints $file"
  filename="$(basename -- ${file})"
  sed 's/{{.Enforcement}}/deny/g' $file > $constraint_test_path/$filename
}

expand_all_constraints() {
  for file in ${constraint_path}/*
  do
    expand_constraint "$file"
  done
}

main() {
  if [[ $1 == "-h" || $1 == "--help" ]]; then
    usage
  fi

  policy_folder="$1"
  constraint_file="$2"

  # If a specific policy folder is provided, ensure the constraint file is provided
  if [[ -n "$policy_folder"  && -z "$constraint_file" ]]; then
    echo "Error: constraint file parameter is mandatory when policy folder is specified."
    usage
  fi

  if [[ ! -d ${constraint_test_path} ]]; then
    mkdir -p "${constraint_test_path}"
  fi

  # Only test specified policy folder if parameters passed
  if [ -n "$policy_folder"  ]; then
    echo "[opa test] -> $library ${template_src_path}/${policy_folder}/*.rego"
    opa test $library "${template_src_path}/${policy_folder}"/*.rego

    echo "[gator verify] -> ${template_src_path}/${policy_folder}"
    expand_constraint "${constraint_path}/${constraint_file}"
    gator verify -v "${template_src_path}/${policy_folder}"
    exit 0
  fi

  # Test all policy folders if no parameter passed
  expand_all_constraints
  for folder in ${template_src_path}/*
    do
      echo "[opa test] -> $library ${folder}/*.rego"
      opa test $library "${folder}"/*.rego

      echo "[gator verify] -> ${folder}"
      gator verify -v "${folder}"
  done
}

main "$@"
