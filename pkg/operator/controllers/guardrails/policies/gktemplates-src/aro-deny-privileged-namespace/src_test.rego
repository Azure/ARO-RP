package aroprivilegednamespace

test_input_allowed_ns {
  input := { "review": input_ns(input_allowed_ns, non_priv_sa, non_priv_user) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := { "review": input_ns(input_disallowed_ns1, non_priv_sa, non_priv_user) }
  results := violation with input as input
  count(results) == 1
}

test_input_allowed_ns2 {
  input := { "review": input_ns(input_disallowed_ns1, priv_sa, priv_user) }
  results := violation with input as input
  count(results) == 0
}

input_ns(ns, serviceAccountName, username) = output {
  output = {
    "object": {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "namespace": ns
      },
      "spec": {
        "serviceAccountName":serviceAccountName,
        "containers":[
            {
              "image":"nginx",
              "name":"test"
            }
        ]        
      }
    },
    "userInfo":{
       "groups":[
          "system:masters",
          "system:authenticated"
       ],
       "username": username # "system:admin"
    }
  }
}

input_allowed_ns = "mytest"

input_disallowed_ns1 = "openshift-config"

priv_sa = "geneva"
non_priv_sa = "testsa"

priv_user = "system:admin"
non_priv_user = "testuser"

test_input_allowed_ns_new {
  input := { "review": input_ns(input_allowed_ns, non_priv_sa, non_priv_user) }
  results := violation with input as input
  count(results) == 0
}

# test_input_disallowed_ns1_new {
#   input := { "review": input_ns(input_disallowed_ns1, non_priv_sa, priv_user) }
#   results := violation with input as input
#   count(results) == 0
# }

