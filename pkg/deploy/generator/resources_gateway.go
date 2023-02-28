package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) gatewayManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     to.StringPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     to.StringPtr("[concat('aro-gateway-', resourceGroup().location)]"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) gatewaySecurityGroup() *arm.Resource {
	return g.securityGroup("gateway-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) gatewayVnet() *arm.Resource {
	return g.virtualNetwork("gateway-vnet", "10.0.8.0/24", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.8.0/24"),
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"),
				},
				ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.ContainerRegistry"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.EventHub"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.Storage"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.KeyVault"),
						Locations: &[]string{"*"},
					},
				},
				PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
			},
			Name: to.StringPtr("gateway-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"})
}

func (g *generator) gatewayLB() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Zones: &[]string{},
						Name:  to.StringPtr("gateway-frontend"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.StringPtr("gateway-backend"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.StringPtr("gateway-lbrule-https"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(80),
							BackendPort:      to.Int32Ptr(80),
						},
						Name: to.StringPtr("gateway-lbrule-http"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTP,
							Port:           to.Int32Ptr(80),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("gateway-probe"),
					},
				},
			},
			Name:     to.StringPtr("gateway-lb-internal"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) gatewayPLS() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PrivateLinkService{
			PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
				LoadBalancerFrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
					},
				},
				IPConfigurations: &[]mgmtnetwork.PrivateLinkServiceIPConfiguration{
					{
						PrivateLinkServiceIPConfigurationProperties: &mgmtnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Name: to.StringPtr("gateway-pls-001-nic"),
					},
				},
				EnableProxyProtocol: to.BoolPtr(true),
			},
			Name:     to.StringPtr("gateway-pls-001"),
			Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/gateway-lb-internal",
		},
	}
}

func (g *generator) gatewayVMSS() *arm.Resource {
	// TODO: there is a lot of duplication with rpVMSS()

	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{
		"acrResourceId",
		"azureCloudName",
		"azureSecPackQualysUrl",
		"azureSecPackVSATenantId",
		"databaseAccountName",
		"dbtokenClientId",
		"dbtokenUrl",
		"mdmFrontendUrl",
		"mdsdEnvironment",
		"fluentbitImage",
		"gatewayMdsdConfigVersion",
		"gatewayDomains",
		"gatewayFeatures",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"rpImage",
		"rpMdmAccount",
		"rpMdsdAccount",
		"rpMdsdNamespace",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	parts = append(parts,
		"'MDMIMAGE=''"+version.MdmImage("")+"''\n'",
	)

	parts = append(parts,
		"'LOCATION=$(base64 -d <<<'''",
		"base64(resourceGroup().location)",
		"''')\n'",
	)

	parts = append(parts,
		"'SUBSCRIPTIONID=$(base64 -d <<<'''",
		"base64(subscription().subscriptionId)",
		"''')\n'",
	)

	parts = append(parts,
		"'RESOURCEGROUPNAME=$(base64 -d <<<'''",
		"base64(resourceGroup().name)",
		"''')\n'",
	)

	trailer := base64.StdEncoding.EncodeToString(scriptGatewayVMSS)

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr("[parameters('gatewayVmSize')]"),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1339),
			},
			Tags: map[string]*string{
				"SkipLinuxAzSecPack": to.StringPtr("true"),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeManual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("[concat('gateway-', parameters('vmssName'), '-')]"),
						AdminUsername:      to.StringPtr("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							DisablePasswordAuthentication: to.BoolPtr(true),
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.StringPtr("[parameters('sshPublicKey')]"),
									},
								},
							},
						},
					},
					StorageProfile: &mgmtcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: to.StringPtr("RedHat"),
							Offer:     to.StringPtr("RHEL"),
							Sku:       to.StringPtr("8-LVM"),
							Version:   to.StringPtr("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("gateway-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									// disabling accelerated networking due to egress issues
									// see icm 271210960 (egress) and 274977072 (accelerated networking team)
									EnableAcceleratedNetworking: to.BoolPtr(false),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("gateway-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("gateway-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
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
								Name: to.StringPtr("gateway-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
									Type:                    to.StringPtr("CustomScript"),
									TypeHandlerVersion:      to.StringPtr("2.0"),
									AutoUpgradeMinorVersion: to.BoolPtr(true),
									Settings:                map[string]interface{}{},
									ProtectedSettings: map[string]interface{}{
										"script": script,
									},
								},
							},
						},
					},
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled:    to.BoolPtr(true),
							StorageURI: to.StringPtr("[concat('https://', parameters('gatewayStorageAccountDomain'), '/')]"),
						},
					},
				},
				Overprovision: to.BoolPtr(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-gateway-', resourceGroup().location))]": {},
				},
			},
			Name:     to.StringPtr("[concat('gateway-vmss-', parameters('vmssName'))]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'gateway-lb-internal')]",
			"[resourceId('Microsoft.Storage/storageAccounts', substring(parameters('gatewayStorageAccountDomain'), 0, indexOf(parameters('gatewayStorageAccountDomain'), '.')))]",
		},
	}
}

func (g *generator) gatewayKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('gatewayServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
			},
		},
	}
}

func (g *generator) gatewayKeyvault() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtkeyvault.Vault{
			Properties: &mgmtkeyvault.VaultProperties{
				EnableSoftDelete: to.BoolPtr(true),
				TenantID:         &tenantUUIDHack,
				Sku: &mgmtkeyvault.Sku{
					Name:   mgmtkeyvault.Standard,
					Family: to.StringPtr("A"),
				},
				AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
					{
						ObjectID: to.StringPtr(gatewayAccessPolicyHack),
					},
				},
			},
			Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.GatewayKeyvaultSuffix + "')]"),
			Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) gatewayRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceRoleAssignment(
			rbac.RoleNetworkContributor,
			"parameters('rpServicePrincipalId')",
			"Microsoft.Network/privateLinkServices",
			"'gateway-pls-001'",
		),
	}
}

func (g *generator) gatewayStorageAccount() *arm.Resource {
	return g.storageAccount("[substring(parameters('gatewayStorageAccountDomain'), 0, indexOf(parameters('gatewayStorageAccountDomain'), '.'))]", nil)
}
