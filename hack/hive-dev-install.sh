#!/bin/bash

HIVE_OPERATOR_NS="hive"
KUBECTL=$( which kubectl 2> /dev/null || which oc 2> /dev/null)

function cleanup {
	[ -f "$(pwd)/kubectl" ] && rm -f "$(pwd)/kubectl"
}

function download_tmp_kubectl {
	curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
	if [ $? -ne 0 ]; then
		echo ": error downloading kubectl"
		exit 1
	fi
	chmod 755 kubectl
	KUBECTL="$(pwd)/kubectl"
}

function verify_tools {
	if [ ! -z "$KUBECTL" ]; then
		return
	fi
	echo -n "kubectl or oc not detected, downloading"
	download_tmp_kubectl
	echo ", done."

	if [ $( $KUBECTL get nodes 2>/dev/null | wc -l ) -eq 0 ]; then
		echo "unable to connect to the cluster"
		exit 1
	fi
}

set -e
trap cleanup EXIT

if [ ! -f go.mod ] || [ ! -d ".git" ]; then
	echo "this script must by run from the repo's root directory"
	exit 1
fi

if [ ! -f "./hack/hive-config/hive-deployment.yaml" ] || [ ! -d "./hack/hive-config/crds" ] ; then
	echo "hive config is missing, generating config, please rerun this script afterwards"
	./hack/hive-generate-config.sh
	if [ $? -ne 0 ]; then
		echo "error generating the hive configs"
		exit 1
	fi
fi

if [ -z "$PULL_SECRET" ]; then
	echo "global pull secret variable required, please source ./env"
	exit
fi

verify_tools
local skip_deployments
skip_deployments=${SKIP_DEPLOYMENTS}

if [ $( $KUBECTL get namespace $HIVE_OPERATOR_NS -o yaml 2>/dev/null | wc -l ) -ne 0 ]; then
	echo "hive is already installed in the namespace"
	if [[ -z $skip_deployments]]; then
		log "SKIP_DEPLOYMENTS was not set, then use default value of 'true'"
        skip_deployments=true
	fi
	source hack/devtools/rp_dev_helper.sh && is_it_boolean $skip_deployments
	# Don't skip deployment creation when SKIP_DEPLOYMENTS is set to "false" 
	if $skip_deployments; then
		log "'SKIP_DEPLOYMENTS' env var is set to true. ⏩📋 Skip Hive installation"
		abort
	else
		log "'SKIP_DEPLOYMENTS' env var is set to false. ❌⏩ Don't skip Hive installation, and try to reinstall it"
	fi
	else
		echo -n "would you like to reapply the configs? (y/N): "
		read answer
		if [[ "$answer" != "y" ]]; then
			abort
		fi
	fi
else
	$KUBECTL create namespace $HIVE_OPERATOR_NS
fi

$KUBECTL apply -f ./hack/hive-config/crds

echo "$PULL_SECRET" > /tmp/.tmp-secret
# Using dry-run allows updates to work seamlessly
$KUBECTL create secret generic hive-global-pull-secret --from-file=.dockerconfigjson=/tmp/.tmp-secret --type=kubernetes.io/dockerconfigjson --namespace $HIVE_OPERATOR_NS -o yaml --dry-run=client | $KUBECTL apply -f - 2>/dev/null
rm -f /tmp/.tmp-secret

sed "s/HIVE_OPERATOR_NS/$HIVE_OPERATOR_NS/g" hack/hive-config/hive-config.yaml | $KUBECTL apply -f -
$KUBECTL apply -f ./hack/hive-config/hive-additional-install-log-regexes.yaml

$KUBECTL apply -f ./hack/hive-config/hive-deployment.yaml

$KUBECTL wait --timeout=5m --for=condition=Available --namespace $HIVE_OPERATOR_NS deployment/hive-operator

echo -e "\nHive is installed."
