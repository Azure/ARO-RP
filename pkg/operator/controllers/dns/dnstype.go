package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/coreos/go-semver/semver"
	"github.com/sirupsen/logrus"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

var (
	// MinCustomDNSVersion is the minimum OCP version that supports CustomDNS (ClusterHostedDNS)
	MinCustomDNSVersion = semver.Version{Major: 4, Minor: 21, Patch: 0}
)

// GetEffectiveDNSType determines the effective DNS type based on the flag value and cluster version
// Returns: operator.DNSTypeClusterHosted or operator.DNSTypeDnsmasq (or empty string for default dnsmasq)
func GetEffectiveDNSType(ctx context.Context, c client.Client, log *logrus.Entry, instance *arov1alpha1.Cluster) string {
	dnsType := instance.Spec.OperatorFlags.GetWithDefault(operator.DNSType, "")

	// Explicit dnsmasq - always use dnsmasq
	if dnsType == operator.DNSTypeDnsmasq {
		return operator.DNSTypeDnsmasq
	}

	// Explicit clusterhosted - check version support
	if dnsType == operator.DNSTypeClusterHosted {
		clusterVersion, err := getClusterVersion(ctx, c)
		if err != nil {
			log.Warnf("failed to get cluster version: %v, falling back to dnsmasq", err)
			return operator.DNSTypeDnsmasq
		}

		if !supportsCustomDNS(clusterVersion) {
			log.Warnf("aro.dns.type=clusterhosted not supported for version %s (requires 4.21+), falling back to dnsmasq", clusterVersion)
			return operator.DNSTypeDnsmasq
		}

		return operator.DNSTypeClusterHosted
	}

	// Blank/default - use dnsmasq (future: could be version-dependent after CustomDNS GA)
	return operator.DNSTypeDnsmasq
}

// getClusterVersion retrieves the current cluster version from the ClusterVersion object
func getClusterVersion(ctx context.Context, c client.Client) (*semver.Version, error) {
	cv := &configv1.ClusterVersion{}
	err := c.Get(ctx, client.ObjectKey{Name: "version"}, cv)
	if err != nil {
		return nil, err
	}

	// Get the current version from status
	if len(cv.Status.History) == 0 {
		return nil, nil
	}

	// The first entry in history is the current version
	versionString := cv.Status.History[0].Version
	version, err := semver.NewVersion(versionString)
	if err != nil {
		return nil, err
	}

	return version, nil
}

// supportsCustomDNS checks if the cluster version supports CustomDNS (4.21+)
func supportsCustomDNS(version *semver.Version) bool {
	if version == nil {
		return false
	}
	return !version.LessThan(MinCustomDNSVersion)
}

// ValidateDNSType validates the aro.dns.type flag value
// Returns true if valid, false otherwise
func ValidateDNSType(dnsType string) bool {
	switch dnsType {
	case "", operator.DNSTypeDnsmasq, operator.DNSTypeClusterHosted:
		return true
	default:
		return false
	}
}
