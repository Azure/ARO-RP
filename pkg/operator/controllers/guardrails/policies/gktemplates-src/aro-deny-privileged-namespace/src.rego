package aroprivilegednamespace

import data.lib.common.is_priv_namespace
import data.lib.common.is_exempted_account
import data.lib.common.get_user

violation[{"msg": msg}] {
  ns := input.review.object.metadata.namespace
  is_priv_namespace(ns)
  not is_exempted_account(input.review)

  userinfo := get_user(input.review)

  msg := sprintf("%v not allowed to operate in namespace %v", [userinfo, ns])
}
