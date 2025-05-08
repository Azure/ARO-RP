package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	infraID      = "infraID"
	location     = "eastus"
	subscription = "00000000-0000-0000-0000-000000000000"
	clusterRG    = "clusterRG"
)

var originalLB = armnetwork.LoadBalancer{
	SKU: &armnetwork.LoadBalancerSKU{
		Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
	},
	Properties: &armnetwork.LoadBalancerPropertiesFormat{
		FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
			{
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
					},
				},
				ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
				Name: pointerutils.ToPtr("public-lb-ip-v4"),
			},
			{
				Name: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973"),
				ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					LoadBalancingRules: []*armnetwork.SubResource{
						{
							ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
						},
						{
							ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
						},
					},
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
					},
				},
			},
			{
				Name: pointerutils.ToPtr("adce98f85c7dd47c5a21263a5e39c083"),
				ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083"),
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083"),
					},
				},
			},
		},
	},
	Name:     pointerutils.ToPtr(infraID),
	Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
	Location: pointerutils.ToPtr(location),
}

func TestDeleteManagedResource(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name        string
		resourceID  string
		currentLB   armnetwork.LoadBalancer
		expectedErr string
		mocks       func(*mock_features.MockResourcesClient, *mock_armnetwork.MockLoadBalancersClient)
	}{
		{
			name:        "remove frontend ip config",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083",
			currentLB:   originalLB,
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: originalLB}, nil)
				loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, infraID, armnetwork.LoadBalancer{
					SKU: &armnetwork.LoadBalancerSKU{
						Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
					},
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &armnetwork.PublicIPAddress{
										ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
									},
								},
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
								Name: pointerutils.ToPtr("public-lb-ip-v4"),
							},
							{
								Name: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973"),
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									LoadBalancingRules: []*armnetwork.SubResource{
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
										},
									},
									PublicIPAddress: &armnetwork.PublicIPAddress{
										ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
									},
								},
							},
						},
					},
					Name:     pointerutils.ToPtr(infraID),
					Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
					Location: pointerutils.ToPtr(location),
				}, nil).Return(nil)
			},
		},
		{
			name:        "delete public IP Address",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083",
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				resources.EXPECT().DeleteByIDAndWait(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(nil)
			},
		},
		{
			name:        "deletion of private link service is forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateLinkServices/infraID-pls",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateLinkServices/infraID-pls is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
			},
		},
		{
			name:        "deletion of private endpoints are forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateEndpoints/infraID-pe",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateEndpoints/infraID-pe is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
			},
		},
		{
			name:        "deletion of Microsoft.Storage resources is forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Storage/someStorageType/infraID",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Storage/someStorageType/infraID is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return(location)

			loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(controller)
			resources := mock_features.NewMockResourcesClient(controller)
			tt.mocks(resources, loadBalancers)

			a := azureActions{
				log: logrus.NewEntry(logrus.StandardLogger()),
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, clusterRG),
						},
					},
				},
				loadBalancers: loadBalancers,
				resources:     resources,
			}

			ctx := context.Background()

			err := a.ResourceDeleteAndWait(ctx, tt.resourceID)
			utilerror.AssertErrorMessage(t, err, tt.expectedErr)
		})
	}
}
