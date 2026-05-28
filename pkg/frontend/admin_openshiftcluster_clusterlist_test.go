package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestGetAdminOpenShiftClusterList(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	otherMockSubID := "00000000-0000-0000-0000-000000000001"

	createdAt := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	lastModified := time.Date(2024, 7, 20, 14, 30, 0, 0, time.UTC)

	type test struct {
		name           string
		fixture        func(*testdatabase.Fixture)
		dbError        error
		wantStatusCode int
		wantResponse   []*adminClusterListEntry
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exist in db",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc1",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster1",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(mockSubID, "cluster1"),
							Name: "cluster1",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateSucceeded,
								ClusterProfile: api.ClusterProfile{
									Version: "4.14.16",
								},
								CreatedBy:     "user@example.com",
								ProvisionedBy: "aro-rp",
								CreatedAt:     createdAt,
							},
							SystemData: api.SystemData{
								LastModifiedAt: &lastModified,
							},
						},
					},
					&api.OpenShiftClusterDocument{
						ID:  "doc2",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/cluster2",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(otherMockSubID, "cluster2"),
							Name: "cluster2",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateUpdating,
								ClusterProfile: api.ClusterProfile{
									Version: "4.15.2",
								},
								CreatedBy:     "admin@example.com",
								ProvisionedBy: "aro-rp",
								CreatedAt:     createdAt,
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: []*adminClusterListEntry{
				{
					Key:                     "doc1",
					ResourceId:              testdatabase.GetResourcePath(mockSubID, "cluster1"),
					Name:                    "cluster1",
					Subscription:            mockSubID,
					ResourceGroup:           "resourceGroup",
					Version:                 "4.14.16",
					CreatedAt:               "2024-06-15T10:00:00Z",
					CreatedBy:               "user@example.com",
					ProvisionedBy:           "aro-rp",
					ProvisioningState:       "Succeeded",
					FailedProvisioningState: "",
					LastModified:            "2024-07-20T14:30:00Z",
				},
				{
					Key:                     "doc2",
					ResourceId:              testdatabase.GetResourcePath(otherMockSubID, "cluster2"),
					Name:                    "cluster2",
					Subscription:            otherMockSubID,
					ResourceGroup:           "resourceGroup",
					Version:                 "4.15.2",
					CreatedAt:               "2024-06-15T10:00:00Z",
					CreatedBy:               "admin@example.com",
					ProvisionedBy:           "aro-rp",
					ProvisioningState:       "Failed",
					FailedProvisioningState: "Updating",
					LastModified:            "",
				},
			},
		},
		{
			name:           "no clusters in db",
			wantStatusCode: http.StatusOK,
			wantResponse:   []*adminClusterListEntry{},
		},
		{
			name: "nil OpenShiftCluster is skipped",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:               "doc-nil",
						Key:              "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/nilcluster",
						OpenShiftCluster: nil,
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   []*adminClusterListEntry{},
		},
		{
			name: "cluster with missing optional fields",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc-sparse",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/sparse",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID: testdatabase.GetResourcePath(mockSubID, "sparse"),
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateCreating,
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: []*adminClusterListEntry{
				{
					Key:               "doc-sparse",
					ResourceId:        testdatabase.GetResourcePath(mockSubID, "sparse"),
					Name:              "sparse",
					Subscription:      mockSubID,
					ResourceGroup:     "resourceGroup",
					ProvisioningState: "Creating",
				},
			},
		},
		{
			name: "cluster with unparseable resource ID",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc-bad-id",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/badcluster",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID: "not-a-valid-arm-id",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateSucceeded,
								CreatedBy:         "someone@example.com",
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: []*adminClusterListEntry{
				{
					Key:               "doc-bad-id",
					ResourceId:        "not-a-valid-arm-id",
					CreatedBy:         "someone@example.com",
					ProvisioningState: "Succeeded",
				},
			},
		},
		{
			name: "results are sorted by resource ID: subscription then resource group then name",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc-1",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg-beta/providers/microsoft.redhatopenshift/openshiftclusters/cluster-b",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-beta/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-b", mockSubID),
							Properties: api.OpenShiftClusterProperties{},
						},
					},
					&api.OpenShiftClusterDocument{
						ID:  "doc-2",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg-alpha/providers/microsoft.redhatopenshift/openshiftclusters/cluster-z",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-z", mockSubID),
							Properties: api.OpenShiftClusterProperties{},
						},
					},
					&api.OpenShiftClusterDocument{
						ID:  "doc-3",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg-alpha/providers/microsoft.redhatopenshift/openshiftclusters/cluster-a",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-a", mockSubID),
							Properties: api.OpenShiftClusterProperties{},
						},
					},
					&api.OpenShiftClusterDocument{
						ID:  "doc-4",
						Key: "/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/rg-alpha/providers/microsoft.redhatopenshift/openshiftclusters/cluster-a",
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-a", otherMockSubID),
							Properties: api.OpenShiftClusterProperties{},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: []*adminClusterListEntry{
				{
					Key:           "doc-3",
					ResourceId:    fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-a", mockSubID),
					Name:          "cluster-a",
					Subscription:  mockSubID,
					ResourceGroup: "rg-alpha",
				},
				{
					Key:           "doc-2",
					ResourceId:    fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-z", mockSubID),
					Name:          "cluster-z",
					Subscription:  mockSubID,
					ResourceGroup: "rg-alpha",
				},
				{
					Key:           "doc-1",
					ResourceId:    fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-beta/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-b", mockSubID),
					Name:          "cluster-b",
					Subscription:  mockSubID,
					ResourceGroup: "rg-beta",
				},
				{
					Key:           "doc-4",
					ResourceId:    fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-alpha/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-a", otherMockSubID),
					Name:          "cluster-a",
					Subscription:  otherMockSubID,
					ResourceGroup: "rg-alpha",
				},
			},
		},
		{
			name:           "database error returns 500",
			dbError:        &cosmosdb.Error{StatusCode: 500, Code: "ERR500", Message: "cosmos unavailable"},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : 500 ERR500: cosmos unavailable`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			if tt.dbError != nil {
				ti.openShiftClustersClient.SetError(tt.dbError)
			}

			aead := testdatabase.NewFakeAEAD()

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, aead, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/clusters",
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
			}

			if tt.wantError != "" {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}
				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
				return
			}

			var got []*adminClusterListEntry
			err = json.Unmarshal(b, &got)
			if err != nil {
				t.Fatalf("failed to unmarshal response: %v\nraw: %s", err, string(b))
			}

			if diff := cmp.Diff(tt.wantResponse, got); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetAdminOpenShiftClusterListFiltering(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	otherMockSubID := "00000000-0000-0000-0000-000000000001"

	type test struct {
		name           string
		queryParams    string
		fixture        func(*testdatabase.Fixture)
		wantStatusCode int
		wantNames      []string
	}

	commonFixture := func(f *testdatabase.Fixture) {
		f.AddOpenShiftClusterDocuments(
			&api.OpenShiftClusterDocument{
				ID:  "doc1",
				Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/prod-cluster",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "prod-cluster"),
					Name: "prod-cluster",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						ClusterProfile:    api.ClusterProfile{Version: "4.14.16"},
						CreatedBy:         "abc123",
						ProvisionedBy:     "def456",
					},
				},
			},
			&api.OpenShiftClusterDocument{
				ID:  "doc2",
				Key: "/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/dev-cluster",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(otherMockSubID, "dev-cluster"),
					Name: "dev-cluster",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateFailed,
						ClusterProfile:    api.ClusterProfile{Version: "4.15.2"},
						CreatedBy:         "xyz789",
						ProvisionedBy:     "def456",
					},
				},
			},
			&api.OpenShiftClusterDocument{
				ID:  "doc3",
				Key: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/staging-cluster",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   testdatabase.GetResourcePath(mockSubID, "staging-cluster"),
					Name: "staging-cluster",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						ClusterProfile:    api.ClusterProfile{Version: "4.14.16"},
						CreatedBy:         "abc123",
						ProvisionedBy:     "ghi012",
					},
				},
			},
		)
	}

	for _, tt := range []*test{
		{
			name:           "no filter returns all clusters",
			queryParams:    "",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"dev-cluster", "prod-cluster", "staging-cluster"},
		},
		{
			name:           "filter by name substring",
			queryParams:    "?name=prod",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"prod-cluster"},
		},
		{
			name:           "filter by name is case-insensitive",
			queryParams:    "?name=PROD",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"prod-cluster"},
		},
		{
			name:           "filter by subscription",
			queryParams:    "?subscription=" + otherMockSubID,
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"dev-cluster"},
		},
		{
			name:           "filter by version",
			queryParams:    "?version=4.15",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"dev-cluster"},
		},
		{
			name:           "filter by created_by",
			queryParams:    "?created_by=abc123",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"prod-cluster", "staging-cluster"},
		},
		{
			name:           "filter by provisioned_by",
			queryParams:    "?provisioned_by=ghi012",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"staging-cluster"},
		},
		{
			name:           "filter by state exact match",
			queryParams:    "?state=Failed",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"dev-cluster"},
		},
		{
			name:           "filter by state is case-insensitive",
			queryParams:    "?state=failed",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"dev-cluster"},
		},
		{
			name:           "combined filters use AND logic",
			queryParams:    "?version=4.14&subscription=" + mockSubID,
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{"prod-cluster", "staging-cluster"},
		},
		{
			name:           "no matches returns empty list",
			queryParams:    "?name=nonexistent",
			fixture:        commonFixture,
			wantStatusCode: http.StatusOK,
			wantNames:      []string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			aead := testdatabase.NewFakeAEAD()

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, aead, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/clusters"+tt.queryParams,
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
			}

			var got []*adminClusterListEntry
			err = json.Unmarshal(b, &got)
			if err != nil {
				t.Fatalf("failed to unmarshal response: %v\nraw: %s", err, string(b))
			}

			gotNames := make([]string, len(got))
			for i, c := range got {
				gotNames[i] = c.Name
			}
			slices.Sort(gotNames)
			slices.Sort(tt.wantNames)

			if diff := cmp.Diff(tt.wantNames, gotNames); diff != "" {
				t.Errorf("cluster names mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
