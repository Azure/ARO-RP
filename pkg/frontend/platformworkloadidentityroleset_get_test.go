package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/api/v20250725"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func TestGetPlatformWorkloadIdentityRoleSet(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()
	availableMinorVersion := "4.14"
	changeFeed := map[string]*api.PlatformWorkloadIdentityRoleSet{
		availableMinorVersion: {
			Properties: api.PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: availableMinorVersion,
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
			Name: availableMinorVersion,
			Type: api.PlatformWorkloadIdentityRoleSetsType,
			ID:   "mockID",
		},
	}

	for _, tt := range []struct {
		name           string
		apiVersion     string
		minorVersion   string
		wantStatusCode int
		wantResponse   *v20250725.PlatformWorkloadIdentityRoleSet
		wantError      string
	}{
		{
			name:           "GET request results in StatusOK",
			apiVersion:     "2025-07-25",
			minorVersion:   availableMinorVersion,
			wantStatusCode: 200,
			wantResponse: &v20250725.PlatformWorkloadIdentityRoleSet{
				PlatformWorkloadIdentityRoleSet: generated.PlatformWorkloadIdentityRoleSet{
					Properties: &generated.PlatformWorkloadIdentityRoleSetProperties{
						OpenShiftVersion: pointerutils.ToPtr(availableMinorVersion),
						PlatformWorkloadIdentityRoles: []*generated.PlatformWorkloadIdentityRole{
							{
								OperatorName:       pointerutils.ToPtr("CloudControllerManager"),
								RoleDefinitionName: pointerutils.ToPtr("Azure RedHat OpenShift Cloud Controller Manager Role"),
								RoleDefinitionID:   pointerutils.ToPtr("/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4"),
							},
						},
					},
					ID:   pointerutils.ToPtr("mockID"),
					Name: pointerutils.ToPtr(availableMinorVersion),
					Type: pointerutils.ToPtr(api.PlatformWorkloadIdentityRoleSetsType),
				},
			},
		},
		{
			name:           "GET request with non-existent API version results in StatusBadRequest",
			apiVersion:     "invalid",
			minorVersion:   availableMinorVersion,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
		{
			name:           "GET request with old API version that doesn't support MIWI results in StatusBadRequest",
			apiVersion:     v20220904.APIVersion,
			minorVersion:   availableMinorVersion,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The endpoint could not be found in the namespace 'microsoft.redhatopenshift' for api version '2022-09-04'.",
		},
		{
			name:           "GET request with not available minor version results in StatusBadRequest",
			apiVersion:     "2025-07-25",
			minorVersion:   "4.13",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: ResourceNotFound: : The Resource platformWorkloadIdentityRoleSet with version '4.13' was not found in the namespace 'microsoft.redhatopenshift' for api version '2025-07-25'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithPlatformWorkloadIdentityRoleSets()
			defer ti.done()

			frontend, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go frontend.Run(ctx, nil, nil)

			frontend.platformWorkloadIdentityRoleSetsMu.Lock()
			frontend.availablePlatformWorkloadIdentityRoleSets = changeFeed
			frontend.platformWorkloadIdentityRoleSetsMu.Unlock()

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/platformWorkloadIdentityRoleSets/%s?api-version=%s", mockSubID, ti.env.Location(), tt.minorVersion, tt.apiVersion),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// unmarshal and marshal the response body to match string content
			if b != nil && resp.StatusCode == http.StatusOK {
				var r v20250725.PlatformWorkloadIdentityRoleSet
				if err = json.Unmarshal(b, &r); err != nil {
					t.Error(err)
				}

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
