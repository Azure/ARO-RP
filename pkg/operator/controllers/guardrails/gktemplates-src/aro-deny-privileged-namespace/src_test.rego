package aroprivilegednamespace

test_input_allowed_ns {
  input := { "review": input_ns(input_allowed_ns) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := { "review": input_review(input_disallowed_ns1) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns2 {
  input := { "review": input_review(input_disallowed_ns2) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns3 {
  input := { "review": input_review(input_disallowed_ns3) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns4 {
  input := { "review": input_review(input_disallowed_ns4) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns5 {
  input := { "review": input_review(input_disallowed_ns5) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns6 {
  input := { "review": input_review(input_disallowed_ns6) }
  results := violation with input as input
  count(results) == 1
}

test_input_disallowed_ns7 {
  input := { "review": input_review(input_disallowed_ns7) }
  results := violation with input as input
  count(results) == 1
}

input_ns(ns) = output {
  output = {
    "object": {
      "metadata": {
        "namespace": ns
      }
    }
  }
}

input_allowed_ns = "mytest"

input_disallowed_ns1 = "openshift-config"
input_disallowed_ns2 = "kube-config"
input_disallowed_ns3 = "redhat-config"
input_disallowed_ns4 = "default"
input_disallowed_ns5 = "com"
input_disallowed_ns6 = "io"
input_disallowed_ns7 = "in"
