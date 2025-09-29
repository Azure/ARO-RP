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
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestMIMOCreateManifest(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
		body           *admin.MaintenanceManifest
		wantStatusCode int
		wantResponse   *admin.MaintenanceManifest
		wantResult     func(f *testdatabase.Checker)
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "no cluster",
			wantError:      "404: NotFound: : cluster not found: 404 : ",
			fixtures:       func(f *testdatabase.Fixture) {},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "cluster being deleted",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateDeleting,
						},
					},
				})
			},
			wantError:      "404: NotFound: : cluster being deleted",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "invalid",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
			},
			body:           &admin.MaintenanceManifest{},
			wantError:      "400: InvalidParameter: maintenanceTaskID: Must be provided",
			wantStatusCode: http.StatusBadRequest,
		},

		{
			name: "good",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
			},
			body: &admin.MaintenanceManifest{
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceManifestStatePending,
				RunAfter:          1,
				RunBefore:         1,
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifest{
				ID:                "07070707-0707-0707-0707-070707070001",
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceManifestStatePending,
				RunAfter:          1,
				RunBefore:         1,
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "default set to pending",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
			},
			body: &admin.MaintenanceManifest{
				MaintenanceTaskID: "exampletask",
				RunAfter:          1,
				RunBefore:         1,
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifest{
				ID:                "07070707-0707-0707-0707-070707070001",
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceManifestStatePending,
				RunAfter:          1,
				RunBefore:         1,
			},
			wantStatusCode: http.StatusCreated,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			now := func() time.Time { return time.Unix(1000, 0) }

			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions().WithMaintenanceManifests(now)
			defer ti.done()

			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err := ti.buildFixtures(tt.fixtures)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantResult != nil {
				tt.wantResult(ti.checker)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.now = now

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPut,
				fmt.Sprintf("https://server/admin%s/maintenancemanifests", resourceID),
				http.Header{
					"Content-Type": []string{"application/json"},
				}, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			for _, err := range ti.checker.CheckMaintenanceManifests(ti.maintenanceManifestsClient) {
				t.Error(err)
			}
		})
	}
}
