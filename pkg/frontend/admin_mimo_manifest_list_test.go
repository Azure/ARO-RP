package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestMIMOListManifests(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
		limit          int
		wantStatusCode int
		wantResponse   *admin.MaintenanceManifestList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "no entries",
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
			wantResponse: &admin.MaintenanceManifestList{
				MaintenanceManifests: []*admin.MaintenanceManifest{},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "single entry",
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
			wantResponse: &admin.MaintenanceManifestList{
				MaintenanceManifests: []*admin.MaintenanceManifest{
					{
						ID:                "07070707-0707-0707-0707-070707070001",
						MaintenanceTaskID: "exampletask",
						State:             admin.MaintenanceManifestStatePending,
						Priority:          0,
						RunAfter:          1,
						RunBefore:         1,
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:  "limit over",
			limit: 1,
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
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "exampleset2",
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          1,
						RunBefore:         1,
					},
				})
			},
			wantResponse: &admin.MaintenanceManifestList{
				NextLink: "https://mockrefererhost/?%24skipToken=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("FAKE1"))),
				MaintenanceManifests: []*admin.MaintenanceManifest{
					{
						ID:                "07070707-0707-0707-0707-070707070001",
						MaintenanceTaskID: "exampletask",
						State:             admin.MaintenanceManifestStatePending,
						Priority:          0,
						RunAfter:          1,
						RunBefore:         1,
					},
				},
			},
			wantStatusCode: http.StatusOK,
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
			name:           "missing cluster",
			wantError:      "404: NotFound: : cluster not found: 404 : ",
			wantStatusCode: http.StatusNotFound,
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

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			if tt.limit == 0 {
				tt.limit = 100
			}

			fmt.Printf("limit: %d", tt.limit)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin%s/maintenancemanifests?limit=%d", resourceID, tt.limit),
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
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
