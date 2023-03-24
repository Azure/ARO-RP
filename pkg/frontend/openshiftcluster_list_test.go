package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func makeDoc(num int) *api.OpenShiftClusterDocument {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceName := fmt.Sprintf("resourceName%02d", num)
	clientSecret := fmt.Sprintf("clientSecret%02d", num)
	return &api.OpenShiftClusterDocument{
		Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, resourceName)),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:   testdatabase.GetResourcePath(mockSubID, resourceName),
			Name: resourceName,
			Type: "Microsoft.RedHatOpenShift/openShiftClusters",
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					PullSecret: "{}",
				},
				ServicePrincipalProfile: api.ServicePrincipalProfile{
					ClientSecret: (api.SecureString)(clientSecret),
				},
			},
		},
	}
}

func TestListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		fixture        func(*testdatabase.Fixture)
		dbError        error
		skipToken      string
		wantEnriched   []string
		wantStatusCode int
		wantResponse   func() *v20200430.OpenShiftClusterList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			fixture: func(f *testdatabase.Fixture) {
				var docs []*api.OpenShiftClusterDocument
				for i := 1; i <= 2; i++ {
					docs = append(docs, makeDoc(i))
				}
				f.AddOpenShiftClusterDocuments(docs...)
			},
			wantEnriched: []string{
				testdatabase.GetResourcePath(mockSubID, "resourceName01"),
				testdatabase.GetResourcePath(mockSubID, "resourceName02"),
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftClusterList {
				return &v20200430.OpenShiftClusterList{
					OpenShiftClusters: []*v20200430.OpenShiftCluster{
						{
							ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName01", mockSubID),
							Name: "resourceName01",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						},
						{
							ID:   testdatabase.GetResourcePath(mockSubID, "resourceName02"),
							Name: "resourceName02",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						},
					},
				}
			},
		},
		{
			name: "clusters exists in db - multiple pages",
			fixture: func(f *testdatabase.Fixture) {
				var docs []*api.OpenShiftClusterDocument
				for i := 1; i <= 11; i++ {
					docs = append(docs, makeDoc(i))
				}
				f.AddOpenShiftClusterDocuments(docs...)
			},
			wantEnriched: []string{
				testdatabase.GetResourcePath(mockSubID, "resourceName01"),
				testdatabase.GetResourcePath(mockSubID, "resourceName02"),
				testdatabase.GetResourcePath(mockSubID, "resourceName03"),
				testdatabase.GetResourcePath(mockSubID, "resourceName04"),
				testdatabase.GetResourcePath(mockSubID, "resourceName05"),
				testdatabase.GetResourcePath(mockSubID, "resourceName06"),
				testdatabase.GetResourcePath(mockSubID, "resourceName07"),
				testdatabase.GetResourcePath(mockSubID, "resourceName08"),
				testdatabase.GetResourcePath(mockSubID, "resourceName09"),
				testdatabase.GetResourcePath(mockSubID, "resourceName10"),
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftClusterList {
				var docs []*v20200430.OpenShiftCluster
				for i := 1; i < 11; i++ {
					docs = append(docs, &v20200430.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, fmt.Sprintf("resourceName%02d", i)),
						Name: fmt.Sprintf("resourceName%02d", i),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					})
				}

				return &v20200430.OpenShiftClusterList{
					OpenShiftClusters: docs,
					NextLink:          "https://mockrefererhost/?%24skipToken=" + base64.StdEncoding.EncodeToString([]byte("FAKE10")),
				}
			},
		},
		{
			name: "request has pagination token",
			fixture: func(f *testdatabase.Fixture) {
				var docs []*api.OpenShiftClusterDocument
				for i := 1; i < 12; i++ {
					docs = append(docs, makeDoc(i))
				}
				f.AddOpenShiftClusterDocuments(docs...)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName11")},
			skipToken:      base64.StdEncoding.EncodeToString([]byte("FAKE10")),
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftClusterList {
				return &v20200430.OpenShiftClusterList{
					OpenShiftClusters: []*v20200430.OpenShiftCluster{
						{
							ID:   testdatabase.GetResourcePath(mockSubID, "resourceName11"),
							Name: "resourceName11",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						},
					},
				}
			},
		},
		{
			name:           "no clusters found in db",
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftClusterList {
				return &v20200430.OpenShiftClusterList{
					OpenShiftClusters: []*v20200430.OpenShiftCluster{},
				}
			},
		},
		{
			name:           "internal error on list",
			dbError:        &cosmosdb.Error{Code: "500"},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
		{
			name:           "internal error while iterating list",
			dbError:        &cosmosdb.Error{Code: "500"},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for name, listPrefix := range map[string]string{
				"subscription list":   fmt.Sprintf("/subscriptions/%s/", mockSubID),
				"resource group list": fmt.Sprintf("/subscriptions/%s/resourcegroups/resourcegroup/", mockSubID),
			} {
				t.Run(name, func(t *testing.T) {
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

					f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, aead, nil, nil, nil, func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Emitter) clusterdata.OpenShiftClusterEnricher {
						return ti.enricher
					})
					if err != nil {
						t.Fatal(err)
					}

					go f.Run(ctx, nil, nil)

					resp, b, err := ti.request(http.MethodGet,
						fmt.Sprintf("https://server%sproviders/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2020-04-30&%%24skipToken=%s", listPrefix, tt.skipToken),
						http.Header{
							"Referer": []string{"https://mockrefererhost/"},
						}, nil)
					if err != nil {
						t.Error(err)
					}

					var wantResponse interface{}
					if tt.wantResponse != nil {
						wantResponse = tt.wantResponse()
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
		})
	}
}
