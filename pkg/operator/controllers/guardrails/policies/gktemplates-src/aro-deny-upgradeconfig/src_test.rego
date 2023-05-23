package arodenyupgradeconfig

test_input_allowed_regular_user_update_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("regular-user","test","UPDATE") }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_regular_user_update_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("regular-user","test","UPDATE") }
  results := violation with input as input
  count(results) == 1
}

test_input_allowed_system_user_update_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("system:admin","test","UPDATE") }
  results := violation with input as input
  count(results) == 0
}

test_allowed_regular_user_delete_upgradeconfig {
  input := { "review": fake_local_upgradeconfig("regular-user","test","DELETE") }
  results := violation with input as input
  count(results) == 0
}

test_disallowed_regular_user_delete_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("regular-user","test","DELETE") }
  results := violation with input as input
  count(results) == 1
}

test_allowed_system_user_delete_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("system:admin","test","DELETE") }
  results := violation with input as input
  count(results) == 0
}

test_allowed_regular_user_create_upgradeconfig {
  input := { "review": fake_local_upgradeconfig("regular-user","test","CREATE") }
  results := violation with input as input
  count(results) == 0
}

test_disallowed_regular_user_create_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("regular-user","test","CREATE") }
  results := violation with input as input
  count(results) == 1
}

test_create_allowed_system_user_create_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("system:admin","test","CREATE") }
  results := violation with input as input
  count(results) == 0
}

fake_ocm_upgradeconfig(group, username, operation) = output {
  output = {
    {
      "apiVersion": "v1",
      "data": {
        "config.yaml": "configManager:\n  source: OCM\n  ocmBaseUrl: https://api.openshift.com\n  \n  watchInterval: 60\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n    controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType: ARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut: 45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n  - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n  ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n  - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n  - openshift-azure-logging\n"
      },
      "kind": "ConfigMap",
      "metadata": {
        "creationTimestamp": "2023-05-25T09:48:14Z",
        "name": "managed-upgrade-operator-config-old",
        "namespace": "openshift-managed-upgrade-operator",
        "ownerReferences": [
          {
            "apiVersion": "aro.openshift.io/v1alpha1",
            "blockOwnerDeletion": true,
            "controller": true,
            "kind": "Cluster",
            "name": "cluster",
            "uid": "c89909e5-3f29-482e-8f8e-50851fc85459"
          }
        ],
        "resourceVersion": "404152",
        "uid": "2e349cc1-034e-4f8b-9377-34ba9620c418"
      },
      "operation": operation,
      "options": null,
      "requestKind": {
        "group": "",
        "kind": "ConfigMap",
        "version": "v1"
      },
      "resource": {
        "group": "",
        "resource": "ConfigMap",
        "version": "v1"
      },
      "uid": "0dc3dee4-fed8-42c8-a089-a6d36477c1c4",
      "userInfo": {
        "uid": "5b7bbd66-0563-4c18-b66b-2771a47959f9",
        "username": username
      }
    }
  }
}

fake_local_upgradeconfig(group, username, operation) = output {
  output = {
        {
      "apiVersion": "admission.k8s.io/v1",
      "kind": "AdmissionReview",
      "request": {
        "dryRun": true,
        "kind": {
          "group": "",
          "kind": "ConfigMap",
          "version": "v1"
        },
        "object": {
          "apiVersion": "v1",
          "data": {
            "config.yaml": "configManager:\n  source: LOCAL\n  ocmBaseUrl: https://api.openshift.com\n  \n  watchInterval: 60\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n    controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType: ARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut: 45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n  - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n  ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n  - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n  - openshift-azure-logging\n"
          },
          "kind": "ConfigMap",
          "metadata": {
            "creationTimestamp": "2023-05-21T09:48:14Z",
            "name": "managed-upgrade-operator-config",
            "namespace": "openshift-managed-upgrade-operator",
            "ownerReferences": [
              {
                "apiVersion": "aro.openshift.io/v1alpha1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "Cluster",
                "name": "cluster",
                "uid": "c89909e5-3f29-482e-8f8e-50851fc85459"
              }
            ],
            "resourceVersion": "404152",
            "uid": "2e349cc1-034e-4f8b-9377-34ba9620c418"
          }
        },
        "oldObject": null,
        "operation": operation,
        "options": null,
        "requestKind": {
          "group": "",
          "kind": "ConfigMap",
          "version": "v1"
        },
        "resource": {
          "group": "",
          "resource": "ConfigMap",
          "version": "v1"
        },
        "uid": "080407a7-d907-4c9a-8b17-67b637b97dce",
        "userInfo": {
          "uid": "0e4ced82-9535-4eaf-b65c-b091754cbd20",
          "username": username
        }
      }
    }
  }