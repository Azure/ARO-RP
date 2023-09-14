package arodenymachineconfig


test_input_not_allowed_with_master_keyword {
    input := { 
        "review": fake_machine_config_input_review("01-master-kubelet", "UPDATE")
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_worker_keyword {
    input := { 
        "review": fake_machine_config_input_review("99-worker-generated-registries", "CREATE")
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_aro_keyword {
    input := { 
        "review": fake_machine_config_input_review("99-worker-aro-dns", "DELETE")
    }
    results := violation with input as input
    count(results) == 1

}

test_input_allowed_with_custom_name {
    input := { 
        "review": fake_machine_config_input_review("new-customer-dns", "CREATE")
    }
    results := violation with input as input
    count(results) == 0
}

test_input_allowed_with_read_operation {
    input := { 
        "review": fake_machine_config_input_review("99-worker-generated-registries", "GET")
    }
    results := violation with input as input
    count(results) == 0
}

fake_machine_config_input_review(name, operation) = review {
    review = {
        "operation": operation,
        "kind": {
            "kind": "MachineConfig"
        },
        "object": {
            "metadata": {
                "name": name
            }
        },
        "userInfo":{
            "username":"testuser"
        }
    }
}
