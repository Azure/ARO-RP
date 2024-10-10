package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNewPlatformWorkloadIdentityRolesByVersion(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		oc                     *api.OpenShiftCluster
		fixture                func(f *testdatabase.Fixture)
		mocks                  func(dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets)
		wantErr                string
		wantPlatformIdentities []string
	}{
		{
			name: "Fail - Exit the func for non MIWI clusters that has ServicePrincipalProfile",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						SPObjectID: "00000000-0000-0000-0000-000000000000",
					},
				},
			},
			wantErr: "PopulatePlatformWorkloadIdentityRolesByVersion called for a Cluster Service Principal cluster",
		},
		{
			name: "Fail - Exit the func for non MIWI clusters that has no PlatformWorkloadIdentityProfile or ServicePrincipalProfile",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			wantErr: "PopulatePlatformWorkloadIdentityRolesByVersion called for a Cluster Service Principal cluster",
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
			name: "Success - The role set document found for the cluster and upgradeable version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{OperatorName: "Dummy1"},
							},
						},
					},
				},
					&api.PlatformWorkloadIdentityRoleSetDocument{
						PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
							Name: "Dummy2",
							Properties: api.PlatformWorkloadIdentityRoleSetProperties{
								OpenShiftVersion: "4.15",
								PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
									{OperatorName: "Dummy1"},
								},
							},
						},
					},
				)
			},
			wantPlatformIdentities: []string{"Dummy1"},
		},
		{
			name: "Success - The role set document found for the cluster and upgradeable version(with new identity)",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{OperatorName: "Dummy1"},
							},
						},
					},
				},
					&api.PlatformWorkloadIdentityRoleSetDocument{
						PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
							Name: "Dummy2",
							Properties: api.PlatformWorkloadIdentityRoleSetProperties{
								OpenShiftVersion: "4.15",
								PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
									{OperatorName: "Dummy2"},
								},
							},
						},
					},
				)
			},
			wantPlatformIdentities: []string{"Dummy1", "Dummy2"},
		},
		{
			name: "Success - Role set document found for cluster version; UpgradeableTo version ignored because it is less than cluster version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						UpgradeableTo: ptr.To(api.UpgradeableTo("4.13.40")),
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{OperatorName: "Dummy1"},
							},
						},
					},
				})
			},
			wantPlatformIdentities: []string{"Dummy1"},
		},
		{
			name: "Success - Role set document found for cluster version; UpgradeableTo version ignored because upgradeable minor version is equal to cluster minor version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.40",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						UpgradeableTo: ptr.To(api.UpgradeableTo("4.14.60")),
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "Dummy1",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{OperatorName: "Dummy1"},
							},
						},
					},
				})
			},
			wantPlatformIdentities: []string{"Dummy1"},
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
		{
			name: "Failed - The role set documents listAll is missing the requested upgradeable version",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "4.14.26",
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
					},
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
			wantErr: "400: InvalidParameter: : No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '4.15'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket",
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
			pir := NewPlatformWorkloadIdentityRolesByVersionService()
			err := pir.PopulatePlatformWorkloadIdentityRolesByVersion(ctx, tt.oc, dbPlatformWorkloadIdentityRoleSets)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantPlatformIdentities != nil {
				platformWorkloadIdentityRolesByRoleName := pir.GetPlatformWorkloadIdentityRolesByRoleName()
				for _, operatorName := range tt.wantPlatformIdentities {
					_, ok := platformWorkloadIdentityRolesByRoleName[operatorName]
					if !ok {
						t.Fatalf("Incorrect platformWorkloadIdentityRolesByRoleName created, %s does not exist. %s", operatorName, platformWorkloadIdentityRolesByRoleName)
					}
				}
			}
		})
	}
}
