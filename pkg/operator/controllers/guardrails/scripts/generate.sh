#!/bin/bash

set -eu

template_src_path="gktemplates-src"
template_path="gktemplates"

main() {

  if [[ ! -d ${template_path} ]]; then
    mkdir -p "${template_path}"
  fi

  for tmpl in $(find ${template_src_path} -name '*.tmpl'); do
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

main
