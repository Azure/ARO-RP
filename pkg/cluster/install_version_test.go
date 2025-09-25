package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
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
			name: "select nonexistent version",
			f: func(f *testdatabase.Fixture) {
				f.AddOpenShiftVersionDocuments(
					&api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version: "4.10.20",
								Enabled: true,
							},
						},
					}, &api.OpenShiftVersionDocument{
						OpenShiftVersion: &api.OpenShiftVersion{
							Properties: api.OpenShiftVersionProperties{
								Version: "4.10.27",
								Enabled: true,
							},
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
				openShiftClusterDocumentVersioner: new(openShiftClusterDocumentVersionerService),
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
				assert.Equal(t, tt.want.Properties.Version, version.Properties.Version, "Version does not match")
				assert.Equal(t, tt.want.Properties.OpenShiftPullspec, version.Properties.OpenShiftPullspec, "properties.OpenShiftPullspec does not match")
				assert.Equal(t, tt.want.Properties.InstallerPullspec, version.Properties.InstallerPullspec, "properties.InstallerPullspec does not match")
			}
		})
	}
}

func TestGetOpenShiftVersionFromVersionWithArbitraryVersions(t *testing.T) {
	const testACRDomain = "acrdomain.io"

	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name                     string
		version                  string
		installViaHive           bool
		arbitraryVersionsEnabled bool
		isLocalDevelopmentMode   bool
		wantErrString            string
		wantInstallerPullspec    string
		wantOpenShiftPullspec    string
		wantVersion              string
	}{
		{
			name:                     "arbitrary version with feature flag enabled - traditional install",
			version:                  "4.15.0-custom.build.123",
			installViaHive:           false,
			arbitraryVersionsEnabled: true,
			wantInstallerPullspec:    "acrdomain.io/aro-installer:4.15",
			wantOpenShiftPullspec:    "quay.io/openshift-release-dev/ocp-release:4.15.0-custom.build.123",
			wantVersion:              "4.15.0-custom.build.123",
		},
		{
			name:                     "arbitrary version with feature flag enabled - hive install",
			version:                  "4.16.5-dev.branch.456",
			installViaHive:           true,
			arbitraryVersionsEnabled: true,
			wantInstallerPullspec:    "acrdomain.io/aro-installer:4.16",
			wantOpenShiftPullspec:    "acrdomain.io/ocp-release:4.16.5-dev.branch.456",
			wantVersion:              "4.16.5-dev.branch.456",
		},
		{
			name:                     "arbitrary version with feature flag disabled",
			version:                  "4.15.0-custom.build.123",
			installViaHive:           false,
			arbitraryVersionsEnabled: false,
			wantErrString:            "400: InvalidParameter: properties.clusterProfile.version: The requested OpenShift version '4.15.0-custom.build.123' is not supported.",
		},
		{
			name:                     "invalid semantic version with feature flag enabled",
			version:                  "not-a-valid-version",
			installViaHive:           false,
			arbitraryVersionsEnabled: true,
			wantErrString:            "400: InvalidParameter: properties.clusterProfile.version: The requested OpenShift version 'not-a-valid-version' is not a valid semantic version.",
		},
		{
			name:                     "version in CosmosDB takes precedence over arbitrary versions",
			version:                  "4.14.38", // This version exists in our test fixture below
			installViaHive:           false,
			arbitraryVersionsEnabled: true,
			wantInstallerPullspec:    "installerimage:1.0.0", // From fixture
			wantOpenShiftPullspec:    "openshiftimage:1.0.0", // From fixture
			wantVersion:              "4.14.38",
		},
		{
			name:                     "prerelease arbitrary version",
			version:                  "4.17.0-0.nightly-2024-12-01-123456",
			installViaHive:           true,
			arbitraryVersionsEnabled: true,
			wantInstallerPullspec:    "acrdomain.io/aro-installer:4.17",
			wantOpenShiftPullspec:    "acrdomain.io/ocp-release:4.17.0-0.nightly-2024-12-01-123456",
			wantVersion:              "4.17.0-0.nightly-2024-12-01-123456",
		},
		{
			name:                   "arbitrary version in development mode - traditional install",
			version:                "4.18.0-dev.custom.build",
			installViaHive:         false,
			isLocalDevelopmentMode: true,
			wantInstallerPullspec:  "acrdomain.io/aro-installer:4.18",
			wantOpenShiftPullspec:  "quay.io/openshift-release-dev/ocp-release:4.18.0-dev.custom.build",
			wantVersion:            "4.18.0-dev.custom.build",
		},
		{
			name:                   "arbitrary version in development mode - hive install",
			version:                "4.19.0-0.nightly-dev-123",
			installViaHive:         true,
			isLocalDevelopmentMode: true,
			wantInstallerPullspec:  "acrdomain.io/aro-installer:4.19",
			wantOpenShiftPullspec:  "acrdomain.io/ocp-release:4.19.0-0.nightly-dev-123",
			wantVersion:            "4.19.0-0.nightly-dev-123",
		},
		{
			name:                   "invalid semantic version in development mode",
			version:                "not-a-valid-version",
			isLocalDevelopmentMode: true,
			wantErrString:          "400: InvalidParameter: properties.clusterProfile.version: The requested OpenShift version 'not-a-valid-version' is not a valid semantic version.",
		},
		{
			name:                     "both AFEC flag and dev mode enabled - should work",
			version:                  "4.20.0-combined.test",
			installViaHive:           false,
			arbitraryVersionsEnabled: true,
			isLocalDevelopmentMode:   true,
			wantInstallerPullspec:    "acrdomain.io/aro-installer:4.20",
			wantOpenShiftPullspec:    "quay.io/openshift-release-dev/ocp-release:4.20.0-combined.test",
			wantVersion:              "4.20.0-combined.test",
		},
		{
			name:                   "dev mode overrides normal validation for arbitrary versions",
			version:                "4.21.0-0.dev-override-123",
			isLocalDevelopmentMode: true,
			wantInstallerPullspec:  "acrdomain.io/aro-installer:4.21",
			wantOpenShiftPullspec:  "quay.io/openshift-release-dev/ocp-release:4.21.0-0.dev-override-123",
			wantVersion:            "4.21.0-0.dev-override-123",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)

			tlc := testliveconfig.NewTestLiveConfig(false, false)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return(testACRDomain)
			_env.EXPECT().LiveConfig().AnyTimes().Return(tlc)
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(tt.isLocalDevelopmentMode)

			uuidGen := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.OPENSHIFT_VERSIONS)
			dbOpenShiftVersions, _ := testdatabase.NewFakeOpenShiftVersions(uuidGen)

			// Add a known version to CosmosDB for precedence testing
			dbOpenShiftVersions.Create(ctx, &api.OpenShiftVersionDocument{
				ID: "1",
				OpenShiftVersion: &api.OpenShiftVersion{
					Properties: api.OpenShiftVersionProperties{
						Version:           "4.14.38",
						OpenShiftPullspec: "openshiftimage:1.0.0",
						InstallerPullspec: "installerimage:1.0.0",
						Enabled:           true,
					},
				},
			})

			// Create subscription document with optional feature flag
			subscription := &api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						RegisteredFeatures: []api.RegisteredFeatureProfile{},
					},
				},
			}

			// Add arbitrary versions feature flag if enabled for this test
			if tt.arbitraryVersionsEnabled {
				subscription.Subscription.Properties.RegisteredFeatures = append(
					subscription.Subscription.Properties.RegisteredFeatures,
					api.RegisteredFeatureProfile{
						Name:  api.FeatureFlagArbitraryVersions,
						State: "Registered",
					},
				)
			}

			m := &manager{
				env:                 _env,
				dbOpenShiftVersions: dbOpenShiftVersions,
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								Version: tt.version,
							},
						},
					},
				},
				subscriptionDoc:                   subscription,
				installViaHive:                    tt.installViaHive,
				openShiftClusterDocumentVersioner: &openShiftClusterDocumentVersionerService{},
			}

			version, err := m.openShiftVersionFromVersion(ctx)

			if len(tt.wantErrString) > 0 {
				assert.Equal(t, tt.wantErrString, err.Error(), "Unexpected error message")
				assert.Nil(t, version, "Expected nil version on error")
			} else {
				assert.NoError(t, err, "Unexpected error")
				assert.NotNil(t, version, "Expected non-nil version")
				assert.Equal(t, tt.wantVersion, version.Properties.Version, "Unexpected version")
				assert.Equal(t, tt.wantInstallerPullspec, version.Properties.InstallerPullspec, "Unexpected installer pullspec")
				assert.Equal(t, tt.wantOpenShiftPullspec, version.Properties.OpenShiftPullspec, "Unexpected OpenShift pullspec")
				assert.True(t, version.Properties.Enabled, "Expected version to be enabled")
			}
		})
	}
}
