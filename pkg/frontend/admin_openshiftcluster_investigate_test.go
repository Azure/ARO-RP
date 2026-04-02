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
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

const (
	mockInvestigateSubID    = "00000000-0000-0000-0000-000000000001"
	mockInvestigateTenantID = "00000000-0000-0000-0000-000000000002"
)

func investigateDatabaseFixture(dbFixture *testdatabase.Fixture) {
	dbFixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "testCluster")),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "testCluster")),
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockInvestigateSubID),
				},
				HiveProfile: api.HiveProfile{
					Namespace: "aro-00000000-0000-0000-0000-000000000001",
				},
				StorageSuffix: "abcdef",
			},
		},
	})

	dbFixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
		ID: mockInvestigateSubID,
		Subscription: &api.Subscription{
			State: api.SubscriptionStateRegistered,
			Properties: &api.SubscriptionProperties{
				TenantID: mockInvestigateTenantID,
			},
		},
	})
}

func investigateDatabaseFixtureNoHiveNamespace(dbFixture *testdatabase.Fixture) {
	dbFixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "testCluster")),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "testCluster")),
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockInvestigateSubID),
				},
			},
		},
	})

	dbFixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
		ID: mockInvestigateSubID,
		Subscription: &api.Subscription{
			State: api.SubscriptionStateRegistered,
			Properties: &api.SubscriptionProperties{
				TenantID: mockInvestigateTenantID,
			},
		},
	})
}

func TestPostAdminOpenShiftClusterInvestigate(t *testing.T) {
	resourceID := strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "testCluster"))

	tests := []struct {
		name           string
		body           string
		resourceID     string
		fixture        func(*testdatabase.Fixture)
		hiveEnabled    bool
		mocks          func(*mock_hive.MockClusterManager)
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "empty body returns bad request",
			body:           "",
			resourceID:     resourceID,
			fixture:        investigateDatabaseFixture,
			hiveEnabled:    true,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "The request body could not be parsed",
		},
		{
			name:           "empty question returns bad request",
			body:           `{"question":""}`,
			resourceID:     resourceID,
			fixture:        investigateDatabaseFixture,
			hiveEnabled:    true,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "The question parameter is required",
		},
		{
			name:           "question too long returns bad request",
			body:           `{"question":"` + strings.Repeat("a", 1001) + `"}`,
			resourceID:     resourceID,
			fixture:        investigateDatabaseFixture,
			hiveEnabled:    true,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "The question must not exceed 1000 characters",
		},
		{
			name:           "cluster not found returns not found",
			body:           `{"question":"what is wrong?"}`,
			resourceID:     strings.ToLower(testdatabase.GetResourcePath(mockInvestigateSubID, "nonexistent")),
			fixture:        investigateDatabaseFixture,
			hiveEnabled:    true,
			wantStatusCode: http.StatusNotFound,
			wantError:      "was not found",
		},
		{
			name:           "hive not enabled returns internal error",
			body:           `{"question":"what is wrong?"}`,
			resourceID:     resourceID,
			fixture:        investigateDatabaseFixture,
			hiveEnabled:    false,
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "hive is not enabled",
		},
		{
			name:           "no hive namespace returns internal error",
			body:           `{"question":"what is wrong?"}`,
			resourceID:     resourceID,
			fixture:        investigateDatabaseFixtureNoHiveNamespace,
			hiveEnabled:    true,
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "cluster does not have a Hive namespace configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			var f *frontend

			if tt.hiveEnabled {
				controller := gomock.NewController(t)
				defer controller.Finish()
				clusterManager := mock_hive.NewMockClusterManager(controller)
				if tt.mocks != nil {
					tt.mocks(clusterManager)
				}
				f, err = NewFrontend(context.Background(), ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, clusterManager, nil, nil, nil, nil, nil)
			} else {
				f, err = NewFrontend(context.Background(), ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			}
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			// The URL must include /investigate — the outer handler strips it via filepath.Dir.
			request := httptest.NewRequest(http.MethodPost, "/admin"+tt.resourceID+"/investigate", nil)

			ctx := context.WithValue(request.Context(), middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger()))
			ctx = context.WithValue(ctx, middleware.ContextKeyBody, []byte(tt.body))
			ctx = context.WithValue(ctx, chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{
					Keys:   []string{"resourceType", "resourceName", "resourceGroupName"},
					Values: []string{"openshiftcluster", "testCluster", "resourceGroup"},
				},
			})
			request = request.WithContext(ctx)

			f.postAdminOpenShiftClusterInvestigate(recorder, request)

			response := recorder.Result()
			require.Equal(t, tt.wantStatusCode, response.StatusCode)

			if tt.wantError != "" {
				bodyBytes, err := io.ReadAll(response.Body)
				require.NoError(t, err)

				var cloudErr struct {
					Error struct {
						Message string `json:"message"`
					} `json:"error"`
				}
				err = json.Unmarshal(bodyBytes, &cloudErr)
				require.NoError(t, err)
				require.Contains(t, cloudErr.Error.Message, tt.wantError)
			}
		})
	}
}
