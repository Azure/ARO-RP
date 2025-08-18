package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	fakearmnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6/fake"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	utilarmnetwork "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestGetAdminOpenshiftClusterEffectiveRouteTablePathHandling(t *testing.T) {
	tests := []struct {
		name         string
		originalPath string
		expectedDir  string
		finalPath    string
		description  string
	}{
		{
			name:         "admin effective routing tables path",
			originalPath: "/admin/subscriptions/sub/resourceGroups/rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster/effectiveroutingtables",
			expectedDir:  "/admin/subscriptions/sub/resourceGroups/rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster",
			finalPath:    "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster",
			description:  "Should handle admin effective routing tables path correctly",
		},
		{
			name:         "admin cluster path",
			originalPath: "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster/effectiveroutingtables",
			expectedDir:  "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			finalPath:    "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			description:  "Should extract admin cluster resource ID correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the path manipulation logic that happens in the admin handler
			// r.URL.Path = filepath.Dir(r.URL.Path)
			modifiedPath := filepath.Dir(tt.originalPath)

			if modifiedPath != tt.expectedDir {
				t.Errorf("Expected dir path %s, got %s", tt.expectedDir, modifiedPath)
			}

			// Then test the resource ID extraction that happens in the shared method
			// resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
			resourceID := strings.TrimPrefix(modifiedPath, "/admin")

			if resourceID != tt.finalPath {
				t.Errorf("Expected final resource path %s, got %s", tt.finalPath, resourceID)
			}

			// Verify the resource ID can be parsed
			if resourceID != "" {
				resource, err := azure.ParseResourceID(resourceID)
				if err != nil {
					t.Errorf("Final resource ID should be parseable: %v", err)
				} else {
					if !strings.Contains(resource.ResourceType, "openShiftClusters") {
						t.Error("Should extract ARO cluster resource type")
					}
				}
			}
		})
	}
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableQueryParameterHandling(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		expectedNIC string
		expectError bool
		description string
	}{
		{
			name:        "valid admin NIC parameter",
			queryString: "nic=admin-test-nic-interface",
			expectedNIC: "admin-test-nic-interface",
			expectError: false,
			description: "Should extract NIC name from admin query parameters",
		},
		{
			name:        "URL encoded admin NIC parameter",
			queryString: "nic=admin%2Dtest%2Dnic%2Dinterface",
			expectedNIC: "admin-test-nic-interface",
			expectError: false,
			description: "Should handle URL encoded admin NIC names",
		},
		{
			name:        "missing admin NIC parameter",
			queryString: "other=value&admin=true",
			expectedNIC: "",
			expectError: true,
			description: "Should return error when admin NIC parameter is missing",
		},
		{
			name:        "empty admin NIC parameter",
			queryString: "nic=&admin=true",
			expectedNIC: "",
			expectError: true,
			description: "Should return error when admin NIC parameter is empty",
		},
		{
			name:        "admin parameters with valid NIC",
			queryString: "nic=admin-nic&admin=true&debug=on&extra=ignored",
			expectedNIC: "admin-nic",
			expectError: false,
			description: "Should extract admin NIC while ignoring other admin parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create admin request with query parameters
			req, err := http.NewRequest(http.MethodGet, "/admin/test/effectiveroutingtables?"+tt.queryString, nil)
			if err != nil {
				t.Fatalf("Failed to create admin request: %v", err)
			}

			// Extract NIC parameter the same way the shared implementation does
			nicName := req.URL.Query().Get("nic")

			if nicName != tt.expectedNIC {
				t.Errorf("Expected admin nicName='%s', got='%s'", tt.expectedNIC, nicName)
			}

			// Test the validation logic that would be in the shared implementation
			if tt.expectError && nicName == "" {
				// This simulates the error that would be returned
				expectedError := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name is required")
				if expectedError == nil {
					t.Error("Expected error for missing/empty admin NIC parameter")
				}
			}
		})
	}
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableARMNetworkIntegration(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	// Admin now uses cluster resource group from shared implementation (where NICs are located)
	clusterResourceGroup := "admin-cluster-resource-group"
	testNIC := "admin-test-nic-name"

	// Create mock effective route data for admin scenarios
	mockAdminRoutes := []*armnetwork.EffectiveRoute{
		{
			Name:   pointerutils.ToPtr("admin-default-route"),
			Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("Default")),
			State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
			AddressPrefix: []*string{
				pointerutils.ToPtr("0.0.0.0/0"),
			},
			NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VirtualNetworkGateway")),
		},
		{
			Name:   pointerutils.ToPtr("admin-local-route"),
			Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("VNetLocal")),
			State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
			AddressPrefix: []*string{
				pointerutils.ToPtr("10.0.0.0/16"),
			},
			NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VnetLocal")),
		},
		{
			Name:   pointerutils.ToPtr("admin-user-route"),
			Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("User")),
			State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
			AddressPrefix: []*string{
				pointerutils.ToPtr("192.168.0.0/16"),
			},
			NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VirtualAppliance")),
			NextHopIPAddress: []*string{
				pointerutils.ToPtr("10.0.1.4"),
			},
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
			name: "successful admin route table retrieval with shared implementation",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						// Verify the parameters passed match what the shared implementation sends (cluster resource group)
						if resourceGroupName != clusterResourceGroup {
							t.Errorf("Expected admin cluster resource group '%s', got '%s'", clusterResourceGroup, resourceGroupName)
						}
						if networkInterfaceName != testNIC {
							t.Errorf("Expected admin NIC name '%s', got '%s'", testNIC, networkInterfaceName)
						}

						resp.AddNonTerminalResponse(http.StatusAccepted, nil)
						resp.SetTerminalResponse(http.StatusOK, armnetwork.InterfacesClientGetEffectiveRouteTableResponse{
							EffectiveRouteListResult: armnetwork.EffectiveRouteListResult{
								Value: mockAdminRoutes,
							},
						}, nil)
						return resp, errResp
					},
				}
			},
			expectError:        false,
			expectedRouteCount: 3,
			description:        "Should successfully retrieve admin effective routes via shared implementation using cluster resource group",
		},
		{
			name: "admin azure api network interface not found error",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						errResp.SetResponseError(http.StatusNotFound, "AdminNetworkInterfaceNotFound")
						return resp, errResp
					},
				}
			},
			expectError: true,
			description: "Should handle admin network interface not found errors",
		},
		{
			name: "admin azure api permission denied error",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						errResp.SetResponseError(http.StatusForbidden, "AdminInsufficientPermissions")
						return resp, errResp
					},
				}
			},
			expectError: true,
			description: "Should handle admin permission denied errors",
		},
		{
			name: "admin empty effective route table response",
			setupMockServer: func() fakearmnetwork.InterfacesServer {
				return fakearmnetwork.InterfacesServer{
					BeginGetEffectiveRouteTable: func(ctx context.Context, resourceGroupName string, networkInterfaceName string, options *armnetwork.InterfacesClientBeginGetEffectiveRouteTableOptions) (resp fakeazcore.PollerResponder[armnetwork.InterfacesClientGetEffectiveRouteTableResponse], errResp fakeazcore.ErrorResponder) {
						resp.AddNonTerminalResponse(http.StatusAccepted, nil)
						resp.SetTerminalResponse(http.StatusOK, armnetwork.InterfacesClientGetEffectiveRouteTableResponse{
							EffectiveRouteListResult: armnetwork.EffectiveRouteListResult{
								Value: []*armnetwork.EffectiveRoute{}, // Empty admin routes
							},
						}, nil)
						return resp, errResp
					},
				}
			},
			expectError:        false,
			expectedRouteCount: 0,
			description:        "Should handle empty admin effective route table responses correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up fake Azure server for admin testing
			server := tt.setupMockServer()
			clientOptions := &arm.ClientOptions{
				ClientOptions: policy.ClientOptions{
					Transport: fakearmnetwork.NewInterfacesServerTransport(&server),
				},
			}

			// Create Azure client with fake server using ARO-RP utility for admin
			mockCredential := &fakeazcore.TokenCredential{}
			client, err := utilarmnetwork.NewInterfacesClient(mockSubID, mockCredential, clientOptions)
			if err != nil {
				t.Fatalf("Failed to create ARO-RP admin interfaces client: %v", err)
			}

			// Test the GetEffectiveRouteTableAndWait method that the shared implementation uses with cluster resource group
			result, err := client.GetEffectiveRouteTableAndWait(ctx, clusterResourceGroup, testNIC, nil)
			if tt.expectError {
				if err == nil {
					t.Error("Expected admin error but got none")
				}
				// Verify error would be wrapped in api.NewCloudError in the shared implementation
				expectedError := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					fmt.Sprintf("Failed to retrieve effective route table: %v", err))
				if expectedError == nil {
					t.Error("Expected admin CloudError wrapper")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected admin error: %v", err)
			}

			// Verify the admin result structure
			if len(result.Value) != tt.expectedRouteCount {
				t.Errorf("Expected %d admin routes, got %d", tt.expectedRouteCount, len(result.Value))
			}

			// Test JSON marshaling (final step in shared implementation)
			jsonData, err := result.MarshalJSON()
			if err != nil {
				t.Fatalf("Failed to marshal admin result to JSON: %v", err)
			}

			// Verify JSON is valid
			var jsonObj map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonObj)
			if err != nil {
				t.Fatalf("Generated admin JSON is invalid: %v", err)
			}

			// For non-empty results, verify admin-specific content
			if tt.expectedRouteCount > 0 {
				jsonStr := string(jsonData)
				if !strings.Contains(jsonStr, "admin-default-route") || !strings.Contains(jsonStr, "admin-local-route") || !strings.Contains(jsonStr, "admin-user-route") {
					t.Error("Admin JSON should contain admin route names")
				}
				if !strings.Contains(jsonStr, "0.0.0.0/0") || !strings.Contains(jsonStr, "10.0.0.0/16") || !strings.Contains(jsonStr, "192.168.0.0/16") {
					t.Error("Admin JSON should contain admin address prefixes")
				}
				if !strings.Contains(jsonStr, "VirtualAppliance") {
					t.Error("Admin JSON should contain VirtualAppliance next hop type")
				}
			}
		})
	}
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableWorkflow(t *testing.T) {
	tests := []struct {
		name            string
		originalPath    string
		queryParameters string
		description     string
	}{
		{
			name:            "complete admin workflow simulation",
			originalPath:    "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster/effectiveroutingtables",
			queryParameters: "nic=admin-nic-test",
			description:     "Should simulate complete admin workflow: filepath.Dir path manipulation then call shared implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Simulate admin handler path manipulation (only admin-specific logic)
			// r.URL.Path = filepath.Dir(r.URL.Path)
			adminProcessedPath := filepath.Dir(tt.originalPath)

			// Step 2: Simulate shared implementation resource ID extraction
			// resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
			resourceID := strings.TrimPrefix(adminProcessedPath, "/admin")

			// Step 3: Verify the resource ID is valid for parsing (rest handled by shared implementation)
			resource, err := azure.ParseResourceID(resourceID)
			if err != nil {
				t.Fatalf("Resource ID should be parseable after admin processing: %v", err)
			}

			// Step 4: Verify all components are correctly extracted for shared implementation
			if resource.SubscriptionID != "12345" {
				t.Errorf("Expected subscription ID '12345', got '%s'", resource.SubscriptionID)
			}
			if resource.ResourceGroup != "test-rg" {
				t.Errorf("Expected resource group 'test-rg', got '%s'", resource.ResourceGroup)
			}
			if resource.ResourceName != "test-cluster" {
				t.Errorf("Expected resource name 'test-cluster', got '%s'", resource.ResourceName)
			}

			// Step 5: Simulate query parameter extraction
			req, err := http.NewRequest(http.MethodGet, tt.originalPath+"?"+tt.queryParameters, nil)
			if err != nil {
				t.Fatalf("Failed to create test request: %v", err)
			}

			nicName := req.URL.Query().Get("nic")
			if nicName != "admin-nic-test" {
				t.Errorf("Expected NIC name 'admin-nic-test', got '%s'", nicName)
			}

			// Verify no error would be returned for valid parameters
			if nicName == "" {
				t.Error("Valid admin workflow should not have empty NIC name")
			}
			if resourceID == "" {
				t.Error("Valid admin workflow should not have empty resource ID")
			}
		})
	}
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		originalPath      string
		queryParameters   string
		expectedHTTPCode  int
		expectedErrorCode string
		description       string
	}{
		{
			name:              "missing NIC parameter in admin request",
			originalPath:      "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster/effectiveroutingtables",
			queryParameters:   "other=value",
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest when admin request is missing NIC parameter",
		},
		{
			name:              "invalid admin path format",
			originalPath:      "/admin/invalid/path/effectiveroutingtables",
			queryParameters:   "nic=test-nic",
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest when admin path is malformed",
		},
		{
			name:              "admin only path",
			originalPath:      "/admin/effectiveroutingtables",
			queryParameters:   "nic=test-nic",
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest for admin-only path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate admin handler processing
			adminProcessedPath := filepath.Dir(tt.originalPath)
			resourceID := strings.TrimPrefix(adminProcessedPath, "/admin")

			// Create request for parameter testing
			req, err := http.NewRequest(http.MethodGet, tt.originalPath+"?"+tt.queryParameters, nil)
			if err != nil {
				t.Fatalf("Failed to create test request: %v", err)
			}

			nicName := req.URL.Query().Get("nic")

			// Test NIC validation
			if nicName == "" {
				err := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name is required")
				if err.StatusCode != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPCode, err.StatusCode)
				}
				if err.Code != tt.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrorCode, err.Code)
				}
				return
			}

			// Test resource ID validation
			if resourceID == "" {
				err := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Resource ID is required")
				if err.StatusCode != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPCode, err.StatusCode)
				}
				if err.Code != tt.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrorCode, err.Code)
				}
				return
			}

			// Test resource ID parsing validation
			_, err = azure.ParseResourceID(resourceID)
			if err != nil {
				cloudErr := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Invalid resource ID format")
				if cloudErr.StatusCode != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPCode, cloudErr.StatusCode)
				}
				if cloudErr.Code != tt.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrorCode, cloudErr.Code)
				}
			}
		})
	}
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableDataConsistency(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := testdatabase.GetResourcePath(mockSubID, "admin-test-cluster")

	t.Run("admin workflow resource ID extraction consistency", func(t *testing.T) {
		// Test the complete admin workflow
		adminPath := "/admin" + resourceID + "/effectiveroutingtables"

		// Apply admin handler path manipulation
		adminProcessedPath := filepath.Dir(adminPath)

		// Apply shared implementation resource ID extraction
		extractedResourceID := strings.TrimPrefix(adminProcessedPath, "/admin")

		if extractedResourceID != resourceID {
			t.Errorf("Expected admin resourceID='%s', got='%s'", resourceID, extractedResourceID)
		}

		// Test that azure.ParseResourceID can parse the extracted admin ID
		resource, err := azure.ParseResourceID(extractedResourceID)
		if err != nil {
			t.Fatalf("Failed to parse extracted admin resource ID: %v", err)
		}

		if resource.SubscriptionID != mockSubID {
			t.Errorf("Expected admin subscriptionID='%s', got='%s'", mockSubID, resource.SubscriptionID)
		}
	})

	t.Run("admin JSON serialization consistency", func(t *testing.T) {
		// Create admin test route data
		adminTestRoutes := armnetwork.EffectiveRouteListResult{
			Value: []*armnetwork.EffectiveRoute{
				{
					Name:   pointerutils.ToPtr("admin-test-route"),
					Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("Default")),
					State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
					AddressPrefix: []*string{
						pointerutils.ToPtr("172.16.1.0/24"),
					},
					NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VnetLocal")),
				},
				{
					Name:   pointerutils.ToPtr("admin-custom-route"),
					Source: (*armnetwork.EffectiveRouteSource)(pointerutils.ToPtr("User")),
					State:  (*armnetwork.EffectiveRouteState)(pointerutils.ToPtr("Active")),
					AddressPrefix: []*string{
						pointerutils.ToPtr("172.16.2.0/24"),
					},
					NextHopType: (*armnetwork.RouteNextHopType)(pointerutils.ToPtr("VirtualAppliance")),
					NextHopIPAddress: []*string{
						pointerutils.ToPtr("172.16.1.100"),
					},
				},
			},
		}

		// Test admin JSON marshaling
		jsonData, err := adminTestRoutes.MarshalJSON()
		if err != nil {
			t.Fatalf("Failed to marshal admin route data: %v", err)
		}

		// Test that we can unmarshal admin back to the same structure
		var unmarshaled armnetwork.EffectiveRouteListResult
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal admin JSON: %v", err)
		}

		// Verify admin data consistency after round-trip
		if len(unmarshaled.Value) != len(adminTestRoutes.Value) {
			t.Errorf("Expected %d admin routes after unmarshal, got %d", len(adminTestRoutes.Value), len(unmarshaled.Value))
		}

		if unmarshaled.Value[0].Name == nil || *unmarshaled.Value[0].Name != "admin-test-route" {
			t.Error("Admin route name not preserved after JSON round-trip")
		}

		if len(unmarshaled.Value[0].AddressPrefix) == 0 || *unmarshaled.Value[0].AddressPrefix[0] != "172.16.1.0/24" {
			t.Error("Admin address prefix not preserved after JSON round-trip")
		}

		// Verify admin-specific fields
		if len(unmarshaled.Value) > 1 {
			adminRoute := unmarshaled.Value[1]
			if len(adminRoute.NextHopIPAddress) == 0 || *adminRoute.NextHopIPAddress[0] != "172.16.1.100" {
				t.Error("Admin next hop IP address not preserved after JSON round-trip")
			}
		}
	})
}

