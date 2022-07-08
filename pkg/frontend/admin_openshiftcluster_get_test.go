package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
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

func TestAdminGetOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		dbError        error
		wantEnriched   []string
		wantStatusCode int
		wantResponse   func(*test) *admin.OpenShiftCluster
		wantError      string
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: testdatabase.GetResourcePath(mockSubID, "resourceName"),
			fixture: func(f *testdatabase.Fixture) {
				clusterDoc := &api.OpenShiftClusterDocument{
					ID:  "00000000-0000-0000-0000-000000000002",
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								PullSecret: "{}",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								ClientSecret: "clientSecret",
							},
						},
					},
				}
				f.AddOpenShiftClusterDocuments(clusterDoc)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: admin.OpenShiftClusterProperties{
						ClusterID: "00000000-0000-0000-0000-000000000002",
					},
				}
			},
		},
		{
			name:           "cluster not found in db",
			resourceID:     testdatabase.GetResourcePath(mockSubID, "resourceName"),
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:           "internal error",
			resourceID:     testdatabase.GetResourcePath(mockSubID, "resourceName"),
			dbError:        &cosmosdb.Error{Code: "500", Message: "oh no"},
			wantStatusCode: http.StatusInternalServerError,
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

			if tt.dbError != nil {
				ti.openShiftClustersClient.SetError(tt.dbError)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, api.APIs, &noop.Noop{}, nil, nil, nil, func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Emitter) clusterdata.OpenShiftClusterEnricher {
				return ti.enricher
			})
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server"+tt.resourceID+"?api-version=admin",
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			var wantResponse interface{}
			if tt.wantResponse != nil {
				wantResponse = tt.wantResponse(tt)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
