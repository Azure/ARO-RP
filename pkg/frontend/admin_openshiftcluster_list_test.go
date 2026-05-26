package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	otherMockSubID := "00000000-0000-0000-0000-000000000001"

	type test struct {
		name           string
		wantEnriched   []string
		throwsError    error
		fixture        func(*testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *admin.OpenShiftClusterList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName1")),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(mockSubID, "resourceName1"),
							Name: "resourceName1",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: &api.ServicePrincipalProfile{
									ClientSecret: "clientSecret1",
								},
							},
						},
					},
					&api.OpenShiftClusterDocument{
						Key: strings.ToLower(testdatabase.GetResourcePath(otherMockSubID, "resourceName2")),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(otherMockSubID, "resourceName2"),
							Name: "resourceName2",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: &api.ServicePrincipalProfile{
									ClientSecret: "clientSecret2",
								},
							},
						},
					})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName1"), testdatabase.GetResourcePath(otherMockSubID, "resourceName2")},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftClusterList{
				OpenShiftClusters: []*admin.OpenShiftCluster{
					{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName1"),
						Name: "resourceName1",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: admin.OpenShiftClusterProperties{
							ServicePrincipalProfile: &admin.ServicePrincipalProfile{},
							NetworkProfile: admin.NetworkProfile{
								PreconfiguredNSG: admin.PreconfiguredNSGDisabled, // ✅ Ensure expected value
							},
						},
					},
					{
						ID:   testdatabase.GetResourcePath(otherMockSubID, "resourceName2"),
						Name: "resourceName2",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: admin.OpenShiftClusterProperties{
							ServicePrincipalProfile: &admin.ServicePrincipalProfile{},
							NetworkProfile: admin.NetworkProfile{
								PreconfiguredNSG: admin.PreconfiguredNSGDisabled, // ✅ Ensure expected value
							},
						},
					},
				},
			},
		},
		{
			name:           "no clusters found in db",
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftClusterList{
				OpenShiftClusters: []*admin.OpenShiftCluster{},
			},
		},
		{
			name:           "internal error while iterating list",
			wantStatusCode: http.StatusInternalServerError,
			throwsError:    &cosmosdb.Error{StatusCode: 500, Code: "ERR500", Message: "random error"},
			wantError:      `500: InternalServerError: : 500 ERR500: random error`,
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

			if tt.throwsError != nil {
				ti.openShiftClustersClient.SetError(tt.throwsError)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, aead, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/providers/Microsoft.RedHatOpenShift/openShiftClusters",
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			if tt.wantError == "" {
				var ocs *admin.OpenShiftClusterList
				err = json.Unmarshal(b, &ocs)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(ocs, tt.wantResponse) {
					b, _ := json.Marshal(ocs)
					t.Error(string(b))
				}
			} else {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}
		})
	}
}

func TestAdminListOpenShiftClusterOverview(t *testing.T) {
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
		wantResponse   *adminClusterOverviewList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "overview returns flat entries",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc1",
						Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "cluster1")),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(mockSubID, "cluster1"),
							Name: "cluster1",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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
						Key: strings.ToLower(testdatabase.GetResourcePath(otherMockSubID, "cluster2")),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   testdatabase.GetResourcePath(otherMockSubID, "cluster2"),
							Name: "cluster2",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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
			wantResponse: &adminClusterOverviewList{
				Clusters: []*adminClusterListEntry{
					{
						Key:                     "doc1",
						ResourceID:              testdatabase.GetResourcePath(mockSubID, "cluster1"),
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
						ResourceID:              testdatabase.GetResourcePath(otherMockSubID, "cluster2"),
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
		},
		{
			name:           "overview with no clusters returns empty array",
			wantStatusCode: http.StatusOK,
			wantResponse: &adminClusterOverviewList{
				Clusters: []*adminClusterListEntry{},
			},
		},
		{
			name: "overview skips nil OpenShiftCluster",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						Key:              strings.ToLower(testdatabase.GetResourcePath(mockSubID, "nilcluster")),
						OpenShiftCluster: nil,
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &adminClusterOverviewList{
				Clusters: []*adminClusterListEntry{},
			},
		},
		{
			name: "overview with missing optional fields",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc-sparse",
						Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "sparse")),
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
			wantResponse: &adminClusterOverviewList{
				Clusters: []*adminClusterListEntry{
					{
						Key:               "doc-sparse",
						ResourceID:        testdatabase.GetResourcePath(mockSubID, "sparse"),
						Name:              "sparse",
						Subscription:      mockSubID,
						ResourceGroup:     "resourceGroup",
						ProvisioningState: "Creating",
					},
				},
			},
		},
		{
			name: "overview with unparseable resource ID",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(
					&api.OpenShiftClusterDocument{
						ID:  "doc-bad-id",
						Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "badcluster")),
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
			wantResponse: &adminClusterOverviewList{
				Clusters: []*adminClusterListEntry{
					{
						Key:               "doc-bad-id",
						ResourceID:        "not-a-valid-arm-id",
						CreatedBy:         "someone@example.com",
						ProvisioningState: "Succeeded",
					},
				},
			},
		},
		{
			name:           "overview with database error returns 500",
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
				"https://server/admin/providers/Microsoft.RedHatOpenShift/openShiftClusters?view=overview",
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
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

			var got adminClusterOverviewList
			err = json.Unmarshal(b, &got)
			if err != nil {
				t.Fatalf("failed to unmarshal response: %v\nraw: %s", err, string(b))
			}

			if diff := cmp.Diff(tt.wantResponse, &got); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
