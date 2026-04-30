package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func TestIsDNSControllerEnabled(t *testing.T) {
	tests := []struct {
		name  string
		flags arov1alpha1.OperatorFlags
		want  bool
	}{
		{
			name: "new flag true",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSEnabled: operator.FlagTrue,
			},
			want: true,
		},
		{
			name: "new flag false",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSEnabled: operator.FlagFalse,
			},
			want: false,
		},
		{
			name: "legacy flag true, new flag absent",
			flags: arov1alpha1.OperatorFlags{
				operator.DnsmasqEnabled: operator.FlagTrue,
			},
			want: true,
		},
		{
			name: "legacy flag false, new flag absent",
			flags: arov1alpha1.OperatorFlags{
				operator.DnsmasqEnabled: operator.FlagFalse,
			},
			want: false,
		},
		{
			name: "both flags present, new flag takes precedence (true)",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSEnabled:     operator.FlagTrue,
				operator.DnsmasqEnabled: operator.FlagFalse,
			},
			want: true,
		},
		{
			name: "both flags present, new flag takes precedence (false)",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSEnabled:     operator.FlagFalse,
				operator.DnsmasqEnabled: operator.FlagTrue,
			},
			want: false,
		},
		{
			name:  "no flags present returns false",
			flags: arov1alpha1.OperatorFlags{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDNSControllerEnabled(tt.flags)
			if got != tt.want {
				t.Errorf("IsDNSControllerEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateDNSType(t *testing.T) {
	tests := []struct {
		name    string
		dnsType string
		want    bool
	}{
		{
			name:    "empty string is valid (default dnsmasq)",
			dnsType: "",
			want:    true,
		},
		{
			name:    "dnsmasq is valid",
			dnsType: operator.DNSTypeDnsmasq,
			want:    true,
		},
		{
			name:    "clusterhosted is valid",
			dnsType: operator.DNSTypeClusterHosted,
			want:    true,
		},
		{
			name:    "invalid value returns false",
			dnsType: "invalid",
			want:    false,
		},
		{
			name:    "uppercase ClusterHosted is invalid",
			dnsType: "ClusterHosted",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateDNSType(tt.dnsType)
			if got != tt.want {
				t.Errorf("ValidateDNSType(%q) = %v, want %v", tt.dnsType, got, tt.want)
			}
		})
	}
}

func TestSupportsCustomDNS(t *testing.T) {
	tests := []struct {
		name    string
		version *semver.Version
		want    bool
	}{
		{
			name:    "nil version returns false",
			version: nil,
			want:    false,
		},
		{
			name:    "version 4.10.0 does not support",
			version: &semver.Version{Major: 4, Minor: 10, Patch: 0},
			want:    false,
		},
		{
			name:    "version 4.20.99 does not support",
			version: &semver.Version{Major: 4, Minor: 20, Patch: 99},
			want:    false,
		},
		{
			name:    "version 4.21.0 supports",
			version: &semver.Version{Major: 4, Minor: 21, Patch: 0},
			want:    true,
		},
		{
			name:    "version 4.21.5 supports",
			version: &semver.Version{Major: 4, Minor: 21, Patch: 5},
			want:    true,
		},
		{
			name:    "version 4.22.0 supports",
			version: &semver.Version{Major: 4, Minor: 22, Patch: 0},
			want:    true,
		},
		{
			name:    "version 5.0.0 supports",
			version: &semver.Version{Major: 5, Minor: 0, Patch: 0},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsCustomDNS(tt.version)
			if got != tt.want {
				t.Errorf("supportsCustomDNS(%v) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestGetEffectiveDNSType(t *testing.T) {
	tests := []struct {
		name      string
		flags     arov1alpha1.OperatorFlags
		cvVersion string // empty means no ClusterVersion object
		want      string
	}{
		{
			name:  "default (blank) returns dnsmasq without querying ClusterVersion",
			flags: arov1alpha1.OperatorFlags{},
			want:  operator.DNSTypeDnsmasq,
		},
		{
			name: "explicit dnsmasq returns dnsmasq without querying ClusterVersion",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeDnsmasq,
			},
			want: operator.DNSTypeDnsmasq,
		},
		{
			name: "clusterhosted with version 4.21.0 returns clusterhosted",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "4.21.0",
			want:      operator.DNSTypeClusterHosted,
		},
		{
			name: "clusterhosted with version 4.22.3 returns clusterhosted",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "4.22.3",
			want:      operator.DNSTypeClusterHosted,
		},
		{
			name: "clusterhosted with version 4.10.11 falls back to dnsmasq",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "4.10.11",
			want:      operator.DNSTypeDnsmasq,
		},
		{
			name: "clusterhosted with version 4.20.0 falls back to dnsmasq",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "4.20.0",
			want:      operator.DNSTypeDnsmasq,
		},
		{
			name: "clusterhosted with no ClusterVersion falls back to dnsmasq",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "",
			want:      operator.DNSTypeDnsmasq,
		},
		{
			name: "clusterhosted with empty status history falls back to dnsmasq",
			flags: arov1alpha1.OperatorFlags{
				operator.DNSType: operator.DNSTypeClusterHosted,
			},
			cvVersion: "empty-history",
			want:      operator.DNSTypeDnsmasq,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []client.Object
			if tt.cvVersion == "empty-history" {
				objects = append(objects, &configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{Name: "version"},
				})
			} else if tt.cvVersion != "" {
				objects = append(objects, &configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{Name: "version"},
					Status: configv1.ClusterVersionStatus{
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: tt.cvVersion,
							},
						},
					},
				})
			}
			c := ctrlfake.NewClientBuilder().WithObjects(objects...).Build()

			instance := &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: tt.flags,
				},
			}

			log := logrus.NewEntry(logrus.StandardLogger())
			got := GetEffectiveDNSType(context.Background(), c, log, instance)

			if got != tt.want {
				t.Errorf("GetEffectiveDNSType() = %q, want %q", got, tt.want)
			}
		})
	}
}
