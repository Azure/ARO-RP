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
			doc.OpenShiftCluster.Properties.OperatorFlags = DefaultOperatorFlags()
		}

		// If there's no userDefinedRouting, set default one
		if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == "" {
			doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType = OutboundTypeLoadbalancer
		}
	}
}

// shorthand
const flagTrue string = "true"
const flagFalse string = "false"

// DefaultOperatorFlags returns flags for new clusters
// and ones that have not been AdminUpdated.
func DefaultOperatorFlags() OperatorFlags {
	// TODO: Get rid of magic strings.
	// We already have constants for all of the below strings.
	// For example `controllerEnabled` in `github.com/Azure/ARO-RP/pkg/operator/controllers/machine`.
	// But if we import packages with constants here we will have a cyclic import issue because controllers
	// import this package. We should probably move this somewhere else.
	// Maybe into a subpackage like `github.com/Azure/ARO-RP/pkg/api/defaults`?
	return OperatorFlags{
		"aro.alertwebhook.enabled":                 flagTrue,
		"aro.azuresubnets.enabled":                 flagTrue,
		"aro.azuresubnets.nsg.managed":             flagTrue,
		"aro.azuresubnets.serviceendpoint.managed": flagTrue,
		"aro.banner.enabled":                       flagFalse,
		"aro.checker.enabled":                      flagTrue,
		"aro.dnsmasq.enabled":                      flagTrue,
		"aro.genevalogging.enabled":                flagTrue,
		"aro.imageconfig.enabled":                  flagTrue,
		"aro.ingress.enabled":                      flagTrue,
		"aro.machine.enabled":                      flagTrue,
		"aro.machineset.enabled":                   flagTrue,
		"aro.machinehealthcheck.enabled":           flagTrue,
		"aro.machinehealthcheck.managed":           flagTrue,
		"aro.monitoring.enabled":                   flagTrue,
		"aro.nodedrainer.enabled":                  flagTrue,
		"aro.pullsecret.enabled":                   flagTrue,
		"aro.pullsecret.managed":                   flagTrue,
		"aro.rbac.enabled":                         flagTrue,
		"aro.routefix.enabled":                     flagTrue,
		"aro.storageaccounts.enabled":              flagTrue,
		"aro.workaround.enabled":                   flagTrue,
		"aro.autosizednodes.enabled":               flagFalse,
		"rh.srep.muo.enabled":                      flagTrue,
		"rh.srep.muo.managed":                      flagTrue,
	}
}
