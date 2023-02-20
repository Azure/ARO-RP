package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestOpenShiftVersionList(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *admin.OpenShiftVersionList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "empty",
			fixture:        func(f *testdatabase.Fixture) {},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftVersionList{
				OpenShiftVersions: []*admin.OpenShiftVersion{},
			},
		},
		{
			name: "happy path",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version:           "4.10.0",
								Enabled:           true,
								OpenShiftPullspec: "a:a/b",
							},
						},
					},
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{

								Version:           "4.9.9",
								Enabled:           true,
								OpenShiftPullspec: "a:a/b",
								InstallerPullspec: "b:b/c",
							},
						},
					},
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{

								Version:           "4.10.1",
								Enabled:           false,
								OpenShiftPullspec: "a:a/b",
								InstallerPullspec: "b:b/c",
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftVersionList{
				OpenShiftVersions: []*admin.OpenShiftVersion{
					{
						Properties: admin.OpenShiftVersionProperties{
							Version:           "4.9.9",
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
							InstallerPullspec: "b:b/c",
						},
					},
					{
						Properties: admin.OpenShiftVersionProperties{
							Version:           "4.10.0",
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
						},
					},
					{
						Properties: admin.OpenShiftVersionProperties{
							Version:           "4.10.1",
							Enabled:           false,
							OpenShiftPullspec: "a:a/b",
							InstallerPullspec: "b:b/c",
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftVersions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, nil, nil, nil, ti.openShiftVersionsDatabase, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet, "https://server/admin/versions",
				nil, nil)
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
