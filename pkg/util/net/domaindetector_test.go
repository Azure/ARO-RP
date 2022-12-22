package net_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	azuretypes "github.com/openshift/installer/pkg/types/azure"

	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

type TestData struct {
	domain string
	want   bool
}

func TestDomainDetectorPublicCloud(t *testing.T) {
	testCases := []TestData{
		{
			domain: "foo.aroapp.io",
			want:   true,
		},
		{
			domain: "foo",
			want:   false,
		},
		{
			domain: "fooaroapp.io",
			want:   false,
		},
		{
			domain: "foo.aroapp.io.something",
			want:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.domain, func(t *testing.T) {
			detector := utilnet.DomainDetectorPublicCloud{}

			got := detector.ClusterHasManagedDomain(tt.domain)
			if got != tt.want {
				t.Errorf("expected %v, but got %v", tt.want, got)
			}
		})
	}
}

func TestDomainDetectorGovCloud(t *testing.T) {
	testCases := []TestData{
		{
			domain: "foo.aroapp.azure.us",
			want:   true,
		},
		{
			domain: "foo",
			want:   false,
		},
		{
			domain: "fooaroapp.azure.us",
			want:   false,
		},
		{
			domain: "foo.aroapp.azure.us.something",
			want:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.domain, func(t *testing.T) {
			detector := utilnet.DomainDetectorGovCloud{}

			got := detector.ClusterHasManagedDomain(tt.domain)
			if got != tt.want {
				t.Errorf("expected %v, but got %v", tt.want, got)
			}
		})
	}
}

func TestNewDomainDetector(t *testing.T) {
	type TestData struct {
		cloudName          string
		wantErr            string
		wantDomainDetector utilnet.DomainDetector
	}

	testCases := []TestData{
		{
			cloudName:          "invalidCloudName",
			wantErr:            `cloud environment "invalidCloudName" is unsupported by ARO`,
			wantDomainDetector: nil,
		},
		{
			cloudName:          azuretypes.PublicCloud.Name(),
			wantErr:            "",
			wantDomainDetector: &utilnet.DomainDetectorPublicCloud{},
		},
		{
			cloudName:          azuretypes.USGovernmentCloud.Name(),
			wantErr:            "",
			wantDomainDetector: &utilnet.DomainDetectorGovCloud{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.cloudName, func(t *testing.T) {
			got, err := utilnet.NewDomainDetector(tt.cloudName)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if got != tt.wantDomainDetector {
				t.Errorf("expected %v, but got %v", tt.wantDomainDetector, got)
			}
		})
	}
}
