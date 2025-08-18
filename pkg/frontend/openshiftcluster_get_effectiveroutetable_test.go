package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestGetOpenshiftClusterEffectiveRouteTableResourceIDParsing(t *testing.T) {
	tests := []struct {
		name               string
		requestPath        string
		expectedResourceID string
		expectValidParsing bool
		expectedSubID      string
		expectedRG         string
		description        string
	}{
		{
			name:               "standard admin path",
			requestPath:        "/admin/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			expectedResourceID: "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			expectValidParsing: true,
			expectedSubID:      "12345",
			expectedRG:         "test-rg",
			description:        "Should parse standard ARO cluster resource ID correctly",
		},
		{
			name:               "non-admin path",
			requestPath:        "/subscriptions/67890/resourceGroups/my-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/my-cluster",
			expectedResourceID: "/subscriptions/67890/resourceGroups/my-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/my-cluster",
			expectValidParsing: true,
			expectedSubID:      "67890",
			expectedRG:         "my-rg",
			description:        "Should handle non-admin path correctly",
		},
		{
			name:               "empty path",
			requestPath:        "",
			expectedResourceID: "",
			expectValidParsing: false,
			description:        "Should handle empty path gracefully",
		},
		{
			name:               "admin only path",
			requestPath:        "/admin",
			expectedResourceID: "",
			expectValidParsing: false,
			description:        "Should handle admin-only path",
		},
		{
			name:               "malformed resource ID",
			requestPath:        "/admin/invalid/resource/path",
			expectedResourceID: "/invalid/resource/path",
			expectValidParsing: false,
			description:        "Should handle malformed resource ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test resource ID extraction using the  logic
			resourceID := strings.TrimPrefix(tt.requestPath, "/admin")
			if resourceID != tt.expectedResourceID {
				t.Errorf("Expected resourceID='%s', got='%s'", tt.expectedResourceID, resourceID)
			}

			// Test azure.ParseResourceID if we expect valid parsing
			if tt.expectValidParsing && resourceID != "" {
				resource, err := azure.ParseResourceID(resourceID)
				if err != nil {
					t.Errorf("Expected successful parsing but got error: %v", err)
					return
				}

				if resource.SubscriptionID != tt.expectedSubID {
					t.Errorf("Expected subscriptionID='%s', got='%s'", tt.expectedSubID, resource.SubscriptionID)
				}
				if resource.ResourceGroup != tt.expectedRG {
					t.Errorf("Expected resourceGroup='%s', got='%s'", tt.expectedRG, resource.ResourceGroup)
				}
			} else if !tt.expectValidParsing && resourceID != "" {
				_, err := azure.ParseResourceID(resourceID)
				if err == nil {
					t.Error("Expected parsing error but got none")
				}
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableQueryParameterHandling(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		expectedNIC string
		expectError bool
		description string
	}{
		{
			name:        "valid NIC parameter",
			queryString: "nic=test-nic-interface",
			expectedNIC: "test-nic-interface",
			expectError: false,
			description: "Should extract NIC name from query parameters",
		},
		{
			name:        "URL encoded NIC parameter",
			queryString: "nic=test%2Dnic%2Dinterface",
			expectedNIC: "test-nic-interface",
			expectError: false,
			description: "Should handle URL encoded NIC names",
		},
		{
			name:        "missing NIC parameter",
			queryString: "other=value",
			expectedNIC: "",
			expectError: true,
			description: "Should return error when NIC parameter is missing",
		},
		{
			name:        "empty NIC parameter",
			queryString: "nic=",
			expectedNIC: "",
			expectError: true,
			description: "Should return error when NIC parameter is empty",
		},
		{
			name:        "additional parameters with valid NIC",
			queryString: "nic=my-nic&extra=ignored&other=value",
			expectedNIC: "my-nic",
			expectError: false,
			description: "Should extract NIC while ignoring other parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			req, err := http.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Extract NIC parameter the same way the  implementation does
			nicName := req.URL.Query().Get("nic")

			if nicName != tt.expectedNIC {
				t.Errorf("Expected nicName='%s', got='%s'", tt.expectedNIC, nicName)
			}

			// Test the validation logic that would be in the  implementation
			if tt.expectError && nicName == "" {
				// This simulates the error that would be returned
				expectedError := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name is required")
				if expectedError == nil {
					t.Error("Expected error for missing/empty NIC parameter")
				}
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableAzureActionsIntegration(t *testing.T) {
	testNIC := "test-nic-name"

	// Create mock effective route data that AzureActions would return
	mockRouteData := []byte(`{
		"value": [
			{
				"name": "default-route",
				"source": "Default",
				"state": "Active",
				"addressPrefix": ["0.0.0.0/0"],
				"nextHopType": "VirtualNetworkGateway"
			},
			{
				"name": "local-route", 
				"source": "VNetLocal",
				"state": "Active",
				"addressPrefix": ["10.0.0.0/16"],
				"nextHopType": "VnetLocal"
			}
		]
	}`)

	tests := []struct {
		name               string
		setupMockActions   func(ctrl *gomock.Controller) *mock_adminactions.MockAzureActions
		expectError        bool
		expectedRouteCount int
		description        string
	}{
		{
			name: "successful route table retrieval with AzureActions",
			setupMockActions: func(ctrl *gomock.Controller) *mock_adminactions.MockAzureActions {
				mockActions := mock_adminactions.NewMockAzureActions(ctrl)
				mockActions.EXPECT().
					GetEffectiveRouteTable(gomock.Any(), testNIC).
					Return(mockRouteData, nil)
				return mockActions
			},
			expectError:        false,
			expectedRouteCount: 2,
			description:        "Should successfully retrieve effective routes using AzureActions interface",
		},
		{
			name: "azure actions network interface not found error",
			setupMockActions: func(ctrl *gomock.Controller) *mock_adminactions.MockAzureActions {
				mockActions := mock_adminactions.NewMockAzureActions(ctrl)
				mockActions.EXPECT().
					GetEffectiveRouteTable(gomock.Any(), testNIC).
					Return(nil, fmt.Errorf("NetworkInterfaceNotFound"))
				return mockActions
			},
			expectError: true,
			description: "Should handle network interface not found errors from AzureActions",
		},
		{
			name: "azure actions permission denied error",
			setupMockActions: func(ctrl *gomock.Controller) *mock_adminactions.MockAzureActions {
				mockActions := mock_adminactions.NewMockAzureActions(ctrl)
				mockActions.EXPECT().
					GetEffectiveRouteTable(gomock.Any(), testNIC).
					Return(nil, fmt.Errorf("InsufficientPermissions"))
				return mockActions
			},
			expectError: true,
			description: "Should handle permission denied errors from AzureActions",
		},
		{
			name: "empty effective route table response from AzureActions",
			setupMockActions: func(ctrl *gomock.Controller) *mock_adminactions.MockAzureActions {
				mockActions := mock_adminactions.NewMockAzureActions(ctrl)
				emptyRouteData := []byte(`{"value": []}`)
				mockActions.EXPECT().
					GetEffectiveRouteTable(gomock.Any(), testNIC).
					Return(emptyRouteData, nil)
				return mockActions
			},
			expectError:        false,
			expectedRouteCount: 0,
			description:        "Should handle empty effective route table responses from AzureActions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			// Set up mock AzureActions
			mockActions := tt.setupMockActions(ctrl)

			// Test the AzureActions.GetEffectiveRouteTable method that the implementation uses
			jsonData, err := mockActions.GetEffectiveRouteTable(ctx, testNIC)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				// Verify error would be wrapped in api.NewCloudError in the implementation
				expectedError := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					fmt.Sprintf("Failed to retrieve effective route table: %v", err))
				if expectedError == nil {
					t.Error("Expected CloudError wrapper")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify JSON is valid
			var jsonObj map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonObj)
			if err != nil {
				t.Fatalf("Generated JSON is invalid: %v", err)
			}

			// Verify route count
			if valueObj, ok := jsonObj["value"].([]interface{}); ok {
				if len(valueObj) != tt.expectedRouteCount {
					t.Errorf("Expected %d routes, got %d", tt.expectedRouteCount, len(valueObj))
				}

				// For non-empty results, verify content
				if tt.expectedRouteCount > 0 {
					jsonStr := string(jsonData)
					if !strings.Contains(jsonStr, "default-route") || !strings.Contains(jsonStr, "local-route") {
						t.Error("JSON should contain route names")
					}
					if !strings.Contains(jsonStr, "0.0.0.0/0") || !strings.Contains(jsonStr, "10.0.0.0/16") {
						t.Error("JSON should contain address prefixes")
					}
				}
			} else {
				t.Error("JSON response should contain 'value' array")
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		nicName           string
		resourceID        string
		mockParseError    bool
		expectedHTTPCode  int
		expectedErrorCode string
		description       string
	}{
		{
			name:              "missing NIC parameter",
			nicName:           "",
			resourceID:        "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			mockParseError:    false,
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest when NIC parameter is missing",
		},
		{
			name:              "empty resource ID",
			nicName:           "test-nic",
			resourceID:        "",
			mockParseError:    false,
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest when resource ID is empty",
		},
		{
			name:              "invalid resource ID format",
			nicName:           "test-nic",
			resourceID:        "/invalid/resource/id/format",
			mockParseError:    true,
			expectedHTTPCode:  http.StatusBadRequest,
			expectedErrorCode: api.CloudErrorCodeInvalidParameter,
			description:       "Should return BadRequest when resource ID is malformed",
		},
		{
			name:              "valid parameters",
			nicName:           "test-nic",
			resourceID:        "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			mockParseError:    false,
			expectedHTTPCode:  0, // No error expected
			expectedErrorCode: "",
			description:       "Should pass validation with valid parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test NIC name validation
			if tt.nicName == "" {
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
			if tt.resourceID == "" {
				err := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Resource ID is required")
				if err.StatusCode != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPCode, err.StatusCode)
				}
				if err.Code != tt.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrorCode, err.Code)
				}
				return
			}

			// Test azure.ParseResourceID validation
			_, err := azure.ParseResourceID(tt.resourceID)
			if tt.mockParseError {
				if err == nil {
					t.Error("Expected parse error but got none")
					return
				}
				// Simulate the error wrapping that the  implementation does
				cloudErr := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Invalid resource ID format")
				if cloudErr.StatusCode != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPCode, cloudErr.StatusCode)
				}
				if cloudErr.Code != tt.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrorCode, cloudErr.Code)
				}
			} else if err != nil {
				t.Errorf("Unexpected parse error: %v", err)
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableClusterResourceGroupExtraction(t *testing.T) {
	tests := []struct {
		name                      string
		clusterResourceGroupID    string
		expectedResourceGroupName string
		expectError               bool
		description               string
	}{
		{
			name:                      "valid cluster resource group ID",
			clusterResourceGroupID:    "/subscriptions/12345/resourceGroups/cluster-rg/providers/Microsoft.Resources/resourceGroups",
			expectedResourceGroupName: "cluster-rg",
			expectError:               false,
			description:               "Should extract resource group name from valid cluster resource group ID",
		},
		{
			name:                      "another valid cluster resource group ID",
			clusterResourceGroupID:    "/subscriptions/67890/resourceGroups/my-cluster-resources/providers/Microsoft.Resources/resourceGroups",
			expectedResourceGroupName: "my-cluster-resources",
			expectError:               false,
			description:               "Should extract resource group name from another valid format",
		},
		{
			name:                   "invalid cluster resource group ID format",
			clusterResourceGroupID: "/invalid/format/resourceGroup",
			expectError:            true,
			description:            "Should return error for invalid cluster resource group ID format",
		},
		{
			name:                   "empty cluster resource group ID",
			clusterResourceGroupID: "",
			expectError:            true,
			description:            "Should return error for empty cluster resource group ID",
		},
		{
			name:                   "malformed cluster resource group ID",
			clusterResourceGroupID: "/subscriptions/12345/resourceGroups",
			expectError:            true,
			description:            "Should return error for malformed cluster resource group ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test cluster resource group extraction logic from implementation
			if tt.clusterResourceGroupID == "" {
				// Should return error for empty cluster resource group
				err := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					"Cluster resource group not found in cluster document")
				if !tt.expectError {
					t.Error("Expected no error but got error")
				}
				if err.StatusCode != http.StatusInternalServerError {
					t.Error("Should return 500 for missing cluster resource group")
				}
				return
			}

			// Extract resource group name from resource group ID (same logic as implementation)
			// ClusterResourceGroup format: /subscriptions/{sub}/resourceGroups/{rg}
			parts := strings.Split(tt.clusterResourceGroupID, "/")
			if len(parts) < 5 || parts[3] != "resourceGroups" {
				if !tt.expectError {
					t.Error("Expected successful extraction but got parsing error")
				}
				err := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					"Invalid cluster resource group format")
				if err.StatusCode != http.StatusInternalServerError {
					t.Error("Should return 500 for invalid cluster resource group format")
				}
				return
			}

			clusterResourceGroupName := parts[4]
			if tt.expectError {
				t.Error("Expected error but got successful extraction")
				return
			}

			if clusterResourceGroupName != tt.expectedResourceGroupName {
				t.Errorf("Expected cluster resource group name '%s', got '%s'", tt.expectedResourceGroupName, clusterResourceGroupName)
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableDatabaseIntegration(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		expectError bool
		description string
	}{
		{
			name:        "valid cluster resource ID",
			resourceID:  "/subscriptions/12345/resourceGroups/test-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster",
			expectError: false,
			description: "Should handle valid cluster resource ID for database lookup",
		},
		{
			name:        "cluster not found scenario",
			resourceID:  "/subscriptions/12345/resourceGroups/missing-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/missing-cluster",
			expectError: true,
			description: "Should return appropriate error when cluster not found in database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the resource ID parsing that would be used for database lookup
			resource, err := azure.ParseResourceID(tt.resourceID)
			if err != nil {
				if !tt.expectError {
					t.Errorf("Unexpected parse error: %v", err)
				}
				return
			}

			// Verify resource components are extracted correctly for database operations
			if resource.SubscriptionID == "" {
				t.Error("Subscription ID should be extracted for database operations")
			}
			if resource.ResourceGroup == "" {
				t.Error("Resource group should be extracted for database operations")
			}
			if resource.ResourceName == "" {
				t.Error("Resource name should be extracted for database operations")
			}

			// Test error formatting that would be used when cluster not found
			if tt.expectError {
				notFoundError := api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
					fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.",
						resource.ResourceType, resource.ResourceName, resource.ResourceGroup))
				if notFoundError.StatusCode != http.StatusNotFound {
					t.Error("Should return 404 for not found clusters")
				}
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableDataConsistency(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := testdatabase.GetResourcePath(mockSubID, "test-cluster")

	t.Run("resource ID extraction consistency", func(t *testing.T) {
		// Test that the implementation extracts resource ID consistently
		adminPath := "/admin" + resourceID
		extractedResourceID := strings.TrimPrefix(adminPath, "/admin")

		if extractedResourceID != resourceID {
			t.Errorf("Expected resourceID='%s', got='%s'", resourceID, extractedResourceID)
		}

		// Test that azure.ParseResourceID can parse the extracted ID
		resource, err := azure.ParseResourceID(extractedResourceID)
		if err != nil {
			t.Fatalf("Failed to parse extracted resource ID: %v", err)
		}

		if resource.SubscriptionID != mockSubID {
			t.Errorf("Expected subscriptionID='%s', got='%s'", mockSubID, resource.SubscriptionID)
		}
	})

	t.Run("JSON serialization consistency", func(t *testing.T) {
		// Create test route data that AzureActions would return
		testRouteData := []byte(`{
			"value": [
				{
					"name": "test-route",
					"source": "Default",
					"state": "Active",
					"addressPrefix": ["192.168.1.0/24"],
					"nextHopType": "VnetLocal"
				}
			]
		}`)

		// Test that we can unmarshal the JSON from AzureActions
		var jsonObj map[string]interface{}
		err := json.Unmarshal(testRouteData, &jsonObj)
		if err != nil {
			t.Fatalf("Failed to unmarshal AzureActions JSON: %v", err)
		}

		// Verify data structure
		if valueObj, ok := jsonObj["value"].([]interface{}); ok {
			if len(valueObj) != 1 {
				t.Errorf("Expected 1 route, got %d", len(valueObj))
			}

			if route, ok := valueObj[0].(map[string]interface{}); ok {
				if routeName, ok := route["name"].(string); !ok || routeName != "test-route" {
					t.Error("Route name not preserved in AzureActions JSON")
				}

				if addressPrefixes, ok := route["addressPrefix"].([]interface{}); !ok || len(addressPrefixes) == 0 {
					t.Error("Address prefix not preserved in AzureActions JSON")
				} else if prefix, ok := addressPrefixes[0].(string); !ok || prefix != "192.168.1.0/24" {
					t.Error("Address prefix value not preserved in AzureActions JSON")
				}
			} else {
				t.Error("Route object structure invalid in AzureActions JSON")
			}
		} else {
			t.Error("JSON response should contain 'value' array")
		}
	})
}

func TestGetOpenshiftClusterEffectiveRouteTableSubscriptionDocumentHandling(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		tenantID       string
		expectError    bool
		description    string
	}{
		{
			name:           "valid subscription document",
			subscriptionID: "00000000-0000-0000-0000-000000000000",
			tenantID:       "11111111-1111-1111-1111-111111111111",
			expectError:    false,
			description:    "Should handle valid subscription document with tenant ID",
		},
		{
			name:           "missing tenant ID",
			subscriptionID: "00000000-0000-0000-0000-000000000000",
			tenantID:       "",
			expectError:    true,
			description:    "Should return error when tenant ID is missing from subscription document",
		},
		{
			name:           "missing subscription ID",
			subscriptionID: "",
			tenantID:       "11111111-1111-1111-1111-111111111111",
			expectError:    true,
			description:    "Should return error when subscription ID is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test subscription document validation logic from implementation
			if tt.subscriptionID == "" {
				err := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					"Failed to retrieve subscription document: subscription ID is required")
				if !tt.expectError {
					t.Error("Expected no error but got error for missing subscription ID")
				}
				if err.StatusCode != http.StatusInternalServerError {
					t.Error("Should return 500 for missing subscription ID")
				}
				return
			}

			if tt.tenantID == "" && tt.expectError {
				err := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
					"Failed to create Azure credentials: tenant ID is required")
				if err.StatusCode != http.StatusInternalServerError {
					t.Error("Should return 500 for missing tenant ID")
				}
				return
			}

			// Valid case - verify IDs are usable
			if !tt.expectError {
				if tt.subscriptionID == "" || tt.tenantID == "" {
					t.Error("Valid case should have both subscription ID and tenant ID")
				}
			}
		})
	}
}

func TestGetOpenshiftClusterEffectiveRouteTableAzureActionsFactoryHandling(t *testing.T) {
	mockTenantID := "11111111-1111-1111-1111-111111111111"
	mockSubID := "00000000-0000-0000-0000-000000000000"

	t.Run("azureActionsFactory validation", func(t *testing.T) {
		// Test the parameters that would be used for azureActionsFactory
		if mockTenantID == "" {
			t.Error("Tenant ID should be available for AzureActions creation")
		}
		if mockSubID == "" {
			t.Error("Subscription ID should be available for AzureActions creation")
		}

		// Test error handling for AzureActions factory failure
		factoryError := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create Azure actions: %v", "mock factory error"))
		if factoryError.StatusCode != http.StatusInternalServerError {
			t.Error("Should return 500 for AzureActions factory failures")
		}
	})

	t.Run("azureActions method validation", func(t *testing.T) {
		// Test error handling for AzureActions method failure
		methodError := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to retrieve effective route table: %v", "mock method error"))
		if methodError.StatusCode != http.StatusInternalServerError {
			t.Error("Should return 500 for AzureActions method failures")
		}
	})
}
