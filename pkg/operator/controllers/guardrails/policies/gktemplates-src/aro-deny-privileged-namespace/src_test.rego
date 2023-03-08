package aroprivilegednamespace

test_input_allowed_ns {
  input := { "review": input_ns(input_allowed_ns, non_priv_user) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := { "review": input_ns(input_disallowed_ns1, non_priv_user) }
  results := violation with input as input
  count(results) == 1
}

test_input_allowed_ns2 {
  input := { "review": input_ns(input_disallowed_ns1, priv_user) }
  results := violation with input as input
  count(results) == 0
}

input_ns(ns, user) = output {
  output = {
    "object": {
      "kind": "Pod",
      "metadata": {
        "namespace": ns
      },
      "spec": {
        "serviceAccountName": user
      }
    }
  }
}

input_allowed_ns = "mytest"

input_disallowed_ns1 = "openshift-config"

priv_user = "geneva"
non_priv_user = "test"