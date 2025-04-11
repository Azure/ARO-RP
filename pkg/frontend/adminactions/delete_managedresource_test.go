package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	infraID      = "infraID"
	location     = "eastus"
	subscription = "00000000-0000-0000-0000-000000000000"
	clusterRG    = "clusterRG"
)

var originalLB = mgmtnetwork.LoadBalancer{
	Sku: &mgmtnetwork.LoadBalancerSku{
		Name: mgmtnetwork.LoadBalancerSkuNameStandard,
	},
	LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
		FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
			{
				FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &mgmtnetwork.PublicIPAddress{
						ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
					},
				},
				ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
				Name: to.StringPtr("public-lb-ip-v4"),
			},
			{
				Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
				ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
				FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
					LoadBalancingRules: &[]mgmtnetwork.SubResource{
						{
							ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
						},
						{
							ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
						},
					},
					PublicIPAddress: &mgmtnetwork.PublicIPAddress{
						ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
					},
				},
			},
			{
				Name: to.StringPtr("adce98f85c7dd47c5a21263a5e39c083"),
				ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083"),
				FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &mgmtnetwork.PublicIPAddress{
						ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083"),
					},
				},
			},
		},
		Probes: &[]mgmtnetwork.Probe{
			{
				Name: to.StringPtr("probe-2"),
				ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/probe-2"),
				ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
					Port:           to.Int32Ptr(80),
					Protocol:       mgmtnetwork.ProbeProtocolHTTP,
					NumberOfProbes: to.Int32Ptr(3),
					RequestPath:    to.StringPtr("/health2"),
					LoadBalancingRules: &[]mgmtnetwork.SubResource{
						{
							ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
						},
					},
				},
			},
		},
	},
	Name:     to.StringPtr(infraID),
	Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
	Location: to.StringPtr(location),
}

var probeID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/probe-2"

func TestDeleteManagedResource(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name        string
		resourceID  string
		currentLB   mgmtnetwork.LoadBalancer
		expectedErr string
		mocks       func(*mock_features.MockResourcesClient, *mock_network.MockLoadBalancersClient)
	}{
		{
			name:        "remove frontend ip config",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083",
			currentLB:   originalLB,
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", "").Return(originalLB, nil)
				loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, infraID, mgmtnetwork.LoadBalancer{
					Sku: &mgmtnetwork.LoadBalancerSku{
						Name: mgmtnetwork.LoadBalancerSkuNameStandard,
					},
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &mgmtnetwork.PublicIPAddress{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
									},
								},
								ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
								Name: to.StringPtr("public-lb-ip-v4"),
							},
							{
								Name: to.StringPtr("ae3506385907e44eba9ef9bf76eac973"),
								ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
								FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
									LoadBalancingRules: &[]mgmtnetwork.SubResource{
										{
											ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
										{
											ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
										},
									},
									PublicIPAddress: &mgmtnetwork.PublicIPAddress{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-default-v4"),
									},
								},
							},
						},
						Probes: &[]mgmtnetwork.Probe{
							{
								Name: to.StringPtr("probe-2"),
								ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/probe-2"),
								ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
									Port:           to.Int32Ptr(80),
									Protocol:       mgmtnetwork.ProbeProtocolHTTP,
									NumberOfProbes: to.Int32Ptr(3),
									RequestPath:    to.StringPtr("/health2"),
									LoadBalancingRules: &[]mgmtnetwork.SubResource{
										{
											ID: to.StringPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
									},
								},
							},
						},
					},
					Name:     to.StringPtr(infraID),
					Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
					Location: to.StringPtr(location),
				}).Return(nil)
			},
		},
		{
			name:        "delete public IP Address",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083",
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				resources.EXPECT().DeleteByIDAndWait(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/adce98f85c7dd47c5a21263a5e39c083", "2020-08-01").Return(nil)
			},
		},
		{
			name:        "deletion of private link service is forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateLinkServices/infraID-pls",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateLinkServices/infraID-pls is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
			},
		},
		{
			name:        "deletion of private endpoints are forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateEndpoints/infraID-pe",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/privateEndpoints/infraID-pe is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
			},
		},
		{
			name:        "deletion of Microsoft.Storage resources is forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Storage/someStorageType/infraID",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "deletion of resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Storage/someStorageType/infraID is forbidden").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
			},
		},
		{
			name:        "prevent deletion of probe still in use",
			resourceID:  probeID,
			expectedErr: fmt.Sprintf("probe %s still in use by load balancing rules, remove references prior to removing the probe", probeID),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_network.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), probeID, "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", "").Return(originalLB, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return(location)

			networkLoadBalancers := mock_network.NewMockLoadBalancersClient(controller)
			resources := mock_features.NewMockResourcesClient(controller)
			tt.mocks(resources, networkLoadBalancers)

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
				loadBalancers: networkLoadBalancers,
				resources:     resources,
			}

			ctx := context.Background()

			err := a.ResourceDeleteAndWait(ctx, tt.resourceID)
			utilerror.AssertErrorMessage(t, err, tt.expectedErr)
		})
	}
}
