package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-12-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// devLBInternal is needed for defining a healthprobe.
// VMSS with auto upgrademode requires a healthprobe from an LB.
func (g *generator) devLBInternal() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameBasic,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.Ptr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
							},
						},
						Name: to.Ptr("not-used"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.Ptr("dev-backend"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.Ptr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'dev-lb-internal', 'not-used')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.Ptr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.Ptr("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.Ptr("dev-lbrule"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolTCP,
							Port:           to.Int32Ptr(443),
							NumberOfProbes: to.Int32Ptr(3),
						},
						Name: to.Ptr("dev-probe"),
					},
				},
			},
			Name:     to.Ptr("dev-lb-internal"),
			Type:     to.Ptr("Microsoft.Network/loadBalancers"),
			Location: to.Ptr("[resourceGroup().location]"),
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
				Name:     to.Ptr(string(mgmtcompute.VirtualMachineSizeTypesStandardF2sV2)),
				Tier:     to.Ptr("Standard"),
				Capacity: to.Int64Ptr(1),
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
						EnableAutomaticOSUpgrade: to.Ptr(true),
					},
				},
				// https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-instance-repairs?tabs=portal-1%2Cportal-2%2Crest-api-4%2Crest-api-5
				AutomaticRepairsPolicy: &mgmtcompute.AutomaticRepairsPolicy{
					Enabled: to.Ptr(true),
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.Ptr("dev-proxy-"),
						AdminUsername:      to.Ptr("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							DisablePasswordAuthentication: to.Ptr(true),
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    to.Ptr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.Ptr("[parameters('sshPublicKey')]"),
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
							Publisher: to.Ptr("MicrosoftCBLMariner"),
							Offer:     to.Ptr("cbl-mariner"),
							Sku:       to.Ptr("cbl-mariner-2-gen2"),
							Version:   to.Ptr("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: to.Int32Ptr(64),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: to.Ptr("[resourceId('Microsoft.Network/loadBalancers/probes', 'dev-lb-internal', 'dev-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.Ptr("dev-proxy-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.Ptr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.Ptr("dev-proxy-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.Ptr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.Ptr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.Ptr("dev-proxy-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: to.Ptr("[parameters('proxyDomainNameLabel')]"),
														},
													},
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: to.Ptr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'dev-lb-internal', 'dev-backend')]"),
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
								Name: to.Ptr("dev-proxy-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:          to.Ptr("Microsoft.Azure.Extensions"),
									Type:               to.Ptr("CustomScript"),
									TypeHandlerVersion: to.Ptr("2.0"),
									ProvisionAfterExtensions: &[]string{
										"Microsoft.Azure.Monitor.AzureMonitorLinuxAgent",
										"Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent",
									},
									AutoUpgradeMinorVersion: to.Ptr(true),
									Settings:                map[string]interface{}{},
									ProtectedSettings: map[string]interface{}{
										"script": customScript,
									},
								},
							},
							{
								Name: to.Ptr("Microsoft.Azure.Monitor.AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.Ptr("Microsoft.Azure.Monitor"),
									Type:                    to.Ptr("AzureMonitorLinuxAgent"),
									TypeHandlerVersion:      to.Ptr("1.0"),
									AutoUpgradeMinorVersion: to.Ptr(true),
									EnableAutomaticUpgrade:  to.Ptr(true),
									Settings: map[string]interface{}{
										"GCS_AUTO_CONFIG": true,
									},
								},
							},
							{
								Name: to.Ptr("Microsoft.Azure.Security.Monitoring.AzureSecurityLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.Ptr("Microsoft.Azure.Security.Monitoring"),
									Type:                    to.Ptr("AzureSecurityLinuxAgent"),
									TypeHandlerVersion:      to.Ptr("2.0"),
									AutoUpgradeMinorVersion: to.Ptr(true),
									EnableAutomaticUpgrade:  to.Ptr(true),
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
				Overprovision: to.Ptr(false),
			},
			Name:     to.Ptr("dev-proxy-vmss"),
			Type:     to.Ptr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.Ptr("[resourceGroup().location]"),
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
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: "[parameters('publicIPAddressSkuName')]",
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: "[parameters('publicIPAddressAllocationMethod')]",
			},
			Name:     to.Ptr("dev-vpn-pip"),
			Type:     to.Ptr("Microsoft.Network/publicIPAddresses"),
			Location: to.Ptr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) devVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "10.0.0.0/16", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.Ptr("10.0.1.0/24"),
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.Ptr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
				},
			},
			Name: to.Ptr("ToolingSubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPNVnet() *arm.Resource {
	return g.virtualNetwork("dev-vpn-vnet", "10.2.0.0/24", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.Ptr("10.2.0.0/24"),
			},
			Name: to.Ptr("GatewaySubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPN() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetworkGateway{
			VirtualNetworkGatewayPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayPropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.VirtualNetworkGatewayIPConfiguration{
					{
						VirtualNetworkGatewayIPConfigurationPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
							Subnet: &mgmtnetwork.SubResource{
								ID: to.Ptr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vpn-vnet', 'GatewaySubnet')]"),
							},
							PublicIPAddress: &mgmtnetwork.SubResource{
								ID: to.Ptr("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
							},
						},
						Name: to.Ptr("default"),
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
								PublicCertData: to.Ptr("[parameters('vpnCACertificate')]"),
							},
							Name: to.Ptr("dev-vpn-ca"),
						},
					},
					VpnClientProtocols: &[]mgmtnetwork.VpnClientProtocol{
						mgmtnetwork.OpenVPN,
					},
				},
			},
			Name:     to.Ptr("dev-vpn"),
			Type:     to.Ptr("Microsoft.Network/virtualNetworkGateways"),
			Location: to.Ptr("[resourceGroup().location]"),
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
			KeySize: to.Int32Ptr(4096),
		},

		Name:     to.Ptr(fmt.Sprintf("[concat(%s, '/', %s)]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName)),
		Type:     to.Ptr("Microsoft.KeyVault/vaults/keys"),
		Location: to.Ptr("[resourceGroup().location]"),
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
				KeyURL: to.Ptr(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', %s, %s), '%s', 'Full').properties.keyUriWithVersion]", sharedDiskEncryptionKeyVaultName, sharedDiskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: to.Ptr(fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedDiskEncryptionKeyVaultName)),
				},
			},
		},

		Name:     to.Ptr(fmt.Sprintf("[%s]", sharedDiskEncryptionSetName)),
		Type:     to.Ptr("Microsoft.Compute/diskEncryptionSets"),
		Location: to.Ptr("[resourceGroup().location]"),
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
					ObjectID: to.Ptr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", sharedDiskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
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

		Name:     to.Ptr(fmt.Sprintf("[concat(%s, '/add')]", sharedDiskEncryptionKeyVaultName)),
		Type:     to.Ptr("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: to.Ptr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", sharedDiskEncryptionSetName)},
	}
}
