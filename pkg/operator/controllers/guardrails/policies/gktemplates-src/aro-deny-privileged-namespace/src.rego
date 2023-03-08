package aroprivilegednamespace

import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_user
import data.lib.common.get_service_account

violation[{"msg": msg}] {
  ns := input.review.object.metadata.namespace
  user := get_service_account(input.review.object)
  is_priv_namespace(ns)
  not is_exempted_user(user)
  msg := sprintf("User %v not allowed to operate in namespace %v", [user, ns])
}