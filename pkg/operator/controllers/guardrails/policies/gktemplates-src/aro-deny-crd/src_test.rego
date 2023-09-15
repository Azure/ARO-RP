package arodenycrd

test_input_not_allowed_with_gatekeeper_suffix {
    input := { 
        "review": fake_crd_input_review("assignmetadata.mutations.gatekeeper.sh", "CREATE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}


test_input_not_allowed_with_openshift_suffix {
    input := { 
        "review": fake_crd_input_review("schedulers.config.openshift.io", "UPDATE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_coreos_suffix {
    input := { 
        "review": fake_crd_input_review("prometheuses.monitoring.coreos.com", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_k8s_suffix {
    input := { 
        "review": fake_crd_input_review("volumesnapshots.snapshot.storage.k8s.io", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_cncf_suffix {
    input := { 
        "review": fake_crd_input_review("ippools.whereabouts.cni.cncf.io", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_metal3_suffix {
    input := { 
        "review": fake_crd_input_review("preprovisioningimages.metal3.io", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_not_allowed_with_ovn_suffix {
    input := { 
        "review": fake_crd_input_review("egressfirewalls.k8s.ovn.org", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 1
}

test_input_non_priv_user_custom_suffix_allowed_create{
    input := { 
        "review": fake_crd_input_review("demo.test.io", "CREATE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 0
}

test_input_non_priv_user_custom_suffix_allowed_update{
    input := { 
        "review": fake_crd_input_review("demo.test.sh", "UPDATE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 0
}

test_input_non_priv_user_custom_suffix_allowed_delete{
    input := { 
        "review": fake_crd_input_review("demo.test.org", "DELETE", non_priv_user)
    }
    results := violation with input as input
    count(results) == 0
}

test_input_priv_user_allowed_create {
    input := { 
        "review": fake_crd_input_review("schedulers.config.openshift.io", "CREATE", priv_user)
    }
    results := violation with input as input
    count(results) == 0
}

test_input_priv_user_allowed_update {
    input := { 
        "review": fake_crd_input_review("prometheuses.monitoring.coreos.com", "UPDATE", priv_user)
    }
    results := violation with input as input
    count(results) == 0
}

test_input_priv_user_allowed_delete {
    input := { 
        "review": fake_crd_input_review("assignmetadata.mutations.gatekeeper.sh", "DELETE", priv_user)
    }
    results := violation with input as input
    count(results) == 0
}


fake_crd_input_review(name, operation, userInfo) = review {
    review = {
        "operation": operation,
        "object": {
            "apiVersion": "apiextensions.k8s.io/v1",
            "kind": "CustomResourceDefinition",
            "metadata": {
                "name": name
            }
        },
        "userInfo":userInfo
    }
}

priv_user = {
    "username":"system:admin"
}

non_priv_user = {
    "username":"testuser"
}
