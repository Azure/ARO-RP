package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
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

func TestGetOutboundIPsFromLB(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name                string
		currentLB           mgmtnetwork.LoadBalancer
		expectedOutboundIPs []api.ResourceReference
	}{
		{
			name: "default",
			currentLB: mgmtnetwork.LoadBalancer{
				Name: to.StringPtr("infraID"),
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
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
							Name: to.StringPtr("public-lb-ip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("api-internal-v4"),
									},
								},
								OutboundRules: &[]mgmtnetwork.SubResource{{
									ID: to.StringPtr(OutboundRuleV4),
								}},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(OutboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
									},
								},
							},
						},
					},
				},
			},
			expectedOutboundIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run GetOutboundIPsFromLB and assert the correct results
			outboundIPs := GetOutboundIPsFromLB(tt.currentLB)
			assert.Equal(t, tt.expectedOutboundIPs, outboundIPs)
		})
	}
}

func TestAddOutboundIPsToLB(t *testing.T) {
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"

	// Run tests
	for _, tt := range []struct {
		name         string
		desiredOBIPs []api.ResourceReference
		currentLB    mgmtnetwork.LoadBalancer
		expectedLB   mgmtnetwork.LoadBalancer
	}{
		{
			name: "add default IP to lb",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
			currentLB: GetClearedLB(),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: to.StringPtr("infraID"),
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
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
							Name: to.StringPtr("public-lb-ip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("api-internal-v4"),
									},
								},
								OutboundRules: &[]mgmtnetwork.SubResource{{
									ID: to.StringPtr(OutboundRuleV4),
								}},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(OutboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "add multiple outbound IPs to LB",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
				},
			},
			currentLB: GetClearedLB(),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: to.StringPtr("infraID"),
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
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
							Name: to.StringPtr("public-lb-ip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("api-internal-v4"),
									},
								},
								OutboundRules: &[]mgmtnetwork.SubResource{{
									ID: to.StringPtr(OutboundRuleV4),
								}},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
						{
							Name: to.StringPtr("uuid1-outbound-pip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(OutboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
									},
									{
										ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run addOutboundIPsToLB and assert the correct results
			AddOutboundIPsToLB(clusterRGID, tt.currentLB, tt.desiredOBIPs)
			assert.Equal(t, tt.expectedLB, tt.currentLB)
		})
	}
}

func TestRemoveOutboundIPsFromLB(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name       string
		currentLB  mgmtnetwork.LoadBalancer
		expectedLB mgmtnetwork.LoadBalancer
	}{
		{
			name:      "remove all outbound-rule-v4 fip config except api server",
			currentLB: FakeLoadBalancersGet(1, api.VisibilityPublic),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: to.StringPtr("infraID"),
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
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
							Name: to.StringPtr("public-lb-ip-v4"),
							ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
							FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
								LoadBalancingRules: &[]mgmtnetwork.SubResource{
									{
										ID: to.StringPtr("api-internal-v4"),
									},
								},
								OutboundRules: &[]mgmtnetwork.SubResource{{
									ID: to.StringPtr(OutboundRuleV4),
								}},
								PublicIPAddress: &mgmtnetwork.PublicIPAddress{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
								},
							},
						},
					},
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(OutboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{},
							},
						},
					},
				},
			},
		},
		{
			name:      "remove all outbound-rule-v4 fip config",
			currentLB: FakeLoadBalancersGet(1, api.VisibilityPrivate),
			expectedLB: mgmtnetwork.LoadBalancer{
				Name: to.StringPtr("infraID"),
				LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
					FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
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
					OutboundRules: &[]mgmtnetwork.OutboundRule{
						{
							Name: to.StringPtr(OutboundRuleV4),
							OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
								FrontendIPConfigurations: &[]mgmtnetwork.SubResource{},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run removeOutboundIPsFromLB and assert correct results
			RemoveOutboundIPsFromLB(tt.currentLB)
			assert.Equal(t, tt.expectedLB, tt.currentLB)
		})
	}
}
