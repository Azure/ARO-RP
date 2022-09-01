package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestListInstallVersions(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *v20220904.InstallVersions
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "default InstallStream version",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20220904.InstallVersions{
				v20220904.InstallVersion((v20220904.InstallVersion)(version.InstallStream.Version.String())),
			},
		},
		{
			name: "only enabled versions",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				})
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Version: "4.10.20",
							Enabled: false,
						},
					}, &api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Version: "4.10.27",
							Enabled: true,
						},
					},
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Version: "4.11.5",
							Enabled: true,
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20220904.InstallVersions{
				"4.10.27",
				"4.11.5",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftVersions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, nil, nil, ti.openShiftVersionsDatabase, api.APIs, &noop.Noop{}, nil, nil, nil, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/listinstallversions?api-version=2022-09-04", mockSubID, ti.env.Location()),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			sort.Slice(*tt.wantResponse, func(i, j int) bool {
				return i < j
			})

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
