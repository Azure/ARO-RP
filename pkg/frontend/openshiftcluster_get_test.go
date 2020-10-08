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
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestGetOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mock_database.MockOpenShiftClusters)
		wantEnriched   []string
		wantStatusCode int
		wantResponse   func(*test) *v20200430.OpenShiftCluster
		wantError      string
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
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
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
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
			resourceID: getResourcePath(mockSubID, "resourceName"),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "internal error",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
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

			openshiftClusters := mock_database.NewMockOpenShiftClusters(ti.controller)

			tt.mocks(tt, openshiftClusters)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, openshiftClusters, nil, api.APIs, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).ocEnricher = ti.enricher

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

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
