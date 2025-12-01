package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

const (
	mockSubscriptionID = "00000000-0000-0000-0000-000000000000"
	mockTenantID       = "00000000-0000-0000-0000-000000000000"
)

func databaseFixture(dbFixture *testdatabase.Fixture) {
	dbFixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubscriptionID),
				},
			},
		},
	})

	dbFixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
		ID: mockSubscriptionID,
		Subscription: &api.Subscription{
			State: api.SubscriptionStateRegistered,
			Properties: &api.SubscriptionProperties{
				TenantID: mockTenantID,
			},
		},
	})
}

func TestGetAdminOpenShiftClusterSerialConsole(t *testing.T) {
	tests := []struct {
		name           string
		vmName         string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		mocks          func(*mock_adminactions.MockAzureActions)
		wantStatusCode int
		wantError      string
		wantResponse   string
	}{
		{
			name:       "valid request returns console output",
			vmName:     "master-0",
			resourceID: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
			fixture:    databaseFixture,
			mocks: func(mockActions *mock_adminactions.MockAzureActions) {
				mockActions.EXPECT().
					ResourceGroupHasVM(gomock.Any(), "master-0").
					Return(true, nil)
				mockActions.EXPECT().
					VMSerialConsole(gomock.Any(), gomock.Any(), "master-0", gomock.Any()).
					DoAndReturn(func(ctx context.Context, log *logrus.Entry, vmName string, writer io.Writer) error {
						_, err := writer.Write([]byte("console output"))
						return err
					})
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   "console output",
		},
		{
			name:           "missing vm name parameter returns error",
			vmName:         "",
			resourceID:     strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
			fixture:        databaseFixture,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "The vmName parameter is required",
		},
		{
			name:           "invalid characters in vm name returns validation error",
			vmName:         "master#0",
			resourceID:     strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
			fixture:        databaseFixture,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "The provided vmName 'master#0' is invalid",
		},
		{
			name:       "non-existent vm returns not found error",
			vmName:     "nonexistent-vm",
			resourceID: strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "resourceName")),
			fixture:    databaseFixture,
			mocks: func(mockActions *mock_adminactions.MockAzureActions) {
				mockActions.EXPECT().
					ResourceGroupHasVM(gomock.Any(), "nonexistent-vm").
					Return(false, nil)
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "The VirtualMachine 'nonexistent-vm' under resource group 'test-cluster' was not found.",
		},
		{
			name:           "cluster not found returns error",
			vmName:         "master-0",
			resourceID:     strings.ToLower(testdatabase.GetResourcePath(mockSubscriptionID, "nonexistent-cluster")),
			fixture:        databaseFixture,
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testInfra := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer testInfra.done()

			a := mock_adminactions.NewMockAzureActions(testInfra.controller)
			if tt.mocks != nil {
				tt.mocks(a)
			}

			err := testInfra.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			frontend, err := NewFrontend(context.Background(), testInfra.auditLog, testInfra.log, testInfra.otelAudit, testInfra.env, testInfra.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/admin"+tt.resourceID+"/serialconsole?vmName="+tt.vmName, nil)

			ctx := context.WithValue(request.Context(), middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger()))
			ctx = context.WithValue(ctx, chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{
					Keys:   []string{"resourceType", "resourceName", "resourceGroupName"},
					Values: []string{"openshiftcluster", "resourceName", "test-cluster"},
				},
			})
			request = request.WithContext(ctx)

			frontend.getAdminOpenShiftClusterSerialConsole(recorder, request)

			response := recorder.Result()
			require.Equal(t, tt.wantStatusCode, response.StatusCode)

			if tt.wantError != "" {
				body := make(map[string]interface{})
				err := json.NewDecoder(response.Body).Decode(&body)
				require.NoError(t, err)
				require.Contains(t, body["error"].(map[string]interface{})["message"], tt.wantError)
			}
		})
	}
}
