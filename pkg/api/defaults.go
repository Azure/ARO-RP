package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/coreos/go-semver/semver"
)

// MinCustomDNSVersion is the minimum OCP version that supports CustomDNS (ClusterHostedDNS)
var MinCustomDNSVersion = semver.Version{Major: 4, Minor: 21, Patch: 0}

// DNS operator flag key and values. These mirror the canonical constants in
// pkg/operator/flags.go (operator.DNSType, operator.DNSTypeDnsmasq,
// operator.DNSTypeClusterHosted). Defined here because pkg/api cannot import
// pkg/operator without creating a circular dependency.
const (
	dnsTypeFlagKey       = "aro.dns.type"  // operator.DNSType
	dnsTypeDnsmasq       = "dnsmasq"       // operator.DNSTypeDnsmasq
	dnsTypeClusterHosted = "clusterhosted" // operator.DNSTypeClusterHosted
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

// setDNSDefaults validates and sets the DNS type operator flag.
// If aro.dns.type is explicitly set to "dnsmasq", it is always accepted.
// If aro.dns.type is explicitly set to "clusterhosted", it is accepted only
// when the cluster version is >= 4.21; otherwise the flag is cleared so the
// cluster falls back to dnsmasq.
// If aro.dns.type is empty or unset, the type is auto-detected from the
// cluster version.
func setDNSDefaults(doc *OpenShiftClusterDocument) {
	if doc.OpenShiftCluster.Properties.OperatorFlags == nil {
		return
	}

	dnsType := doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey]
	clusterVersion := doc.OpenShiftCluster.Properties.ClusterProfile.Version
	meetsMinVersion := meetsMinCustomDNSVersion(clusterVersion)

	switch dnsType {
	case dnsTypeDnsmasq:
		// Switching to dnsmasq is always allowed regardless of version
		return

	case dnsTypeClusterHosted:
		// Switching to clusterhosted requires version >= 4.21
		if !meetsMinVersion {
			// Version too old or unparseable — reject the switch, clear the flag
			doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = ""
		}
		return

	default:
		// Empty or unset — auto-detect from version
		if clusterVersion == "" {
			return
		}

		if meetsMinVersion {
			doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
		}
	}
}

// meetsMinCustomDNSVersion returns true if clusterVersion is >= MinCustomDNSVersion.
// Returns false if the version string is empty or unparseable.
func meetsMinCustomDNSVersion(clusterVersion string) bool {
	if clusterVersion == "" {
		return false
	}

	version, err := semver.NewVersion(clusterVersion)
	if err != nil {
		return false
	}

	return !version.LessThan(MinCustomDNSVersion)
}
