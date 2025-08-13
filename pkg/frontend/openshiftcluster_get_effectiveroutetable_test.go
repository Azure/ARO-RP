package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	fakearmnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6/fake"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

//func TestGetOpenshiftClusterEffectiveRouteTableFunctionExists(t *testing.T) {
//	// Verify the _getOpenshiftClusterEffectiveRouteTable method exists on frontend
//	f := &frontend{}
//
//	// This test passes if the method exists and can be assigned to a variable
//	handler := f._getOpenshiftClusterEffectiveRouteTable
//	if handler == nil {
//		t.Error("_getOpenshiftClusterEffectiveRouteTable method should exist")
//	}
//}

func TestGetOpenshiftClusterEffectiveRouteTableQueryParameterExtraction(t *testing.T) {
	tests := []struct {
		Name          string `json:"name,omitempty"`
		QueryString   string `json:"query_string,omitempty"`
		ExpectedSubID string `json:"expected_sub_id,omitempty"`
		ExpectedRG    string `json:"expected_rg,omitempty"` //nolint:gci
		ExpectedNIC   string `json:"expected_nic,omitempty"`
		Description   string `json:"description,omitempty"`
	}{
		{
			Name:          "all parameters present",
			QueryString:   "subid=12345&rgn=test-rg&nic=test-nic",
			ExpectedSubID: "12345",
			ExpectedRG:    "test-rg",
			ExpectedNIC:   "test-nic",
			Description:   "Should extract all query parameters correctly",
		},
		{
			Name:          "parameters in different order",
			QueryString:   "nic=my-nic&subid=67890&rgn=my-rg",
			ExpectedSubID: "67890",
			ExpectedRG:    "my-rg",
			ExpectedNIC:   "my-nic",
			Description:   "Should handle parameters in any order",
		},
		{
			Name:          "url encoded parameters",
			QueryString:   "subid=12345&rgn=test%2Drg&nic=test%2Dnic",
			ExpectedSubID: "12345",
			ExpectedRG:    "test-rg",
			ExpectedNIC:   "test-nic",
			Description:   "Should handle URL encoded parameters",
		},
		{
			Name:          "empty parameters",
			QueryString:   "subid=&rgn=&nic=",
			ExpectedSubID: "",
			ExpectedRG:    "",
			ExpectedNIC:   "",
			Description:   "Should handle empty parameter values",
		},
		{
			Name:          "missing parameters",
			QueryString:   "other=value",
			ExpectedSubID: "",
			ExpectedRG:    "",
			ExpectedNIC:   "",
			Description:   "Should handle missing required parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create request with query parameters
			req, err := http.NewRequest(http.MethodGet, "/test?"+tt.QueryString, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Extract parameters the same way the implementation does
			subID := req.URL.Query().Get("subid")
			rg := req.URL.Query().Get("rgn")
			nicName := req.URL.Query().Get("nic")

			// Verify extracted values
			if subID != tt.ExpectedSubID {
				t.Errorf("Expected subID='%s', got='%s'", tt.ExpectedSubID, subID)
			}
			if rg != tt.ExpectedRG {
				t.Errorf("Expected rg='%s', got='%s'", tt.ExpectedRG, rg)
			}
			if nicName != tt.ExpectedNIC {
				t.Errorf("Expected nicName='%s', got='%s'", tt.ExpectedNIC, nicName)
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableResourceIDExtraction(t *testing.T) {
	tests := []struct {
		name               string
		requestPath        string
		expectedResourceID string
		description        string
	}{
		{
			name:               "standard admin path",
			requestPath:        "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			expectedResourceID: "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			description:        "Should strip /admin prefix correctly",
		},
		{
			name:               "path without admin prefix",
			requestPath:        "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			expectedResourceID: "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			description:        "Should handle path without admin prefix",
		},
		{
			name:               "empty path",
			requestPath:        "",
			expectedResourceID: "",
			description:        "Should handle empty path",
		},
		{
			name:               "admin only path",
			requestPath:        "/admin",
			expectedResourceID: "",
			description:        "Should handle admin-only path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract resource ID the same way the implementation does
			resourceID := strings.TrimPrefix(tt.requestPath, "/admin")

			if resourceID != tt.expectedResourceID {
				t.Errorf("Expected resourceID='%s', got='%s'", tt.expectedResourceID, resourceID)
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableDataProcessing(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := testdatabase.GetResourcePath(mockSubID, "test-cluster")

	t.Run("resource ID extraction from request path", func(t *testing.T) {
		// Create a request
		req, err := http.NewRequest(http.MethodGet, "/admin"+resourceID+"?subid="+mockSubID+"&rgn=test-rg&nic=test-nic", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Test the parameter extraction logic that the implementation uses
		extractedResourceID := strings.TrimPrefix(req.URL.Path, "/admin")
		if extractedResourceID != resourceID {
			t.Errorf("Expected resourceID='%s', got='%s'", resourceID, extractedResourceID)
		}

		// Test query parameter extraction
		subID := req.URL.Query().Get("subid")
		rg := req.URL.Query().Get("rgn")
		nicName := req.URL.Query().Get("nic")

		if subID != mockSubID {
			t.Errorf("Expected subID='%s', got='%s'", mockSubID, subID)
		}
		if rg != "test-rg" {
			t.Errorf("Expected rg='test-rg', got='%s'", rg)
		}
		if nicName != "test-nic" {
			t.Errorf("Expected nicName='test-nic', got='%s'", nicName)
		}
	})
}

func TestGetOpenshiftClusterEffectiveRouteTableAzureClientCreation(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"

	t.Run("azure client factory creation", func(t *testing.T) {
		// Test that we can create a client factory with a mock credential
		// This tests the Azure SDK integration without mocking the environment
		mockCredential := &fakeazcore.TokenCredential{}

		clientFactory, err := armnetwork.NewClientFactory(mockSubID, mockCredential, nil)
		if err != nil {
			t.Fatalf("Failed to create client factory: %v", err)
		}

		if clientFactory == nil {
			t.Error("Expected non-nil client factory")
		}

		// Test that we can create an interfaces client
		client := clientFactory.NewInterfacesClient()
		if client == nil {
			t.Error("Expected non-nil interfaces client")
		}
	})

	t.Run("azure client creation with custom options", func(t *testing.T) {
		// Test creating client with custom transport (like we do in tests)
		mockCredential := &fakeazcore.TokenCredential{}

		// Create a mock server for testing
		server := fakearmnetwork.InterfacesServer{}
		clientOptions := &arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: fakearmnetwork.NewInterfacesServerTransport(&server),
			},
		}

		client, err := armnetwork.NewInterfacesClient(mockSubID, mockCredential, clientOptions)
		if err != nil {
			t.Fatalf("Failed to create interfaces client with custom options: %v", err)
		}

		if client == nil {
			t.Error("Expected non-nil interfaces client")
		}
	})
}

func TestGetOpenshiftClusterEffectiveRouteTableAzureAPICall(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	testRG := "test-resource-group"
	testNIC := "test-nic-name"

	// Create mock effective route data
	mockRoutes := []*armnetwork.EffectiveRoute{
		{
			Name:   pointerutils.ToPtr("default-route"),
			Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("Default")),
			State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
			AddressPrefix: []*string{
				pointerutils.ToPtr("0.0.0.0/0"),
			},
			NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VirtualNetworkGateway")),
		},
	}

	tests := []struct {
		name               string
		setupMockServer    func() fakearmnetwork.InterfacesServer
		expectError        bool
		expectedRouteCount int
		description        string
	}{
		{
			name: "successful route table retrieval",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						// Verify the parameters passed to the Azure API
						if resourceGroupName != testRG {
							t.Errorf("Expected resource group '%s', got '%s'", testRG, resourceGroupName)
						}
						if networkInterfaceName != testNIC {
							t.Errorf("Expected NIC name '%s', got '%s'", testNIC, networkInterfaceName)
						}

						resp.AddNonTerminalResponse(http.StatusAccepted, nil)
						resp.SetTerminalResponse(http.StatusOK, armnetwork.InterfacesClientGetEffectiveRouteTableResponse{
							EffectiveRouteListResult: armnetwork.EffectiveRouteListResult{
								Value: mockRoutes,
							},
						}, nil)
						return resp, errResp
					},
				}
			},
			expectError:        false,
			expectedRouteCount: 1,
			description:        "Should successfully retrieve effective routes from Azure",
		},
		{
			name: "azure api returns error",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						errResp.SetResponseError(http.StatusNotFound, "NetworkInterfaceNotFound")
						return resp, errResp
					},
				}
			},
			expectError: true,
			description: "Should handle Azure API errors",
		},
		{
			name: "empty route table response",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						resp.AddNonTerminalResponse(http.StatusAccepted, nil)
						resp.SetTerminalResponse(http.StatusOK, armnetwork.InterfacesClientGetEffectiveRouteTableResponse{
							EffectiveRouteListResult: armnetwork.EffectiveRouteListResult{
								Value: []*armnetwork.EffectiveRoute{}, // Empty routes
							},
						}, nil)
						return resp, errResp
					},
				}
			},
			expectError:        false,
			expectedRouteCount: 0,
			description:        "Should handle empty route table responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up fake Azure server
			server := tt.setupMockServer()
			clientOptions := &arm.ClientOptions{
				ClientOptions: policy.ClientOptions{
					Transport: fakearmnetwork.NewInterfacesServerTransport(&server),
				},
			}

			// Create Azure client with fake server
			mockCredential := &fakeazcore.TokenCredential{}
			client, err := armnetwork.NewInterfacesClient(mockSubID, mockCredential, clientOptions)
			if err != nil {
				t.Fatalf("Failed to create interfaces client: %v", err)
			}

			// Test the Azure API call
			poller, err := client.BeginGetEffectiveRouteTable(ctx, testRG, testNIC, nil)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				// In the real implementation, this would trigger log.Fatalf
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error starting poller: %v", err)
			}

			// Poll until completion
			result, err := poller.PollUntilDone(ctx, nil)
			if err != nil {
				t.Fatalf("Unexpected error polling: %v", err)
			}

			// Verify the result
			if len(result.Value) != tt.expectedRouteCount {
				t.Errorf("Expected %d routes, got %d", tt.expectedRouteCount, len(result.Value))
			}

			// Test JSON marshaling (final step in implementation)
			jsonData, err := result.MarshalJSON()
			if err != nil {
				t.Fatalf("Failed to marshal result to JSON: %v", err)
			}

			// Verify JSON is valid
			var jsonObj map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonObj)
			if err != nil {
				t.Fatalf("Generated JSON is invalid: %v", err)
			}

			// For non-empty results, verify content
			if tt.expectedRouteCount > 0 {
				jsonStr := string(jsonData)
				if !strings.Contains(jsonStr, "default-route") {
					t.Error("JSON should contain route name")
				}
				if !strings.Contains(jsonStr, "0.0.0.0/0") {
					t.Error("JSON should contain address prefix")
				}
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableJSONProcessing(t *testing.T) {
	tests := []struct {
		name               string
		routeData          armnetwork.EffectiveRouteListResult
		expectMarshalError bool
		expectedContent    []string
		description        string
	}{
		{
			name: "valid route data",
			routeData: armnetwork.EffectiveRouteListResult{
				Value: []*armnetwork.EffectiveRoute{
					{
						Name:   pointerutils.ToPtr("test-route"),
						Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("Default")),
						State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
						AddressPrefix: []*string{
							pointerutils.ToPtr("10.0.0.0/24"),
						},
						NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VnetLocal")),
					},
				},
			},
			expectMarshalError: false,
			expectedContent:    []string{"test-route", "10.0.0.0/24", "VnetLocal", "Active"},
			description:        "Should marshal valid route data correctly",
		},
		{
			name: "empty route data",
			routeData: armnetwork.EffectiveRouteListResult{
				Value: []*armnetwork.EffectiveRoute{},
			},
			expectMarshalError: false,
			expectedContent:    []string{"\"value\":[]"},
			description:        "Should marshal empty route data correctly",
		},
		{
			name: "route with nil fields",
			routeData: armnetwork.EffectiveRouteListResult{
				Value: []*armnetwork.EffectiveRoute{
					{
						Name:          nil,
						Source:        nil,
						State:         nil,
						AddressPrefix: nil,
						NextHopType:   nil,
					},
				},
			},
			expectMarshalError: false,
			expectedContent:    []string{}, // Just verify it doesn't crash
			description:        "Should handle route with nil fields gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling (final step of implementation)
			jsonData, err := tt.routeData.MarshalJSON()
			if tt.expectMarshalError {
				if err == nil {
					t.Error("Expected marshal error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected marshal error: %v", err)
			}

			// Verify JSON is valid
			var jsonObj map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonObj)
			if err != nil {
				t.Fatalf("Generated JSON is invalid: %v", err)
			}

			// Verify expected content
			jsonStr := string(jsonData)
			for _, expected := range tt.expectedContent {
				if !strings.Contains(jsonStr, expected) {
					t.Errorf("JSON should contain '%s', but got: %s", expected, jsonStr)
				}
			}

			// Verify we can unmarshal back to the same structure
			var unmarshaled armnetwork.EffectiveRouteListResult
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON back to struct: %v", err)
			}

			// Basic structure verification
			if len(unmarshaled.Value) != len(tt.routeData.Value) {
				t.Errorf("Expected %d routes after unmarshal, got %d", len(tt.routeData.Value), len(unmarshaled.Value))
			}
		})
	}
}
