package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-12-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// devLBInternal is needed for defining a healthprobe.
// VMSS with auto upgrademode requires a healthprobe from an LB.
func (g *generator) devLBInternal() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.LoadBalancer{
			SKU: &armnetwork.LoadBalancerSKU{
				Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameBasic),
			},
			Properties: &armnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &armnetwork.Subnet{
								ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
							},
						},
						Name: new("not-used"),
					},
				},
				BackendAddressPools: []*armnetwork.BackendAddressPool{
					{
						Name: new("dev-backend"),
					},
				},
				LoadBalancingRules: []*armnetwork.LoadBalancingRule{
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'dev-lb-internal', 'not-used')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     new(int32(443)),
							BackendPort:      new(int32(443)),
						},
						Name: new("dev-lbrule"),
					},
				},
				Probes: []*armnetwork.Probe{
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
							Port:           new(int32(443)),
							NumberOfProbes: new(int32(3)),
						},
						Name: new("dev-probe"),
					},
				},
			},
			Name:     new("dev-lb-internal"),
			Type:     new("Microsoft.Network/loadBalancers"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) devProxyVMSS() *arm.Resource {
	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{"proxyImage", "proxyImageAuth"} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	for _, variable := range []string{"proxyCert", "proxyClientCert", "proxyKey"} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
		)
	}

	trailer := base64.StdEncoding.EncodeToString([]byte(scriptDevProxyVMSS))
	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))
	customScript := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     pointerutils.ToPtr(string(mgmtcompute.VirtualMachineSizeTypesStandardF2sV2)),
				Tier:     new("Standard"),
				Capacity: new(int64(1)),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('AzSecPackAutoConfigRG', 'Microsoft.ManagedIdentity/userAssignedIdentities', 'AzSecPackAutoConfigUA-eastus')]": {},
				},
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeRolling,
					AutomaticOSUpgradePolicy: &mgmtcompute.AutomaticOSUpgradePolicy{
						EnableAutomaticOSUpgrade: new(true),
					},
				},
				// https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-instance-repairs?tabs=portal-1%2Cportal-2%2Crest-api-4%2Crest-api-5
				AutomaticRepairsPolicy: &mgmtcompute.AutomaticRepairsPolicy{
					Enabled: new(true),
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: new("dev-proxy-"),
						AdminUsername:      new("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							DisablePasswordAuthentication: new(true),
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    new("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: new("[parameters('sshPublicKey')]"),
									},
								},
							},
						},
					},
					SecurityProfile: &mgmtcompute.SecurityProfile{
						SecurityType: mgmtcompute.SecurityTypesTrustedLaunch,
					},
					StorageProfile: &mgmtcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: new("MicrosoftCBLMariner"),
							Offer:     new("cbl-mariner"),
							Sku:       new("cbl-mariner-2-gen2"),
							Version:   new("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: new(int32(64)),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: new("dev-proxy-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: new(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: new("dev-proxy-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: new(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: new("dev-proxy-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: new("[parameters('proxyDomainNameLabel')]"),
														},
													},
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					ExtensionProfile: &mgmtcompute.VirtualMachineScaleSetExtensionProfile{
						Extensions: &[]mgmtcompute.VirtualMachineScaleSetExtension{
							{
								Name: new("dev-proxy-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:          new("Microsoft.Azure.Extensions"),
									Type:               new("CustomScript"),
									TypeHandlerVersion: new("2.0"),
									ProvisionAfterExtensions: &[]string{
										"Microsoft.Azure.Monitor.AzureMonitorLinuxAgent",
										"Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent",
									},
									AutoUpgradeMinorVersion: new(true),
									Settings:                map[string]any{},
									ProtectedSettings: map[string]any{
										"script": customScript,
									},
								},
							},
							{
								Name: new("Microsoft.Azure.Monitor.AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               new("Microsoft.Azure.Monitor"),
									Type:                    new("AzureMonitorLinuxAgent"),
									TypeHandlerVersion:      new("1.0"),
									AutoUpgradeMinorVersion: new(true),
									EnableAutomaticUpgrade:  new(true),
									Settings: map[string]any{
										"GCS_AUTO_CONFIG": true,
									},
								},
							},
							{
								Name: new("Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               new("Microsoft.Azure.Security.Monitoring"),
									Type:                    new("AzureSecurityLinuxAgent"),
									TypeHandlerVersion:      new("2.0"),
									AutoUpgradeMinorVersion: new(true),
									EnableAutomaticUpgrade:  new(true),
									Settings: map[string]any{
										"enableGenevaUpload":               true,
										"enableAutoConfig":                 true,
										"reportSuccessOnUnsupportedDistro": true,
									},
								},
							},
						},
					},
				},
				Overprovision: new(false),
			},
			Name:     new("dev-proxy-vmss"),
			Type:     new("Microsoft.Compute/virtualMachineScaleSets"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		Tags: map[string]any{
			"azsecpack": "nonprod",
		},
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'dev-lb-internal')]",
		},
	}
}

