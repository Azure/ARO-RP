package arodenyupgradeconfig

test_input_allowed_regular_user_update_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "UPDATE")}
	inv := inv_data(create_data_local([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

test_input_disallowed_regular_user_update_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "UPDATE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 1
}

test_input_allowed_system_user_update_upgradeconfig {
	input := {"review": input_configmap("system:admin", "system:admin", "UPDATE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

test_allowed_regular_user_delete_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "DELETE")}
	inv := inv_data(create_data_local([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

test_disallowed_regular_user_delete_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "DELETE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 1
}

test_allowed_system_user_delete_upgradeconfig {
	input := {"review": input_configmap("system:admin", "system:admin", "DELETE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

test_allowed_regular_user_create_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "CREATE")}
	inv := inv_data(create_data_local([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

test_disallowed_regular_user_create_upgradeconfig {
	input := {"review": input_configmap("regular-user", "regular-user", "CREATE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 1
}

test_create_allowed_system_user_create_upgradeconfig {
	input := {"review": input_configmap("system:admin", "system:admin", "CREATE")}
	inv := inv_data(create_data_ocm([]))
	results := violation with input as input with data.inventory as inv
	count(results) == 0
}

input_configmap(group, username, operation) = output {
	output = {
		"operation": operation,
		"uid": "d4cb640e-dc2f-42b0-95e2-c2f91dbc74d9",
		"userInfo": {
			"uid": "109561ea-68ee-45ca-82be-96733b504593",
			"username": username
		}
	}
}

inv_data(obj) = output {
	output := {"namespace": {"openshift-managed-upgrade-operator": {obj.apiVersion: {obj.kind: obj}}}}
}

create_data_ocm([]) = output {
	output = {
		"apiVersion": "v1",
              "managed-upgrade-operator-config" : {
                "data": {
                    "config.yaml": "configManager:\n  source: OCM\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
                }
              },
		          "kind": "ConfigMap"
	}
}

create_data_local([]) = output {
	output = {
		"apiVersion": "v1",
              "managed-upgrade-operator-config" : {
                "data": {
                    "config.yaml": "configManager:\n  source: LOCAL\n  \n  localConfigName: managed-upgrade-config\n\n watchInterval: 15\nmaintenance:\n  controlPlaneTime: 90\n  ignoredAlerts:\n\n   controlPlaneCriticals:\n    - ClusterOperatorDown\n    - ClusterOperatorDegraded\nupgradeType:\nARO\nupgradeWindow:\n  delayTrigger: 30\n  timeOut: 120\nnodeDrain:\n  timeOut:\n45\n  expectedNodeDrainTime: 8\nscale:\n  timeOut: 30\nhealthCheck:\n  ignoredCriticals:\n\n - PrometheusRuleFailures\n  - CannotRetrieveUpdates\n  - FluentdNodeDown\n\n ignoredNamespaces:\n  - openshift-logging\n  - openshift-redhat-marketplace\n\n - openshift-operators\n  - openshift-user-workload-monitoring\n  - openshift-pipelines\n\n - openshift-azure-logging\n"
                }
              },
              "kind": "ConfigMap"
	}
}
