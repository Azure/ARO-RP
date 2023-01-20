package aroprivilegednamespace

violation[{"msg": msg}] {
  input_priv_namespace(input.review.object.metadata.namespace)
  msg := sprintf("Operation in privileged namespace %v is not allowed", [input.review.object.metadata.namespace])
}

input_priv_namespace(ns) {
  any([ns == "default",
  ns == "com",
  ns == "io",
  ns == "in",
  startswith(ns, "openshift"),
  startswith(ns, "kube"),
  startswith(ns, "redhat")])
}
