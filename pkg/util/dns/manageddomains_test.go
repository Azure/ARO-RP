package dns_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/dns"
)

func TestIsManagedDomain(t *testing.T) {
	type TestData struct {
		testName string
		domain   string
		want     bool
	}

	testCases := []TestData{
		{
			testName: "Custom domain",
			domain:   "custom.domain.io",
			want:     false,
		},
		{
			testName: "Public Cloud Managed Domain",
			domain:   "foo.location.aroapp.io",
			want:     true,
		},
		{
			testName: "US Gov Cloud Managed Domain",
			domain:   "foo.aroapp.azure.us",
			want:     true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			got := dns.IsManagedDomain(tt.domain)

			if got != tt.want {
				t.Errorf("expected %v, but got %v", tt.want, got)
			}
		})
	}
}
