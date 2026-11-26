package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

func TestRefererWithMaintenanceManifestListOptions(t *testing.T) {
	before := int64(200)
	got, err := refererWithMaintenanceManifestListOptions("https://mockrefererhost/", database.MaintenanceManifestListOptions{
		After:  100,
		Before: &before,
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("after") != "100" {
		t.Errorf("after=%q, want 100", u.Query().Get("after"))
	}
	if u.Query().Get("before") != "200" {
		t.Errorf("before=%q, want 200", u.Query().Get("before"))
	}

	got, err = refererWithMaintenanceManifestListOptions("https://mockrefererhost/?before=999", database.MaintenanceManifestListOptions{
		After: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("after") != "100" {
		t.Errorf("after=%q, want 100", u.Query().Get("after"))
	}
	if u.Query().Has("before") {
		t.Errorf("before=%q, want omitted", u.Query().Get("before"))
	}
}

func TestMIMOListManifestsPaginationPreservesDefaultAfterWithAdvancingClock(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	ctx := context.Background()

	t0 := time.Unix(1_700_000_000, 0)
	current := t0
	clock := func() time.Time { return current }

	defaultAfter := t0.Add(-maintenanceManifestListDefaultLookback).Unix()
	boundaryAfter := defaultAfter
	recentAfter := t0.Add(-1 * time.Hour).Unix()
	oldAfter := t0.Add(-31 * 24 * time.Hour).Unix()

	ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions().WithMaintenanceManifests(clock)
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
		// Sorted by ID: old (...001), boundary (...002), recent (...003)
		f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
			ClusterResourceID: strings.ToLower(resourceID),
			MaintenanceManifest: api.MaintenanceManifest{
				MaintenanceTaskID: "oldtask",
				State:             api.MaintenanceManifestStateCompleted,
				RunAfter:          oldAfter,
				RunBefore:         oldAfter + 3600,
			},
		})
		f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
			ClusterResourceID: strings.ToLower(resourceID),
			MaintenanceManifest: api.MaintenanceManifest{
				MaintenanceTaskID: "boundarytask",
				State:             api.MaintenanceManifestStateCompleted,
				RunAfter:          boundaryAfter,
				RunBefore:         boundaryAfter + 3600,
			},
		})
		f.AddMaintenanceManifestDocuments(&api.MaintenanceManifestDocument{
			ClusterResourceID: strings.ToLower(resourceID),
			MaintenanceManifest: api.MaintenanceManifest{
				MaintenanceTaskID: "recenttask",
				State:             api.MaintenanceManifestStatePending,
				RunAfter:          recentAfter,
				RunBefore:         recentAfter + 3600,
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
	f.now = clock

	go f.Run(ctx, nil, nil)

	// Page 1: default window, limit 1. Expect boundary manifest and a nextLink that pins after.
	resp, b, err := ti.request(http.MethodGet,
		fmt.Sprintf("https://server/admin%s/maintenancemanifests?limit=1", resourceID),
		http.Header{"Referer": []string{"https://mockrefererhost/"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page1 status=%d body=%s", resp.StatusCode, string(b))
	}

	var page1 admin.MaintenanceManifestList
	if err := json.Unmarshal(b, &page1); err != nil {
		t.Fatal(err)
	}
	if len(page1.MaintenanceManifests) != 1 || page1.MaintenanceManifests[0].MaintenanceTaskID != "boundarytask" {
		t.Fatalf("page1 unexpected manifests: %+v", page1.MaintenanceManifests)
	}
	if page1.NextLink == "" {
		t.Fatal("page1 missing nextLink")
	}

	nextURL, err := url.Parse(page1.NextLink)
	if err != nil {
		t.Fatal(err)
	}
	pinnedAfter := nextURL.Query().Get("after")
	if pinnedAfter != strconv.FormatInt(defaultAfter, 10) {
		t.Fatalf("nextLink after=%q, want %d", pinnedAfter, defaultAfter)
	}
	skipToken := nextURL.Query().Get("$skipToken")
	if skipToken == "" {
		t.Fatal("nextLink missing $skipToken")
	}

	// Advance clock so a recomputed default after would exclude the boundary manifest.
	current = t0.Add(48 * time.Hour)

	// Page 2 using the pinned after from nextLink must still succeed and return the recent manifest.
	resp, b, err = ti.request(http.MethodGet,
		fmt.Sprintf("https://server/admin%s/maintenancemanifests?limit=1&after=%s&%%24skipToken=%s",
			resourceID, pinnedAfter, url.QueryEscape(skipToken)),
		http.Header{"Referer": []string{"https://mockrefererhost/"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page2 status=%d body=%s", resp.StatusCode, string(b))
	}

	var page2 admin.MaintenanceManifestList
	if err := json.Unmarshal(b, &page2); err != nil {
		t.Fatal(err)
	}
	if len(page2.MaintenanceManifests) != 1 || page2.MaintenanceManifests[0].MaintenanceTaskID != "recenttask" {
		t.Fatalf("page2 unexpected manifests: %+v", page2.MaintenanceManifests)
	}

	// Without the pinned after, the drifted default window excludes the boundary item from page 1's set.
	driftedAfter := current.Add(-maintenanceManifestListDefaultLookback).Unix()
	if driftedAfter <= boundaryAfter {
		t.Fatalf("test setup error: drifted after %d should be > boundary %d", driftedAfter, boundaryAfter)
	}
}
