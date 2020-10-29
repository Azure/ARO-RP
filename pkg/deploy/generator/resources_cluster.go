package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func clusterVnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.0.0/9",
					},
				},
			},
			Name:     to.StringPtr("dev-vnet"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  "[parameters('fullDeploy')]",
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func clusterRouteTable() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.RouteTable{
			Name:     to.StringPtr("[concat(parameters('clusterName'), '-rt')]"),
			Type:     to.StringPtr("Microsoft.Network/routeTables"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func clusterMasterSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('masterAddressPrefix')]"),
				ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
					{
						Service: to.StringPtr("Microsoft.ContainerRegistry"),
					},
				},
				PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
				RouteTable: &mgmtnetwork.RouteTable{
					ID: to.StringPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-master')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
			"[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]",
		},
	}
}

func clusterWorkerSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('workerAddressPrefix')]"),
				ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
					{
						Service: to.StringPtr("Microsoft.ContainerRegistry"),
					},
				},
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-worker')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', concat(parameters('clusterName'), '-master'))]",
		},
	}
}
