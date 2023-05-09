package arodenyregularuserapigroups

import future.keywords.in
import data.lib.common.get_user

# Prevent regular users from managing certain resources in restricted API group
violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]
    user := get_user(input.review)
    not regex.match("^(system|kube):.+", user)
    msg := "User is not allowed to manage the resource in the API group"
}
