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
	"github.com/Azure/ARO-RP/pkg/database"
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
		queryExtra     string
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
			queryExtra: "&after=0",
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
			queryExtra: "&after=0",
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
			queryExtra: "&after=0",
			wantResponse: &admin.MaintenanceManifestList{
				NextLink: "https://mockrefererhost/?%24skipToken=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("FAKE1"))) + "&after=0",
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
			name: "after and before filter",
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
						MaintenanceTaskID: "too-old",
						State:             api.MaintenanceManifestStateCompleted,
						RunAfter:          100,
						RunBefore:         200,
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "in-range",
						State:             api.MaintenanceManifestStateCompleted,
						RunAfter:          500,
						RunBefore:         600,
					},
				})
				f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
					ClusterResourceID: strings.ToLower(resourceID),
					MaintenanceManifest: api.MaintenanceManifest{
						MaintenanceTaskID: "too-new",
						State:             api.MaintenanceManifestStatePending,
						RunAfter:          900,
						RunBefore:         1000,
					},
				})
			},
			queryExtra: "&after=400&before=800",
			wantResponse: &admin.MaintenanceManifestList{
				MaintenanceManifests: []*admin.MaintenanceManifest{
					{
						ID:                "07070707-0707-0707-0707-070707070002",
						MaintenanceTaskID: "in-range",
						State:             admin.MaintenanceManifestStateCompleted,
						Priority:          0,
						RunAfter:          500,
						RunBefore:         600,
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid after",
			queryExtra:     "&after=not-a-number",
			wantError:      "400: InvalidParameter: after: after must be a unix epoch timestamp",
			wantStatusCode: http.StatusBadRequest,
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
		},
		{
			name:           "before less than after",
			queryExtra:     "&after=100&before=50",
			wantError:      "400: InvalidParameter: before: before must be greater than or equal to after",
			wantStatusCode: http.StatusBadRequest,
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

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			if tt.limit == 0 {
				tt.limit = 100
			}

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin%s/maintenancemanifests?limit=%d%s", resourceID, tt.limit, tt.queryExtra),
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

func TestMIMOListManifestsDefaultTimeWindow(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()
	now := time.Now()
	recentAfter := now.Add(-1 * time.Hour).Unix()
	recentBefore := now.Add(7 * 24 * time.Hour).Unix()

	ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions().WithMaintenanceManifests(func() time.Time { return now })
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

	err := ti.buildFixtures(func(f *testdatabase.Fixture) {
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
				MaintenanceTaskID: "oldtask",
				State:             api.MaintenanceManifestStateCompleted,
				RunAfter:          now.Add(-60 * 24 * time.Hour).Unix(),
				RunBefore:         now.Add(-53 * 24 * time.Hour).Unix(),
			},
		})
		f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
			ClusterResourceID: strings.ToLower(resourceID),
			MaintenanceManifest: api.MaintenanceManifest{
				MaintenanceTaskID: "recenttask",
				State:             api.MaintenanceManifestStatePending,
				RunAfter:          recentAfter,
				RunBefore:         recentBefore,
			},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	go f.Run(ctx, nil, nil)

	resp, b, err := ti.request(http.MethodGet,
		fmt.Sprintf("https://server/admin%s/maintenancemanifests?limit=100", resourceID),
		http.Header{
			"Referer": []string{"https://mockrefererhost/"},
		}, nil)
	if err != nil {
		t.Fatal(err)
	}

	wantResponse := &admin.MaintenanceManifestList{
		MaintenanceManifests: []*admin.MaintenanceManifest{
			{
				ID:                "07070707-0707-0707-0707-070707070002",
				MaintenanceTaskID: "recenttask",
				State:             admin.MaintenanceManifestStatePending,
				Priority:          0,
				RunAfter:          recentAfter,
				RunBefore:         recentBefore,
			},
		},
	}

	err = validateResponse(resp, b, http.StatusOK, "", wantResponse)
	if err != nil {
		t.Error(err)
	}
}

func TestParseMaintenanceManifestListOptions(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	defaultAfter := now.Add(-maintenanceManifestListDefaultLookback).Unix()

	for _, tt := range []struct {
		name      string
		rawQuery  string
		want      database.MaintenanceManifestListOptions
		wantError string
	}{
		{
			name: "defaults to last 30 days",
			want: database.MaintenanceManifestListOptions{After: defaultAfter},
		},
		{
			name:     "explicit after only",
			rawQuery: "after=100",
			want:     database.MaintenanceManifestListOptions{After: 100},
		},
		{
			name:     "before only uses default after",
			rawQuery: fmt.Sprintf("before=%d", now.Unix()),
			want: database.MaintenanceManifestListOptions{
				After:  defaultAfter,
				Before: new(now.Unix()),
			},
		},
		{
			name:     "after and before",
			rawQuery: "after=100&before=200",
			want: database.MaintenanceManifestListOptions{
				After:  100,
				Before: new(int64(200)),
			},
		},
		{
			name:      "invalid after",
			rawQuery:  "after=abc",
			wantError: "400: InvalidParameter: after: after must be a unix epoch timestamp",
		},
		{
			name:      "invalid before",
			rawQuery:  "before=abc",
			wantError: "400: InvalidParameter: before: before must be a unix epoch timestamp",
		},
		{
			name:      "before less than after",
			rawQuery:  "after=100&before=50",
			wantError: "400: InvalidParameter: before: before must be greater than or equal to after",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "https://example/?"+tt.rawQuery, nil)
			if err != nil {
				t.Fatal(err)
			}

			got, err := parseMaintenanceManifestListOptions(r, now)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.After != tt.want.After {
				t.Errorf("After=%d, want %d", got.After, tt.want.After)
			}
			switch {
			case tt.want.Before == nil && got.Before != nil:
				t.Errorf("Before=%v, want nil", *got.Before)
			case tt.want.Before != nil && got.Before == nil:
				t.Errorf("Before=nil, want %d", *tt.want.Before)
			case tt.want.Before != nil && got.Before != nil && *got.Before != *tt.want.Before:
				t.Errorf("Before=%d, want %d", *got.Before, *tt.want.Before)
			}
		})
	}
}
