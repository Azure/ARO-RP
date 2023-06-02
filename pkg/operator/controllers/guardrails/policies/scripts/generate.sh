#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

template_src_path="gktemplates-src"
template_path="gktemplates"

generate_template() {
  local tmpl=$1
  local filename="$(basename -- ${tmpl} .tmpl).yaml"
  echo "Generating ${template_path}/${filename} from ${tmpl}"
  gomplate -f "${tmpl}" > "${template_path}/${filename}"
}

check_required_files() {
  local src_dir=$1
  for req in src.rego src_test.rego; do
    if [[ ! -f "${src_dir}/${req}" ]]; then
      echo "${src_dir}/${req} is missing"
      return 1
    fi
  done
}

main() {
  local sub_dir="${1:-}"
  local search_path="${template_src_path}"

  if [[ ! -z "${sub_dir}" ]]; then
    search_path="${search_path}/${sub_dir}"
  fi

  if [[ ! -d ${template_path} ]]; then
    mkdir -p "${template_path}"
  fi

  find "${search_path}" -name '*.tmpl' | while read -r tmpl; do
    generate_template "${tmpl}" || exit 1
    check_required_files "$(dirname "${tmpl}")" || exit 1
  done
}

main "$@"
