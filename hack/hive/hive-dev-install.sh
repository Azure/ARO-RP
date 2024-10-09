#!/bin/bash

set -o errexit \
    -o nounset \
    -o monitor

declare -r utils=hack/util.sh
if [ -f "$utils" ]; then
    # shellcheck source=../util.sh  
    source "$utils"  
fi

HIVE_OPERATOR_NS="hive"
HIVE_CONFIG_DIR="hack/hive/hive-config"
HIVE_CONFIG_CRDS="${HIVE_CONFIG_DIR}/crds"
HIVE_CONFIG_DEP="${HIVE_CONFIG_DIR}/hive-deployment.yaml"

if [ ! -f go.mod ] || [ ! -d ".git" ]; then
	abort "this script must by run from the repo's root directory"
fi


function cleanup() {
	[ -f "$(pwd)/kubectl" ] && rm -f "$(pwd)/kubectl"
}
trap cleanup EXIT

main() {
	log "enter hive installation"
    err_str="Usage $0 <PULL_SECRET> [SKIP_DEPLOYMENTS]. Please try again"
	local pull_secret="${1?$err_str}"
    local skip_deployments="${2:-"none"}"

	if [ ! -f "./${HIVE_CONFIG_DEP}" ] || [ ! -d "./${HIVE_CONFIG_CRDS}" ] ; then
		log "hive config is missing, generating config, please rerun this script afterwards"
		if ! ./hack/hive/hive-generate-config.sh; then
			abort "error generating the hive configs"
		fi
	fi

    local kubectl
    set_kubectl_binary kubectl
    if ! $kubectl get nodes >/dev/null 2>&1; then
		abort "unable to connect to the cluster"
	fi
    log "Connected to the AKS cluster"
    if $kubectl get namespace $HIVE_OPERATOR_NS >/dev/null 2>&1; then
        log "Hive is already installed in namespace '$HIVE_OPERATOR_NS'"
        if [ "${skip_deployments}" = false ]; then
            log "'skip_deployments' is set to 'false'. ❌⏩ Don't skip Hive installation, and try to reinstall it"
        elif [ "${skip_deployments}" = true ]; then
            log "'skip_deployments' is set to 'true'. ⏩📋 Skip Hive installation"
            exit
        else
            echo -n "would you like to reapply the configs? (y/N): "
            read answer
            if [[ "$answer" != "y" ]]; then
                exit
            fi
        fi
    else
        $kubectl create namespace $HIVE_OPERATOR_NS
    fi

	log "Hive is ready to be installed"
	$kubectl apply -f ./${HIVE_CONFIG_CRDS}
	# Using dry-run allows updates to work seamlessly
	$kubectl create secret generic hive-global-pull-secret \
        --from-literal=.dockerconfigjson="${pull_secret}" \
        --type=kubernetes.io/dockerconfigjson \
        --namespace $HIVE_OPERATOR_NS \
        -o yaml \
        --dry-run=client \
        | $kubectl apply -f -

	sed "s/HIVE_OPERATOR_NS/$HIVE_OPERATOR_NS/g" ${HIVE_CONFIG_DIR}/hive-config.yaml | $kubectl apply -f -
	$kubectl apply -f ./${HIVE_CONFIG_DIR}/hive-additional-install-log-regexes.yaml
	$kubectl apply -f ./${HIVE_CONFIG_DEP}
	$kubectl wait --timeout=5m --for=condition=Available --namespace $HIVE_OPERATOR_NS deployment/hive-operator

	log "Hive is installed but to check Hive readiness use one of the following options to monitor the deployment rollout:
        'kubectl wait --timeout=5m --for=condition=Available --namespace $HIVE_OPERATOR_NS deployment/hive-controllers' 
        or 'kubectl wait --timeout=5m  --for=condition=Ready --namespace $HIVE_OPERATOR_NS pod --selector control-plane=clustersync'"
}


function set_kubectl_binary() {
    local -n tmp_kubectl="$1"
    tmp_kubectl="$( which kubectl 2> /dev/null || true)"
    local oc="$( which oc 2> /dev/null || true)"
    if [[ -n "$tmp_kubectl" ]]; then
		log "'kubectl' was detected"
        return
    elif [[ -n "$oc" ]]; then
		log "'oc' was detected"
        tmp_kubectl="$oc"
        return
	fi
	log "'kubectl' and 'oc' were not detected, downloading kubectl"
	if ! curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; then  
		abort "error downloading kubectl"
	fi
	chmod 755 kubectl
    log "done: downloading kubectl was completed"
	tmp_kubectl="$(pwd)/kubectl"
}

main "$@"
