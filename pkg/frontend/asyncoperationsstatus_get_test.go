package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestGetAsyncOperationsStatus(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockOpID := "11111111-1111-1111-1111-111111111111"
	mockOpStartTime := time.Now().Add(-time.Hour).UTC()
	mockOpEndTime := time.Now().Add(-time.Minute).UTC()

	type test struct {
		name           string
		fixture        func(*testdatabase.Fixture)
		dbError        error
		wantStatusCode int
		wantResponse   *api.AsyncOperation
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "operation and cluster exist in db - final result is available",
			fixture: func(f *testdatabase.Fixture) {
				f.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					ID:                  mockOpID,
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resource1")),
					AsyncOperation: &api.AsyncOperation{
						ID:                       "fakeoppath",
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
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeoppath",
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
			fixture: func(f *testdatabase.Fixture) {
				f.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					ID:                  mockOpID,
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resource1")),
					AsyncOperation: &api.AsyncOperation{
						ID:                       "fakeoppath",
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
				})

				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:              strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resource1")),
					AsyncOperationID: mockOpID,
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeoppath",
				Name:              mockOpID,
				ProvisioningState: api.ProvisioningStateUpdating,
				StartTime:         mockOpStartTime,
			},
		},
		{
			name:           "operation not found in db",
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: : The entity was not found.`,
		},
		{
			name: "operation exists in db, but no cluster",
			fixture: func(f *testdatabase.Fixture) {
				f.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					ID:                  mockOpID,
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resource1")),
					AsyncOperation: &api.AsyncOperation{
						ID:                       "fakeoppath",
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
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.AsyncOperation{
				ID:                "fakeoppath",
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
			name: "operation exists in db, but no subscription match",
			fixture: func(f *testdatabase.Fixture) {
				f.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					ID:                  mockOpID,
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath("33333333-3333-3333-3333-333333333333", "resource1")),
					AsyncOperation: &api.AsyncOperation{
						ID:                       "fakeoppath",
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
				})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: : The entity was not found.`,
		},
		{
			name:           "internal error",
			dbError:        &cosmosdb.Error{Code: "500", Message: "blorb"},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithAsyncOperations().WithOpenShiftClusters()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			if tt.dbError != nil {
				ti.openShiftClustersClient.SetError(tt.dbError)
				ti.asyncOperationsClient.SetError(tt.dbError)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
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
