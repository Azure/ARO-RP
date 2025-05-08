package arodenyprivilegednamespace

import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account
import data.lib.common.get_username

violation[{"msg": msg}] {
  is_namespace(input.review)
  ns := input.review.name
  is_priv_namespace(ns)
  not is_exempted_account(input.review)
  username := get_username(input.review)
  msg := sprintf("user %v not allowed to %v namespace %v", [username, input.review.operation, ns])
} {
  not is_namespace(input.review)
  ns := input.review.object.metadata.namespace
  is_priv_namespace(ns)
  not is_exempted_account(input.review)
  username := get_username(input.review)
  msg := sprintf("user %v not allowed to %v %v in namespace %v", [username, input.review.operation, input.review.name, ns])
}

is_namespace(review) {
  review.kind.kind == "Namespace"
}
