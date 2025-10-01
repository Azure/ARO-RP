package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-12-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) rpManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     pointerutils.ToPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     pointerutils.ToPtr("[concat('aro-rp-', resourceGroup().location)]"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) rpSecurityGroupForPortalSourceAddressPrefixes() *arm.Resource {
	return g.securityRules("rp-nsg/portal_in", &armnetwork.SecurityRulePropertiesFormat{
		Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
		SourcePortRange:          pointerutils.ToPtr("*"),
		DestinationPortRange:     pointerutils.ToPtr("444"),
		SourceAddressPrefixes:    []*string{},
		DestinationAddressPrefix: pointerutils.ToPtr("*"),
		Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
		Priority:                 pointerutils.ToPtr(int32(142)),
		Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
	}, "[not(empty(parameters('rpNsgPortalSourceAddressPrefixes')))]")
}

func (g *generator) rpSecurityGroup() *arm.Resource {
	rules := []*armnetwork.SecurityRule{
		{
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
				SourcePortRange:          pointerutils.ToPtr("*"),
				DestinationPortRange:     pointerutils.ToPtr("443"),
				SourceAddressPrefix:      pointerutils.ToPtr("AzureResourceManager"),
				DestinationAddressPrefix: pointerutils.ToPtr("*"),
				Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
				Priority:                 pointerutils.ToPtr(int32(120)),
				Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
			},
			Name: pointerutils.ToPtr("rp_in_arm"),
		},
		{
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
				SourcePortRange:          pointerutils.ToPtr("*"),
				DestinationPortRange:     pointerutils.ToPtr("443"),
				SourceAddressPrefix:      pointerutils.ToPtr("GenevaActions"),
				DestinationAddressPrefix: pointerutils.ToPtr("*"),
				Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
				Priority:                 pointerutils.ToPtr(int32(130)),
				Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
			},
			Name: pointerutils.ToPtr("rp_in_geneva"),
		},
	}

	if !g.production {
		// override production ARM flag for more open configuration in development
		rules[0].Properties.SourceAddressPrefix = pointerutils.ToPtr("*")

		rules = append(rules, &armnetwork.SecurityRule{
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
				SourcePortRange:          pointerutils.ToPtr("*"),
				DestinationPortRange:     pointerutils.ToPtr("22"),
				SourceAddressPrefix:      pointerutils.ToPtr("*"),
				DestinationAddressPrefix: pointerutils.ToPtr("*"),
				Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessAllow),
				Priority:                 pointerutils.ToPtr(int32(125)),
				Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
			},
			Name: pointerutils.ToPtr("ssh_in"),
		})
	} else {
		rules = append(rules,
			&armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					Protocol:                 pointerutils.ToPtr(armnetwork.SecurityRuleProtocolTCP),
					SourcePortRange:          pointerutils.ToPtr("*"),
					DestinationPortRange:     pointerutils.ToPtr("*"),
					SourceAddressPrefix:      pointerutils.ToPtr("10.0.8.0/24"),
					DestinationAddressPrefix: pointerutils.ToPtr("*"),
					Access:                   pointerutils.ToPtr(armnetwork.SecurityRuleAccessDeny),
					Priority:                 pointerutils.ToPtr(int32(145)),
					Direction:                pointerutils.ToPtr(armnetwork.SecurityRuleDirectionInbound),
				},
				Name: pointerutils.ToPtr("deny_in_gateway"),
			},
		)
	}

	return g.securityGroup("rp-nsg", rules, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpPESecurityGroup() *arm.Resource {
	return g.securityGroup("rp-pe-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpVnet() *arm.Resource {
	addressPrefix := "10.1.0.0/24"
	if g.production {
		addressPrefix = "10.0.0.0/24"
	}

	subnet := &armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: pointerutils.ToPtr(addressPrefix),
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
			},
			ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
				{
					Service:   pointerutils.ToPtr("Microsoft.Storage"),
					Locations: []*string{pointerutils.ToPtr("*")},
				},
			},
		},
		Name: pointerutils.ToPtr("rp-subnet"),
	}

	if g.production {
		subnet.Properties.ServiceEndpoints = append(subnet.Properties.ServiceEndpoints, []*armnetwork.ServiceEndpointPropertiesFormat{
			{
				Service:   pointerutils.ToPtr("Microsoft.KeyVault"),
				Locations: []*string{pointerutils.ToPtr("*")},
			},
			{
				Service:   pointerutils.ToPtr("Microsoft.AzureCosmosDB"),
				Locations: []*string{pointerutils.ToPtr("*")},
			},
		}...)
	}

	return g.virtualNetwork("rp-vnet", addressPrefix, []*armnetwork.Subnet{subnet}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"})
}

