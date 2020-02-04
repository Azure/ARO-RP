package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (i *installer) apiServerPublicLoadBalancer(location string, visibility api.Visibility) *arm.Resource {
	lb := &mgmtnetwork.LoadBalancer{
		Sku: &mgmtnetwork.LoadBalancerSku{
			Name: mgmtnetwork.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &mgmtnetwork.PublicIPAddress{
							ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'aro-pip')]"),
						},
					},
					Name: to.StringPtr("public-lb-ip"),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
				{
					Name: to.StringPtr("aro-public-lb-control-plane"),
				},
			},
			OutboundRules: &[]mgmtnetwork.OutboundRule{
				{
					OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-public-lb', 'public-lb-ip')]"),
							},
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
						},
						Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
						IdleTimeoutInMinutes: to.Int32Ptr(30),
					},
					Name: to.StringPtr("api-internal-outboundrule"),
				},
			},
		},
		Name:     to.StringPtr("aro-public-lb"),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: &location,
	}

	if visibility == api.VisibilityPublic {
		lb.LoadBalancingRules = &[]mgmtnetwork.LoadBalancingRule{
			{
				LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-public-lb', 'public-lb-ip')]"),
					},
					BackendAddressPool: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
					},
					Probe: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-public-lb', 'api-internal-probe')]"),
					},
					Protocol:             mgmtnetwork.TransportProtocolTCP,
					LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
					FrontendPort:         to.Int32Ptr(6443),
					BackendPort:          to.Int32Ptr(6443),
					IdleTimeoutInMinutes: to.Int32Ptr(30),
					DisableOutboundSnat:  to.BoolPtr(true),
				},
				Name: to.StringPtr("api-internal"),
			},
		}
		lb.Probes = &[]mgmtnetwork.Probe{
			{
				ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
					Protocol:          mgmtnetwork.ProbeProtocolTCP,
					Port:              to.Int32Ptr(6443),
					IntervalInSeconds: to.Int32Ptr(10),
					NumberOfProbes:    to.Int32Ptr(3),
				},
				Name: to.StringPtr("api-internal-probe"),
				Type: to.StringPtr("Microsoft.Network/loadBalancers/probes"),
			},
		}
	}

	return &arm.Resource{
		Resource:   lb,
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			"Microsoft.Network/publicIPAddresses/aro-pip",
		},
	}
}
