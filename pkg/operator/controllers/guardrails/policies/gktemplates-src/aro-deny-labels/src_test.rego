package arodenylabels

test_input_master_nonpriv_user {
  input := get_input_with_label(master_label, nonpriv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 1
}

test_input_master_priv_user {
  input := get_input_with_label(master_label, priv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_worker_nonpriv_user {
  input := get_input_with_label(worker_label, nonpriv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_worker_priv_user {
  input := get_input_with_label(worker_label, priv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_other_priv_user {
  input := get_input_with_label(other_label, priv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

test_input_other_nonpriv_user {
  input := get_input_with_label(other_label, nonpriv_username_nonpriv_group_userinfo)
  results := violation with input as input
  count(results) == 0
}

get_input_with_label(label, userinfo) = output {
  output := {
    "parameters": {
        "labels": [
            {
                "key": "machine.openshift.io/cluster-api-machine-role",
                "denyRegex": "master"
            }
        ]
    },
    "review": {
            "operation": "DELETE",
            "object": {
                "kind": {
                    "kind": "Pod",
                    "version": "v1"
                },
                "metadata": {
                    "name": "myapp",
                    "namespace": "default",
                    "labels": {
                        "machine.openshift.io/cluster-api-machine-role": label,
                    }
                },
                "spec": {
                    "containers": []
                }
            },
            "uid":"d914431c-547c-4714-927e-309576e99b48",
            "userInfo": userinfo
        }
    }
}

master_label = "master"
worker_label = "worker"
other_label = "infra"

priv_user = "system:admin"
non_priv_user = "testuser"

priv_groups = ["system:serviceaccount:openshift-machine-config-operator:machine-config-controller", "system:authenticated"]
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
