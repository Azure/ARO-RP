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

// SetDNSDefaults sets the DNS type based on cluster version for new clusters
// For 4.21+ clusters, set aro.dns.type to "clusterhosted" to enable CustomDNS
// For older clusters, leave blank (default dnsmasq behavior)
func SetDNSDefaults(doc *OpenShiftClusterDocument) {
	if doc.OpenShiftCluster == nil || doc.OpenShiftCluster.Properties.OperatorFlags == nil {
		return
	}

	// Check if aro.dns.type is already explicitly set
	if dnsType, exists := doc.OpenShiftCluster.Properties.OperatorFlags["aro.dns.type"]; exists && dnsType != "" {
		// Already explicitly set, don't override
		return
	}

	// Get cluster version
	clusterVersion := doc.OpenShiftCluster.Properties.ClusterProfile.Version
	if clusterVersion == "" {
		// No version yet (during creation), don't set
		return
	}

	// Parse version
	version, err := semver.NewVersion(clusterVersion)
	if err != nil {
		// Invalid version, skip
		return
	}

	// For 4.21+, set to "clusterhosted" to enable CustomDNS
	if !version.LessThan(MinCustomDNSVersion) {
		doc.OpenShiftCluster.Properties.OperatorFlags["aro.dns.type"] = "clusterhosted"
	}
	// For older versions, leave blank (default to dnsmasq)
}
