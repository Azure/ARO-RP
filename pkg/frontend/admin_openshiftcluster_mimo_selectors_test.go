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

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testjson "github.com/Azure/ARO-RP/test/util/json"
)

func TestGetMIMOSelectors(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		wantStatusCode int
		wantResponse   func() []byte
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
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
			wantStatusCode: http.StatusOK,
			wantResponse: func() []byte {
				r, err := json.Marshal(map[string]string{
					"subscriptionState":   "Registered",
					"APIServerVisibility": "Public",
					"architectureVersion": "0",
					"authenticationType":  "ServicePrincipal",
					"isManagedDomain":     "false",
					"outboundType":        "Loadbalancer",
					"provisioningState":   "Succeeded",
					"resourceID":          "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
					"subscriptionID":      "00000000-0000-0000-0000-000000000000",
				})
				if err != nil {
					panic(err)
				}
				return r
			},
		},
		{
			name:       "subscription type suspended",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
						},
					},
				})

				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateSuspended,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() []byte {
				r, err := json.Marshal(map[string]string{
					"subscriptionState":   "Suspended",
					"APIServerVisibility": "Public",
					"architectureVersion": "0",
					"authenticationType":  "ServicePrincipal",
					"isManagedDomain":     "false",
					"outboundType":        "Loadbalancer",
					"provisioningState":   "Succeeded",
					"resourceID":          "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
					"subscriptionID":      "00000000-0000-0000-0000-000000000000",
				})
				if err != nil {
					panic(err)
				}
				return r
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin%s/selectors", tt.resourceID),
				nil, nil)
			if err != nil {
				t.Error(err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("got %d, wanted %d", resp.StatusCode, tt.wantStatusCode)
			}

			testjson.AssertJsonMatches(t, tt.wantResponse(), b)
		})
	}
}

func TestGetMIMOSelectorsErrors(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		wantStatusCode int
		wantResponse   func() []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster not found",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName2"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
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
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename2' under resource group 'resourcegroup' was not found.",
		},
		{
			name:       "subscription not found",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
							},
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: ResourceNotFound: : The subscription ID '" + mockSubID + "' was not found.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin%s/selectors", tt.resourceID),
				nil, nil)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
