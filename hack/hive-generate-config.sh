#!/bin/bash

# This is the commit sha that the image was built from and ensures we use the correct configs for the release
HIVE_IMAGE_COMMIT_HASH="56adaaa"

# For now we'll use the quay hive image, but this will change to an ACR once the quay.io -> ACR mirroring is setup
# Note: semi-scientific way to get the latest image: `podman search --list-tags --limit 10000 quay.io/app-sre/hive | tail -n1`
HIVE_IMAGE="quay.io/app-sre/hive:${HIVE_IMAGE_COMMIT_HASH}"

HIVE_OPERATOR_NS="hive"

# This version is specified in the hive repo and is the only hard dependency for this script
# https://github.com/openshift/hive/blob/master/vendor/github.com/openshift/build-machinery-go/make/targets/openshift/kustomize.mk#L7
KUSTOMIZE_VERSION=4.1.3
KUSTOMIZE=$( which kustomize 2>/dev/null )
TMPDIR=$( mktemp -d )

function cleanup {
	popd >& /dev/null
	[ -d "$TMPDIR" ] && rm -fr "$TMPDIR"
}

function verify_kustomize {
	if [ ! -z "$KUSTOMIZE" ]; then
		return
	fi
	echo -n "kustomize not detected, downloading "
	curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/kustomize/v${KUSTOMIZE_VERSION}/hack/install_kustomize.sh" | bash -s "$KUSTOMIZE_VERSION" "$TMPDIR"
	if [ $? -ne 0 ]; then
		echo "error downloading kustomize"
		exit 1
	fi
	KUSTOMIZE="${TMPDIR}/kustomize"
}

function hive_repo_clone {
	echo -n "Cloning hive repo into tmp for config generation"
	CLONE_ERROR=$(git clone https://github.com/openshift/hive.git "$TMPDIR" 2>/dev/null )
	if [ $? -ne 0 ]; then
		echo ": error cloning the hive repo: ${CLONE_ERROR}"
		exit 1
	fi
	echo ", done."
}

function hive_repo_hash_checkout {
	# go into $TMPDIR and checkout the commit the image was built with
	pushd $TMPDIR >& /dev/null
	git reset --hard $HIVE_IMAGE_COMMIT_HASH
	if [ $? -ne 0 ] || [[ $( git rev-parse --short=${#HIVE_IMAGE_COMMIT_HASH} HEAD ) != ${HIVE_IMAGE_COMMIT_HASH} ]]; then
		echo "error resetting the hive repo to the correct git hash '${HIVE_IMAGE_COMMIT_HASH}'"
		exit 1
	fi
}

function generate_hive_config {
	# Create the hive operator install config using kustomize
	mkdir -p overlays/deploy
	cp overlays/template/kustomization.yaml overlays/deploy
	pushd overlays/deploy >& /dev/null
	$KUSTOMIZE edit set image registry.ci.openshift.org/openshift/hive-v4.0:hive=$HIVE_IMAGE
	$KUSTOMIZE edit set namespace $HIVE_OPERATOR_NS
	popd >& /dev/null

	$KUSTOMIZE build overlays/deploy > hive-deployment.yaml

	# return to the repo directory to copy the generated config from $TMPDIR
	popd >& /dev/null
	mv "$TMPDIR/hive-deployment.yaml" ./hack/hive-config/

	if [ -d ./hack/hive-config/crds ]; then
		rm -fr ./hack/hive-config/crds
	fi
	cp -R "$TMPDIR/config/crds" ./hack/hive-config/
}

set -e
trap cleanup EXIT

if [ ! -f go.mod ] || [ ! -d ".git" ]; then
	echo "this script must by run from the repo's root directory"
	exit 1
fi
if [[ ! "$TMPDIR" || ! -d "$TMPDIR" ]]; then
	echo "could not create temp working dir"
	exit 1
fi

hive_repo_clone
hive_repo_hash_checkout
verify_kustomize
generate_hive_config

echo -e "\nHive config generated."
