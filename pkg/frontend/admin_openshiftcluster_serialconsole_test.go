package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	mockSubID    = "00000000-0000-0000-0000-000000000000"
	mockTenantID = "00000000-0000-0000-0000-000000000000"
)

func databaseFixture(f *testdatabase.Fixture) {
	f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
		Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
				},
			},
		},
	})

	f.AddSubscriptionDocuments(&api.SubscriptionDocument{
		ID: mockSubID,
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
	}{
		{
			name:       "valid request returns console output",
			vmName:     "master-0",
			resourceID: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
			fixture:    databaseFixture,
			mocks: func(mockActions *mock_adminactions.MockAzureActions) {
				mockActions.EXPECT().
					VMSerialConsole(gomock.Any(), gomock.Any(), "master-0", gomock.Any()).
					DoAndReturn(func(ctx context.Context, log *logrus.Entry, vmName string, w io.Writer) error {
						_, err := w.Write([]byte("console output"))
						return err
					})
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:       "invalid vm name returns error",
			vmName:     "invalid-vm",
			resourceID: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
			fixture:    databaseFixture,
			mocks: func(mockActions *mock_adminactions.MockAzureActions) {
				mockActions.EXPECT().
					VMSerialConsole(gomock.Any(), gomock.Any(), "invalid-vm", gomock.Any()).
					Return(&api.CloudError{
						StatusCode: http.StatusBadRequest,
						CloudErrorBody: &api.CloudErrorBody{
							Message: "invalid VM name",
						},
					})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "invalid VM name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			a := mock_adminactions.NewMockAzureActions(ti.controller)
			if tt.mocks != nil {
				tt.mocks(a)
			}

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(context.Background(), ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error) {
				return a, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/admin"+tt.resourceID+"/serialconsole?vmName="+tt.vmName, nil)
			r = r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger())))

			f.getAdminOpenShiftClusterSerialConsole(w, r)

			response := w.Result()
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
