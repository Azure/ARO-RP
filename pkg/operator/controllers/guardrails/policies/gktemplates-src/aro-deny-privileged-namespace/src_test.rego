package aroprivilegednamespace

test_input_allowed_ns {
  input := { "review": input_ns(input_allowed_ns, non_priv_sa, nonpriv_username_nonpriv_group_userinfo) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1 {
  input := { "review": input_ns(input_disallowed_ns1, non_priv_sa, nonpriv_username_nonpriv_group_userinfo) }
  results := violation with input as input
  count(results) == 1
}

test_input_allowed_ns2 {
  input := { "review": input_ns(input_disallowed_ns1, priv_sa, priv_username_nonpriv_group_userinfo) }
  results := violation with input as input
  count(results) == 0
}

test_input_allowed_ns3 {
  input := { "review": input_ns(input_disallowed_ns1, priv_sa, nonpriv_username_nonpriv_group_userinfo) }
  results := violation with input as input
  count(results) == 1
}

test_input_allowed_ns_new {
  input := { "review": input_ns(input_allowed_ns, non_priv_sa, nonpriv_username_nonpriv_group_userinfo) }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_ns1_new {
  input := { "review": input_ns(input_disallowed_ns1, non_priv_sa, nonpriv_username_priv_group_userinfo) }
  results := violation with input as input
  count(results) == 0
}

input_ns(ns, serviceAccountName, userinfo) = output {
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
    "userInfo":userinfo
  }
}


test_input_disallow_pullsecret_deletion1 {
  input := get_input_with_userinfo(nonpriv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 1
}

test_input_disallow_pullsecret_deletion2 {
  input := get_input_with_userinfo(nonpriv_username_empty_priv_group_userinfo)
  results := violation with input as input
  count(results) == 1
}

test_input_allow_pullsecret_deletion1 {
  input := get_input_with_userinfo(priv_username_priv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_allow_pullsecret_deletion2 {
  input := get_input_with_userinfo(nonpriv_username_priv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_allow_pullsecret_deletion3 {
  input := get_input_with_userinfo(priv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_allow_pullsecret_deletion4 {
  input := get_input_with_userinfo(priv_username_empty_priv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}


get_input_with_userinfo(userinfo) = output {
  output := {
    "parameters":{},
    "review":{
        "dryRun":false,
        "kind":{
          "group":"",
          "kind":"Secret",
          "version":"v1"
        },
        "name":"pull-secret",
        "namespace":"openshift-config",
        "object":{
          "apiVersion":"v1",
          "data":{
              ".dockerconfigjson":"eyJhdXRocyI6eyJhcm9pbnRzdmMuYXp1cmVjci5pbyI6eyJhdXRoIjoiWlRKbExYQjFiR3c2Y1RsaVVGSnJRMVJ5VFd0WmNDOXZORXhzYW1aTmFsSmlSazVEUVhSU1NXZz0ifSwiYXJvc3ZjLmF6dXJlY3IuaW8iOnsiYXV0aCI6Ik9UTTVNRFE1WWpRdE5UbGxNUzAwWXpsaExXSmxZemd0TWpBeVpUQXhaamMyTVdGbE9qWkNMa3BGT21aUFQyaHZMVEkzUDI0NFRsWXliRFpxUVM5VWRqQk1kMWhtIn19fQ=="
          },
          "kind":"Secret",
          "metadata":{
              "name":"pull-secret",
              "namespace":"openshift-config",
              "resourceVersion":"1944",
              "uid":"84a0214c-1ee7-4ed7-bd7f-e7ed69dc6374"
          },
          "type":"kubernetes.io/dockerconfigjson"
        },
        "operation":"DELETE",
        "options":{
          "apiVersion":"meta.k8s.io/v1",
          "kind":"DeleteOptions",
          "propagationPolicy":"Background"
        },
        "requestKind":{
          "group":"",
          "kind":"Secret",
          "version":"v1"
        },
        "requestResource":{
          "group":"",
          "resource":"secrets",
          "version":"v1"
        },
        "resource":{
          "group":"",
          "resource":"secrets",
          "version":"v1"
        },
        "uid":"d914431c-547c-4714-927e-309576e99b48",
        "userInfo": userinfo
    }
  }
}

input_allowed_ns = "mytest"

input_disallowed_ns1 = "openshift-config"

priv_sa = "geneva"
non_priv_sa = "testsa"

priv_user = "system:admin"
non_priv_user = "testuser"

priv_groups = ["system:masters", "system:authenticated"]
non_priv_groups = ["system:cluster-admins", "system:authenticated"]

priv_username_nonpriv_group_userinfo = {
          "groups":non_priv_groups,
          "username":priv_user
        }

nonpriv_username_nonpriv_group_userinfo = {
          "groups":non_priv_groups,
          "username":non_priv_user
        }

priv_username_priv_group_userinfo = {
          "groups":priv_groups,
          "username":priv_user
        }

nonpriv_username_priv_group_userinfo = {
          "groups":priv_groups,
          "username":non_priv_user
        }

nonpriv_username_empty_priv_group_userinfo = {
          "username":non_priv_user
        }

priv_username_empty_priv_group_userinfo = {
          "username":priv_user
        }
