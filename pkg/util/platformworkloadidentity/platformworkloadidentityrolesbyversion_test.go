package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const fakeClusterSPObjectId = "00000000-0000-0000-0000-000000000000"

func TestNewPlatformWorkloadIdentityRolesByVersion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftCluster
		fixture func(f *testdatabase.Fixture)
		mocks   func(dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets)
		wantErr string
	}{
		{
			name: "Success - Exit the func for non MIWI clusters that has ServicePrincipalProfile",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						SPObjectID: fakeClusterSPObjectId,
					},
				},
			},
		},
		{
			name: "Success - Exit the func for non MIWI clusters that has no PlatformWorkloadIdentityProfile or ServicePrincipalProfile",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
		},
		{
			name: "Success - The role set document found for the cluster version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion:              "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{},
						},
					},
				})
			},
		},
		{
			name: "Failed - The role set documents listAll returns empty",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			wantErr: "400: InvalidParameter: : No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '4.14'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket",
		},
		{
			name: "Failed - The role set documents listAll is missing the requested version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.26",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion:              "4.15",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{},
						},
					},
				})
			},
			wantErr: "400: InvalidParameter: : No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '4.14'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket",
		},
	} {
		ctx := context.Background()

		uuidGen := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.OPENSHIFT_VERSIONS)

		dbPlatformWorkloadIdentityRoleSets, _ := testdatabase.NewFakePlatformWorkloadIdentityRoleSets(uuidGen)

		f := testdatabase.NewFixture().WithPlatformWorkloadIdentityRoleSets(dbPlatformWorkloadIdentityRoleSets, uuidGen)

		if tt.fixture != nil {
			tt.fixture(f)
		}
		err := f.Create()
		if err != nil {
			t.Fatal(err)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlatformWorkloadIdentityRolesByVersion(ctx, tt.oc, dbPlatformWorkloadIdentityRoleSets)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
