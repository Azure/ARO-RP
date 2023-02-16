package arodenymastertolerationtaints


test_input_allowed_in_privileged_ns_with_master_taint {
    input := { 
        "review": fake_input_review("openshift-config", "CREATE"), 
        "request": fake_input_request("node-role.kubernetes.io/worker", "node-role.kubernetes.io/master") 
    }
    results := violation with input as input
    count(results) == 0
}

test_input_allowed_in_nonprivileged_ns_with_no_master_taint {
    input := { 
        "review": fake_input_review("customer", "CREATE"), 
        "request": fake_input_request("node-role.kubernetes.io/worker", "node-role.kubernetes.io/worker") 
    }
    results := violation with input as input
    count(results) == 0
}

test_input_allowed_in_nonprivileged_ns_with_delete_operation {
    input := { 
        "review": fake_input_review("customer", "DELETE"), 
        "request": fake_input_request("node-role.kubernetes.io/worker", "node-role.kubernetes.io/control-plane") 
    }
    results := violation with input as input
    count(results) == 0
}

test_input_not_allowed_in_nonprivileged_ns_with_create_operation {
    input := { 
        "review": fake_input_review("customer", "CREATE"), 
        "request": fake_input_request("node-role.kubernetes.io/worker", "node-role.kubernetes.io/master") 
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_in_nonprivileged_ns_with_update_operation {
    input := { 
        "review": fake_input_review("customer", "UPDATE"), 
        "request": fake_input_request("node-role.kubernetes.io/worker", "node-role.kubernetes.io/control-plane") 
    }
    results := violation with input as input
    count(results) == 1
}


fake_input_review(namespace, operation) = review {
    review = {
        "operation": operation,
        "object": {
            "metadata": {
                "namespace": namespace
            }
        }
        
    }
}
  
fake_input_request(taint_key_one, taint_key_two) = request {
    request = {
        "kind": {
            "kind": "Pod"
        },
        "object": {
            "spec": {
                "tolerations": [
                    {
                        "key": taint_key_one,
                        "effect": "NoExecute"
                    },
                    {
                        "key": taint_key_two,
                        "effect": "NoExecute"
                    }
                ]
            }
        }
    }
}