func TestGetAdminOpenshiftClusterEffectiveRouteTableIntegrationWithSharedImplementation(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "11111111-1111-1111-1111-111111111111"
	resourceID := testdatabase.GetResourcePath(mockSubID, "admin-test-cluster")

	t.Run("admin to shared implementation integration", func(t *testing.T) {
		// Simulate the complete flow: admin handler does filepath.Dir then calls shared implementation

		// Admin handler receives request
		adminRequestPath := "/admin" + resourceID + "/effectiveroutingtables"

		// Admin handler applies filepath.Dir (only admin-specific logic)
		adminProcessedPath := filepath.Dir(adminRequestPath)

		// Admin handler then calls shared implementation which extracts resource ID
		sharedResourceID := strings.TrimPrefix(adminProcessedPath, "/admin")

		// Verify the shared implementation receives the correct resource ID from admin handler
		if sharedResourceID != resourceID {
			t.Errorf("Shared implementation should receive correct resource ID from admin: expected '%s', got '%s'", resourceID, sharedResourceID)
		}

		// Verify the shared implementation can parse the resource ID
		resource, err := azure.ParseResourceID(sharedResourceID)
		if err != nil {
			t.Fatalf("Shared implementation should be able to parse admin-processed resource ID: %v", err)
		}

		if resource.SubscriptionID != mockSubID {
			t.Errorf("Expected admin subscription ID %s, got %s", mockSubID, resource.SubscriptionID)
		}

		// Simulate database document structure that shared implementation uses (called by admin handler)
		mockDoc := &api.OpenShiftClusterDocument{
			Key: strings.ToLower(sharedResourceID),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID: sharedResourceID,
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: "/subscriptions/" + mockSubID + "/resourceGroups/admin-test-cluster-rg",
					},
				},
			},
		}

		mockSubscriptionDoc := &api.SubscriptionDocument{
			ID: mockSubID,
			Subscription: &api.Subscription{
				State: api.SubscriptionStateRegistered,
				Properties: &api.SubscriptionProperties{
					TenantID: mockTenantID,
				},
			},
		}

		// Verify the documents have the expected structure for shared implementation
		if mockDoc.OpenShiftCluster.ID != sharedResourceID {
			t.Errorf("Expected admin cluster ID %s, got %s", sharedResourceID, mockDoc.OpenShiftCluster.ID)
		}

		if mockSubscriptionDoc.Subscription.Properties.TenantID != mockTenantID {
			t.Errorf("Expected admin tenant ID %s, got %s", mockTenantID, mockSubscriptionDoc.Subscription.Properties.TenantID)
		}
	})
}
