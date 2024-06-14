package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) devDatabaseTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.database("parameters('databaseName')", false)...)

	t.Parameters = map[string]*arm.TemplateParameter{
		"databaseAccountName": {
			Type: "string",
		},
		"databaseName": {
			Type: "string",
		},
	}

	return t
}

func (g *generator) devSharedTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.devVPNPip(),
		g.devVnet(),
		g.devVPNVnet(),
		g.devVPN(),
		g.devLBInternal(),
		g.devDiskEncryptionKeyvault(),
		g.devDiskEncryptionKey(),
		g.devDiskEncryptionKeyVaultAccessPolicy(),
		g.devDiskEncryptionSet(),
		g.devProxyVMSS())

	t.Resources = append(t.Resources,
		g.virtualNetworkPeering("dev-vpn-vnet/peering-dev-vnet",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
			true,
			false,
			[]string{
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworkGateways', 'dev-vpn')]",
			},
		),
		g.virtualNetworkPeering("dev-vnet/peering-dev-vpn-vnet",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
			false,
			true,
			[]string{
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworkGateways', 'dev-vpn')]",
			},
		),
		g.virtualNetworkPeering("dev-vpn-vnet/peering-rp-vnet",
			"[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]",
			true,
			false,
			[]string{
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworkGateways', 'dev-vpn')]",
			},
		),
		g.virtualNetworkPeering("rp-vnet/peering-dev-vpn-vnet",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
			false,
			true,
			[]string{
				"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
				"[resourceId('Microsoft.Network/virtualNetworkGateways', 'dev-vpn')]",
			},
		))

	for _, param := range []string{
		"proxyCert",
		"proxyClientCert",
		"proxyDomainNameLabel",
		"proxyImage",
		"proxyImageAuth",
		"proxyKey",
		"publicIPAddressAllocationMethod",
		"publicIPAddressSkuName",
		"sshPublicKey",
		"vpnCACertificate",
	} {
		typ := "string"
		var defaultValue interface{}
		switch param {
		case "proxyImageAuth", "proxyKey":
			typ = "securestring"
		case "publicIPAddressAllocationMethod":
			defaultValue = "Static"
		case "publicIPAddressSkuName":
			defaultValue = "Standard"
		case "vpnCACertificate":
			defaultValue = ""
		}
		t.Parameters[param] = &arm.TemplateParameter{
			Type:         typ,
			DefaultValue: defaultValue,
		}
	}

	return t
}