func (g *generator) rpPEVnet() *arm.Resource {
	return g.virtualNetwork("rp-pe-vnet-001", "10.0.4.0/22", []*armnetwork.Subnet{
		{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: pointerutils.ToPtr("10.0.4.0/22"),
				NetworkSecurityGroup: &armnetwork.SecurityGroup{
					ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"),
				},
				PrivateEndpointNetworkPolicies: pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
				ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   pointerutils.ToPtr("Microsoft.Storage"),
						Locations: []*string{pointerutils.ToPtr("*")},
					},
				},
			},
			Name: pointerutils.ToPtr("rp-pe-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"})
}

func (g *generator) rpLB() *arm.Resource {
	return &arm.Resource{
		Resource: &armnetwork.LoadBalancer{
			SKU: &armnetwork.LoadBalancerSKU{
				Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
			},
			Properties: &armnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
					{
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &armnetwork.PublicIPAddress{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]"),
							},
						},
						Name: pointerutils.ToPtr("rp-frontend"),
					},
					{
						Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &armnetwork.PublicIPAddress{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'portal-pip')]"),
							},
						},
						Name: pointerutils.ToPtr("portal-frontend"),
					},
				},
				BackendAddressPools: []*armnetwork.BackendAddressPool{
					{
						Name: pointerutils.ToPtr("rp-backend"),
					},
				},
				LoadBalancingRules: []*armnetwork.LoadBalancingRule{
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'rp-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(443)),
							BackendPort:      pointerutils.ToPtr(int32(443)),
						},
						Name: pointerutils.ToPtr("rp-lbrule"),
					},
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-https')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(443)),
							BackendPort:      pointerutils.ToPtr(int32(444)),
						},
						Name: pointerutils.ToPtr("portal-lbrule"),
					},
					{
						Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &armnetwork.SubResource{
								ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-ssh')]"),
							},
							Protocol:         pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
							LoadDistribution: pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
							FrontendPort:     pointerutils.ToPtr(int32(22)),
							BackendPort:      pointerutils.ToPtr(int32(2222)),
						},
						Name: pointerutils.ToPtr("portal-lbrule-ssh"),
					},
				},
				Probes: []*armnetwork.Probe{
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolHTTPS),
							Port:           pointerutils.ToPtr(int32(443)),
							NumberOfProbes: pointerutils.ToPtr(int32(2)),
							RequestPath:    pointerutils.ToPtr("/healthz/ready"),
						},
						Name: pointerutils.ToPtr("rp-probe"),
					},
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolHTTPS),
							Port:           pointerutils.ToPtr(int32(444)),
							NumberOfProbes: pointerutils.ToPtr(int32(2)),
							RequestPath:    pointerutils.ToPtr("/healthz/ready"),
						},
						Name: pointerutils.ToPtr("portal-probe-https"),
					},
					{
						Properties: &armnetwork.ProbePropertiesFormat{
							Protocol:       pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
							Port:           pointerutils.ToPtr(int32(2222)),
							NumberOfProbes: pointerutils.ToPtr(int32(2)),
						},
						Name: pointerutils.ToPtr("portal-probe-ssh"),
					},
				},
			},
			Name:     pointerutils.ToPtr("rp-lb"),
			Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'portal-pip')]",
			"[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]",
		},
	}
}

