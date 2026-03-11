#!/bin/bash

set -o errexit \
    -o nounset

main() {
    local -r hive_commit_override="${1:-}"

    # This is the commit sha that the image was built from and ensures we use the correct configs for the release
    # Ensure it is the latest when testing pre production deployments
    local -r default_commit="f48f47857f6a1dda25ad46957927ee6fe3afe1eb"
    if [ -z "$hive_commit_override" ]; then
        warn "Using default hive commit hash: $default_commit"
        warn "Hive commit hashes can be found here: https://quay.io/repository/redhat-user-workloads/crt-redhat-acm-tenant/hive-operator/hive?tab=tags"
    fi
    local -r hive_image_commit_hash="${hive_commit_override:-$default_commit}"

    info "Using hive commit: $hive_image_commit_hash"
    # shellcheck disable=SC2034

    local -r tmpdir="$(mktemp -d)"
    # shellcheck disable=SC2064
    trap "cleanup $tmpdir" EXIT

    # shellcheck disable=SC2034
    kustomize_bin="$(which kustomize 2> /dev/null)" \
    || install_kustomize "$tmpdir" kustomize_bin

    hive_repo_clone "$tmpdir"

    hive_repo_hash_checkout "$tmpdir" "$hive_image_commit_hash"

    local -r hive_image="arointsvc.azurecr.io/redhat-services-prod/crt-redhat-acm-tenant/hive-operator/hive:${hive_image_commit_hash}"
    generate_hive_config kustomize_bin \
                         "$HIVE_OPERATOR_NS" \
                         "$hive_image" \
                         "$tmpdir"

    info "Hive config generated."
}

install_kustomize() {
    local tmpd="$1"
    local -n kustomize="$2"
    info "starting"

    pushd "$tmpd" 1> /dev/null

    # This version is specified in the hive repo and is the only hard dependency for this script
    # https://github.com/openshift/hive/blob/master/vendor/github.com/openshift/build-machinery-go/make/targets/openshift/kustomize.mk#L7
    local -r kustomize_version="4.1.3"
    info "kustomize not detected, downloading..."
    if ! curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/kustomize/v${kustomize_version}/hack/install_kustomize.sh" | bash -s "$kustomize_version" "$tmpd" 1> /dev/null; then
        fatal "error downloading kustomize"
    fi

    [ -d "${HOME}/bin" ] || mkdir -p "${HOME}/bin"

    kustomize_new="${tmpd}/kustomize"
    kustomize_dest="${HOME}/bin/kustomize"
    info "Installing $kustomize_new into $kustomize_dest"
    mv "$kustomize_new" "$kustomize_dest"

    popd 1> /dev/null

    kustomize="$(which kustomize)"
}

hive_repo_clone() {
    local tmpd="$1"
    info "starting"

    local -r repo="https://github.com/openshift/hive.git"
    info "Cloning $repo into $tmpd"
    if ! git clone "$repo" "$tmpd"; then
        fatal "error cloning the hive repo"
    fi
}

hive_repo_hash_checkout() {
    local tmpd="$1"
    local commit="$2"
    info "starting"
    info "Attempting to use commit: $commit"

    pushd "$tmpd" 1> /dev/null

    if git reset --hard "$commit" && [ "$( git rev-parse --short="${#commit}" HEAD )" != "$commit" ]; then
        fatal "error resetting the hive repo to the correct git hash '${commit}'"
    fi

    popd 1> /dev/null
}

# generate_hive_config()
generate_hive_config() {
    local -n kustomize="$1"
    local namespace="$2"
    local image="$3"
    local tmpd="$4"
    info "starting"
    
    pushd "$tmpd" 1> /dev/null
    mkdir -p overlays/deploy

    debug "copying template kustomization.yaml"
    cp overlays/template/kustomization.yaml overlays/deploy
    pushd overlays/deploy 1> /dev/null
    debug "Setting hive image."
    $kustomize edit set image registry.ci.openshift.org/openshift/hive-v4.0:hive="$image"
    $kustomize edit set namespace "$namespace"
    popd 1> /dev/null

    info "Building hive deployment"
    $kustomize build overlays/deploy > hive-deployment.yaml

    popd 1> /dev/null
    mv "$tmpd/hive-deployment.yaml" "$HIVE_CONFIG"

    debug "Verifying hive deployment pull secret exists in deployment."
    yq -i 'select(.kind == "ServiceAccount").imagePullSecrets = [{"name": "hive-global-pull-secret"}]' "$HIVE_CONFIG/hive-deployment.yaml"

    crds_old="$HIVE_CONFIG/crds"
    if [ -d "$crds_old" ]; then
        info "Deleting $crds_old"
        rm -rf "$HIVE_CONFIG/crds"
    fi

    crds_new="$tmpd/config/crds"
    info "Copying $crds_new into $HIVE_CONFIG"
    cp -R "$crds_new" "$HIVE_CONFIG/"
}

# declare -r source_file_not_found_err="not found. Are you in the ARO-RP repository root?"
declare -r source_file_not_found_err="not found. Are you in the ARO-RP repository root?"

declare -r util_lib="hack/util.sh"
[ -f "$util_lib" ] || "$(echo "$util_lib $source_file_not_found_err"; exit 1)"
# shellcheck source=../util.sh
[ "${__hack_util_sourced:-'false'}" == "false" ] || . "$util_lib"

declare -r hive_env="hack/hive/hive.env"
[ -f "$hive_env" ] || fatal "$hive_env $source_file_not_found_err"
# shellcheck source=./hive.env
[ "${__hive_env_sourced:-'false'}" == "false" ] || . "$hive_env"

main "$@"
