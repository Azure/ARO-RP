package arodenymastertolerationtaints

import future.keywords.in
import future.keywords.contains
import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account

violation[{"msg": msg}] {
    # Check if the input namespace is a non-privileged namespace
    ns := input.review.object.metadata.namespace
    not is_priv_namespace(ns)

    # Check if the input operation is CREATE or UPDATE
    input.review.operation in ["CREATE", "UPDATE"]

    # Check if it is a regular user
    not is_exempted_account(input.review)

    # Check if pod object has master toleration taints
    tolerations := input.review.object.spec.tolerations
    some toleration in tolerations
    is_master_toleration(toleration.key)

    msg := "Create or update resources to have master toleration taints is not allowed in non-privileged namespaces"
}


is_master_toleration(toleration_key){
    contains(toleration_key,"node-role.kubernetes.io/master")
}

is_master_toleration(toleration_key){
    contains(toleration_key,"node-role.kubernetes.io/control-plane")
}
