package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

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
		Probes: []*armnetwork.Probe{
			{
				Name: pointerutils.ToPtr("testProbeInUse"),
				ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse"),
				Properties: &armnetwork.ProbePropertiesFormat{
					Port: pointerutils.ToPtr(int32(8443)),
					LoadBalancingRules: []*armnetwork.SubResource{
						{
							ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
						},
						{
							ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
						},
					},
				},
			},
			{
				Name: pointerutils.ToPtr("testProbeToDelete"),
				ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeToDelete"),
				Properties: &armnetwork.ProbePropertiesFormat{
					Port: pointerutils.ToPtr(int32(8080)),
				},
			},
		},
	},
	Name:     &infraID,
	Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
	Location: &location,
}

func loadBalancerWithRuleReferences() armnetwork.LoadBalancer {
	rule80ID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/loadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-80"
	rule443ID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/loadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-443"

	return armnetwork.LoadBalancer{
		SKU: &armnetwork.LoadBalancerSKU{
			Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973"),
					ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						LoadBalancingRules: []*armnetwork.SubResource{
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80")},
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443")},
						},
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: pointerutils.ToPtr("test-backend-pool"),
					Properties: &armnetwork.BackendAddressPoolPropertiesFormat{
						LoadBalancingRules: []*armnetwork.SubResource{
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80")},
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443")},
						},
					},
				},
			},
			Probes: []*armnetwork.Probe{
				{
					Name: pointerutils.ToPtr("testProbeInUse"),
					Properties: &armnetwork.ProbePropertiesFormat{
						Port: pointerutils.ToPtr(int32(8443)),
						LoadBalancingRules: []*armnetwork.SubResource{
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80")},
							{ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443")},
						},
					},
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
					ID:   pointerutils.ToPtr(rule80ID),
				},
				{
					Name: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
					ID:   pointerutils.ToPtr(rule443ID),
				},
			},
		},
		Name:     &infraID,
		Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
		Location: &location,
	}
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
						Probes: []*armnetwork.Probe{
							{
								Name: pointerutils.ToPtr("testProbeInUse"),
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse"),
								Properties: &armnetwork.ProbePropertiesFormat{
									Port: pointerutils.ToPtr(int32(8443)),
									LoadBalancingRules: []*armnetwork.SubResource{
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
										},
									},
								},
							},
							{
								Name: pointerutils.ToPtr("testProbeToDelete"),
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeToDelete"),
								Properties: &armnetwork.ProbePropertiesFormat{
									Port: pointerutils.ToPtr(int32(8080)),
								},
							},
						},
					},
					Name:     &infraID,
					Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
					Location: &location,
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
		{
			name:        "deletion of health probes in use is forbidden",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse",
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "Load balancer health probe /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse is used by load balancing rules, remove the referencing load balancing rules before removing the health probe").Error(),
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: originalLB}, nil)
			},
		},
		{
			name:        "health probe delete",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeToDelete",
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeToDelete", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
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
						Probes: []*armnetwork.Probe{
							{
								Name: pointerutils.ToPtr("testProbeInUse"),
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/probes/testProbeInUse"),
								Properties: &armnetwork.ProbePropertiesFormat{
									Port: pointerutils.ToPtr(int32(8443)),
									LoadBalancingRules: []*armnetwork.SubResource{
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-80"),
										},
										{
											ID: pointerutils.ToPtr("ae3506385907e44eba9ef9bf76eac973-TCP-443"),
										},
									},
								},
							},
						},
					},
					Name:     &infraID,
					Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
					Location: &location,
				}, nil).Return(nil)
			},
		},
		{
			name:        "load balancing rule delete",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/loadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-80",
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				lbFixture := loadBalancerWithRuleReferences()
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/loadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-80", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: lbFixture}, nil)
				loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, infraID, gomock.AssignableToTypeOf(armnetwork.LoadBalancer{}), nil).DoAndReturn(
					func(ctx context.Context, resourceGroupName, loadBalancerName string, parameters armnetwork.LoadBalancer, options *armnetwork.LoadBalancersClientBeginCreateOrUpdateOptions) error {
						expectedRemainingRuleID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/loadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-443"

						if parameters.Properties == nil {
							t.Fatal("expected load balancer properties to be set")
						}
						if len(parameters.Properties.LoadBalancingRules) != 1 || parameters.Properties.LoadBalancingRules[0].ID == nil || *parameters.Properties.LoadBalancingRules[0].ID != expectedRemainingRuleID {
							t.Fatalf("expected exactly one remaining top-level load balancing rule %s, got %#v", expectedRemainingRuleID, parameters.Properties.LoadBalancingRules)
						}
						if len(parameters.Properties.FrontendIPConfigurations) != 1 || parameters.Properties.FrontendIPConfigurations[0].Properties == nil ||
							len(parameters.Properties.FrontendIPConfigurations[0].Properties.LoadBalancingRules) != 1 ||
							parameters.Properties.FrontendIPConfigurations[0].Properties.LoadBalancingRules[0].ID == nil ||
							*parameters.Properties.FrontendIPConfigurations[0].Properties.LoadBalancingRules[0].ID != "ae3506385907e44eba9ef9bf76eac973-TCP-443" {
							t.Fatalf("expected frontend IP config references to retain only the remaining rule, got %#v", parameters.Properties.FrontendIPConfigurations)
						}
						if len(parameters.Properties.BackendAddressPools) != 1 || parameters.Properties.BackendAddressPools[0].Properties == nil ||
							len(parameters.Properties.BackendAddressPools[0].Properties.LoadBalancingRules) != 1 ||
							parameters.Properties.BackendAddressPools[0].Properties.LoadBalancingRules[0].ID == nil ||
							*parameters.Properties.BackendAddressPools[0].Properties.LoadBalancingRules[0].ID != "ae3506385907e44eba9ef9bf76eac973-TCP-443" {
							t.Fatalf("expected backend pool references to retain only the remaining rule, got %#v", parameters.Properties.BackendAddressPools)
						}
						if len(parameters.Properties.Probes) != 1 || parameters.Properties.Probes[0].Properties == nil ||
							len(parameters.Properties.Probes[0].Properties.LoadBalancingRules) != 1 ||
							parameters.Properties.Probes[0].Properties.LoadBalancingRules[0].ID == nil ||
							*parameters.Properties.Probes[0].Properties.LoadBalancingRules[0].ID != "ae3506385907e44eba9ef9bf76eac973-TCP-443" {
							t.Fatalf("expected probe references to retain only the remaining rule, got %#v", parameters.Properties.Probes)
						}
						return nil
					},
				)
			},
		},
		{
			name:        "load balancing rule delete with mixed-case resource type",
			resourceID:  "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/LoadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-80",
			expectedErr: "",
			mocks: func(resources *mock_features.MockResourcesClient, loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				lbFixture := loadBalancerWithRuleReferences()
				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/LoadBalancingRules/ae3506385907e44eba9ef9bf76eac973-TCP-80", "2020-08-01").Return(mgmtfeatures.GenericResource{}, nil)
				loadBalancers.EXPECT().Get(gomock.Any(), "clusterRG", "infraID", nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: lbFixture}, nil)
				loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, infraID, gomock.AssignableToTypeOf(armnetwork.LoadBalancer{}), nil).Return(nil)
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