// rpLBAlert generates an alert resource for the rp-lb healthprobe metric
func (g *generator) rpLBAlert(threshold float64, severity int32, name string, evalFreq string, windowSize string, metric string) *arm.Resource {
	return &arm.Resource{
		Resource: mgmtinsights.MetricAlertResource{
			MetricAlertProperties: &mgmtinsights.MetricAlertProperties{
				Actions: &[]mgmtinsights.MetricAlertAction{
					{
						ActionGroupID: pointerutils.ToPtr("[resourceId(parameters('subscriptionResourceGroupName'), 'Microsoft.Insights/actionGroups', 'rp-health-ag')]"),
					},
				},
				Enabled:             pointerutils.ToPtr(true),
				EvaluationFrequency: pointerutils.ToPtr(evalFreq),
				Severity:            pointerutils.ToPtr(severity),
				Scopes: &[]string{
					"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
				},
				WindowSize:         pointerutils.ToPtr(windowSize),
				TargetResourceType: pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
				AutoMitigate:       pointerutils.ToPtr(true),
				Criteria: mgmtinsights.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf: &[]mgmtinsights.MetricCriteria{
						{
							CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
							MetricName:      pointerutils.ToPtr(metric),
							MetricNamespace: pointerutils.ToPtr("microsoft.network/loadBalancers"),
							Name:            pointerutils.ToPtr("HealthProbeCheck"),
							Operator:        mgmtinsights.OperatorLessThan,
							Threshold:       pointerutils.ToPtr(threshold),
							TimeAggregation: mgmtinsights.Average,
						},
					},
					OdataType: mgmtinsights.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     pointerutils.ToPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     pointerutils.ToPtr("Microsoft.Insights/metricAlerts"),
			Location: pointerutils.ToPtr("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) rpVMSS() *arm.Resource {
	// TODO: there is a lot of duplication with gatewayVMSS() (and other places)

	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{
		"acrResourceId",
		"adminApiClientCertCommonName",
		"armApiClientCertCommonName",
		"armClientId",
		"azureCloudName",
		"azureSecPackQualysUrl",
		"azureSecPackVSATenantId",
		"clusterMdmAccount",
		"clusterMdsdAccount",
		"clusterMdsdConfigVersion",
		"clusterMdsdNamespace",
		"clusterParentDomainName",
		"databaseAccountName",
		"fluentbitImage",
		"fpClientId",
		"fpTenantId",
		"fpServicePrincipalId",
		"gatewayDomains",
		"gatewayResourceGroupName",
		"gatewayServicePrincipalId",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"mdmFrontendUrl",
		"mdsdEnvironment",
		"msiRpEndpoint",
		"portalAccessGroupIds",
		"portalClientId",
		"portalElevatedGroupIds",
		"rpFeatures",
		"rpImage",
		"rpMdmAccount",
		"rpMdsdAccount",
		"rpMdsdConfigVersion",
		"rpMdsdNamespace",
		"rpParentDomainName",
		"oidcStorageAccountName",
		"otelAuditQueueSize",
		"tokenContributorRoleID",
		"tokenContributorRoleName",

		// TODO: Replace with Live Service Configuration in KeyVault
		"clustersInstallViaHive",
		"clustersAdoptByHive",
		"clusterDefaultInstallerPullspec",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	// convert array variables to string using ARM string() function to be passed via customScript later
	for _, variable := range []string{
		"miseValidAudiences",
		"miseValidAppIDs",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(string(parameters('%s')))", variable),
			"''')\n'",
		)
	}

	for _, variable := range []string{
		"adminApiCaBundle",
		"armApiCaBundle",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
		)
	}

	parts = append(parts,
		"'MDMIMAGE=''"+version.MdmImage("")+"''\n'",
	)

	parts = append(parts,
		"'OTELIMAGE=''"+version.OTelImage("")+"''\n'",
	)

	parts = append(parts,
		"'MISEIMAGE=''"+version.MiseImage("")+"''\n'",
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
		scriptRpVMSS
	trailer := base64.StdEncoding.EncodeToString([]byte(bootstrapScript))
	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))
	customScript := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     pointerutils.ToPtr("[parameters('vmSize')]"),
				Tier:     pointerutils.ToPtr("Standard"),
				Capacity: pointerutils.ToPtr(int64(1338)),
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
						ComputerNamePrefix: pointerutils.ToPtr("[concat('rp-', parameters('vmssName'), '-')]"),
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
						// https://eng.ms/docs/products/azure-linux/gettingstarted/azurevm/azurevm
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
							ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: pointerutils.ToPtr("rp-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: pointerutils.ToPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: pointerutils.ToPtr("rp-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: pointerutils.ToPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: pointerutils.ToPtr("rp-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
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
								Name: pointerutils.ToPtr("rp-vmss-cse"),
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
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-rp-', resourceGroup().location))]": {},
				},
			},
			Name:     pointerutils.ToPtr("[concat('rp-vmss-', parameters('vmssName'))]"),
			Type:     pointerutils.ToPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Authorization/roleAssignments', guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader'))]",
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) rpParentDNSZone() *arm.Resource {
	return g.dnsZone("[parameters('rpParentDomainName')]")
}

func (g *generator) rpClusterParentDNSZone() *arm.Resource {
	return g.dnsZone("[parameters('clusterParentDomainName')]")
}

func (g *generator) rpDNSZone() *arm.Resource {
	return g.dnsZone("[concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))]")
}

func (g *generator) rpClusterKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: pointerutils.ToPtr("[parameters('fpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
				Certificates: &[]mgmtkeyvault.CertificatePermissions{
					mgmtkeyvault.Create,
					mgmtkeyvault.Delete,
					mgmtkeyvault.Get,
					mgmtkeyvault.Update,
				},
			},
		},
	}
}

func (g *generator) rpPortalKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: pointerutils.ToPtr("[parameters('rpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
			},
		},
	}
}

func (g *generator) rpServiceKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: pointerutils.ToPtr("[parameters('rpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
					mgmtkeyvault.SecretPermissionsList,
				},
			},
		},
	}
}

