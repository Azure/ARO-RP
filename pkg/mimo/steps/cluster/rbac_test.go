package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armauthorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEnsureClusterServicePrincipalRBAC(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(100, 0) }

	const (
		subscriptionID = "00000000-0000-0000-0000-000000000000"
		spObjectID     = "00000000-0000-0000-0000-000000000001"
		clusterRGName  = "cluster-rg"
	)

	resourceGroupID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + clusterRGName
	contributorID := "/subscriptions/" + subscriptionID + "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleContributor

	assignment := func(scope, principalID, roleDefinitionID string) *sdkauthorization.RoleAssignment {
		return &sdkauthorization.RoleAssignment{
			Properties: &sdkauthorization.RoleAssignmentProperties{
				Scope:            &scope,
				PrincipalID:      &principalID,
				RoleDefinitionID: &roleDefinitionID,
			},
		}
	}

	wantCreateParameters := sdkauthorization.RoleAssignmentCreateParameters{
		Properties: &sdkauthorization.RoleAssignmentProperties{
			RoleDefinitionID: &contributorID,
			PrincipalID:      pointerutils.ToPtr(spObjectID),
			PrincipalType:    pointerutils.ToPtr(sdkauthorization.PrincipalTypeServicePrincipal),
		},
	}

	for _, tt := range []struct {
		name                 string
		usesWorkloadIdentity bool
		noServicePrincipal   bool
		mocks                func(*mock_armauthorization.MockRoleAssignmentsClient)
		wantErr              string
		wantMessage          string
	}{
		{
			name:                 "workload identity cluster is skipped without calling Azure",
			usesWorkloadIdentity: true,
			wantMessage:          "cluster uses workload identity and has no cluster service principal",
		},
		{
			name:               "missing service principal object id is terminal",
			noServicePrincipal: true,
			wantErr:            "TerminalError: cluster service principal object id is not populated",
		},
		{
			name: "listing failure is retryable",
			mocks: func(ra *mock_armauthorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, nil).Return(nil, errors.New("arm is down"))
			},
			wantErr: "TransientError: listing role assignments for resource group cluster-rg: arm is down",
		},
		{
			name: "existing assignment means no write",
			mocks: func(ra *mock_armauthorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, nil).Return([]*sdkauthorization.RoleAssignment{
					assignment(resourceGroupID, spObjectID, contributorID),
				}, nil)
			},
			wantMessage: "cluster service principal already has Contributor on the managed resource group",
		},
		{
			name: "assignment is created when only unrelated ones exist",
			mocks: func(ra *mock_armauthorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, nil).Return([]*sdkauthorization.RoleAssignment{
					// inherited from the subscription, so out of scope
					assignment("/subscriptions/"+subscriptionID, spObjectID, contributorID),
					// right scope and principal, wrong role
					assignment(resourceGroupID, spObjectID, "/subscriptions/"+subscriptionID+"/providers/Microsoft.Authorization/roleDefinitions/"+rbac.RoleReader),
					// must not panic
					nil,
					{},
				}, nil)
				ra.EXPECT().Create(gomock.Any(), resourceGroupID, gomock.Any(), wantCreateParameters, nil).
					Return(sdkauthorization.RoleAssignmentsClientCreateResponse{}, nil)
			},
			wantMessage: "restored Contributor on the managed resource group for the cluster service principal",
		},
		{
			name: "creation failure is retryable",
			mocks: func(ra *mock_armauthorization.MockRoleAssignmentsClient) {
				ra.EXPECT().ListForResourceGroup(gomock.Any(), clusterRGName, nil).Return(nil, nil)
				ra.EXPECT().Create(gomock.Any(), resourceGroupID, gomock.Any(), wantCreateParameters, nil).
					Return(sdkauthorization.RoleAssignmentsClientCreateResponse{}, errors.New("arm is down"))
			},
			wantErr: "TransientError: creating Contributor role assignment for cluster service principal: arm is down",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			_, log := testlog.New()

			properties := api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{ResourceGroupID: resourceGroupID},
			}

			// a workload identity cluster is one with no service principal at all
			if tt.usesWorkloadIdentity {
				properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
			} else if !tt.noServicePrincipal {
				properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{SPObjectID: spObjectID}
			}

			options := []testtasks.Option{
				testtasks.WithOpenShiftClusterDocument(&api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{Properties: properties},
				}),
				testtasks.WithSubscriptionDocument(&api.SubscriptionDocument{ID: subscriptionID}),
			}

			if tt.mocks != nil {
				roleAssignments := mock_armauthorization.NewMockRoleAssignmentsClient(gomock.NewController(t))
				tt.mocks(roleAssignments)
				options = append(options, testtasks.WithRoleAssignmentsClient(roleAssignments))
			}

			tc := testtasks.NewFakeTestContext(ctx, nil, log, now, options...)

			err := EnsureClusterServicePrincipalRBAC(tc)

			if tt.wantErr != "" {
				g.Expect(err).To(MatchError(tt.wantErr))
				return
			}

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(tc.GetResultMessage()).To(Equal(tt.wantMessage))
		})
	}
}
