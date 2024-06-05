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

func TestPlatformWorkloadIdentityRoleSetList(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *admin.PlatformWorkloadIdentityRoleSetList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "GET request returns empty result with StatusOK",
			fixture:        func(f *testdatabase.Fixture) {},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.PlatformWorkloadIdentityRoleSetList{
				PlatformWorkloadIdentityRoleSets: []*admin.PlatformWorkloadIdentityRoleSet{},
			},
		},
		{
			name: "GET request returns non-empty result with StatusOK",
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
					&api.PlatformWorkloadIdentityRoleSetDocument{
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
								},
							},
						},
					},
				)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.PlatformWorkloadIdentityRoleSetList{
				PlatformWorkloadIdentityRoleSets: []*admin.PlatformWorkloadIdentityRoleSet{
					{
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
					{
						Properties: admin.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.15",
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

			resp, b, err := ti.request(http.MethodGet, "https://server/admin/platformworkloadidentityrolesets",
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
