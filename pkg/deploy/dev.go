package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
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
				Resource: &network.PublicIPAddress{
					Sku: &network.PublicIPAddressSku{
						Name: network.PublicIPAddressSkuNameStandard,
					},
					PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
						PublicIPAllocationMethod: network.Static,
					},
					Name:     to.StringPtr("dev-vpn-pip"),
					Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
					Location: to.StringPtr("[resourceGroup().location]"),
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &network.VirtualNetwork{
					VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
						AddressSpace: &network.AddressSpace{
							AddressPrefixes: &[]string{
								"10.0.0.0/9",
							},
						},
						Subnets: &[]network.Subnet{
							{
								SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
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
				Resource: &network.VirtualNetworkGateway{
					VirtualNetworkGatewayPropertiesFormat: &network.VirtualNetworkGatewayPropertiesFormat{
						IPConfigurations: &[]network.VirtualNetworkGatewayIPConfiguration{
							{
								VirtualNetworkGatewayIPConfigurationPropertiesFormat: &network.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
									Subnet: &network.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', 'GatewaySubnet')]"),
									},
									PublicIPAddress: &network.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
									},
								},
								Name: to.StringPtr("default"),
							},
						},
						VpnType: network.RouteBased,
						Sku: &network.VirtualNetworkGatewaySku{
							Name: network.VirtualNetworkGatewaySkuNameVpnGw1,
							Tier: network.VirtualNetworkGatewaySkuTierVpnGw1,
						},
						VpnClientConfiguration: &network.VpnClientConfiguration{
							VpnClientAddressPool: &network.AddressSpace{
								AddressPrefixes: &[]string{"192.168.255.0/24"},
							},
							VpnClientRootCertificates: &[]network.VpnClientRootCertificate{
								{
									VpnClientRootCertificatePropertiesFormat: &network.VpnClientRootCertificatePropertiesFormat{
										PublicCertData: to.StringPtr("[parameters('vpnCACertificate')]"),
									},
									Name: to.StringPtr("dev-vpn-ca"),
								},
							},
							VpnClientProtocols: &[]network.VpnClientProtocol{
								network.OpenVPN,
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
		"sshPublicKey",
		"vpnCACertificate",
	} {
		typ := "string"
		switch param {
		case "proxyImageAuth", "proxyKey":
			typ = "securestring"
		}
		t.Parameters[param] = &arm.TemplateParameter{Type: typ}
	}

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile("env-development.json", b, 0666)
}
