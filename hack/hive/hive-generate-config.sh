#!/bin/bash

set -o errexit \
    -o nounset

main() {
    local -r tmpdir="$(mktemp -d)"
    # shellcheck disable=SC2064
    trap "cleanup $tmpdir" EXIT

    # This is the commit sha that the image was built from and ensures we use the correct configs for the release
    local -r default_commit="87bff5947f"
    local -r hive_image_commit_hash="${1:-$default_commit}"
    log "Using hive commit: $hive_image_commit_hash"
    # shellcheck disable=SC2034
    local -r hive_operator_namespace="hive"

    # For now we'll use the quay hive image, but this will change to an ACR once the quay.io -> ACR mirroring is setup
    # Note: semi-scientific way to get the latest image: `podman search --list-tags --limit 10000 quay.io/app-sre/hive | tail -n1`
    # shellcheck disable=SC2034
    local -r hive_image="quay.io/app-sre/hive:${hive_image_commit_hash}"


    # shellcheck disable=SC2034
    local kustomize_bin
    install_kustomize tmpdir \
                      kustomize_bin
    hive_repo_clone tmpdir
    hive_repo_hash_checkout tmpdir \
                            hive_image_commit_hash
    generate_hive_config kustomize_bin \
                         hive_operator_namespace \
                         hive_image \
                         tmpdir

    log "Hive config generated."
}

install_kustomize() {
    local -n tmpd="$1"
    local -n kustomize="$2"
    log "starting"

    if kustomize="$(which kustomize 2> /dev/null)"; then
        return 0
    fi

    pushd "$tmpd" 1> /dev/null

    # This version is specified in the hive repo and is the only hard dependency for this script
    # https://github.com/openshift/hive/blob/master/vendor/github.com/openshift/build-machinery-go/make/targets/openshift/kustomize.mk#L7
    local -r kustomize_version="4.1.3"
    log "kustomize not detected, downloading..."
    if ! curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/kustomize/v${kustomize_version}/hack/install_kustomize.sh" | bash -s "$kustomize_version" "$tmpd" 1> /dev/null; then
        abort "error downloading kustomize"
    fi

    if [ ! -d "${HOME}/bin" ]; then
        mkdir -p "${HOME}/bin"
    fi

    kustomize_new="${tmpd}/kustomize"
    kustomize_dest="${HOME}/bin/kustomize"
    log "Installing $kustomize_new into $kustomize_dest"
    mv "$kustomize_new" "$kustomize_dest"

    popd 1> /dev/null

    kustomize="$(which kustomize)"
}

hive_repo_clone() {
    local -n tmpd="$1"
    log "starting"

    local -r repo="https://github.com/openshift/hive.git"
    log "Cloning $repo into $tmpd for config generation"
    if ! git clone "$repo" "$tmpd"; then
        log "error cloning the hive repo"
        return 1
    fi
}

hive_repo_hash_checkout() {
    local -n tmpd="$1"
    local -n commit="$2"
    log "starting"
    log "Attempting to use commit: $commit"

    pushd "$tmpd" 1> /dev/null
    git reset --hard "$commit"
    if [ "$?" -ne 0 ] || [ "$( git rev-parse --short="${#commit}" HEAD )" != "$commit" ]; then
        abort "error resetting the hive repo to the correct git hash '${commit}'"
    fi

    popd 1> /dev/null
}

generate_hive_config() {
    local -n kustomize="$1"
    local -n namespace="$2"
    local -n image="$3"
    local -n tmpd="$4"
    log "starting"
    
    pushd "$tmpd" 1> /dev/null
    # Create the hive operator install config using kustomize
    mkdir -p overlays/deploy
    cp overlays/template/kustomization.yaml overlays/deploy
    pushd overlays/deploy >& /dev/null
    $kustomize edit set image registry.ci.openshift.org/openshift/hive-v4.0:hive="$image"
    $kustomize edit set namespace "$namespace"
    popd >& /dev/null

    $kustomize build overlays/deploy > hive-deployment.yaml

    # return to the repo directory to copy the generated config from $TMPDIR
    popd 1> /dev/null
    mv "$tmpd/hive-deployment.yaml" ./hack/hive/hive-config/

    if [ -d ./hack/hive/hive-config/crds ]; then
        rm -rf ./hack/hive/hive-config/crds
    fi
    cp -R "$tmpd/config/crds" ./hack/hive/hive-config/
}

if [ ! -f go.mod ] || [ ! -d ".git" ]; then
    echo "this script must by run from the repo's root directory"
    exit 1
fi

declare -r util_script="hack/util.sh"
if [ -f $util_script ]; then
    # shellcheck source=util.sh
    source "$util_script"
fi

cleanup() {
    local tmpd="$1"
    [ -d "$tmpd" ] && rm -fr "$tmpd"
}

main "$@"
