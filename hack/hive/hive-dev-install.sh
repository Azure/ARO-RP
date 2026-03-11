#!/bin/bash

set -o errexit \
    -o nounset

main() {
	info "starting"

	if [ ! -f "$HIVE_CONFIG/hive-deployment.yaml" ] || [ ! -d "$HIVE_CONFIG/crds" ]; then
        fatal "hive-config is incomplete. Regenerate by running hack/hive/hive-generate-config.sh"
	fi

    kubectl="$(which kubectl 2> /dev/null)" \
    || kubectl_install kubectl

    info "Creating $HIVE_OPERATOR_NS"
	$kubectl get namespace "$HIVE_OPERATOR_NS" > /dev/null 2>&1 || $kubectl create namespace "$HIVE_OPERATOR_NS"

    info "Applying $HIVE_CONFIG/crds"
	$kubectl apply -f "$HIVE_CONFIG/crds"

	[ -z "$PULL_SECRET" ] && fatal "global pull secret variable required, please source ./env"
    info "Generating and applying secret $HIVE_GLOBAL_PULL_SECRET_NAME"
    kubectl create \
            secret \
            generic \
            "$HIVE_GLOBAL_PULL_SECRET_NAME" \
            --from-literal .dockerconfigjson="$PULL_SECRET" \
            --type=kubernetes.io/dockerconfigjson \
            --namespace="$HIVE_OPERATOR_NS" \
            -o yaml \
            --dry-run=client \
    | $kubectl apply -f -

	sed "s/HIVE_OPERATOR_NS/$HIVE_OPERATOR_NS/g" "$HIVE_CONFIG/hive-config.yaml" | $kubectl apply -f -
	$kubectl apply -f "$HIVE_CONFIG/hive-additional-install-log-regexes.yaml"
	$kubectl apply -f "$HIVE_CONFIG/hive-deployment.yaml"

    # shellcheck disable=SC2329
	kubectl_get() { $kubectl -n "$HIVE_OPERATOR_NS" get "$1"; }

    info "Getting $HIVE_DEPLOYMENT_OPERATOR"
    # Ensure the deployment exists before we wait on it
    retry "kubectl_get $HIVE_DEPLOYMENT_OPERATOR" || fatal "$HIVE_DEPLOYMENT_CONTROLLERS failed to create."
    info "Waiting for $HIVE_DEPLOYMENT_OPERATOR to become available."
	$kubectl wait \
        --timeout=5m \
        --for=condition=Available \
        --namespace \
        "$HIVE_OPERATOR_NS" \
        "$HIVE_DEPLOYMENT_OPERATOR"

    info "Getting $HIVE_DEPLOYMENT_CONTROLLERS"
    # Ensure the deployment exists before we wait on it
    retry "kubectl_get $HIVE_DEPLOYMENT_CONTROLLERS" || fatal "$HIVE_DEPLOYMENT_CONTROLLERS failed to create."
	info "Waiting for $HIVE_DEPLOYMENT_CONTROLLERS to be available..."
	$kubectl wait \
        --timeout=5m \
        --for=condition=Available \
        --namespace "$HIVE_OPERATOR_NS" \
        "$HIVE_DEPLOYMENT_CONTROLLERS"
}

kubectl_install() {
    local -n kubectl_bin="$1"
    info "starting"

    kubectl_stable_version="$(curl -L -s https://dl.k8s.io/release/stable.txt)"
    kubectl_stable_url="https://dl.k8s.io/release/$kubectl_stable_version/bin/linux/amd64/kubectl"

    info "Downloading $kubectl_stable_url"
	curl -sLO \
        --create-dirs \
        --output-dir "$HOME/bin" \
         \
    || error "Failed to download $kubectl_stable_url"

    # shellcheck disable=SC2034
    kubectl_bin="$HOME/bin/kubectl"
	chmod 755 "$kubectl_bin" || fatal "failed to mark $kubectl_bin as executable."
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
