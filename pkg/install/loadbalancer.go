package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (i *Installer) apiServerPublicLoadBalancer(location string) *arm.Resource {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	lb := &mgmtnetwork.LoadBalancer{
		Sku: &mgmtnetwork.LoadBalancerSku{
			Name: mgmtnetwork.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &mgmtnetwork.PublicIPAddress{
							ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', '" + infraID + "-pip-v4')]"),
						},
					},
					Name: to.StringPtr("public-lb-ip-v4"),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
				{
					Name: to.StringPtr(infraID + "-public-lb-control-plane-v4"),
				},
			},
			OutboundRules: &[]mgmtnetwork.OutboundRule{
				{
					OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '" + infraID + "-public-lb', 'public-lb-ip-v4')]"),
							},
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '" + infraID + "-public-lb', '" + infraID + "-public-lb-control-plane-v4')]"),
						},
						Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
						IdleTimeoutInMinutes: to.Int32Ptr(30),
					},
					Name: to.StringPtr("api-internal-outboundrule"),
				},
			},
		},
		Name:     to.StringPtr(infraID + "-public-lb"),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: &location,
	}

	if i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		lb.LoadBalancingRules = &[]mgmtnetwork.LoadBalancingRule{
			{
				LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '" + infraID + "-public-lb', 'public-lb-ip-v4')]"),
					},
					BackendAddressPool: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '" + infraID + "-public-lb', '" + infraID + "-public-lb-control-plane-v4')]"),
					},
					Probe: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', '" + infraID + "-public-lb', 'api-internal-probe')]"),
					},
					Protocol:             mgmtnetwork.TransportProtocolTCP,
					LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
					FrontendPort:         to.Int32Ptr(6443),
					BackendPort:          to.Int32Ptr(6443),
					IdleTimeoutInMinutes: to.Int32Ptr(30),
					DisableOutboundSnat:  to.BoolPtr(true),
				},
				Name: to.StringPtr("api-internal-v4"),
			},
		}
		lb.Probes = &[]mgmtnetwork.Probe{
			{
				ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
					Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
					Port:              to.Int32Ptr(6443),
					IntervalInSeconds: to.Int32Ptr(10),
					NumberOfProbes:    to.Int32Ptr(3),
					RequestPath:       to.StringPtr("/readyz"),
				},
				Name: to.StringPtr("api-internal-probe"),
				Type: to.StringPtr("Microsoft.Network/loadBalancers/probes"),
			},
		}
	}

	return &arm.Resource{
		Resource:   lb,
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"Microsoft.Network/publicIPAddresses/" + infraID + "-pip-v4",
		},
	}
}
