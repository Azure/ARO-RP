package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
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
		Name: to.Ptr(armnetwork.LoadBalancerSKUNameStandard),
	},
	Properties: &armnetwork.LoadBalancerPropertiesFormat{
		FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
			{
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
					},
				},
				ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
				Name: to.Ptr("public-lb-ip-v4"),
			},
			{
				Name: to.Ptr("ae3506385907e44eba9ef9bf76eac973"),
				ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					LoadBalancingRules: []*armnetwork.SubResource{
						{
							ID: to.Ptr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
						},
						{
							ID: to.Ptr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
						},
					},
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
					},
				},
			},
			{
				Name: to.Ptr("adce98f85c7dd47c5a21263a5e39c083"),
				ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083"),
				Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &armnetwork.PublicIPAddress{
						ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083"),
					},
				},
			},
		},
	},
	Name:     to.Ptr(infraID),
	Type:     to.Ptr("Microsoft.Network/loadBalancers"),
	Location: to.Ptr(location),
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
						Name: to.Ptr(armnetwork.LoadBalancerSKUNameStandard),
					},
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &armnetwork.PublicIPAddress{
										ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
									},
								},
								ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
								Name: to.Ptr("public-lb-ip-v4"),
							},
							{
								Name: to.Ptr("ae3506385907e44eba9ef9bf76eac973"),
								ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									LoadBalancingRules: []*armnetwork.SubResource{
										{
											ID: to.Ptr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
										{
											ID: to.Ptr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
										},
									},
									PublicIPAddress: &armnetwork.PublicIPAddress{
										ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
									},
								},
							},
						},
					},
					Name:     to.Ptr(infraID),
					Type:     to.Ptr("Microsoft.Network/loadBalancers"),
					Location: to.Ptr(location),
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