func (g *generator) rpClusterKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: pointerutils.ToPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: pointerutils.ToPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: pointerutils.ToPtr(clusterAccessPolicyHack),
				},
			},
		},
		Name:     pointerutils.ToPtr("[concat(parameters('keyvaultPrefix'), '" + env.ClusterKeyvaultSuffix + "')]"),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpClusterKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: pointerutils.ToPtr("[parameters('adminObjectId')]"),
				Permissions: &mgmtkeyvault.Permissions{
					Certificates: &[]mgmtkeyvault.CertificatePermissions{
						mgmtkeyvault.Get,
						mgmtkeyvault.List,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpPortalKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: pointerutils.ToPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: pointerutils.ToPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: pointerutils.ToPtr(portalAccessPolicyHack),
				},
			},
		},
		Name:     pointerutils.ToPtr("[concat(parameters('keyvaultPrefix'), '" + env.PortalKeyvaultSuffix + "')]"),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpPortalKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: pointerutils.ToPtr("[parameters('adminObjectId')]"),
				Permissions: &mgmtkeyvault.Permissions{
					Certificates: &[]mgmtkeyvault.CertificatePermissions{
						mgmtkeyvault.Delete,
						mgmtkeyvault.Get,
						mgmtkeyvault.Import,
						mgmtkeyvault.List,
					},
					Secrets: &[]mgmtkeyvault.SecretPermissions{
						mgmtkeyvault.SecretPermissionsSet,
						mgmtkeyvault.SecretPermissionsList,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpServiceKeyvaultDynamic() *arm.Resource {
	vaultAccessPoliciesResource := &arm.DeploymentTemplateResource{
		Name:       "[concat(parameters('keyvaultPrefix'), '" + env.ServiceKeyvaultSuffix + "/add')]",
		Type:       "Microsoft.KeyVault/vaults/accessPolicies",
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault/vaults/accessPolicies"),
		Properties: &mgmtkeyvault.VaultProperties{
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: pointerutils.ToPtr("[reference(resourceId('Microsoft.ContainerService/managedClusters', 'aro-aks-cluster-001'), '2020-12-01', 'Full').properties.identityProfile.kubeletidentity.objectId]"),
					Permissions: &mgmtkeyvault.Permissions{
						Secrets: &[]mgmtkeyvault.SecretPermissions{
							mgmtkeyvault.SecretPermissionsGet,
							mgmtkeyvault.SecretPermissionsList,
						},
						Certificates: &[]mgmtkeyvault.CertificatePermissions{
							mgmtkeyvault.Get,
						},
					},
				},
			},
		},
	}

	rpServiceKeyvaultDynamicDeployment := &arm.Deployment{
		Properties: &arm.DeploymentProperties{
			Template: &arm.DeploymentTemplate{
				Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
				ContentVersion: "1.0.0.0",
				Parameters: map[string]*arm.TemplateParameter{
					"keyvaultPrefix": {
						Type: "string",
					},
				},
				Resources: []*arm.DeploymentTemplateResource{vaultAccessPoliciesResource},
			},
			Parameters: map[string]*arm.DeploymentTemplateResourceParameter{
				"keyvaultPrefix": {
					Value: "[parameters('keyvaultPrefix')]",
				},
			},
			Mode: "Incremental",
			ExpressionEvaluationOptions: map[string]*string{
				"scope": pointerutils.ToPtr("inner"),
			},
		},
	}

	return &arm.Resource{
		Name:       "rpServiceKeyvaultDynamic",
		Type:       "Microsoft.Resources/deployments",
		APIVersion: azureclient.APIVersion("Microsoft.Resources/deployments"),
		DependsOn:  []string{"[concat(parameters('keyvaultPrefix'), '" + env.ServiceKeyvaultSuffix + "')]"},
		Resource:   rpServiceKeyvaultDynamicDeployment,
	}
}

