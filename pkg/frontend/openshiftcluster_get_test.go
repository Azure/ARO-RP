package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestGetOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mock_database.MockOpenShiftClusters, *mock_clusterdata.MockOpenShiftClusterEnricher)
		wantStatusCode int
		wantResponse   func(*test) *v20200430.OpenShiftCluster
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
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

				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)

				enricher.EXPECT().Enrich(gomock.Any(), clusterDoc.OpenShiftCluster)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
				}
			},
		},
		{
			name:       "cluster not found in db",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "internal error",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, errors.New("random error"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := newTestInfra(t)
			if err != nil {
				t.Fatal(err)
			}
			defer ti.done()

			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			enricher := mock_clusterdata.NewMockOpenShiftClusterEnricher(ti.controller)

			tt.mocks(tt, dbopenshiftclusters, enricher)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, nil, nil, dbopenshiftclusters, nil, ti.l, api.APIs, &noop.Noop{}, nil, nil, clientauthorizer.NewOne(clientcerts[0].Raw), nil)
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).ocEnricher = enricher

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server"+tt.resourceID+"?api-version=2020-04-30",
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
		})
	}
}
