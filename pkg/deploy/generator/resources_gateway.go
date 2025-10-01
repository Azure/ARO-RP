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
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) gatewayManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     pointerutils.ToPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     pointerutils.ToPtr("[concat('aro-gateway-', resourceGroup().location)]"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) gatewaySecurityGroup() *arm.Resource {
	return g.securityGroup("gateway-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) gatewayVnet() *arm.Resource {
	return g.virtualNetwork("gateway-vnet", "10.0.8.0/24", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointerutils.ToPtr("10.0.8.0/24"),
				NetworkSecurityGroup: &armnetwork.SecurityGroup{
					ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"),
				},
				ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   pointerutils.ToPtr("Microsoft.AzureCosmosDB"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
					{
						Service:   pointerutils.ToPtr("Microsoft.ContainerRegistry"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
					{
						Service:   pointerutils.ToPtr("Microsoft.EventHub"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
					{
						Service:   pointerutils.ToPtr("Microsoft.Storage"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
					{
						Service:   pointerutils.ToPtr("Microsoft.KeyVault"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
				},
				PrivateLinkServiceNetworkPolicies: pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled),
			},
			Name: pointerutils.ToPtr("gateway-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"})
}

func (g *generator) gatewayLB() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.LoadBalancer{
			SKU: &armnetwork.LoadBalancerSKU{
				Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
			},
			Properties: &armnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &armnetwork.Subnet{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Zones: []*string{},
						Name:  pointerutils.ToPtr("gateway-frontend"),
					},
				},
				BackendAddressPools: []*armnetwork.BackendAddressPool{
					{
						Name: pointerutils.ToPtr("gateway-backend"),
					},
				},
				LoadBalancingRules: []*armnetwork.LoadBalancingRule{
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(443)),
							BackendPort:      pointerutils.ToPtr(int32(443)),
						},
						Name: pointerutils.ToPtr("gateway-lbrule-https"),
					},
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(80)),
							BackendPort:      pointerutils.ToPtr(int32(80)),
						},
						Name: pointerutils.ToPtr("gateway-lbrule-http"),
					},
				},
				Probes: []*armnetwork.Probe{
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolHTTP),
							Port:           pointerutils.ToPtr(int32(80)),
							NumberOfProbes: pointerutils.ToPtr(int32(2)),
							RequestPath:    pointerutils.ToPtr("/healthz/ready"),
						},
						Name: pointerutils.ToPtr("gateway-probe"),
					},
				},
			},
			Name:     pointerutils.ToPtr("gateway-lb-internal"),
			Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) gatewayPLS() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.PrivateLinkService{
			Properties: &armnetwork.PrivateLinkServiceProperties{
				LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
					},
				},
				IPConfigurations: []*armnetwork.PrivateLinkServiceIPConfiguration{
					{
						Properties: &armnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &armnetwork.Subnet{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Name: pointerutils.ToPtr("gateway-pls-001-nic"),
					},
				},
				EnableProxyProtocol: pointerutils.ToPtr(true),
			},
			Name:     pointerutils.ToPtr("gateway-pls-001"),
			Type:     pointerutils.ToPtr("Microsoft.Network/privateLinkServices"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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

	// VMSS extensions only support one custom script
	// Because of this, the util-*.sh scripts are prefixed to the bootstrapping script
	// main is called at the end of the bootstrapping script, so appending them will not work
	bootstrapScript := scriptUtilCommon +
		scriptUtilPackages +
		scriptUtilServices +
		scriptUtilSystem +
		scriptGatewayVMSS
	trailer := base64.StdEncoding.EncodeToString([]byte(bootstrapScript))
	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))
	customScript := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     pointerutils.ToPtr("[parameters('gatewayVmSize')]"),
				Tier:     pointerutils.ToPtr("Standard"),
				Capacity: pointerutils.ToPtr(int64(1339)),
			},
			Tags: map[string]*string{},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				// Reference: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-upgrade#arm-templates
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeAutomatic,
					RollingUpgradePolicy: &mgmtcompute.RollingUpgradePolicy{
						// Percentage equates to 1.02 instances out of 3
						MaxBatchInstancePercent:             pointerutils.ToPtr(int32(34)),
						MaxUnhealthyInstancePercent:         pointerutils.ToPtr(int32(34)),
						MaxUnhealthyUpgradedInstancePercent: pointerutils.ToPtr(int32(34)),
						PauseTimeBetweenBatches:             pointerutils.ToPtr("PT10M"),
					},
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
						ComputerNamePrefix: pointerutils.ToPtr("[concat('gateway-', parameters('vmssName'), '-')]"),
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
					StorageProfile: &mgmtcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							// cbl-mariner-2-gen2-fips is not supported by Automatic OS Updates
							// therefore the non fips image is used, and fips is configured manually
							// Reference: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-upgrade
							// https://eng.ms/docs/cloud-ai-platform/azure-core/azure-compute/compute-platform-arunki/azure-compute-artifacts/azure-compute-artifacts-docs/project-standard/1pgalleryusageinstructions#vmss-deployment-with-1p-image-galleryarm-template
							// https://eng.ms/docs/cloud-ai-platform/azure-core/core-compute-and-host/compute-platform-arunki/azure-compute-artifacts/azure-compute-artifacts-docs/project-standard/1pgalleryimagereference#cbl-mariner-2-images
							SharedGalleryImageID: pointerutils.ToPtr("/sharedGalleries/CblMariner.1P/images/cbl-mariner-2-gen2/versions/latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: pointerutils.ToPtr(int32(1024)),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: pointerutils.ToPtr("gateway-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: pointerutils.ToPtr(true),
									// disabling accelerated networking due to egress issues
									// see icm 271210960 (egress) and 274977072 (accelerated networking team)
									EnableAcceleratedNetworking: pointerutils.ToPtr(false),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: pointerutils.ToPtr("gateway-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
												},
												Primary: pointerutils.ToPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: pointerutils.ToPtr("gateway-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
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
								Name: pointerutils.ToPtr("gateway-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               pointerutils.ToPtr("Microsoft.Azure.Extensions"),
									Type:                    pointerutils.ToPtr("CustomScript"),
									TypeHandlerVersion:      pointerutils.ToPtr("2.0"),
									AutoUpgradeMinorVersion: pointerutils.ToPtr(true),
									Settings:                map[string]interface{}{},
									ProtectedSettings: map[string]interface{}{
										"script": customScript,
									},
								},
							},
							{
								// az-secmonitor package no longer needs to be manually installed
								// References:
								// 		https://eng.ms/docs/products/azure-linux/gettingstarted/aks/monitoring
								//		https://msazure.visualstudio.com/ASMDocs/_wiki/wikis/ASMDocs.wiki/179541/Linux-AzSecPack-AutoConfig-Onboarding-(manual-for-C-AI)?anchor=3.1.1-using-arm-template-resource-elements
								Name: pointerutils.ToPtr("AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               pointerutils.ToPtr("Microsoft.Azure.Monitor"),
									EnableAutomaticUpgrade:  pointerutils.ToPtr(true),
									AutoUpgradeMinorVersion: pointerutils.ToPtr(true),
									TypeHandlerVersion:      pointerutils.ToPtr("1.0"),
									Type:                    pointerutils.ToPtr("AzureMonitorLinuxAgent"),
									Settings: map[string]interface{}{
										"GCS_AUTO_CONFIG": true,
									},
								},
							},
						},
					},
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled: pointerutils.ToPtr(true),
						},
					},
					SecurityProfile: &mgmtcompute.SecurityProfile{
						// Required for 1P Image Gallery Use
						// https://eng.ms/docs/cloud-ai-platform/azure-core/azure-compute/compute-platform-arunki/azure-compute-artifacts/azure-compute-artifacts-docs/project-standard/1pgalleryusageinstructions#enable-trusted-launch-for-vmss
						SecurityType: mgmtcompute.SecurityTypesTrustedLaunch,
					},
				},
				Overprovision: pointerutils.ToPtr(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-gateway-', resourceGroup().location))]": {},
				},
			},
			Name:     pointerutils.ToPtr("[concat('gateway-vmss-', parameters('vmssName'))]"),
			Type:     pointerutils.ToPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'gateway-lb-internal')]",
		},
	}
}

func (g *generator) gatewayKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: pointerutils.ToPtr("[parameters('gatewayServicePrincipalId')]"),
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
				EnableSoftDelete: pointerutils.ToPtr(true),
				TenantID:         &tenantUUIDHack,
				Sku: &mgmtkeyvault.Sku{
					Name:   mgmtkeyvault.Standard,
					Family: pointerutils.ToPtr("A"),
				},
				AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
					{
						ObjectID: pointerutils.ToPtr(gatewayAccessPolicyHack),
					},
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('keyvaultPrefix'), '" + env.GatewayKeyvaultSuffix + "')]"),
			Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
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
