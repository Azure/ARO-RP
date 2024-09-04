package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestUpdateFromIteratorOcpVersions(t *testing.T) {
	for _, tt := range []struct {
		name           string
		docsInIterator []*api.OpenShiftVersionDocument
		versions       map[string]*api.OpenShiftVersion
		wantVersions   map[string]*api.OpenShiftVersion
	}{
		{
			name: "Add a new doc from the changefeed to an empty frontend cache",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.2.0",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.2.0": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.2.0",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "Docs in changefeed match docs in frontend cache - no changes needed",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "Add a new doc from the iterator to a non-empty frontend cache",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.6.7",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
				"4.6.7": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.6.7",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "A doc present in the frontend cache is marked deleting in the changefeed - remove it from the cache",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: true,
						},
					},
				},
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "5.0.0",
							Enabled: true,
						},
						Deleting: true,
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
				"5.0.0": {
					Properties: api.OpenShiftVersionProperties{
						Version: "5.0.0",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "A doc present in the frontend cache is marked disabled in the changefeed - remove it from the cache",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: false,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(20 * time.Millisecond)
			ctx, cancel := context.WithCancel(context.TODO())

			frontend := frontend{
				enabledOcpVersions: tt.versions,
			}

			fakeIterator := cosmosdb.NewFakeOpenShiftVersionDocumentIterator(tt.docsInIterator, 0)

			go frontend.updateFromIteratorOcpVersions(ctx, ticker, fakeIterator)
			time.Sleep(10 * time.Millisecond)
			cancel()

			if !reflect.DeepEqual(frontend.enabledOcpVersions, tt.wantVersions) {
				t.Error(cmp.Diff(frontend.enabledOcpVersions, tt.wantVersions))
			}
		})
	}
}

func TestUpdateFromIteratorRoleSets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		docsInIterator []*api.PlatformWorkloadIdentityRoleSetDocument
		roleSets       map[string]*api.PlatformWorkloadIdentityRoleSet
		wantRoleSets   map[string]*api.PlatformWorkloadIdentityRoleSet
	}{
		{
			name: "add to empty",
			docsInIterator: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
							},
						},
					},
				},
			},
			roleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{},
			wantRoleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "do nothing",
			docsInIterator: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
							},
						},
					},
				},
			},
			roleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
			},
			wantRoleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "add to not empty",
			docsInIterator: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
							},
						},
					},
				},
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.15",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
								{
									OperatorName:       "ClusterIngressOperator",
									RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
									ServiceAccounts: []string{
										"openshift-ingress-operator:ingress-operator",
									},
								},
							},
						},
					},
				},
			},
			roleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
			},
			wantRoleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
				"4.15": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.15",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
							{
								OperatorName:       "ClusterIngressOperator",
								RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
								ServiceAccounts: []string{
									"openshift-ingress-operator:ingress-operator",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove existing",
			docsInIterator: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
							},
						},
						Deleting: true,
					},
				},
				{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.15",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
									ServiceAccounts: []string{
										"openshift-cloud-controller-manager:cloud-controller-manager",
									},
								},
								{
									OperatorName:       "ClusterIngressOperator",
									RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
									ServiceAccounts: []string{
										"openshift-ingress-operator:ingress-operator",
									},
								},
							},
						},
					},
				},
			},
			roleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.14": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.14",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
						},
					},
				},
				"4.15": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.15",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
							{
								OperatorName:       "ClusterIngressOperator",
								RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
								ServiceAccounts: []string{
									"openshift-ingress-operator:ingress-operator",
								},
							},
						},
					},
				},
			},
			wantRoleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{
				"4.15": {
					Properties: api.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: "4.15",
						PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
							{
								OperatorName:       "CloudControllerManager",
								RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								ServiceAccounts: []string{
									"openshift-cloud-controller-manager:cloud-controller-manager",
								},
							},
							{
								OperatorName:       "ClusterIngressOperator",
								RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
								RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
								ServiceAccounts: []string{
									"openshift-ingress-operator:ingress-operator",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(20 * time.Millisecond)
			ctx, cancel := context.WithCancel(context.TODO())

			frontend := frontend{
				availablePlatformWorkloadIdentityRoleSets: tt.roleSets,
			}

			fakeIterator := cosmosdb.NewFakePlatformWorkloadIdentityRoleSetDocumentIterator(tt.docsInIterator, 0)

			go frontend.updateFromIteratorRoleSets(ctx, ticker, fakeIterator)
			time.Sleep(10 * time.Millisecond)
			cancel()

			if !reflect.DeepEqual(frontend.availablePlatformWorkloadIdentityRoleSets, tt.wantRoleSets) {
				t.Error(cmp.Diff(frontend.availablePlatformWorkloadIdentityRoleSets, tt.wantRoleSets))
			}
		})
	}
}
