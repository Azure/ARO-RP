package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestGetAsyncOperationsStatus(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockClusterDocKey := "22222222-2222-2222-2222-222222222222"
	mockOpID := "11111111-1111-1111-1111-111111111111"
	mockOpStartTime := time.Now().Add(-time.Hour).UTC()
	mockOpEndTime := time.Now().Add(-time.Minute).UTC()

	type test struct {
		name           string
		mocks          func(*mock_database.MockOpenShiftClusters, *mock_database.MockAsyncOperations)
		wantStatusCode int
		wantResponse   *api.AsyncOperation
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "operation and cluster exist in db - final result is available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateUpdating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeOpPath",
				Name:              mockOpID,
				ProvisioningState: api.ProvisioningStateFailed,
				StartTime:         mockOpStartTime,
				EndTime:           &mockOpEndTime,
				Error: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "Some error.",
				},
			},
		},
		{
			name: "operation and cluster exist in db - final result is not yet available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateUpdating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{
						AsyncOperationID: mockOpID,
					}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeOpPath",
				Name:              mockOpID,
				ProvisioningState: api.ProvisioningStateUpdating,
				StartTime:         mockOpStartTime,
			},
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
			name: "operation exists in db, but no cluster",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateCreating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeOpPath",
				Name:              mockOpID,
				ProvisioningState: api.ProvisioningStateFailed,
				StartTime:         mockOpStartTime,
				EndTime:           &mockOpEndTime,
				Error: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "Some error.",
				},
			},
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

			asyncOperations := mock_database.NewMockAsyncOperations(ti.controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(ti.controller)

			tt.mocks(openshiftClusters, asyncOperations)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, asyncOperations, openshiftClusters, nil, api.APIs, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/operationsstatus/%s?api-version=2020-04-30", mockSubID, ti.env.Location(), mockOpID),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