func (g *generator) rpServiceKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: pointerutils.ToPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: pointerutils.ToPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: pointerutils.ToPtr(serviceAccessPolicyHack),
				},
			},
		},
		Name:     pointerutils.ToPtr("[concat(parameters('keyvaultPrefix'), '" + env.ServiceKeyvaultSuffix + "')]"),
		Type:     pointerutils.ToPtr("Microsoft.KeyVault/vaults"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpServiceKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: pointerutils.ToPtr("[parameters('adminObjectId')]"),
				Permissions: &mgmtkeyvault.Permissions{
					Certificates: &[]mgmtkeyvault.CertificatePermissions{
						mgmtkeyvault.Delete,
						mgmtkeyvault.Get,
						mgmtkeyvault.Import,
						mgmtkeyvault.List,
					},
					Secrets: &[]mgmtkeyvault.SecretPermissions{
						mgmtkeyvault.SecretPermissionsGet,
						mgmtkeyvault.SecretPermissionsSet,
						mgmtkeyvault.SecretPermissionsList,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpCosmosDB() []*arm.Resource {
	dbType := sdkcosmos.DatabaseAccountKindGlobalDocumentDB
	consistency := sdkcosmos.DefaultConsistencyLevelStrong
	backupPolicy := sdkcosmos.BackupPolicyTypePeriodic
	minTLSVersion := sdkcosmos.MinimalTLSVersionTls12

	cosmosdb := &sdkcosmos.DatabaseAccountCreateUpdateParameters{
		Kind: &dbType,
		Properties: &sdkcosmos.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy: &sdkcosmos.ConsistencyPolicy{
				DefaultConsistencyLevel: &consistency,
			},
			Locations: []*sdkcosmos.Location{
				{
					LocationName: pointerutils.ToPtr("[resourceGroup().location]"),
				},
			},
			DatabaseAccountOfferType: pointerutils.ToPtr("Standard"),
			BackupPolicy: &sdkcosmos.PeriodicModeBackupPolicy{
				Type: &backupPolicy,
				PeriodicModeProperties: &sdkcosmos.PeriodicModeProperties{
					BackupIntervalInMinutes:        pointerutils.ToPtr(int32(240)), //4 hours
					BackupRetentionIntervalInHours: pointerutils.ToPtr(int32(720)), //30 days
				},
			},
			MinimalTLSVersion: &minTLSVersion,
			DisableLocalAuth:  pointerutils.ToPtr(true), // Disable local authentication
		},
		Name:     pointerutils.ToPtr("[parameters('databaseAccountName')]"),
		Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts"),
		Location: pointerutils.ToPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			"defaultExperience": pointerutils.ToPtr("Core (SQL)"),
		},
	}

	r := &arm.Resource{
		Resource:   cosmosdb,
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
	}
	if g.production {
		cosmosdb.Properties.IPRules = []*sdkcosmos.IPAddressOrRange{}
		cosmosdb.Properties.IsVirtualNetworkFilterEnabled = pointerutils.ToPtr(true)
		cosmosdb.Properties.VirtualNetworkRules = []*sdkcosmos.VirtualNetworkRule{}
		cosmosdb.Properties.DisableKeyBasedMetadataWriteAccess = pointerutils.ToPtr(true)
	}

	rs := []*arm.Resource{
		r,
	}

	if g.production {
		rs = append(rs, g.database("'ARO'", true)...)
		rs = append(rs, g.rpCosmosDBAlert(10, 90, 3, "rp-cosmosdb-alert", "PT5M", "PT1H"))
		rs = append(rs, g.CosmosDBDataContributorRoleAssignment("'ARO'", "rp"))
		rs = append(rs, g.CosmosDBDataContributorRoleAssignment("'ARO'", "gateway"))
		rs = append(rs, g.CosmosDBDataContributorRoleAssignment("'ARO'", "globalDevops"))
	} else {
		rs = append(rs, g.CosmosDBDataContributorRoleAssignment("''", "rp"))
		rs = append(rs, g.CosmosDBDataContributorRoleAssignment("'ARO'", "globalDevops"))
	}

	return rs
}

func (g *generator) CosmosDBDataContributorRoleAssignment(databaseName, component string) *arm.Resource {
	var scope string
	if g.production {
		scope = "[resourceId('Microsoft.DocumentDB/databaseAccounts/dbs', parameters('databaseAccountName'), " + databaseName + ")]"
	} else {
		scope = "[resourceId('Microsoft.DocumentDB/databaseAccounts/', parameters('databaseAccountName'))]"
	}

	roleAssignment := &arm.Resource{
		Condition: pointerutils.ToPtr("[not(equals(parameters('" + component + "ServicePrincipalId'), ''))]"),
		Resource: mgmtauthorization.RoleAssignment{
			Name: pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('" + component + "ServicePrincipalId'), 'DocumentDB Data Contributor'))]"),
			Type: pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlRoleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            &scope,
				RoleDefinitionID: pointerutils.ToPtr("[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlRoleDefinitions', parameters('databaseAccountName'), '" + rbac.RoleDocumentDBDataContributor + "')]"),
				PrincipalID:      pointerutils.ToPtr("[parameters('" + component + "ServicePrincipalId')]"),
				PrincipalType:    mgmtauthorization.ServicePrincipal,
			},
		},
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
	}
	return roleAssignment
}

