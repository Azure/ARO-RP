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
	"github.com/Azure/ARO-RP/pkg/mimo"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminMdsdCertificateRenew(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/reesourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1000, 0) }

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
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: mimo.MDSD_CERT_ROTATION_ID,
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifest{
				ID:                "07070707-0707-0707-0707-070707070001",
				MaintenanceTaskID: mimo.MDSD_CERT_ROTATION_ID,
				State:             admin.MaintenanceManifestStatePending,
				RunAfter:          int(now().Unix()),
				RunBefore:         int(now().Add(time.Hour * 7 * 24).Unix()),
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
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ID:                "07070707-0707-0707-0707-070707070001",
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: mimo.MDSD_CERT_ROTATION_ID,
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifest{
				ID:                "07070707-0707-0707-0707-070707070001",
				MaintenanceTaskID: mimo.MDSD_CERT_ROTATION_ID,
				State:             admin.MaintenanceManifestStatePending,
				RunAfter:          int(now().Unix()),
				RunBefore:         int(now().Add(time.Hour * 7 * 24).Unix()),
			},
			wantStatusCode: http.StatusCreated,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
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
				fmt.Sprintf("https://server/admin%s/mdsdcertificaterenew", resourceID), nil, nil)
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
