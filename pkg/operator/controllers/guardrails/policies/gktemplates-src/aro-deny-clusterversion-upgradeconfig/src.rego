package arodenyclusterversionupgradeconfig

import data.lib.common.is_exempted_account
import future.keywords.in

# Use object
# To retrieve from a different resource, data.inventory.namespace["openshift-managed-upgrade-operator"]["v1"]["ConfigMap"]["managed-upgrade-operator-config"]["data"]["config.yaml"]
violation[{"msg": msg}] {
	input.review.operation in ["CREATE", "UPDATE", "DELETE"]

	# ## Check user type
	not is_exempted_account(input.review)

	# ## If regular user and
	# ## has NO cloud.openshift.com entry in openshift-config/pull-secret Secret
	# ## ALLOW EDITING
	# ## If regular user and 
	# ## HAS cloud.openshift.com entry (`source: OCM` indicates pull-secret exists) in openshift-config/pull-secret Secret
	# ## NOT ALLOWED
	config_data := data.inventory.namespace["openshift-managed-upgrade-operator"]["v1"]["ConfigMap"]["managed-upgrade-operator-config"]["data"]["config.yaml"]
	regex.match("source: OCM", config_data)
	msg := "Modifying this resource is not allowed for regular users if have updated your Azure Red Hat OpenShift pull secret for your cluster, and including the cloud.openshift.com entry from your pull secret."
}
