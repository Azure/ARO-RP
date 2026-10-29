package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-12-01/compute"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) gatewayManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     new("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     new("[concat('aro-gateway-', resourceGroup().location)]"),
			Location: new("[resourceGroup().location]"),
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
				AddressPrefix: new("10.0.8.0/24"),
				NetworkSecurityGroup: &armnetwork.SecurityGroup{
					ID: new("[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"),
				},
				ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   new("Microsoft.AzureCosmosDB"),
						Locations: []*string{new("*")},
					},
					{
						Service:   new("Microsoft.ContainerRegistry"),
						Locations: []*string{new("*")},
					},
					{
						Service:   new("Microsoft.EventHub"),
						Locations: []*string{new("*")},
					},
					{
						Service:   new("Microsoft.Storage"),
						Locations: []*string{new("*")},
					},
					{
						Service:   new("Microsoft.KeyVault"),
						Locations: []*string{new("*")},
					},
				},
				PrivateLinkServiceNetworkPolicies: new(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled),
			},
			Name: new("gateway-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"})
}

func (g *generator) gatewayLB() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.LoadBalancer{
			SKU: &armnetwork.LoadBalancerSKU{
				Name: new(armnetwork.LoadBalancerSKUNameStandard),
			},
			Properties: &armnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &armnetwork.Subnet{
								ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Zones: []*string{},
						Name:  new("gateway-frontend"),
					},
				},
				BackendAddressPools: []*armnetwork.BackendAddressPool{
					{
						Name: new("gateway-backend"),
					},
				},
				LoadBalancingRules: []*armnetwork.LoadBalancingRule{
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         new(armnetwork.TransportProtocolTCP),
							LoadDistribution: new(armnetwork.LoadDistributionDefault),
							FrontendPort:     new(int32(443)),
							BackendPort:      new(int32(443)),
						},
						Name: new("gateway-lbrule-https"),
					},
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         new(armnetwork.TransportProtocolTCP),
							LoadDistribution: new(armnetwork.LoadDistributionDefault),
							FrontendPort:     new(int32(80)),
							BackendPort:      new(int32(80)),
						},
						Name: new("gateway-lbrule-http"),
					},
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe-otel')]"),
							},
							Protocol:         new(armnetwork.TransportProtocolTCP),
							LoadDistribution: new(armnetwork.LoadDistributionDefault),
							FrontendPort:     new(int32(4317)),
							BackendPort:      new(int32(4317)),
						},
						Name: new("gateway-lbrule-otel"),
					},
				},
				Probes: []*armnetwork.Probe{
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       new(armnetwork.ProbeProtocolHTTP),
							Port:           new(int32(80)),
							NumberOfProbes: new(int32(2)),
							RequestPath:    new("/healthz/ready"),
						},
						Name: new("gateway-probe"),
					},
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       new(armnetwork.ProbeProtocolTCP),
							Port:           new(int32(13133)),
							NumberOfProbes: new(int32(2)),
						},
						Name: new("gateway-probe-otel"),
					},
				},
			},
			Name:     new("gateway-lb-internal"),
			Type:     new("Microsoft.Network/loadBalancers"),
			Location: new("[resourceGroup().location]"),
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
						ID: new("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
					},
				},
				IPConfigurations: []*armnetwork.PrivateLinkServiceIPConfiguration{
					{
						Properties: &armnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &armnetwork.Subnet{
								ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Name: new("gateway-pls-001-nic"),
					},
				},
				EnableProxyProtocol: new(true),
			},
			Name:     new("gateway-pls-001"),
			Type:     new("Microsoft.Network/privateLinkServices"),
			Location: new("[resourceGroup().location]"),
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
		"clusterMdsdAccount",
		"clusterMdsdNamespace",
		"databaseAccountName",
		"otelClusterMdsdConfigVersion",
		"environment",
		"fluentbitImage",
		"gatewayDomains",
		"gatewayFeatures",
		"gatewayLogLevel",
		"gatewayMdsdConfigVersion",
		"gatewayOtelCollectorImage",
		"gatewayOtelKustoIngestionEndpoint",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"mdmFrontendUrl",
		"mdsdEnvironment",
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
		"'CLUSTERMDSDIMAGE=''"+version.MdsdImage("")+"''\n'",
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

	parts = append(parts,
		"'GATEWAYUSERASSIGNEDIDENTITYRESOURCEID=$(base64 -d <<<'''",
		"base64(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-gateway-', resourceGroup().location)))",
		"''')\n'",
	)

	parts = append(parts,
		"'GATEWAYCLIENTID=$(base64 -d <<<'''",
		"base64(parameters('gatewayClientId'))",
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

	customScript, err := g.createStartupScript("gateway", parts, bootstrapScript)
	if err != nil {
		panic(err)
	}

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     new("[parameters('gatewayVmSize')]"),
				Tier:     new("Standard"),
				Capacity: new(int64(1339)),
			},
			Tags: map[string]*string{
				"AMA_Tenant_AROClusterLogs": new("/etc/amatenants/AROClusterLogs"),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				// Reference: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-upgrade#arm-templates
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeAutomatic,
					RollingUpgradePolicy: &mgmtcompute.RollingUpgradePolicy{
						// Percentage equates to 1.02 instances out of 3
						MaxBatchInstancePercent:             new(int32(34)),
						MaxUnhealthyInstancePercent:         new(int32(34)),
						MaxUnhealthyUpgradedInstancePercent: new(int32(34)),
						PauseTimeBetweenBatches:             new("PT10M"),
					},
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
						ComputerNamePrefix: new("[concat('gateway-', parameters('vmssName'), '-')]"),
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
					StorageProfile: &mgmtcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							SharedGalleryImageID: new("/sharedGalleries/CblMariner.1P/images/azure-linux-3-gen2-fips/versions/latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: new(int32(1024)),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: new("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: new("gateway-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: new(true),
									// disabling accelerated networking due to egress issues
									// see icm 271210960 (egress) and 274977072 (accelerated networking team)
									EnableAcceleratedNetworking: new(false),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: new("gateway-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: new("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
												},
												Primary: new(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: new("gateway-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														IPTags: &[]mgmtcompute.VirtualMachineScaleSetIPTag{},
													},
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: new("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
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
								Name: new("gateway-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               new("Microsoft.Azure.Extensions"),
									Type:                    new("CustomScript"),
									TypeHandlerVersion:      new("2.0"),
									AutoUpgradeMinorVersion: new(true),
									Settings:                map[string]any{},
									ProtectedSettings: map[string]any{
										"script": customScript,
									},
								},
							},
							{
								// az-secmonitor package no longer needs to be manually installed
								// References:
								// 		https://eng.ms/docs/products/azure-linux/gettingstarted/aks/monitoring
								//		https://msazure.visualstudio.com/ASMDocs/_wiki/wikis/ASMDocs.wiki/179541/Linux-AzSecPack-AutoConfig-Onboarding-(manual-for-C-AI)?anchor=3.1.1-using-arm-template-resource-elements
								Name: new("AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               new("Microsoft.Azure.Monitor"),
									EnableAutomaticUpgrade:  new(true),
									AutoUpgradeMinorVersion: new(true),
									TypeHandlerVersion:      new("1.0"),
									Type:                    new("AzureMonitorLinuxAgent"),
									Settings: map[string]any{
										"genevaConfiguration": map[string]any{
											"enable": true,
										},
									},
								},
							},
						},
					},
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled: new(true),
						},
					},
					SecurityProfile: &mgmtcompute.SecurityProfile{
						// Required for 1P Image Gallery Use
						// https://eng.ms/docs/cloud-ai-platform/azure-core/azure-compute/compute-platform-arunki/azure-compute-artifacts/azure-compute-artifacts-docs/project-standard/1pgalleryusageinstructions#enable-trusted-launch-for-vmss
						SecurityType: mgmtcompute.SecurityTypesTrustedLaunch,
					},
				},
				Overprovision: new(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-gateway-', resourceGroup().location))]": {},
				},
			},
			Name:     new("[concat('gateway-vmss-', parameters('vmssName'))]"),
			Type:     new("Microsoft.Compute/virtualMachineScaleSets"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'gateway-lb-internal')]",
		},
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

