package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEnsureClusterServicePrincipalRBAC(t *testing.T) {
	ctx := context.Background()

	const (
		spObjectID     = "00000000-0000-0000-0000-000000000001"
		clusterRGName  = "cluster-rg"
		subscriptionID = "00000000-0000-0000-0000-000000000000"
	)
	resourceGroupID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, clusterRGName)

	baseOC := func() *api.OpenShiftCluster {
		return &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
				ServicePrincipalProfile: &api.ServicePrincipalProfile{
					SPObjectID: spObjectID,
				},
			},
		}
	}

	for _, tt := range []struct {
		name            string
		oc              *api.OpenShiftCluster
		roleAssignments []mgmtauthorization.RoleAssignment
		mockAuthz       func(*mock_authorization.MockRoleAssignmentsClient)
		mockDeploy      func(*mock_features.MockDeploymentsClient)
		wantErr         string
	}{
		{
			name: "workload identity cluster returns 400",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
				},
			},
			wantErr: `400: InvalidParameter: : Cluster uses workload identity and has no cluster service principal`,
		},
		{
			name: "nil service principal profile returns 400",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
				},
			},
			wantErr: `400: InvalidParameter: : Cluster service principal object id is not populated`,
		},
		{
			name: "empty SPObjectID returns 400",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						SPObjectID: "",
					},
				},
			},
			wantErr: `400: InvalidParameter: : Cluster service principal object id is not populated`,
		},
		{
			name: "ListForResourceGroup error propagates",
			oc:   baseOC(),
			mockAuthz: func(ra *mock_authorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, "").Return(nil, fmt.Errorf("list error"))
			},
			wantErr: "list error",
		},
		{
			name: "SP already has Contributor - no deployment",
			oc:   baseOC(),
			mockAuthz: func(ra *mock_authorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, "").Return([]mgmtauthorization.RoleAssignment{
					{
						RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
							Scope:            pointerutils.ToPtr(resourceGroupID),
							PrincipalID:      pointerutils.ToPtr(spObjectID),
							RoleDefinitionID: pointerutils.ToPtr(rbac.RoleContributor),
						},
					},
				}, nil)
			},
		},
		{
			name: "SP missing Contributor - deployment created",
			oc:   baseOC(),
			mockAuthz: func(ra *mock_authorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, "").Return(nil, nil)
			},
			mockDeploy: func(d *mock_features.MockDeploymentsClient) {
				d.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, "clustersp", gomock.Any()).Return(nil)
			},
		},
		{
			name: "deployment failure propagates",
			oc:   baseOC(),
			mockAuthz: func(ra *mock_authorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, "").Return(nil, nil)
			},
			mockDeploy: func(d *mock_features.MockDeploymentsClient) {
				d.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, "clustersp", gomock.Any()).Return(fmt.Errorf("deploy error"))
			},
			wantErr: "deploy error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.LogForTesting(t)
			controller := gomock.NewController(t)

			roleAssignments := mock_authorization.NewMockRoleAssignmentsClient(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)

			if tt.mockAuthz != nil {
				tt.mockAuthz(roleAssignments)
			}
			if tt.mockDeploy != nil {
				tt.mockDeploy(deployments)
			}

			a := azureActions{
				log:             log,
				oc:              tt.oc,
				roleAssignments: roleAssignments,
				deployments:     deployments,
			}

			err := a.EnsureClusterServicePrincipalRBAC(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
