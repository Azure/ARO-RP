package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var infraID = "infraID"
var location = "eastus"
var publicIngressFIPConfigID = to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973")
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
				ID:   publicIngressFIPConfigID,
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
	},
	Name:     to.StringPtr(infraID),
	Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
	Location: to.StringPtr(location),
}

func TestRemoveLoadBalancerFrontendIPConfiguration(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name          string
		fipResourceID string
		currentLB     mgmtnetwork.LoadBalancer
		expectedLB    mgmtnetwork.LoadBalancer
		expectedErr   string
	}{
		{
			name:          "remove frontend ip config",
			fipResourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083",
			currentLB:     originalLB,
			expectedLB: mgmtnetwork.LoadBalancer{
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
				},
				Name:     to.StringPtr(infraID),
				Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
				Location: to.StringPtr(location),
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
