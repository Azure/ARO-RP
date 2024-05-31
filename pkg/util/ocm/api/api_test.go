package api_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
)

func TestAPI(t *testing.T) {
	clusterList := api.ClusterList{
		Items: []api.ClusterInfo{{
			Id:   "testClusterID",
			Name: "testClusterName",
		}},
	}
	clusterListBytes, _ := json.Marshal(clusterList)

	upgradePolicies := api.UpgradePolicyList{
		Items: []api.UpgradePolicy{{
			Id: "testPolicyID",
		}},
	}
	upgradePoliciesBytes, _ := json.Marshal(upgradePolicies)

	upgradePolicyState := api.UpgradePolicyState{
		Kind: "testKind",
		UpgradePolicyStatus: api.UpgradePolicyStatus{
			State:       "testState",
			Description: "testDescription",
		},
	}
	upgradePolicyStateBytes, _ := json.Marshal(upgradePolicyState)

	cancelUpgradeResponse := api.CancelUpgradeResponse{
		Kind:        "testKind",
		Value:       "cancelled",
		Description: "Manually cancelled by SRE",
	}
	cancelUpgradeResponseBytes, _ := json.Marshal(cancelUpgradeResponse)

	getClusterInfoEndpoint, err := api.BuildEndpoint(api.GetClusterListEndpointV1, map[string]string{})
	if err != nil {
		t.Fatalf("BuildEndpoint failed: %v", err)
	}
	getClusterUpgradePoliciesEndpoint, err := api.BuildEndpoint(api.GetClusterUpgradePoliciesEndpointV1, map[string]string{"ocmClusterID": "testClusterID"})
	if err != nil {
		t.Fatalf("BuildEndpoint failed: %v", err)
	}
	getClusterUpgradePolicyStateEndpoint, err := api.BuildEndpoint(api.GetClusterUpgradePolicyStateEndpointV1, map[string]string{"ocmClusterID": "testClusterID", "policyID": "testPolicyID"})
	if err != nil {
		t.Fatalf("BuildEndpoint failed: %v", err)
	}
	cancelClusterUpgradePolicyEndpoint, err := api.BuildEndpoint(api.CancelClusterUpgradePolicyEndpointV1, map[string]string{"ocmClusterID": "testClusterID", "policyID": "testPolicyID"})
	if err != nil {
		t.Fatalf("BuildEndpoint failed: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Path == getClusterInfoEndpoint && req.Method == http.MethodGet:
			_, _ = rw.Write(clusterListBytes)
		case req.URL.Path == getClusterUpgradePoliciesEndpoint && req.Method == http.MethodGet:
			_, _ = rw.Write(upgradePoliciesBytes)
		case req.URL.Path == cancelClusterUpgradePolicyEndpoint && req.Method == http.MethodPatch:
			_, _ = rw.Write(cancelUpgradeResponseBytes)
		case req.URL.Path == getClusterUpgradePolicyStateEndpoint && req.Method == http.MethodGet:
			_, _ = rw.Write(upgradePolicyStateBytes)
		case req.URL.Path == "/non-200":
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		default:
			http.Error(rw, "Not Found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	testAPI := api.NewClient("testClusterID", server.URL, "testToken")

	testCases := []struct {
		name      string
		runTest   func() (interface{}, error)
		expected  interface{}
		expectErr bool
	}{
		{
			name: "Test GetClusterList",
			runTest: func() (interface{}, error) {
				return testAPI.GetClusterList(context.Background(), map[string]string{"key": "value"})
			},
			expected:  &clusterList,
			expectErr: false,
		},
		{
			name: "Test GetClusterUpgradePolicies",
			runTest: func() (interface{}, error) {
				return testAPI.GetClusterUpgradePolicies(context.Background(), "testClusterID")
			},
			expected:  &upgradePolicies,
			expectErr: false,
		},
		{
			name: "Test CancelClusterUpgradePolicy",
			runTest: func() (interface{}, error) {
				return testAPI.CancelClusterUpgradePolicy(context.Background(), "testClusterID", "testPolicyID")
			},
			expected:  &cancelUpgradeResponse,
			expectErr: false,
		},
		{
			name: "Test GetClusterUpgradePolicyState",
			runTest: func() (interface{}, error) {
				return testAPI.GetClusterUpgradePolicyState(context.Background(), "testClusterID", "testPolicyID")
			},
			expected:  &upgradePolicyState,
			expectErr: false,
		},
		{
			name: "Test Non-200 HTTP Status Code",
			runTest: func() (interface{}, error) {
				rb := api.NewRequestBuilder(http.MethodGet, server.URL).
					SetEndpoint("/non-200").
					AddHeader("Content-Type", "application/json").
					AddHeader("Accept", "application/json")

				request, err := rb.Build()
				if err != nil {
					return nil, err
				}

				_, err = testAPI.Send(request)
				return nil, err
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "Test GetBaseURL",
			runTest: func() (interface{}, error) {
				return testAPI.GetBaseURL(), nil
			},
			expected:  server.URL,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.runTest()
			if (err != nil) != tc.expectErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectErr, err)
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Got %v, expect %v", actual, tc.expected)
			}
		})
	}
}
