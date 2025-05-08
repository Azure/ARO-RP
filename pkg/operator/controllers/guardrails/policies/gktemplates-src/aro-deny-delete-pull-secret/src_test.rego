package arodenydeletepullsecret


test_input_not_allowed_with_delete {
    input := { 
        "review": fake_pull_secret_input_review("DELETE")
    }
    results := violation with input as input
    count(results) == 1
}

test_input_allowed_with_update {
    input := { 
        "review": fake_pull_secret_input_review("CREATE")
    }
    results := violation with input as input
    count(results) == 0
}

test_input_allowed_with_create {
    input := { 
        "review": fake_pull_secret_input_review("CREATE")
    }
    results := violation with input as input
    count(results) == 0
}

test_input_allowed_with_get {
    input := { 
        "review": fake_pull_secret_input_review("GET")
    }
    results := violation with input as input
    count(results) == 0
}

fake_pull_secret_input_review(operation) = review {
    review = {
        "operation": operation,
        "kind": {
            "kind": "Secret"
        },
        "object": {
            "metadata": {
                "name": "pull-secret",
                "namespace": "openshift-config"
            }
        },
        "userInfo":{
            "username":"testuser"
        }
    }
}
