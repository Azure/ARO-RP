package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPlatformWorkloadIdentityRoleSetList(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixture        func(f *testdatabase.Fixture)
		cosmosdb       func(c *cosmosdb.FakePlatformWorkloadIdentityRoleSetDocumentClient)
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
		{
			name: "GET request with StatusOK returns results in correct order even if Cosmos DB returns them in a different order",
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
			cosmosdb: func(c *cosmosdb.FakePlatformWorkloadIdentityRoleSetDocumentClient) {
				// Sort the documents in descending order rather than ascending order, which
				// is the order we expect to see in the response.
				c.SetSorter(func(roleSets []*api.PlatformWorkloadIdentityRoleSetDocument) {
					sort.Slice(roleSets, func(i, j int) bool {
						return version.CreateSemverFromMinorVersionString(roleSets[j].PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion).LessThan(*version.CreateSemverFromMinorVersionString(roleSets[i].PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion))
					})
				})
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
		{
			name:    "GET request results in StatusInternalServerError due to issues with Cosmos DB",
			fixture: func(f *testdatabase.Fixture) {},
			cosmosdb: func(c *cosmosdb.FakePlatformWorkloadIdentityRoleSetDocumentClient) {
				c.SetError(errors.New("Well shoot, Cosmos DB isn't working!"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantResponse: &admin.PlatformWorkloadIdentityRoleSetList{
				PlatformWorkloadIdentityRoleSets: []*admin.PlatformWorkloadIdentityRoleSet{},
			},
			wantError: api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.").Error(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithPlatformWorkloadIdentityRoleSets()
			defer ti.done()

			if tt.cosmosdb != nil {
				tt.cosmosdb(ti.platformWorkloadIdentityRoleSetsClient)
			}

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
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
