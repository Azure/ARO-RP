package fake

import (
	"github.com/Azure/ARO-RP/pkg/api"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const OutboundRuleV4 = "outbound-rule-v4"

var FakeDefaultIngressFrontendIPConfig = mgmtnetwork.FrontendIPConfiguration{
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
}

// returns a default public loadbalancer
func NewFakePublicLoadBalancer(apiServerVisibility api.Visibility) mgmtnetwork.LoadBalancer {
	defaultOutboundFIPConfig := mgmtnetwork.FrontendIPConfiguration{
		Name: to.StringPtr("public-lb-ip-v4"),
		ID:   to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
		FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
			OutboundRules: &[]mgmtnetwork.SubResource{{
				ID: to.StringPtr(OutboundRuleV4),
			}},
			PublicIPAddress: &mgmtnetwork.PublicIPAddress{
				ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4"),
			},
		},
	}
	if apiServerVisibility == api.VisibilityPublic {
		defaultOutboundFIPConfig.FrontendIPConfigurationPropertiesFormat.LoadBalancingRules = &[]mgmtnetwork.SubResource{
			{
				ID: to.StringPtr("api-internal-v4"),
			},
		}
	}
	return mgmtnetwork.LoadBalancer{
		Name: to.StringPtr("infraID"),
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				FakeDefaultIngressFrontendIPConfig,
				defaultOutboundFIPConfig,
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
	}
}
