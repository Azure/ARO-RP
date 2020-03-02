package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func GenerateDevelopmentTemplate() error {
	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.TemplateParameter{},
		Resources: []*arm.Resource{
			{
				Resource: &mgmtnetwork.PublicIPAddress{
					Sku: &mgmtnetwork.PublicIPAddressSku{
						Name: "[parameters('publicIPAddressSkuName')]",
					},
					PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
						PublicIPAllocationMethod: "[parameters('publicIPAddressAllocationMethod')]",
					},
					Name:     to.StringPtr("dev-vpn-pip"),
					Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
					Location: to.StringPtr("[resourceGroup().location]"),
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &mgmtnetwork.VirtualNetwork{
					VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
						AddressSpace: &mgmtnetwork.AddressSpace{
							AddressPrefixes: &[]string{
								"10.0.0.0/9",
							},
						},
						Subnets: &[]mgmtnetwork.Subnet{
							{
								SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
									AddressPrefix: to.StringPtr("10.0.0.0/24"),
								},
								Name: to.StringPtr("GatewaySubnet"),
							},
						},
					},
					Name:     to.StringPtr("dev-vnet"),
					Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
					Location: to.StringPtr("[resourceGroup().location]"),
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &mgmtnetwork.VirtualNetworkGateway{
					VirtualNetworkGatewayPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayPropertiesFormat{
						IPConfigurations: &[]mgmtnetwork.VirtualNetworkGatewayIPConfiguration{
							{
								VirtualNetworkGatewayIPConfigurationPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
									Subnet: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', 'GatewaySubnet')]"),
									},
									PublicIPAddress: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
									},
								},
								Name: to.StringPtr("default"),
							},
						},
						VpnType: mgmtnetwork.RouteBased,
						Sku: &mgmtnetwork.VirtualNetworkGatewaySku{
							Name: mgmtnetwork.VirtualNetworkGatewaySkuNameVpnGw1,
							Tier: mgmtnetwork.VirtualNetworkGatewaySkuTierVpnGw1,
						},
						VpnClientConfiguration: &mgmtnetwork.VpnClientConfiguration{
							VpnClientAddressPool: &mgmtnetwork.AddressSpace{
								AddressPrefixes: &[]string{"192.168.255.0/24"},
							},
							VpnClientRootCertificates: &[]mgmtnetwork.VpnClientRootCertificate{
								{
									VpnClientRootCertificatePropertiesFormat: &mgmtnetwork.VpnClientRootCertificatePropertiesFormat{
										PublicCertData: to.StringPtr("[parameters('vpnCACertificate')]"),
									},
									Name: to.StringPtr("dev-vpn-ca"),
								},
							},
							VpnClientProtocols: &[]mgmtnetwork.VpnClientProtocol{
								mgmtnetwork.OpenVPN,
							},
						},
					},
					Name:     to.StringPtr("dev-vpn"),
					Type:     to.StringPtr("Microsoft.Network/virtualNetworkGateways"),
					Location: to.StringPtr("[resourceGroup().location]"),
				},
				APIVersion: apiVersions["network"],
				DependsOn: []string{
					"[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]",
					"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
				},
			},
			proxyVmss(),
		},
	}

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
		}
		t.Parameters[param] = &arm.TemplateParameter{
			Type:         typ,
			DefaultValue: defaultValue,
		}
	}

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile(fileEnvDevelopment, b, 0666)
}
