package arohostmount

test_priv_ns_pod {
  input := { "review": input_pod(input_priv_ns, non_priv_sa, non_priv_username, non_privileged, false) }
  results := violation with input as input
  count(results) == 0
}

test_non_priv_ns1_pod {
  input := { "review": input_pod(input_non_priv_ns1, non_priv_sa, non_priv_username, privileged, false) }
  results := violation with input as input
  count(results) == 1
}

test_non_priv_ns2_pod {
  input := { "review": input_pod(input_non_priv_ns1, priv_sa, non_priv_username, privileged, true) }
  results := violation with input as input
  count(results) == 0
}

input_pod(ns, account, username, priv, ro_host_mount) = output {
  output = {
    "object": {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "namespace": ns,
        "name": "test1"
      },
      "spec": {
        "serviceAccountName": account,
        "containers" : [
          {
            "name": "nginx",
            "image": "nginx",
            "securityContext": {
              "privileged": priv
            },
            "volumeMounts": [
              {
                "mountPath": "/cache",
                "name": "host",
                "readOnly": ro_host_mount
              }
            ],
          }
        ],
        "volumes": [
          {
            "name": "host",
            "hostPath": {
                "path": "/test1"
            }
          }
        ],
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

input_priv_ns = "openshift-config"

input_non_priv_ns1 = "mytest"

priv_sa = "aro-sre"
non_priv_sa = "test"

privileged = true
non_privileged = false

priv_username = "system:admin"
non_priv_username = "testuser"

test_rw_pv {
  input := {"review": get_input_pv_with_username(non_priv_username, "ReadWriteOnce")}
  results := violation with input as input
  count(results) == 1
}

test_rw_pv1 {
  input := {"review": get_input_pv_with_username(non_priv_username, "ReadWriteMany")}
  results := violation with input as input
  count(results) == 1
}

test_rw_pv2 {
  input := {"review": get_input_pv_with_username(non_priv_username, "ReadWriteOncePod")}
  results := violation with input as input
  count(results) == 1
}

test_ro_pv {
  input := {"review": get_input_pv_with_username(non_priv_username, "ReadOnlyMany")}
  results := violation with input as input
  count(results) == 0
}

test_rw_pv3 {
  input := {"review": get_input_pv_with_username(priv_username, "ReadWriteMany")}
  results := violation with input as input
  count(results) == 0
}

get_input_pv_with_username(username, access) = output {
  output = {
    "object": {
      "apiVersion": "v1",
      "kind": "PersistentVolume",
      "metadata": {
        "name": "test_pv1"
      },
      "spec": {
        "accessModes": [
          access
        ],
        "storageClassName": "manual",
        "hostPath": {
            "path": "/test1"
        },
        "capacity": {
            "storage": "10Gi"
        }
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
