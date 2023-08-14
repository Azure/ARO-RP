package arodenypvrwhostpath

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
      #  "groups":[
      #     "system:masters",
      #     "system:authenticated"
      #  ],
       "username": username # "system:admin"
    }
  }
}
