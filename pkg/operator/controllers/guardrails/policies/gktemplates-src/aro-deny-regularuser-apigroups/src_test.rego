package arodenyregularuserapigroups

# Test cases
test_create_violation {
    input := {
        "review": fake_regularuser_validation_input_review("CREATE", "user:restricted")
    }

    results := violation with input as input
    count(results) == 1
}

test_update_violation {
    input := {
        "review": fake_regularuser_validation_input_review("UPDATE", "user:restricted")
    }

    results := violation with input as input
    count(results) == 1
}

test_delete_violation {
    input := {
        "review": fake_regularuser_validation_input_review("DELETE", "user:restricted")
    }

    results := violation with input as input
    count(results) == 1
}

test_no_violation {
    input := {
        "review": fake_regularuser_validation_input_review("CREATE", "system:allowed")
    }

    results := violation with input as input
    count(results) == 0
}

test_no_violation {
    input := {
        "review": fake_regularuser_validation_input_review("CREATE", "kube:allowed")
    }

    results := violation with input as input
    count(results) == 0
}

fake_regularuser_validation_input_review(operation, name) = review {
    review = {
        "operation": operation,
        "userInfo": {
            "username": name
        }
    }
}
