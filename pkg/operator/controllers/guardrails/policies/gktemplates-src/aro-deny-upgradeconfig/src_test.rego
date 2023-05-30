package arodenyupgradeconfig

test_input_allowed_regular_user_update_upgradeconfig {
  input := { "review": fake_local_update_delete_upgradeconfig("regular-user","regular-user","UPDATE") }
  results := violation with input as input
  count(results) == 0
}

test_input_disallowed_regular_user_update_upgradeconfig {
  input := { "review": fake_ocm_update_delete_upgradeconfig("regular-user","regular-user","UPDATE") }
  inv := ocm_inventory_data([])
  results := violation with input as input with data.inventory as inv
  count(results) == 1
}

ocm_inventory_data([]) = out {
  out = {
    "namespace": {
      "openshift-managed-upgrade-operator": {
        "v1": {
          "data": {
              "config.yaml": "configManager:\n  source: OCM\n  ocmBaseUrl: https://api.openshift.com\n  \n  watchInterval: 60\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n    controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType: ARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut: 45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n  - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n  ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n  - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n  - openshift-azure-logging\n"
          },
          "kind": "ConfigMap",
        }
      }
    }
          {
            "apiVersion": "v1",
            "data": {
                "config.yaml": "configManager:\n  source: OCM\n  ocmBaseUrl: https://api.openshift.com\n  \n  watchInterval: 60\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n    controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType: ARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut: 45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n  - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n  ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n  - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n  - openshift-azure-logging\n"
            },
            "kind": "ConfigMap"
          }
    }
  }
}

test_input_allowed_system_user_update_upgradeconfig {
  input := { "review": fake_ocm_update_delete_upgradeconfig("system:admin","system:admin","UPDATE") }
  results := violation with input as input
  count(results) == 0
}

test_allowed_regular_user_delete_upgradeconfig {
  input := { "review": fake_local_update_delete_upgradeconfig("regular-user","regular-user","DELETE") }
  results := violation with input as input
  count(results) == 0
}

test_disallowed_regular_user_delete_upgradeconfig {
  input := { "review": fake_ocm_update_delete_upgradeconfig("regular-user","regular-user","DELETE") }
  results := violation with input as input
  count(results) == 1
}

test_allowed_system_user_delete_upgradeconfig {
  input := { "review": fake_ocm_update_delete_upgradeconfig("system:admin","system:admin","DELETE") }
  results := violation with input as input
  count(results) == 0
}

test_allowed_regular_user_create_upgradeconfig {
  input := { "review": fake_local_upgradeconfig("regular-user","regular-user","CREATE") }
  results := violation with input as input
  count(results) == 0
}

test_disallowed_regular_user_create_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("regular-user","regular-user","CREATE") }
  results := violation with input as input
  count(results) == 1
}

test_create_allowed_system_user_create_upgradeconfig {
  input := { "review": fake_ocm_upgradeconfig("system:admin","system:admin","CREATE") }
  results := violation with input as input
  count(results) == 0
}