func (g *generator) database(databaseName string, addDependsOn bool) []*arm.Resource {
	database := &arm.Resource{
		Resource: &sdkcosmos.SQLDatabaseCreateUpdateParameters{
			Properties: &sdkcosmos.SQLDatabaseCreateUpdateProperties{
				Resource: &sdkcosmos.SQLDatabaseResource{
					ID: pointerutils.ToPtr("[" + databaseName + "]"),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: pointerutils.ToPtr(int32(cosmosDbStandardProvisionedThroughputHack)),
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
			Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
	}
	hashPartitionKey := sdkcosmos.PartitionKindHash
	portal := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: pointerutils.ToPtr("Portal"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							pointerutils.ToPtr("/id"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: pointerutils.ToPtr(int32(-1)),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: pointerutils.ToPtr(int32(cosmosDbPortalProvisionedThroughputHack)),
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Portal')]"),
			Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
	}

	gateway := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: pointerutils.ToPtr("Gateway"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							pointerutils.ToPtr("/id"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: pointerutils.ToPtr(int32(-1)),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: pointerutils.ToPtr(int32(cosmosDbGatewayProvisionedThroughputHack)),
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Gateway')]"),
			Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
	}

	mimo := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: pointerutils.ToPtr("MaintenanceManifests"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							pointerutils.ToPtr("/clusterResourceID"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: pointerutils.ToPtr(int32(-1)),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: pointerutils.ToPtr(int32(cosmosDbGatewayProvisionedThroughputHack)),
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/MaintenanceManifests')]"),
			Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
	}

	if !g.production {
		database.Resource.(*sdkcosmos.SQLDatabaseCreateUpdateParameters).Properties.Options = &sdkcosmos.CreateUpdateOptions{
			AutoscaleSettings: &sdkcosmos.AutoscaleSettings{
				MaxThroughput: pointerutils.ToPtr(int32(1000)),
			},
		}
		portal.Resource.(*sdkcosmos.SQLContainerCreateUpdateParameters).Properties.Options = &sdkcosmos.CreateUpdateOptions{}
		gateway.Resource.(*sdkcosmos.SQLContainerCreateUpdateParameters).Properties.Options = &sdkcosmos.CreateUpdateOptions{}
		mimo.Resource.(*sdkcosmos.SQLContainerCreateUpdateParameters).Properties.Options = &sdkcosmos.CreateUpdateOptions{}
	}

	rs := []*arm.Resource{
		database,
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("AsyncOperations"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: pointerutils.ToPtr(int32(7 * 86400)), // 7 days
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/AsyncOperations')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("OpenShiftVersions"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: pointerutils.ToPtr(int32(-1)),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftVersions')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("PlatformWorkloadIdentityRoleSets"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: pointerutils.ToPtr(int32(-1)),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/PlatformWorkloadIdentityRoleSets')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("Billing"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Billing')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		gateway,
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("Monitors"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: pointerutils.ToPtr(int32(-1)),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Monitors')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("OpenShiftClusters"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/partitionKey"),
							},
							Kind: &hashPartitionKey,
						},
						UniqueKeyPolicy: &sdkcosmos.UniqueKeyPolicy{
							UniqueKeys: []*sdkcosmos.UniqueKey{
								{
									Paths: []*string{
										pointerutils.ToPtr("/key"),
									},
								},
								{
									Paths: []*string{
										pointerutils.ToPtr("/clusterResourceGroupIdKey"),
									},
								},
								{
									Paths: []*string{
										pointerutils.ToPtr("/clientIdKey"),
									},
								},
							},
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftClusters')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		portal,
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: pointerutils.ToPtr("Subscriptions"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								pointerutils.ToPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: pointerutils.ToPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		mimo,
	}

	// Adding Triggers
	rs = append(rs,
		// Subscription
		g.rpCosmosDBTriggers(databaseName, "Subscriptions", "renewLease", renewLeaseTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
		g.rpCosmosDBTriggers(databaseName, "Subscriptions", "retryLater", retryLaterTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
		// Billing
		g.rpCosmosDBTriggers(databaseName, "Billing", "setCreationBillingTimeStamp", setCreationBillingTimeStampTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationCreate),
		g.rpCosmosDBTriggers(databaseName, "Billing", "setDeletionBillingTimeStamp", setDeletionBillingTimeStampTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationReplace),
		// OpenShiftClusters
		g.rpCosmosDBTriggers(databaseName, "OpenShiftClusters", "renewLease", renewLeaseTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
		// Monitors
		g.rpCosmosDBTriggers(databaseName, "Monitors", "renewLease", renewLeaseTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
		// MIMO DB triggers
		g.rpCosmosDBTriggers(databaseName, "MaintenanceManifests", "renewLease", renewLeaseTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
	)

	if addDependsOn {
		for i := range rs {
			rs[i].DependsOn = append(rs[i].DependsOn,
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			)
		}
	}

	return rs
}

func (g *generator) rpCosmosDBTriggers(databaseName, containerName, triggerID, triggerFunction string, triggerType sdkcosmos.TriggerType, triggerOperation sdkcosmos.TriggerOperation) *arm.Resource {
	return &arm.Resource{
		Resource: &sdkcosmos.SQLTriggerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLTriggerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLTriggerResource{
					ID:               pointerutils.ToPtr(triggerID),
					Body:             pointerutils.ToPtr(triggerFunction),
					TriggerOperation: &triggerOperation,
					TriggerType:      &triggerType,
				},
			},
			Name:     pointerutils.ToPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/" + containerName + "/" + triggerID + "')]"),
			Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/triggers"),
			Location: pointerutils.ToPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers', parameters('databaseAccountName'), " + databaseName + ", '" + containerName + "')]",
		},
	}
}

func (g *generator) rpCosmosDBAlert(throttledRequestThreshold float64, ruConsumptionThreshold float64, severity int32, name string, evalFreq string, windowSize string) *arm.Resource {
	throttledRequestMetricCriteria := mgmtinsights.MetricCriteria{
		CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
		MetricName:      pointerutils.ToPtr("TotalRequests"),
		MetricNamespace: pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts"),
		Name:            pointerutils.ToPtr("ThrottledRequestCheck"),
		Operator:        mgmtinsights.OperatorGreaterThan,
		Threshold:       pointerutils.ToPtr(throttledRequestThreshold),
		TimeAggregation: mgmtinsights.Count,
		Dimensions: &[]mgmtinsights.MetricDimension{
			{
				Name:     pointerutils.ToPtr("StatusCode"),
				Operator: pointerutils.ToPtr("Include"),
				Values:   &[]string{"429"},
			},
		},
	}

	ruConsumptionMetricCriteria := mgmtinsights.MetricCriteria{
		CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
		MetricName:      pointerutils.ToPtr("NormalizedRUConsumption"),
		MetricNamespace: pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts"),
		Name:            pointerutils.ToPtr("RUConsumptionCheck"),
		Operator:        mgmtinsights.OperatorGreaterThan,
		Threshold:       pointerutils.ToPtr(ruConsumptionThreshold),
		TimeAggregation: mgmtinsights.Average,
	}

	return &arm.Resource{
		Resource: mgmtinsights.MetricAlertResource{
			MetricAlertProperties: &mgmtinsights.MetricAlertProperties{
				Actions: &[]mgmtinsights.MetricAlertAction{
					{
						ActionGroupID: pointerutils.ToPtr("[resourceId(parameters('subscriptionResourceGroupName'), 'Microsoft.Insights/actionGroups', 'rp-health-ag')]"),
					},
				},
				Enabled:             pointerutils.ToPtr(true),
				EvaluationFrequency: pointerutils.ToPtr(evalFreq),
				Severity:            pointerutils.ToPtr(severity),
				Scopes: &[]string{
					"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
				},
				WindowSize:         pointerutils.ToPtr(windowSize),
				TargetResourceType: pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts"),
				AutoMitigate:       pointerutils.ToPtr(true),
				Criteria: mgmtinsights.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf:     &[]mgmtinsights.MetricCriteria{throttledRequestMetricCriteria, ruConsumptionMetricCriteria},
					OdataType: mgmtinsights.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     pointerutils.ToPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     pointerutils.ToPtr("Microsoft.Insights/metricAlerts"),
			Location: pointerutils.ToPtr("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
		},
	}
}

func (g *generator) rpRoleDefinitionTokenContributor() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtauthorization.RoleDefinition{
			Name: pointerutils.ToPtr("[parameters('tokenContributorRoleID')]"),
			Type: pointerutils.ToPtr("Microsoft.Authorization/roleDefinitions"),
			RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
				RoleName:         pointerutils.ToPtr("[parameters('tokenContributorRoleName')]"),
				AssignableScopes: &[]string{"[subscription().id]"},
				Permissions: &[]mgmtauthorization.Permission{
					{
						Actions: &[]string{
							"Microsoft.ContainerRegistry/registries/generateCredentials/action",
							"Microsoft.ContainerRegistry/registries/scopeMaps/read",
							"Microsoft.ContainerRegistry/registries/tokens/delete",
							"Microsoft.ContainerRegistry/registries/tokens/operationStatuses/read",
							"Microsoft.ContainerRegistry/registries/tokens/read",
							"Microsoft.ContainerRegistry/registries/tokens/write",
						},
					},
				},
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization/roleDefinitions"),
	}
}

func (g *generator) rpRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceGroupRoleAssignmentWithName(
			rbac.RoleReader,
			"parameters('rpServicePrincipalId')",
			"guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader')",
		),
		rbac.ResourceGroupRoleAssignmentWithName(
			rbac.RoleNetworkContributor,
			"parameters('fpServicePrincipalId')",
			"guid(resourceGroup().id, 'FP / Network Contributor')",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleDocumentDBAccountContributor,
			"parameters('rpServicePrincipalId')",
			"Microsoft.DocumentDB/databaseAccounts",
			"parameters('databaseAccountName')",
			"concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('rpServicePrincipalId'), 'RP / DocumentDB Account Contributor'))",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleDNSZoneContributor,
			"parameters('fpServicePrincipalId')",
			"Microsoft.Network/dnsZones",
			"concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))",
			"concat(resourceGroup().location, '.', parameters('clusterParentDomainName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/dnsZones', concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))), 'FP / DNS Zone Contributor'))",
		),
	}
}

func (g *generator) rpACR() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Registry{
			Sku: &mgmtcontainerregistry.Sku{
				Name: mgmtcontainerregistry.Premium,
			},
			RegistryProperties: &mgmtcontainerregistry.RegistryProperties{
				// enable data hostname stability: https://azure.microsoft.com/en-gb/blog/azure-container-registry-mitigating-data-exfiltration-with-dedicated-data-endpoints/
				DataEndpointEnabled: pointerutils.ToPtr(true),
			},
			Name: pointerutils.ToPtr("[substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))]"),
			Type: pointerutils.ToPtr("Microsoft.ContainerRegistry/registries"),
			// TODO: INT ACR has wrong location - should be redeployed at globalResourceGroupLocation then remove acrLocationOverride configurable.
			Location: pointerutils.ToPtr("[if(equals(parameters('acrLocationOverride'), ''), resourceGroup().location, parameters('acrLocationOverride'))]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ContainerRegistry"),
	}
}

func (g *generator) rpACRReplica() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Replication{
			Name:     pointerutils.ToPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', parameters('location'))]"),
			Type:     pointerutils.ToPtr("Microsoft.ContainerRegistry/registries/replications"),
			Location: pointerutils.ToPtr("[parameters('location')]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ContainerRegistry"),
	}
}

func (g *generator) rpACRRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleACRPull,
			"parameters('rpServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), parameters('rpServicePrincipalId'), 'RP / AcrPull')))",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleACRPull,
			"parameters('gatewayServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), parameters('gatewayServicePrincipalId'), 'RP / AcrPull')))",
		),
		rbac.ResourceRoleAssignmentWithName(
			"parameters('tokenContributorRoleID')",
			"parameters('fpServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), 'FP / ARO v4 ContainerRegistry Token Contributor')))",
		),
	}
}

func (g *generator) rpVersionStorageAccount() []*arm.Resource {
	storageAccountName := "parameters('rpVersionStorageAccountName')"
	return []*arm.Resource{
		g.storageAccount(
			fmt.Sprintf("[%s]", storageAccountName),
			&mgmtstorage.AccountProperties{
				AllowBlobPublicAccess: pointerutils.ToPtr(false),
				MinimumTLSVersion:     mgmtstorage.MinimumTLSVersionTLS12,
				AllowSharedKeyAccess:  pointerutils.ToPtr(false),
			},
			map[string]*string{},
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleStorageAccountContributor,
			"parameters('globalDevopsServicePrincipalId')",
			resourceTypeStorageAccount,
			storageAccountName,
			fmt.Sprintf("concat(%s, '/Microsoft.Authorization/', guid(resourceId('%s', %s)))", storageAccountName, resourceTypeStorageAccount, storageAccountName),
			"[not(equals(parameters('globalDevopsServicePrincipalId'), ''))]",
		),
		g.storageAccountBlobContainer(
			fmt.Sprintf("concat(%s, '/default', '/$web')", storageAccountName),
			storageAccountName,
			&mgmtstorage.ContainerProperties{},
		),
		rbac.ResourceRoleAssignmentWithScope(
			rbac.RoleStorageBlobDataContributor,
			"parameters('globalDevopsServicePrincipalId')",
			fmt.Sprintf("%s/%s", resourceTypeStorageAccount, resourceTypeBlobContainer),
			fmt.Sprintf("concat(resourceId('Microsoft.Storage/storageAccounts', %s), '/blobServices/default/containers/$web')", storageAccountName),
			fmt.Sprintf("concat(%s, '/default/$web/Microsoft.Authorization/', guid(%s))", storageAccountName, storageAccountName),
			"[not(equals(parameters('globalDevopsServicePrincipalId'), ''))]",
		),
	}
}
