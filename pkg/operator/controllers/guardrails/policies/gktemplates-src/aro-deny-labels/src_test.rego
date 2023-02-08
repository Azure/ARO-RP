package arodenylabels

test_input_master {
  input := get_input_with_label(master_label)
  results := violation with input as input
  count(results) == 1
}

test_input_worker {
  input := get_input_with_label(worker_label)
  results := violation with input as input
  count(results) == 0
}

test_input_other {
  input := get_input_with_label(other_label)
  results := violation with input as input
  count(results) == 0
}

get_input_with_label(label) = output {
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
            }
        }
    }
}

master_label = "master"
worker_label = "worker"
other_label = "infra"
