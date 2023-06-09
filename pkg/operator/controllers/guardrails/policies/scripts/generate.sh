#!/bin/bash

set -e
IFS=$'\n\t'

usage() {
  echo "Usage: $0 [policy_folder]"
  echo "  policy_folder: Optional parameter to specify a specific policy folder to generate templates for."
  echo "                 If not provided, all policy folders will be generated."
  exit 1
}

template_src_path="gktemplates-src"
template_path="gktemplates"

main() {
  if [[ $1 == "-h" || $1 == "--help" ]]; then
    usage
  fi

  if [[ ! -d ${template_path} ]]; then
    mkdir -p "${template_path}"
  fi

  # Optional policy folder param
  policy_folder="$1"
  if [[ -n $policy_folder ]]; then
    # If policy_folder was provided, prepend it with path and trailing slash
    policy_folder="${template_src_path}/${policy_folder}/"
  else
    # Otherwise, just operate on the whole template source path
    policy_folder="${template_src_path}/"
  fi

  # Go through all the .tmpl and .rego files and generate constraint templates 
  for tmpl in $(find ${policy_folder} -name '*.tmpl'); do
    filename="$(basename -- ${tmpl} .tmpl).yaml"
    echo "Generating ${template_path}/${filename} from ${tmpl}"
    gomplate -f "${tmpl}" > "${template_path}/${filename}"

    src_dir="$(dirname "${tmpl}")"
    for req in src.rego src_test.rego; do
      if [[ ! -f "${src_dir}/${req}" ]]; then
        echo "${src_dir}/${req} is missing"
        exit 1
      fi
    done
  done
}

main "$@"
