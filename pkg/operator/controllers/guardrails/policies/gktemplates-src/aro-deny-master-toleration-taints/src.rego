package arodenymastertolerationtaints

import future.keywords.in
import data.lib.common.is_priv_namespace

violation[{"msg": msg}] {
    # Check if the input namespace is a non-privileged namespace
    ns := input.review.object.metadata.namespace
    not is_priv_namespace(ns)

    # Check if the input operation is CREATE or UPDATE
    input.review.operation in ["CREATE", "UPDATE"]

    # Check if pod object has master toleration taints
    tolerations := input.review.object.spec.tolerations
    some toleration in tolerations
    toleration.key in ["node-role.kubernetes.io/master", "node-role.kubernetes.io/control-plane"]

    msg := "Create or update resources to have master toleration taints is not allowed in non-privileged namespaces"
}