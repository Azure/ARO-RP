package arodenyregularuserapigroups

import future.keywords.in
import data.lib.common.is_exempted_account

# Prevent regular users from managing certain resources in restricted API group
violation[{"msg": msg}] {
    input.review.operation in ["CREATE", "UPDATE", "DELETE"]
    not is_exempted_account(input.review)
    msg := "User is not allowed to manage the resource in the API group"
}