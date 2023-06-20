package aroprivilegednamespace

import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account
import data.lib.common.get_username

violation[{"msg": msg}] {
  ns := input.review.object.metadata.namespace
  is_priv_namespace(ns)
  not is_exempted_account(input.review)
  username := get_username(input.review)
  msg := sprintf("user %v not allowed to operate in namespace %v", [username, ns])
}
