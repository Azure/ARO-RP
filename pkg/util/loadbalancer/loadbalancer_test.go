package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var infraID = "infraID"
var location = "eastus"
var publicIngressFIPConfigID = to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973")
var originalLB = armnetwork.LoadBalancer{
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
				ID:   publicIngressFIPConfigID,
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

func TestRemoveLoadBalancerFrontendIPConfiguration(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name          string
		fipResourceID string
		currentLB     armnetwork.LoadBalancer
		expectedLB    armnetwork.LoadBalancer
		expectedErr   string
	}{
		{
			name:          "remove frontend ip config",
			fipResourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083",
			currentLB:     originalLB,
			expectedLB: armnetwork.LoadBalancer{
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
			},
		},
		{
			name:          "removal of frontend ip config fails when frontend ip config has references",
			fipResourceID: *publicIngressFIPConfigID,
			currentLB:     originalLB,
			expectedLB:    originalLB,
			expectedErr:   fmt.Sprintf("frontend IP Configuration %s has external references, remove the external references prior to removing the frontend IP configuration", *publicIngressFIPConfigID),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveFrontendIPConfiguration(&tt.currentLB, tt.fipResourceID)
			assert.Equal(t, tt.expectedLB, tt.currentLB)
			utilerror.AssertErrorMessage(t, err, tt.expectedErr)
		})
	}
}
