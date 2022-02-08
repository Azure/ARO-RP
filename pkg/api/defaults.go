package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// SetDefaults sets the default values for older api version
// when interacting with newer api versions. This together with
// database migration will make sure we have right values in the cluster documents
// when moving between old and new versions
func SetDefaults(doc *OpenShiftClusterDocument) {
	if doc.OpenShiftCluster != nil {
		// SoftwareDefinedNetwork was introduced in 2021-09-01-preview
		if doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork == "" {
			doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = SoftwareDefinedNetworkOpenShiftSDN
		}

		// EncryptionAtHost was introduced in 2021-09-01-preview.
		// It can't be changed post cluster creation
		if doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost == "" {
			doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostDisabled
		}

		for i, wp := range doc.OpenShiftCluster.Properties.WorkerProfiles {
			if wp.EncryptionAtHost == "" {
				doc.OpenShiftCluster.Properties.WorkerProfiles[i].EncryptionAtHost = EncryptionAtHostDisabled
			}
		}

		if doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules == "" {
			doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = FipsValidatedModulesDisabled
		}

		// When ProvisioningStateAdminUpdating is set, it needs a MaintenanceTask
		if doc.OpenShiftCluster.Properties.ProvisioningState == ProvisioningStateAdminUpdating {
			if doc.OpenShiftCluster.Properties.MaintenanceTask == "" {
				doc.OpenShiftCluster.Properties.MaintenanceTask = MaintenanceTaskEverything
			}
		}
		// If there's no operator flags, set the default ones
		if doc.OpenShiftCluster.Properties.OperatorFlags == nil {
			doc.OpenShiftCluster.Properties.OperatorFlags = DefaultOperatorFlags.Copy()
		}
	}
}

// shorthand
const FLAG_TRUE string = "true"
const FLAG_FALSE string = "false"

// Default flags for new clusters & ones that have not been AdminUpdated
var DefaultOperatorFlags OperatorFlags = OperatorFlags{
	"aro.alertwebhook.enabled": FLAG_TRUE,

	"aro.azuresubnets.enabled": FLAG_TRUE,

	"aro.banner.enabled": FLAG_FALSE,

	"aro.checker.enabled": FLAG_TRUE,

	"aro.dnsmasq.enabled": FLAG_TRUE,

	"aro.genevalogging.enabled": FLAG_TRUE,

	"aro.imageconfig.enabled": FLAG_TRUE,

	"aro.machine.enabled": FLAG_TRUE,

	"aro.machineset.enabled": FLAG_TRUE,

	"aro.monitoring.enabled": FLAG_TRUE,

	"aro.nodedrainer.enabled": FLAG_TRUE,

	"aro.pullsecret.enabled": FLAG_TRUE,
	"aro.pullsecret.managed": FLAG_TRUE,

	"aro.rbac.enabled": FLAG_TRUE,

	"aro.routefix.enabled": FLAG_TRUE,

	"aro.workaround.enabled": FLAG_TRUE,
}

func (d OperatorFlags) Copy() OperatorFlags {
	newFlags := make(OperatorFlags)
	for k, v := range d {
		newFlags[k] = v
	}
	return newFlags
}
