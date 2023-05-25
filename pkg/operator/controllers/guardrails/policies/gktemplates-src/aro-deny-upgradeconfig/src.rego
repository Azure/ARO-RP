package arodenyupgradeconfig

import data.lib.common.is_exempted_account
import future.keywords.in

# Use object
violation[{"msg": msg}] {
	input.review.operation in ["CREATE"]
	name := input.review.object.metadata.name

	## Check user type
	not is_exempted_account(input.review)

	## If regular user and
	## has NO cloud.openshift.com entry in openshift-config/pull-secret Secret
	## ALLOW EDITING

	## If regular user and 
	## HAS cloud.openshift.com entry (`source: OCM` indicates pull-secret exists) in openshift-config/pull-secret Secret
	## NOT ALLOWED
	config_data := input.review.object.data["config.yaml"]
	regex.match("source: OCM", config_data)
	msg := "Modifying the UpgradeConfig is not allowed for regular users. This can include creating, deleting, and updating UpgradeConfig."
}

# Use oldObject
violation[{"msg": msg}] {
	input.review.operation in ["UPDATE", "DELETE"]
	name := input.review.object.metadata.name

	## Check user type
	not is_exempted_account(input.review)

	## If regular user and
	## has NO cloud.openshift.com entry in openshift-config/pull-secret Secret
	## ALLOW EDITING

	## If regular user and 
	## HAS cloud.openshift.com entry (`source: OCM` indicates pull-secret exists) in openshift-config/pull-secret Secret
	## NOT ALLOWED
	config_data := input.review.oldObject.data["config.yaml"]
	regex.match("source: OCM", config_data)
	msg := "Modifying the UpgradeConfig is not allowed for regular users. This can include creating, deleting, and updating UpgradeConfig."
}