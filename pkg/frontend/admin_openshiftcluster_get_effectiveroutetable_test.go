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

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminGetEffectiveRouteTable(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		nicName        string
		extraParams    string // for additional query parameters
		mocks          func(*test, *mock_adminactions.MockAzureActions)
		wantStatusCode int
		wantResponse   interface{}
		validateJSON   bool
		wantError      string
	}

	mockRouteTableData := []byte(`{"value":[{"name":"default-route","source":"Default","state":"Active","addressPrefix":["0.0.0.0/0"],"nextHopIpAddress":["10.0.0.1"],"nextHopType":"VirtualNetworkGateway"},{"name":"subnet-route","source":"VnetLocal","state":"Active","addressPrefix":["10.0.0.0/24"],"nextHopIpAddress":[],"nextHopType":"VnetLocal"}]}`)

	for _, tt := range []*test{
		{
			name:       "successful effective route table retrieval",
			nicName:    "aro-worker-nic-123",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetEffectiveRouteTable(gomock.Any(), tt.nicName).Return(mockRouteTableData, nil)
			},
			wantStatusCode: http.StatusOK,
			validateJSON:   true,
		},
		{
			name:        "successful with additional query parameters",
			nicName:     "aro-worker-nic-123",
			extraParams: "subid=00000000-0000-0000-0000-000000000000&rgn=resourceName",
			resourceID:  testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetEffectiveRouteTable(gomock.Any(), tt.nicName).Return(mockRouteTableData, nil)
			},
			wantStatusCode: http.StatusOK,
			validateJSON:   true,
		},
		{
			name:       "missing nic parameter",
			nicName:    "",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				// No expectations - should fail before calling Azure
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: nic: Network interface name is required",
		},
		{
			name:       "cluster not found",
			nicName:    "aro-worker-nic-123",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/notfound/providers/Microsoft.RedHatOpenShift/openShiftClusters/notfound",
			fixture: func(f *testdatabase.Fixture) {
				// Don't add any cluster documents
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				// No expectations - should fail before calling Azure
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: ResourceNotFound: : The Resource 'openshiftclusters/notfound' under resource group 'notfound' was not found.",
		},
		{
			name:       "azure actions creation failure",
			nicName:    "aro-worker-nic-123",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				// Don't add subscription document to trigger azure actions creation failure
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				// No expectations - should fail during azure actions creation
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : Failed to retrieve subscription document: 400: InvalidSubscriptionState: : Request is not allowed in unregistered subscription '00000000-0000-0000-0000-000000000000'.",
		},
		{
			name:       "azure effective route table retrieval failure",
			nicName:    "aro-worker-nic-123",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetEffectiveRouteTable(gomock.Any(), tt.nicName).Return(nil, fmt.Errorf("network interface not found"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : Failed to retrieve effective route table: network interface not found",
		},
		{
			name:       "empty effective route table response",
			nicName:    "aro-worker-nic-empty",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				a.EXPECT().GetEffectiveRouteTable(gomock.Any(), tt.nicName).Return([]byte(`{"value":[]}`), nil)
			},
			wantStatusCode: http.StatusOK,
			validateJSON:   true,
		},
		{
			name:       "invalid nic name too long",
			nicName:    strings.Repeat("a", 81), // 81 characters, exceeds 80 limit
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			mocks: func(tt *test, a *mock_adminactions.MockAzureActions) {
				// No expectations - should fail before calling Azure
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: nic: Network interface name must be between 1-80 characters",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			tt.mocks(tt, a)

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			// Build the URL with the nic parameter and any extra parameters
			url := fmt.Sprintf("https://server/admin%s/effectiveRouteTable", tt.resourceID)
			if tt.nicName != "" {
				url += fmt.Sprintf("?nic=%s", tt.nicName)
				if tt.extraParams != "" {
					url += "&" + tt.extraParams
				}
			} else if tt.extraParams != "" {
				url += "?" + tt.extraParams
			}

			resp, b, err := ti.request(http.MethodGet, url, nil, nil)
			if err != nil {
				t.Error(err)
			}

			// Use custom validation for JSON responses
			if tt.validateJSON {
				if resp.StatusCode != tt.wantStatusCode {
					t.Errorf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
				}

				if tt.wantError != "" {
					t.Errorf("expected error but got success response")
				}
			} else {
				err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
				if err != nil {
					t.Error(err)
				}
			}

			// Additional validation for successful responses
			if tt.wantStatusCode == http.StatusOK && len(b) > 0 && tt.validateJSON {
				// Verify response is valid JSON
				var result map[string]interface{}
				if err := json.Unmarshal(b, &result); err != nil {
					t.Errorf("Response is not valid JSON: %v", err)
				}

				// Verify response contains expected structure
				if value, ok := result["value"]; ok {
					if valueSlice, ok := value.([]interface{}); ok {
						// For tests with route data, verify structure
						if tt.name == "successful effective route table retrieval" && len(valueSlice) > 0 {
							route := valueSlice[0].(map[string]interface{})
							expectedFields := []string{"name", "source", "state", "addressPrefix", "nextHopType"}
							for _, field := range expectedFields {
								if _, exists := route[field]; !exists {
									t.Errorf("Missing expected field '%s' in route entry", field)
								}
							}
						}
					} else {
						t.Error("Expected 'value' field to be an array")
					}
				} else {
					t.Error("Expected 'value' field in response")
				}
			}
		})
	}
}
