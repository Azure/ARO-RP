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

func TestMIMOCancelManifest(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
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
			name: "no item",
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
			wantError:      "404: NotFound: : manifest not found: 404 : ",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "cancel pending",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStateCancelled,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifest{
				ID:                "07070707-0707-0707-0707-070707070001",
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceManifestStateCancelled,
				Priority:          0,
				RunAfter:          1,
				RunBefore:         1,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "cannot cancel failed",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStateFailed,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceManifestStateFailed,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantError:      "406: PropertyChangeNotAllowed: : cannot cancel task in state Failed",
			wantStatusCode: http.StatusNotAcceptable,
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

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/maintenancemanifests/07070707-0707-0707-0707-070707070001/cancel", resourceID),
				nil, nil)
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

type cancelResp struct {
	Cancelled int `json:"cancelled"`
	Errors    int `json:"errors"`
}

func TestMIMOCancelManifestBatch(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *cancelResp
		wantResult     func(f *testdatabase.Checker)
		wantError      string
		scheduleID     string
	}

	for _, tt := range []*test{
		{
			name:       "cancel pending",
			scheduleID: "testschedule",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampletask",
						CreatedBySchedule: "testschedule",
						State:             api.MaintenanceManifestStateCompleted,
						RunAfter:          1,
						RunBefore:         1,
					},
				},
					&api.MaintenanceManifestDocument{
						ClusterResourceID: strings.ToLower(resourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							MaintenanceTaskID: "exampletask",
							CreatedBySchedule: "testschedule",
							State:             api.MaintenanceManifestStatePending,
							RunAfter:          10000,
							RunBefore:         19999,
						},
					})
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(
					&api.MaintenanceManifestDocument{
						ID:                "07070707-0707-0707-0707-070707070001",
						ClusterResourceID: strings.ToLower(resourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							CreatedBySchedule: "testschedule",
							MaintenanceTaskID: "exampletask",
							State:             api.MaintenanceManifestStateCompleted,
							RunAfter:          1,
							RunBefore:         1,
						},
					},
					&api.MaintenanceManifestDocument{
						ID:                "07070707-0707-0707-0707-070707070002",
						ClusterResourceID: strings.ToLower(resourceID),
						MaintenanceManifest: api.MaintenanceManifest{
							CreatedBySchedule: "testschedule",
							MaintenanceTaskID: "exampletask",
							State:             api.MaintenanceManifestStateCancelled,
							RunAfter:          10000,
							RunBefore:         19999,
						},
					},
				)
			},
			wantResponse:   &cancelResp{Cancelled: 1},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "no schedule ID",
			wantError:      "400: InvalidParameter: scheduleID: Parameter is missing",
			wantStatusCode: http.StatusBadRequest,
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

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				"https://server/admin/maintenancemanifests/cancel?scheduleID="+tt.scheduleID,
				nil, nil)
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