func (g *generator) devVPNPip() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.PublicIPAddress{
			SKU: &armnetwork.PublicIPAddressSKU{
				Name: pointerutils.ToPtr(armnetwork.PublicIPAddressSKUName("[parameters('publicIPAddressSkuName')]")),
			},
			Properties: &armnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethod("[parameters('publicIPAddressAllocationMethod')]")),
			},
			Name:     new("dev-vpn-pip"),
			Type:     new("Microsoft.Network/publicIPAddresses"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) devVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "10.0.0.0/16", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: new("10.0.1.0/24"),
				NetworkSecurityGroup: &armnetwork.SecurityGroup{
					ID: new("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
				},
			},
			Name: new("ToolingSubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPNVnet() *arm.Resource {
	return g.virtualNetwork("dev-vpn-vnet", "10.2.0.0/24", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: new("10.2.0.0/24"),
			},
			Name: new("GatewaySubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPN() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.VirtualNetworkGateway{
			Properties: &armnetwork.VirtualNetworkGatewayPropertiesFormat{
				IPConfigurations: []*armnetwork.VirtualNetworkGatewayIPConfiguration{
					{
						Properties: &armnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
							Subnet: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vpn-vnet', 'GatewaySubnet')]"),
							},
							PublicIPAddress: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
							},
						},
						Name: new("default"),
					},
				},
				VPNType: pointerutils.ToPtr(armnetwork.VPNTypeRouteBased),
				SKU: &armnetwork.VirtualNetworkGatewaySKU{
					Name: pointerutils.ToPtr(armnetwork.VirtualNetworkGatewaySKUNameVPNGw1),
					Tier: pointerutils.ToPtr(armnetwork.VirtualNetworkGatewaySKUTierVPNGw1),
				},
				VPNClientConfiguration: &armnetwork.VPNClientConfiguration{
					VPNClientAddressPool: &armnetwork.AddressSpace{
						AddressPrefixes: []*string{new("192.168.255.0/24")},
					},
					VPNClientRootCertificates: []*armnetwork.VPNClientRootCertificate{
						{
							Properties: &armnetwork.VPNClientRootCertificatePropertiesFormat{
								PublicCertData: new("[parameters('vpnCACertificate')]"),
							},
							Name: new("dev-vpn-ca"),
						},
					},
					VPNClientProtocols: []*armnetwork.VPNClientProtocol{
						pointerutils.ToPtr(armnetwork.VPNClientProtocolOpenVPN),
					},
				},
			},
			Name:     new("dev-vpn"),
			Type:     new("Microsoft.Network/virtualNetworkGateways"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vpn-vnet')]",
		},
	}
}

const (
	sharedDiskEncryptionKeyVaultName       = "concat(take(resourceGroup().name,10), '" + SharedDiskEncryptionKeyVaultNameSuffix + "')"
	sharedDiskEncryptionSetName            = "concat(resourceGroup().name, '" + SharedDiskEncryptionSetNameSuffix + "')"
	sharedDiskEncryptionKeyName            = "concat(resourceGroup().name, '-disk-encryption-key')"
	SharedDiskEncryptionKeyVaultNameSuffix = "-dev-disk-enc"
	SharedDiskEncryptionSetNameSuffix      = "-disk-encryption-set"
)

// shared keyvault for keys used for disk encryption sets when creating clusters locally
func (g *generator) devDiskEncryptionKeyvault() *arm.Resource {
	return g.keyVault(fmt.Sprintf("[%s]", sharedDiskEncryptionKeyVaultName), &[]mgmtkeyvault.AccessPolicyEntry{}, nil, false, nil)
}

func (g *generator) devDiskEncryptionKey() *arm.Resource {
	key := &mgmtkeyvault.Key{
		KeyProperties: &mgmtkeyvault.KeyProperties{
			Kty:     mgmtkeyvault.RSA,
			KeySize: new(int32(4096)),
		},

		Name:     new(fmt.Sprintf("[concat(%s, '/', %s)]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName)),
		Type:     new("Microsoft.KeyVault/vaults/keys"),
		Location: new("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   key,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedDiskEncryptionKeyVaultName)},
	}
}

func (g *generator) devDiskEncryptionSet() *arm.Resource {
	diskEncryptionSet := &mgmtcompute.DiskEncryptionSet{
		EncryptionSetProperties: &mgmtcompute.EncryptionSetProperties{
			ActiveKey: &mgmtcompute.KeyForDiskEncryptionSet{
				KeyURL: new(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', %s, %s), '%s', 'Full').properties.keyUriWithVersion]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: new(fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedDiskEncryptionKeyVaultName)),
				},
			},
		},

		Name:     new(fmt.Sprintf("[%s]", sharedDiskEncryptionSetName)),
		Type:     new("Microsoft.Compute/diskEncryptionSets"),
		Location: new("[resourceGroup().location]"),
		Identity: &mgmtcompute.EncryptionSetIdentity{Type: mgmtcompute.DiskEncryptionSetIdentityTypeSystemAssigned},
	}

	return &arm.Resource{
		Resource:   diskEncryptionSet,
		APIVersion: azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"),
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults/keys', %s, %s)]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName),
		},
	}
}

func (g *generator) devDiskEncryptionKeyVaultAccessPolicy() *arm.Resource {
	accessPolicy := &mgmtkeyvault.VaultAccessPolicyParameters{
		Properties: &mgmtkeyvault.VaultAccessPolicyProperties{
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: new(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", sharedDiskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
					Permissions: &mgmtkeyvault.Permissions{
						Keys: &[]mgmtkeyvault.KeyPermissions{
							mgmtkeyvault.KeyPermissionsGet,
							mgmtkeyvault.KeyPermissionsWrapKey,
							mgmtkeyvault.KeyPermissionsUnwrapKey,
						},
					},
				},
			},
		},

		Name:     new(fmt.Sprintf("[concat(%s, '/add')]", sharedDiskEncryptionKeyVaultName)),
		Type:     new("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: new("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", sharedDiskEncryptionSetName)},
	}
}
