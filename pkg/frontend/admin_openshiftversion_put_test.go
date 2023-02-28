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
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestOpenShiftVersionPut(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		body           *admin.OpenShiftVersion
		wantStatusCode int
		wantResponse   *admin.OpenShiftVersion
		wantError      string
		wantDocuments  []*api.OpenShiftVersionDocument
	}

	for _, tt := range []*test{
		{
			name: "updating known version",
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
				)
			},
			body: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.0",
					Enabled:           false,
					OpenShiftPullspec: "c:c/d",
					InstallerPullspec: "d:d/e",
				},
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.0",
					Enabled:           false,
					OpenShiftPullspec: "c:c/d",
					InstallerPullspec: "d:d/e",
				},
			},
			wantDocuments: []*api.OpenShiftVersionDocument{
				{
					ID: "07070707-0707-0707-0707-070707070001",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           "4.10.0",
							Enabled:           false,
							OpenShiftPullspec: "c:c/d",
							InstallerPullspec: "d:d/e",
						},
					},
				},
			},
		},
		{
			name: "creating new version",
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
				)
			},
			body: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.1",
					Enabled:           true,
					OpenShiftPullspec: "f:f/g",
					InstallerPullspec: "g:g/h",
				},
			},
			wantStatusCode: http.StatusCreated,
			wantResponse: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.1",
					Enabled:           true,
					OpenShiftPullspec: "f:f/g",
					InstallerPullspec: "g:g/h",
				},
			},
			wantDocuments: []*api.OpenShiftVersionDocument{
				{
					ID: "07070707-0707-0707-0707-070707070001",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           "4.10.0",
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
						},
					},
				},
				{
					ID: "07070707-0707-0707-0707-070707070002",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           "4.10.1",
							Enabled:           true,
							OpenShiftPullspec: "f:f/g",
							InstallerPullspec: "g:g/h",
						},
					},
				},
			},
		},
		{
			name: "updating known version requires installer pullspec",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version:           "4.10.0",
								Enabled:           true,
								OpenShiftPullspec: "a:a/b",
								InstallerPullspec: "d:d/e",
							},
						},
					},
				)
			},
			body: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.0",
					Enabled:           true,
					OpenShiftPullspec: "c:c/d",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.installerPullspec: Must be provided",
			wantDocuments: []*api.OpenShiftVersionDocument{
				{
					ID: "07070707-0707-0707-0707-070707070001",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           "4.10.0",
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
							InstallerPullspec: "d:d/e",
						},
					},
				},
			},
		},
		{
			name: "updating known version requires openshift pullspec",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version:           "4.10.0",
								Enabled:           true,
								OpenShiftPullspec: "a:a/b",
								InstallerPullspec: "d:d/e",
							},
						},
					},
				)
			},
			body: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           "4.10.0",
					Enabled:           true,
					InstallerPullspec: "c:c/d",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.openShiftPullspec: Must be provided",
			wantDocuments: []*api.OpenShiftVersionDocument{
				{
					ID: "07070707-0707-0707-0707-070707070001",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           "4.10.0",
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
							InstallerPullspec: "d:d/e",
						},
					},
				},
			},
		},
		{
			name:           "creating new version needs body",
			fixture:        func(f *testdatabase.Fixture) {},
			body:           &admin.OpenShiftVersion{},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.version: Must be provided",
			wantDocuments:  []*api.OpenShiftVersionDocument{},
		},
		{
			name: "can not disable default install version",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version:           version.DefaultInstallStream.Version.String(),
								Enabled:           true,
								OpenShiftPullspec: "a:a/b",
							},
						},
					},
				)
			},
			body: &admin.OpenShiftVersion{
				Properties: admin.OpenShiftVersionProperties{
					Version:           version.DefaultInstallStream.Version.String(),
					Enabled:           false,
					OpenShiftPullspec: "c:c/d",
					InstallerPullspec: "d:d/e",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.enabled: You cannot disable the default installation version.",
			wantDocuments: []*api.OpenShiftVersionDocument{
				{
					ID: "07070707-0707-0707-0707-070707070001",
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version:           version.DefaultInstallStream.Version.String(),
							Enabled:           true,
							OpenShiftPullspec: "a:a/b",
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

			resp, b, err := ti.request(http.MethodPut, "https://server/admin/versions",
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

			if tt.wantDocuments != nil {
				ti.checker.AddOpenShiftVersionDocuments(tt.wantDocuments...)
				for _, err := range ti.checker.CheckOpenShiftVersions(ti.openShiftVersionsClient) {
					t.Error(err)
				}
			}
		})
	}
}