fake_ocm_upgradeconfig(group, username, operation) = output {
    output = {
 
            "data": {
                "config.yaml": "configManager:\n  source: OCM\n  ocmBaseUrl: https://api.openshift.com\n  \n  watchInterval: 60\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n    controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType: ARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut: 45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n  - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n  ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n  - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n  - openshift-azure-logging\n"
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
            "uid": "b65854be-5d3f-4e79-959c-3055d7cc530a",
            "userInfo": {
              "uid": "ada3819c-bb2b-46c8-8b80-7073c379ba4b",
              "username": username
            }
        }
      }
  
  

fake_ocm_update_delete_upgradeconfig(group, username, operation) = output {
    output = {
            "object": {
              "apiVersion": "v1",
              "data": {
                  "config.yaml": "configManager:\n  source: OCM\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
              },
              "kind": "ConfigMap",
              "metadata": {
                  "creationTimestamp": "2023-05-17T23:51:02Z",
                  "name": "managed-upgrade-operator-config",
                  "namespace": "openshift-managed-upgrade-operator",
                  "ownerReferences": [
                    {
                        "apiVersion": "aro.openshift.io/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "Cluster",
                        "name": "cluster",
                        "uid": "a7ed5f21-7396-46f5-92a8-62a282ab84a3"
                    }
                  ],
                  "resourceVersion": "29355",
                  "uid": "e9072b15-9119-4ef9-a7fe-c187ba03dde7"
              }
            },
            "oldObject": {
              "apiVersion": "v1",
              "data": {
                  "config.yaml": "configManager:\n  source: OCM\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
              },
              "kind": "ConfigMap",
              "metadata": {
                  "creationTimestamp": "2023-05-17T23:51:02Z",
                  "name": "managed-upgrade-operator-config-old",
                  "namespace": "openshift-managed-upgrade-operator",
                  "ownerReferences": [
                    {
                        "apiVersion": "aro.openshift.io/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "Cluster",
                        "name": "cluster",
                        "uid": "a7ed5f21-7396-46f5-92a8-62a282ab84a3"
                    }
                  ],
                  "resourceVersion": "29355",
                  "uid": "e9072b15-9119-4ef9-a7fe-c187ba03dde7"
              }
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
            "uid": "d4cb640e-dc2f-42b0-95e2-c2f91dbc74d9",
            "userInfo": {
              "uid": "109561ea-68ee-45ca-82be-96733b504593",
              "username": username
            }
        }
    }



fake_local_update_delete_upgradeconfig(group, username, operation) = output {
    output = {
            "object": {
              "apiVersion": "v1",
              "data": {
                  "config.yaml": "configManager:\n  source: LOCAL\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
              },
              "kind": "ConfigMap",
              "metadata": {
                  "creationTimestamp": "2023-05-17T23:51:02Z",
                  "name": "managed-upgrade-operator-config",
                  "namespace": "openshift-managed-upgrade-operator",
                  "ownerReferences": [
                    {
                        "apiVersion": "aro.openshift.io/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "Cluster",
                        "name": "cluster",
                        "uid": "a7ed5f21-7396-46f5-92a8-62a282ab84a3"
                    }
                  ],
                  "resourceVersion": "29355",
                  "uid": "e9072b15-9119-4ef9-a7fe-c187ba03dde7"
              }
            },
            "oldObject": {
              "apiVersion": "v1",
              "data": {
                  "config.yaml": "configManager:\n  source: LOCAL\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
              },
              "kind": "ConfigMap",
              "metadata": {
                  "creationTimestamp": "2023-05-17T23:51:02Z",
                  "name": "managed-upgrade-operator-config-old",
                  "namespace": "openshift-managed-upgrade-operator",
                  "ownerReferences": [
                    {
                        "apiVersion": "aro.openshift.io/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "Cluster",
                        "name": "cluster",
                        "uid": "a7ed5f21-7396-46f5-92a8-62a282ab84a3"
                    }
                  ],
                  "resourceVersion": "29355",
                  "uid": "e9072b15-9119-4ef9-a7fe-c187ba03dde7"
              }
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
            "uid": "d4cb640e-dc2f-42b0-95e2-c2f91dbc74d9",
            "userInfo": {
              "uid": "109561ea-68ee-45ca-82be-96733b504593",
              "username": username
            }
        }
      }



fake_local_upgradeconfig(group, username, operation) = output {
  output = {
            "object": {
              "apiVersion": "v1",
              "data": {
                  "config.yaml": "configManager:\n  source: LOCAL\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
              },
              "kind": "ConfigMap",
              "metadata": {
                  "creationTimestamp": "2023-05-17T23:51:02Z",
                  "name": "managed-upgrade-operator-config",
                  "namespace": "openshift-managed-upgrade-operator",
                  "ownerReferences": [
                    {
                        "apiVersion": "aro.openshift.io/v1alpha1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "Cluster",
                        "name": "cluster",
                        "uid": "a7ed5f21-7396-46f5-92a8-62a282ab84a3"
                    }
                  ],
                  "resourceVersion": "29355",
                  "uid": "e9072b15-9119-4ef9-a7fe-c187ba03dde7"
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
            "uid": "b65854be-5d3f-4e79-959c-3055d7cc530a",
            "userInfo": {
              "uid": "ada3819c-bb2b-46c8-8b80-7073c379ba4b",
              "username": username
            }
        }
      }
