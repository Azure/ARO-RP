package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/coreos/go-semver/semver"
)

var (
	// MinCustomDNSVersion is the minimum OCP version that supports CustomDNS (ClusterHostedDNS)
	MinCustomDNSVersion = semver.Version{Major: 4, Minor: 21, Patch: 0}
)

// SetDefaults sets the default values for older api version
// when interacting with newer api versions. This together with
// database migration will make sure we have right values in the cluster documents
// when moving between old and new versions
func SetDefaults(doc *OpenShiftClusterDocument, defaultOperatorFlags func() map[string]string) {
	if doc.OpenShiftCluster != nil {
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

		for i, wp := range doc.OpenShiftCluster.Properties.WorkerProfilesStatus {
			if wp.EncryptionAtHost == "" {
				doc.OpenShiftCluster.Properties.WorkerProfilesStatus[i].EncryptionAtHost = EncryptionAtHostDisabled
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
			doc.OpenShiftCluster.Properties.OperatorFlags = OperatorFlags(defaultOperatorFlags())
		}

		// If there's no OutboundType, set default one
		if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == "" {
			doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType = OutboundTypeLoadbalancer
		}

		// If there's no PreconfiguredNSG, set to disabled
		if doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == "" {
			doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG = PreconfiguredNSGDisabled
		}

		// If OutboundType is Loadbalancer and there is no LoadBalancerProfile, set default one
		if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == OutboundTypeLoadbalancer && doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile == nil {
			doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
				ManagedOutboundIPs: &ManagedOutboundIPs{
					Count: 1,
				},
			}
		}

		// Set DNS type based on cluster version for new clusters.
		// For 4.21+ clusters, set aro.dns.type to "clusterhosted" to enable CustomDNS.
		// For older clusters, leave blank (default dnsmasq behavior).
		setDNSDefaults(doc)
	}
}

// setDNSDefaults sets the DNS type operator flag based on cluster version.
func setDNSDefaults(doc *OpenShiftClusterDocument) {
	if doc.OpenShiftCluster.Properties.OperatorFlags == nil {
		return
	}

	// Don't override if aro.dns.type is already explicitly set
	if dnsType, exists := doc.OpenShiftCluster.Properties.OperatorFlags["aro.dns.type"]; exists && dnsType != "" {
		return
	}

	clusterVersion := doc.OpenShiftCluster.Properties.ClusterProfile.Version
	if clusterVersion == "" {
		return
	}

	version, err := semver.NewVersion(clusterVersion)
	if err != nil {
		return
	}

	if !version.LessThan(MinCustomDNSVersion) {
		doc.OpenShiftCluster.Properties.OperatorFlags["aro.dns.type"] = "clusterhosted"
	}
}
