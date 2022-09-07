package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

func TestGetOpenShiftVersionFromVersion(t *testing.T) {
	const testACRDomain = "acrdomain.io"

	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name          string
		f             func(f *testdatabase.Fixture)
		m             manager
		wantErrString string
		want          *api.OpenShiftVersion
	}{
		{
			name: "no versions gets default version",
			f:    func(f *testdatabase.Fixture) {},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								Version: version.InstallStream.Version.String(),
							},
						},
					},
				},
			},
			wantErrString: "",
			want: &api.OpenShiftVersion{
				Version:           version.InstallStream.Version.String(),
				OpenShiftPullspec: version.InstallStream.PullSpec,
				InstallerPullspec: fmt.Sprintf("%s/aro-installer:release-%d.%d", testACRDomain, version.InstallStream.Version.V[0], version.InstallStream.Version.V[1]),
			},
		},
		{
			name: "select nonexistent version",
			f: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Version: "4.10.20",
							Enabled: true,
						},
					}, &api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Version: "4.10.27",
							Enabled: true,
						},
					},
				)
			},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								Version: "4.11.5",
							},
						},
					},
				},
			},
			wantErrString: "400: InvalidParameter: properties.clusterProfile.version: The requested OpenShift version '4.11.5' is not supported.",
			want:          nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			tlc := testliveconfig.NewTestLiveConfig(false, false)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return(testACRDomain)
			_env.EXPECT().LiveConfig().AnyTimes().Return(tlc)
			tt.m.env = _env

			uuidGen := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.OPENSHIFT_VERSIONS)
			dbOpenShiftVersions, _ := testdatabase.NewFakeOpenShiftVersions(uuidGen)
			fixture := testdatabase.NewFixture().WithOpenShiftVersions(dbOpenShiftVersions, uuidGen)

			if tt.f != nil {
				tt.f(fixture)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			tt.m.dbOpenShiftVersions = dbOpenShiftVersions

			version, err := tt.m.openShiftVersionFromVersion(ctx)

			if len(tt.wantErrString) > 0 {
				assert.Equal(t, tt.wantErrString, err.Error(), "Unexpected error exception")
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.Version, version.Version, "Version does not match")
				assert.Equal(t, tt.want.OpenShiftPullspec, version.OpenShiftPullspec, "OpenShiftPullspec does not match")
				assert.Equal(t, tt.want.InstallerPullspec, version.InstallerPullspec, "InstallerPullspec does not match")
			}
		})
	}
}
