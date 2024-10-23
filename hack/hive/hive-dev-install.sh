#!/bin/bash

set -o errexit \
    -o nounset

declare -r utils=hack/util.sh
if [ -f "$utils" ]; then
    # shellcheck source=../util.sh  
    source "$utils"  
fi

HIVE_OPERATOR_NS="hive"
KUBECTL="$( which kubectl 2> /dev/null || which oc 2> /dev/null)"

if [ ! -f go.mod ] || [ ! -d ".git" ]; then
	abort "this script must by run from the repo's root directory"
fi


function cleanup() {
	[ -f "$(pwd)/kubectl" ] && rm -f "$(pwd)/kubectl"
}
trap cleanup EXIT

main() {
	log "enter hive installation"
	local skip_deployments=${1:-"none"}

	if [ ! -f "./hack/hive/hive-config/hive-deployment.yaml" ] || [ ! -d "./hack/hive/hive-config/crds" ] ; then
		log "hive config is missing, generating config, please rerun this script afterwards"
		./hack/hive/hive-generate-config.sh
		if [ $? -ne 0 ]; then
			abort "error generating the hive configs"
		fi
	fi

	if [[ -z "$PULL_SECRET" ]]; then
		log "global pull secret variable required, please source ./env"
		exit
	fi
	verify_tools

	if [ "$( $KUBECTL get namespace $HIVE_OPERATOR_NS -o yaml 2>/dev/null | wc -l )" -ne 0 ]; then
		log "hive is already installed in namespace $HIVE_OPERATOR_NS"
			log "would you like to reapply the configs? (y/N): "
			read answer
			if [[ "$answer" != "y" ]]; then
				exit
		    fi
	else
		$KUBECTL create namespace $HIVE_OPERATOR_NS
	fi

	log "Hive is ready to be installed"
	$KUBECTL apply -f ./hack/hive/hive-config/crds
	echo "$PULL_SECRET" > /tmp/.tmp-secret
	# Using dry-run allows updates to work seamlessly
	$KUBECTL create secret generic hive-global-pull-secret \
        --from-file=.dockerconfigjson=/tmp/.tmp-secret \
        --type=kubernetes.io/dockerconfigjson \
        --namespace $HIVE_OPERATOR_NS \
        -o yaml \
        --dry-run=client \
        | $KUBECTL apply -f - 2>/dev/null
	rm -f /tmp/.tmp-secret

	sed "s/HIVE_OPERATOR_NS/$HIVE_OPERATOR_NS/g" hack/hive/hive-config/hive-config.yaml | $KUBECTL apply -f -
	$KUBECTL apply -f ./hack/hive/hive-config/hive-additional-install-log-regexes.yaml
	$KUBECTL apply -f ./hack/hive/hive-config/hive-deployment.yaml
	$KUBECTL wait --timeout=5m --for=condition=Available --namespace $HIVE_OPERATOR_NS deployment/hive-operator

	log "Hive is installed but to check Hive readiness use one of the following options to monitor the deployment rollout:
        'kubectl wait --timeout=5m --for=condition=Available --namespace "$HIVE_OPERATOR_NS" deployment/hive-controllers' 
        or 'kubectl wait --timeout=5m  --for=condition=Ready --namespace "$HIVE_OPERATOR_NS" pod --selector control-plane=clustersync'"
}

function download_tmp_kubectl() {
	if curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; then  
		abort ": error downloading kubectl"
	fi
	chmod 755 kubectl
	KUBECTL="$(pwd)/kubectl"
}

function verify_tools() {
	if [ -n "$KUBECTL" ]; then
		return
	fi
	log "kubectl or oc not detected, downloading"
	download_tmp_kubectl
	log "done: downloading kubectl/oc was completed"

	if [ "$( $KUBECTL get nodes 2>/dev/null | wc -l )" -eq 0 ]; then
		abort "unable to connect to the cluster"
	fi
}

main "$@"
