package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestGetAsyncOperationResult(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockClusterDocKey := "22222222-2222-2222-2222-222222222222"
	mockOpID := "11111111-1111-1111-1111-111111111111"

	type test struct {
		name           string
		mocks          func(*mock_database.MockOpenShiftClusters, *mock_database.MockAsyncOperations)
		wantStatusCode int
		wantAsync      bool
		wantResponse   *v20200430.OpenShiftCluster
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "operation and cluster exist in db - final result is available with content",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
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

				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						OpenShiftCluster:    clusterDoc.OpenShiftCluster,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20200430.OpenShiftCluster{
				ID:   "fakeClusterID",
				Name: "resourceName",
				Type: "Microsoft.RedHatOpenShift/openshiftClusters",
			},
		},
		{
			name: "operation exists in db, but no cluster - final result is available with no content",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name: "operation and cluster exist in db - final result is not yet available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{
						AsyncOperationID: mockOpID,
					}, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusAccepted,
		},
		{
			name: "operation not found in db",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: : The entity was not found.`,
		},
		{
			name: "internal error",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
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

			dbasyncoperations := mock_database.NewMockAsyncOperations(ti.controller)
			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)

			tt.mocks(dbopenshiftclusters, dbasyncoperations)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, dbasyncoperations, dbopenshiftclusters, nil, api.APIs, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			referer := fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationresults/%s", mockSubID, ti.env.Location(), mockOpID)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/operationresults/%s?api-version=2020-04-30", mockSubID, ti.env.Location(), mockOpID),
				http.Header{
					"Content-Type": []string{"application/json"},
					"Referer":      []string{referer},
				}, nil)
			if err != nil {
				t.Fatal(err)
			}

			location := resp.Header.Get("Location")
			if tt.wantAsync {
				if location != referer {
					t.Error(location)
				}
			} else {
				if location != "" {
					t.Error(location)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
