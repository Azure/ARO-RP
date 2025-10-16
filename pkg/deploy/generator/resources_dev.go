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
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
							},
						},
						Name: pointerutils.ToPtr("not-used"),
					},
				},
				BackendAddressPools: []*armnetwork.BackendAddressPool{
					{
						Name: pointerutils.ToPtr("dev-backend"),
					},
				},
				LoadBalancingRules: []*armnetwork.LoadBalancingRule{
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'dev-lb-internal', 'not-used')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(443)),
							BackendPort:      pointerutils.ToPtr(int32(443)),
						},
						Name: pointerutils.ToPtr("dev-lbrule"),
					},
				},
				Probes: []*armnetwork.Probe{
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
							Port:           pointerutils.ToPtr(int32(443)),
							NumberOfProbes: pointerutils.ToPtr(int32(3)),
						},
						Name: pointerutils.ToPtr("dev-probe"),
					},
				},
			},
			Name:     pointerutils.ToPtr("dev-lb-internal"),
			Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
				Tier:     pointerutils.ToPtr("Standard"),
				Capacity: pointerutils.ToPtr(int64(1)),
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
						EnableAutomaticOSUpgrade: pointerutils.ToPtr(true),
					},
				},
				// https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-instance-repairs?tabs=portal-1%2Cportal-2%2Crest-api-4%2Crest-api-5
				AutomaticRepairsPolicy: &mgmtcompute.AutomaticRepairsPolicy{
					Enabled: pointerutils.ToPtr(true),
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: pointerutils.ToPtr("dev-proxy-"),
						AdminUsername:      pointerutils.ToPtr("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							DisablePasswordAuthentication: pointerutils.ToPtr(true),
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    pointerutils.ToPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: pointerutils.ToPtr("[parameters('sshPublicKey')]"),
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
							Publisher: pointerutils.ToPtr("MicrosoftCBLMariner"),
							Offer:     pointerutils.ToPtr("azure-linux-3"),
							Sku:       pointerutils.ToPtr("azure-linux-3-gen2"),
							Version:   pointerutils.ToPtr("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: pointerutils.ToPtr(int32(64)),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: pointerutils.ToPtr("dev-proxy-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: pointerutils.ToPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: pointerutils.ToPtr("dev-proxy-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: pointerutils.ToPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: pointerutils.ToPtr("dev-proxy-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: pointerutils.ToPtr("[parameters('proxyDomainNameLabel')]"),
														},
													},
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
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
								Name: pointerutils.ToPtr("dev-proxy-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:          pointerutils.ToPtr("Microsoft.Azure.Extensions"),
									Type:               pointerutils.ToPtr("CustomScript"),
									TypeHandlerVersion: pointerutils.ToPtr("2.0"),
									ProvisionAfterExtensions: &[]string{
										"Microsoft.Azure.Monitor.AzureMonitorLinuxAgent",
										"Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent",
									},
									AutoUpgradeMinorVersion: pointerutils.ToPtr(true),
									Settings:                map[string]interface{}{},
									ProtectedSettings: map[string]interface{}{
										"script": customScript,
									},
								},
							},
							{
								Name: pointerutils.ToPtr("Microsoft.Azure.Monitor.AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               pointerutils.ToPtr("Microsoft.Azure.Monitor"),
									Type:                    pointerutils.ToPtr("AzureMonitorLinuxAgent"),
									TypeHandlerVersion:      pointerutils.ToPtr("1.0"),
									AutoUpgradeMinorVersion: pointerutils.ToPtr(true),
									EnableAutomaticUpgrade:  pointerutils.ToPtr(true),
									Settings: map[string]interface{}{
										"GCS_AUTO_CONFIG": true,
									},
								},
							},
							{
								Name: pointerutils.ToPtr("Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               pointerutils.ToPtr("Microsoft.Azure.Security.Monitoring"),
									Type:                    pointerutils.ToPtr("AzureSecurityLinuxAgent"),
									TypeHandlerVersion:      pointerutils.ToPtr("2.0"),
									AutoUpgradeMinorVersion: pointerutils.ToPtr(true),
									EnableAutomaticUpgrade:  pointerutils.ToPtr(true),
									Settings: map[string]interface{}{
										"enableGenevaUpload":               true,
										"enableAutoConfig":                 true,
										"reportSuccessOnUnsupportedDistro": true,
									},
								},
							},
						},
					},
				},
				Overprovision: pointerutils.ToPtr(false),
			},
			Name:     pointerutils.ToPtr("dev-proxy-vmss"),
			Type:     pointerutils.ToPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
			Name:     pointerutils.ToPtr("dev-vpn-pip"),
			Type:     pointerutils.ToPtr("Microsoft.Network/publicIPAddresses"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) devVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "10.0.0.0/16", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointerutils.ToPtr("10.0.1.0/24"),
				NetworkSecurityGroup: &armnetwork.SecurityGroup{
					ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
				},
			},
			Name: pointerutils.ToPtr("ToolingSubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPNVnet() *arm.Resource {
	return g.virtualNetwork("dev-vpn-vnet", "10.2.0.0/24", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointerutils.ToPtr("10.2.0.0/24"),
			},
			Name: pointerutils.ToPtr("GatewaySubnet"),
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
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vpn-vnet', 'GatewaySubnet')]"),
							},
							PublicIPAddress: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
							},
						},
						Name: pointerutils.ToPtr("default"),
					},
				},
				VPNType: pointerutils.ToPtr(armnetwork.VPNTypeRouteBased),
				SKU: &armnetwork.VirtualNetworkGatewaySKU{
					Name: pointerutils.ToPtr(armnetwork.VirtualNetworkGatewaySKUNameVPNGw1),
					Tier: pointerutils.ToPtr(armnetwork.VirtualNetworkGatewaySKUTierVPNGw1),
				},
				VPNClientConfiguration: &armnetwork.VPNClientConfiguration{
					VPNClientAddressPool: &armnetwork.AddressSpace{
						AddressPrefixes: []*string{pointerutils.ToPtr("192.168.255.0/24")},
					},
					VPNClientRootCertificates: []*armnetwork.VPNClientRootCertificate{
						{
							Properties: &armnetwork.VPNClientRootCertificatePropertiesFormat{
								PublicCertData: pointerutils.ToPtr("[parameters('vpnCACertificate')]"),
							},
							Name: pointerutils.ToPtr("dev-vpn-ca"),
						},
					},
					VPNClientProtocols: []*armnetwork.VPNClientProtocol{
						pointerutils.ToPtr(armnetwork.VPNClientProtocolOpenVPN),
					},
				},
			},
			Name:     pointerutils.ToPtr("dev-vpn"),
			Type:     pointerutils.ToPtr("Microsoft.Network/virtualNetworkGateways"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
			KeySize: pointerutils.ToPtr(int32(4096)),
		},

		Name:     pointerutils.ToPtr(fmt.Sprintf("[concat(%s, '/', %s)]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName)),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults/keys"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
				KeyURL: pointerutils.ToPtr(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', %s, %s), '%s', 'Full').properties.keyUriWithVersion]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedDiskEncryptionKeyVaultName)),
				},
			},
		},

		Name:     pointerutils.ToPtr(fmt.Sprintf("[%s]", sharedDiskEncryptionSetName)),
		Type:     pointerutils.ToPtr("Microsoft.Compute/diskEncryptionSets"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
					ObjectID: pointerutils.ToPtr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", sharedDiskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
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

		Name:     pointerutils.ToPtr(fmt.Sprintf("[concat(%s, '/add')]", sharedDiskEncryptionKeyVaultName)),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", sharedDiskEncryptionSetName)},
	}
}
