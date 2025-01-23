package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestListPlatformWorkloadIdentityRoleSets(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		changeFeed     map[string]*api.PlatformWorkloadIdentityRoleSet
		apiVersion     string
		wantStatusCode int
		wantResponse   v20240812preview.PlatformWorkloadIdentityRoleSetList
		wantError      string
	}{
		{
			name: "GET request results in StatusOK",
			changeFeed: map[string]*api.PlatformWorkloadIdentityRoleSet{
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
					Name: "4.14",
					Type: api.PlatformWorkloadIdentityRoleSetsType,
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
					Name: "4.15",
					Type: api.PlatformWorkloadIdentityRoleSetsType,
				},
			},
			apiVersion:     "2024-08-12-preview",
			wantStatusCode: 200,
			wantResponse: v20240812preview.PlatformWorkloadIdentityRoleSetList{
				PlatformWorkloadIdentityRoleSets: []*v20240812preview.PlatformWorkloadIdentityRoleSet{
					{
						Properties: v20240812preview.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []v20240812preview.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								},
							},
						},
						Name: "4.14",
						Type: api.PlatformWorkloadIdentityRoleSetsType,
					},
					{
						Properties: v20240812preview.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.15",
							PlatformWorkloadIdentityRoles: []v20240812preview.PlatformWorkloadIdentityRole{
								{
									OperatorName:       "CloudControllerManager",
									RoleDefinitionName: "Azure RedHat OpenShift Cloud Controller Manager Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
								},
								{
									OperatorName:       "ClusterIngressOperator",
									RoleDefinitionName: "Azure RedHat OpenShift Cluster Ingress Operator Role",
									RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c",
								},
							},
						},
						Name: "4.15",
						Type: api.PlatformWorkloadIdentityRoleSetsType,
					},
				},
			},
		},
		{
			name:           "GET request with non-existent API version results in StatusBadRequest",
			apiVersion:     "invalid",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
		{
			name:           "GET request with old API version that doesn't support MIWI results in StatusBadRequest",
			apiVersion:     "2022-09-04",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The endpoint could not be found in the namespace 'microsoft.redhatopenshift' for api version '2022-09-04'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithPlatformWorkloadIdentityRoleSets()
			defer ti.done()

			frontend, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go frontend.Run(ctx, nil, nil)

			frontend.platformWorkloadIdentityRoleSetsMu.Lock()
			frontend.availablePlatformWorkloadIdentityRoleSets = tt.changeFeed
			frontend.platformWorkloadIdentityRoleSetsMu.Unlock()

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/platformworkloadidentityrolesets?api-version=%s", mockSubID, ti.env.Location(), tt.apiVersion),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// sort the response as the version order might be changed
			if b != nil && resp.StatusCode == http.StatusOK {
				var r v20240812preview.PlatformWorkloadIdentityRoleSetList
				if err = json.Unmarshal(b, &r); err != nil {
					t.Error(err)
				}

				sort.Slice(r.PlatformWorkloadIdentityRoleSets, func(i, j int) bool {
					return version.CreateSemverFromMinorVersionString(r.PlatformWorkloadIdentityRoleSets[i].Properties.OpenShiftVersion).LessThan(*version.CreateSemverFromMinorVersionString(r.PlatformWorkloadIdentityRoleSets[j].Properties.OpenShiftVersion))
				})

				b, err = json.Marshal(r)
				if err != nil {
					t.Error(err)
				}
			}

			// marshal the expected response into a []byte otherwise
			// it will compare zero values to omitempty json tags
			want, err := json.Marshal(tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, want)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
