package api_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

func TestBuildEndpoint(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		params   map[string]string
		exp      string
		expErr   bool
	}{
		{
			name:     "Test Case 1 - No Error",
			template: api.GetClusterUpgradePolicyStateEndpointV1,
			params:   map[string]string{"ocmClusterID": "123", "policyID": "abc"},
			exp:      "/api/clusters_mgmt/v1/clusters/123/upgrade_policies/abc/state",
			expErr:   false,
		},
		{
			name:     "Test Case 2 - Template Parse Error",
			template: "Invalid {{ .template",
			params:   map[string]string{"ocmClusterID": "123"},
			exp:      "",
			expErr:   true,
		},
		{
			name:     "Test Case 3 - Template Execute Error",
			template: api.GetClusterUpgradePoliciesEndpointV1,
			params:   map[string]string{"nonexistentPlaceholder": "456"},
			exp:      "",
			expErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, err := api.BuildEndpoint(tc.template, tc.params)
			if (err != nil) != tc.expErr {
				t.Fatalf("BuildEndpoint error: %v, expect error %v", err, tc.expErr)
			}
			if endpoint != tc.exp {
				t.Errorf("BuildEndpoint got %v, expect %v", endpoint, tc.exp)
			}
		})
	}
}
