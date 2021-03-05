package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestDenyAssignments(t *testing.T) {
	for _, tt := range []struct {
		name         string
		featureFlags []string
		want         []string
	}{
		{
			name: "Not registered for snapshots feature",
			want: []string{
				"Microsoft.Network/networkSecurityGroups/join/action",
				"Microsoft.Compute/disks/beginGetAccess/action",
				"Microsoft.Compute/disks/endGetAccess/action",
				"Microsoft.Compute/disks/write",
				"Microsoft.Compute/snapshots/beginGetAccess/action",
				"Microsoft.Compute/snapshots/endGetAccess/action",
				"Microsoft.Compute/snapshots/write",
				"Microsoft.Compute/snapshots/delete",
			},
		},
		{
			name:         "Registered for engineering feature flag",
			featureFlags: []string{"Microsoft.RedHatOpenShift/RedHatEngineering"},
			want: []string{
				"Microsoft.Network/networkSecurityGroups/join/action",
				"Microsoft.Compute/disks/beginGetAccess/action",
				"Microsoft.Compute/disks/endGetAccess/action",
				"Microsoft.Compute/disks/write",
				"Microsoft.Compute/snapshots/beginGetAccess/action",
				"Microsoft.Compute/snapshots/endGetAccess/action",
				"Microsoft.Compute/snapshots/write",
				"Microsoft.Compute/snapshots/delete",
				"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var features = []api.RegisteredFeatureProfile{}
			for i := range tt.featureFlags {
				features = append(features, api.RegisteredFeatureProfile{
					Name:  tt.featureFlags[i],
					State: "Registered",
				})
			}
			m := &manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "testing",
							},
						},
					},
				},
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: features,
						},
					},
				},
			}
			exceptionsToDeniedActions := *(*((m.denyAssignment().Resource).(*mgmtauthorization.DenyAssignment).
				DenyAssignmentProperties.Permissions))[0].NotActions
			if !reflect.DeepEqual(exceptionsToDeniedActions, tt.want) {
				t.Error(exceptionsToDeniedActions)
			}
		})
	}
}

func TestCreateAndUpdateErrors(t *testing.T) {
	ctx := context.Background()
	clusterID := "test-cluster"
	resourceGroupName := "fakeResourceGroup"
	resourceGroup := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", resourceGroupName)
	location := "eastus"

	group := mgmtfeatures.ResourceGroup{
		Location:  &location,
		ManagedBy: &clusterID,
	}

	disallowedByPolicy := autorest.NewErrorWithError(&azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "RequestDisallowedByPolicy"},
	}, "", "", nil, "")

	for _, tt := range []struct {
		name    string
		result  mgmtfeatures.ResourceGroup
		mocks   func(*mock_features.MockResourceGroupsClient, interface{})
		wantErr string
	}{
		{
			name: "ResourceGroup creation was fine",
			mocks: func(rg *mock_features.MockResourceGroupsClient, result interface{}) {
				rg.EXPECT().
					CreateOrUpdate(ctx, resourceGroupName, group).
					Return(result, nil)
			},
		},
		{
			name: "ResourceGroup creation failed with RequestDisallowedByPolicy",
			mocks: func(rg *mock_features.MockResourceGroupsClient, result interface{}) {
				rg.EXPECT().
					CreateOrUpdate(ctx, resourceGroupName, group).
					Return(result, disallowedByPolicy)
			},
			wantErr: `400: DeploymentFailed: : Deployment failed. Details: : : {"code":"RequestDisallowedByPolicy","message":"","target":null,"details":null,"innererror":null,"additionalInfo":null}`,
		},
		{
			name: "ResourceGroup creation failed with other error",
			mocks: func(rg *mock_features.MockResourceGroupsClient, result interface{}) {
				rg.EXPECT().
					CreateOrUpdate(ctx, resourceGroupName, group).
					Return(result, fmt.Errorf("Any other error"))
			},
			wantErr: "Any other error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			resourceGroupsClient := mock_features.NewMockResourceGroupsClient(controller)
			tt.mocks(resourceGroupsClient, tt.result)

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return(location)
			env.EXPECT().EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			env.EXPECT().DeploymentMode().Return(deployment.Production)

			m := &manager{
				log:            logrus.NewEntry(logrus.StandardLogger()),
				resourceGroups: resourceGroupsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroup,
							},
						},
						Location: location,
						ID:       clusterID,
					},
				},
				env: env,
			}

			err := m.ensureResourceGroup(ctx)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
