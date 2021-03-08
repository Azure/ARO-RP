package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (g *generator) clusterVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "[parameters('vnetAddressPrefix')]", nil, "[parameters('ci')]", nil)
}

func (g *generator) clusterRouteTable() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.RouteTable{
			Name:     to.StringPtr("[concat(parameters('clusterName'), '-rt')]"),
			Type:     to.StringPtr("Microsoft.Network/routeTables"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) clusterMasterSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('masterAddressPrefix')]"),
				RouteTable: &mgmtnetwork.RouteTable{
					ID: to.StringPtr("[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]"),
				},
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-master')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
			"[resourceid('Microsoft.Network/routeTables', concat(parameters('clusterName'), '-rt'))]",
		},
	}
}

func (g *generator) clusterWorkerSubnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Subnet{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('workerAddressPrefix')]"),
			},
			Name: to.StringPtr("[concat('dev-vnet/', parameters('clusterName'), '-worker')]"),
		},
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Location:   "[resourceGroup().location]",
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceid('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', concat(parameters('clusterName'), '-master'))]",
		},
	}
}