func (g *generator) gatewayVMSSAutoscale() *arm.Resource {
	vmssID := "[resourceId('Microsoft.Compute/virtualMachineScaleSets', concat('gateway-vmss-', parameters('vmssName')))]"

	cpuTrigger := func(op mgmtinsights.ComparisonOperationType, threshold float64) *mgmtinsights.MetricTrigger {
		return &mgmtinsights.MetricTrigger{
			MetricName:        new("Percentage CPU"),
			MetricNamespace:   new("microsoft.compute/virtualmachinescalesets"),
			MetricResourceURI: new(vmssID),
			TimeGrain:         new("PT1M"),
			Statistic:         mgmtinsights.MetricStatisticTypeAverage,
			TimeWindow:        new("PT5M"),
			TimeAggregation:   mgmtinsights.TimeAggregationTypeAverage,
			Operator:          op,
			Threshold:         new(threshold),
		}
	}

	scaleAction := func(dir mgmtinsights.ScaleDirection) *mgmtinsights.ScaleAction {
		return &mgmtinsights.ScaleAction{
			Direction: dir,
			Type:      mgmtinsights.ChangeCount,
			Value:     new("1"),
			Cooldown:  new("PT5M"),
		}
	}

	return &arm.Resource{
		Resource: mgmtinsights.AutoscaleSettingResource{
			AutoscaleSetting: &mgmtinsights.AutoscaleSetting{
				Name:              new("[concat('gateway-vmss-', parameters('vmssName'), '-autoscale')]"),
				Enabled:           new(true),
				TargetResourceURI: new(vmssID),
				Profiles: &[]mgmtinsights.AutoscaleProfile{
					{
						Name: new("gateway-cpu-autoscale"),
						Capacity: &mgmtinsights.ScaleCapacity{
							Minimum: new("3"),
							Maximum: new("7"),
							Default: new("3"),
						},
						Rules: &[]mgmtinsights.ScaleRule{
							{
								MetricTrigger: cpuTrigger(mgmtinsights.GreaterThan, 66),
								ScaleAction:   scaleAction(mgmtinsights.ScaleDirectionIncrease),
							},
							{
								MetricTrigger: cpuTrigger(mgmtinsights.LessThan, 40),
								ScaleAction:   scaleAction(mgmtinsights.ScaleDirectionDecrease),
							},
						},
					},
				},
			},
			Name:     new("[concat('gateway-vmss-', parameters('vmssName'), '-autoscale')]"),
			Type:     new("Microsoft.Insights/autoscaleSettings"),
			Location: new("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights/autoscaleSettings"),
		DependsOn: []string{
			vmssID,
		},
	}
}
