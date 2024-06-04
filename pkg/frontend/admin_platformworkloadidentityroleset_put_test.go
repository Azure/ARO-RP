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

func TestPlatformWorkloadIdentityRoleSetPut(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		body           *admin.PlatformWorkloadIdentityRoleSet
		wantStatusCode int
		wantResponse   *admin.PlatformWorkloadIdentityRoleSet
		wantError      string
		wantDocuments  []*api.PlatformWorkloadIdentityRoleSetDocument
	}

	for _, tt := range []*test{
		{
			name: "updating known version",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
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
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
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
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "creating new version",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.15",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
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
			wantStatusCode: http.StatusCreated,
			wantResponse: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.15",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
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
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
				{
					ID: "08080808-0808-0808-0808-080808080002",
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.15",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
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
		},
		{
			name:           "creating new version needs body",
			fixture:        func(f *testdatabase.Fixture) {},
			body:           &admin.PlatformWorkloadIdentityRoleSet{},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.openShiftVersion: Must be provided",
			wantDocuments:  []*api.PlatformWorkloadIdentityRoleSetDocument{},
		},
		{
			name: "updating known version requires OpenShiftVersion",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
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
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.openShiftVersion: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRoles",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles: Must be provided and must be non-empty",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRole.OperatorName",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
						{
							RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
							RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
							ServiceAccounts: []string{
								"openshift-ingress-operator:ingress-operator",
							},
						},
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles[0].operatorName: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRole.RoleDefinitionName",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
						{
							OperatorName:     "ClusterIngressOperator",
							RoleDefinitionID: "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
							ServiceAccounts: []string{
								"openshift-ingress-operator:ingress-operator",
							},
						},
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles[0].roleDefinitionName: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRole.RoleDefinitionID",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
						{
							OperatorName:       "ClusterIngressOperator",
							RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
							ServiceAccounts: []string{
								"openshift-ingress-operator:ingress-operator",
							},
						},
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles[0].roleDefinitionId: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRole.ServiceAccounts",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
						{
							OperatorName:       "ClusterIngressOperator",
							RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
							RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
						},
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles[0].serviceAccounts: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
		{
			name: "updating known version requires PlatformWorkloadIdentityRole.RoleDefinitionId and PlatformWorkloadIdentityRole.ServiceAccounts (tests the case where multiple attributes are missing and error message consists of missing properties joined together)",
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
				)
			},
			body: &admin.PlatformWorkloadIdentityRoleSet{
				Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
					OpenShiftVersion: "4.14",
					PlatformWorkloadIdentityRoles: []admin.PlatformWorkloadIdentityRole{
						{
							OperatorName:       "ClusterIngressOperator",
							RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
						},
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: properties.platformWorkloadIdentityRoles[0].roleDefinitionId, properties.platformWorkloadIdentityRoles[0].serviceAccounts: Must be provided",
			wantDocuments: []*api.PlatformWorkloadIdentityRoleSetDocument{
				{
					ID: "08080808-0808-0808-0808-080808080001",
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
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithPlatformWorkloadIdentityRoleSets()

			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, nil, nil, nil, nil, nil, ti.platformWorkloadIdentityRoleSetsDatabase, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPut, "https://server/admin/platformworkloadidentityrolesets",
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
				ti.checker.AddPlatformWorkloadIdentityRoleSetDocuments(tt.wantDocuments...)
				for _, err := range ti.checker.CheckPlatformWorkloadIdentityRoleSets(ti.platformWorkloadIdentityRoleSetsClient) {
					t.Error(err)
				}
			}
		})
	}
}
