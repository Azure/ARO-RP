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

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
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
								ServicePrincipalProfile: api.ServicePrincipalProfile{
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
								ServicePrincipalProfile: api.ServicePrincipalProfile{
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
					},
					{
						ID:   testdatabase.GetResourcePath(otherMockSubID, "resourceName2"),
						Name: "resourceName2",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
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
			throwsError:    &cosmosdb.Error{Code: "500", Message: "random error"},
			wantError:      `500: InternalServerError: : Internal server error.`,
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

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, aead, nil, nil, nil, func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Emitter) clusterdata.OpenShiftClusterEnricher {
				return ti.enricher
			})
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

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
