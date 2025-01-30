package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) rpManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     to.StringPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     to.StringPtr("[concat('aro-rp-', resourceGroup().location)]"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) rpSecurityGroupForPortalSourceAddressPrefixes() *arm.Resource {
	return g.securityRules("rp-nsg/portal_in", &mgmtnetwork.SecurityRulePropertiesFormat{
		Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
		SourcePortRange:          to.StringPtr("*"),
		DestinationPortRange:     to.StringPtr("444"),
		SourceAddressPrefixes:    &[]string{},
		DestinationAddressPrefix: to.StringPtr("*"),
		Access:                   mgmtnetwork.SecurityRuleAccessAllow,
		Priority:                 to.Int32Ptr(142),
		Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
	}, "[not(empty(parameters('rpNsgPortalSourceAddressPrefixes')))]")
}

func (g *generator) rpSecurityGroup() *arm.Resource {
	rules := []mgmtnetwork.SecurityRule{
		{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("443"),
				SourceAddressPrefix:      to.StringPtr("AzureResourceManager"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(120),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("rp_in_arm"),
		},
		{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("443"),
				SourceAddressPrefix:      to.StringPtr("GenevaActions"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(130),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("rp_in_geneva"),
		},
	}

	if !g.production {
		// override production ARM flag for more open configuration in development
		rules[0].SecurityRulePropertiesFormat.SourceAddressPrefix = to.StringPtr("*")

		rules = append(rules, mgmtnetwork.SecurityRule{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(125),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("ssh_in"),
		})
	} else {
		rules = append(rules,
			mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("*"),
					SourceAddressPrefix:      to.StringPtr("10.0.8.0/24"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessDeny,
					Priority:                 to.Int32Ptr(145),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("deny_in_gateway"),
			},
		)
	}

	return g.securityGroup("rp-nsg", &rules, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpPESecurityGroup() *arm.Resource {
	return g.securityGroup("rp-pe-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpVnet() *arm.Resource {
	addressPrefix := "10.1.0.0/24"
	if g.production {
		addressPrefix = "10.0.0.0/24"
	}

	subnet := mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr(addressPrefix),
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
			},
			ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
				{
					Service:   to.StringPtr("Microsoft.Storage"),
					Locations: &[]string{"*"},
				},
			},
		},
		Name: to.StringPtr("rp-subnet"),
	}

	if g.production {
		*subnet.ServiceEndpoints = append(*subnet.ServiceEndpoints, []mgmtnetwork.ServiceEndpointPropertiesFormat{
			{
				Service:   to.StringPtr("Microsoft.KeyVault"),
				Locations: &[]string{"*"},
			},
			{
				Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
				Locations: &[]string{"*"},
			},
		}...)
	}

	return g.virtualNetwork("rp-vnet", addressPrefix, &[]mgmtnetwork.Subnet{subnet}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"})
}

func (g *generator) rpPEVnet() *arm.Resource {
	return g.virtualNetwork("rp-pe-vnet-001", "10.0.4.0/22", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.4.0/22"),
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"),
				},
				PrivateEndpointNetworkPolicies: to.StringPtr("Disabled"),
				ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   to.StringPtr("Microsoft.Storage"),
						Locations: &[]string{"*"},
					},
				},
			},
			Name: to.StringPtr("rp-pe-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"})
}

func (g *generator) rpLB() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &mgmtnetwork.PublicIPAddress{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]"),
							},
						},
						Name: to.StringPtr("rp-frontend"),
					},
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &mgmtnetwork.PublicIPAddress{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'portal-pip')]"),
							},
						},
						Name: to.StringPtr("portal-frontend"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.StringPtr("rp-backend"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'rp-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.StringPtr("rp-lbrule"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-https')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(444),
						},
						Name: to.StringPtr("portal-lbrule"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-ssh')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(22),
							BackendPort:      to.Int32Ptr(2222),
						},
						Name: to.StringPtr("portal-lbrule-ssh"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTPS,
							Port:           to.Int32Ptr(443),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("rp-probe"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTPS,
							Port:           to.Int32Ptr(444),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("portal-probe-https"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolTCP,
							Port:           to.Int32Ptr(2222),
							NumberOfProbes: to.Int32Ptr(2),
						},
						Name: to.StringPtr("portal-probe-ssh"),
					},
				},
			},
			Name:     to.StringPtr("rp-lb"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
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
						ActionGroupID: to.StringPtr("[resourceId(parameters('subscriptionResourceGroupName'), 'Microsoft.Insights/actionGroups', 'rp-health-ag')]"),
					},
				},
				Enabled:             to.BoolPtr(true),
				EvaluationFrequency: to.StringPtr(evalFreq),
				Severity:            to.Int32Ptr(severity),
				Scopes: &[]string{
					"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
				},
				WindowSize:         to.StringPtr(windowSize),
				TargetResourceType: to.StringPtr("Microsoft.Network/loadBalancers"),
				AutoMitigate:       to.BoolPtr(true),
				Criteria: mgmtinsights.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf: &[]mgmtinsights.MetricCriteria{
						{
							CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
							MetricName:      to.StringPtr(metric),
							MetricNamespace: to.StringPtr("microsoft.network/loadBalancers"),
							Name:            to.StringPtr("HealthProbeCheck"),
							Operator:        mgmtinsights.OperatorLessThan,
							Threshold:       to.Float64Ptr(threshold),
							TimeAggregation: mgmtinsights.Average,
						},
					},
					OdataType: mgmtinsights.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     to.StringPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     to.StringPtr("Microsoft.Insights/metricAlerts"),
			Location: to.StringPtr("global"),
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
				Name:     to.StringPtr("[parameters('vmSize')]"),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1338),
			},
			Tags: map[string]*string{},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				// Reference: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-upgrade#arm-templates
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeAutomatic,
					RollingUpgradePolicy: &mgmtcompute.RollingUpgradePolicy{
						// Percentage equates to 1.02 instances out of 3
						MaxBatchInstancePercent:             to.Int32Ptr(34),
						MaxUnhealthyInstancePercent:         to.Int32Ptr(34),
						MaxUnhealthyUpgradedInstancePercent: to.Int32Ptr(34),
						PauseTimeBetweenBatches:             to.StringPtr("PT10M"),
					},
					AutomaticOSUpgradePolicy: &mgmtcompute.AutomaticOSUpgradePolicy{
						EnableAutomaticOSUpgrade: to.BoolPtr(true),
					},
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("[concat('rp-', parameters('vmssName'), '-')]"),
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
						// https://eng.ms/docs/products/azure-linux/gettingstarted/azurevm/azurevm
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: to.StringPtr("MicrosoftCBLMariner"),
							Offer:     to.StringPtr("cbl-mariner"),
							// cbl-mariner-2-gen2-fips is not supported by Automatic OS Updates
							// therefore the non fips image is used, and fips is configured manually
							// Reference: https://learn.microsoft.com/en-us/azure/virtual-machine-scale-sets/virtual-machine-scale-sets-automatic-upgrade
							Sku:     to.StringPtr("cbl-mariner-2-gen2"),
							Version: to.StringPtr("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
							DiskSizeGB: to.Int32Ptr(1024),
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("rp-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("rp-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("rp-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
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
								Name: to.StringPtr("rp-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
									Type:                    to.StringPtr("CustomScript"),
									TypeHandlerVersion:      to.StringPtr("2.0"),
									AutoUpgradeMinorVersion: to.BoolPtr(true),
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
								Name: to.StringPtr("AzureMonitorLinuxAgent"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.StringPtr("Microsoft.Azure.Monitor"),
									EnableAutomaticUpgrade:  to.BoolPtr(true),
									AutoUpgradeMinorVersion: to.BoolPtr(true),
									TypeHandlerVersion:      to.StringPtr("1.0"),
									Type:                    to.StringPtr("AzureMonitorLinuxAgent"),
									Settings: map[string]interface{}{
										"GCS_AUTO_CONFIG": true,
									},
								},
							},
						},
					},
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled: to.BoolPtr(true),
						},
					},
				},
				Overprovision: to.BoolPtr(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-rp-', resourceGroup().location))]": {},
				},
			},
			Name:     to.StringPtr("[concat('rp-vmss-', parameters('vmssName'))]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
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
			ObjectID: to.StringPtr("[parameters('fpServicePrincipalId')]"),
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
			ObjectID: to.StringPtr("[parameters('rpServicePrincipalId')]"),
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
			ObjectID: to.StringPtr("[parameters('rpServicePrincipalId')]"),
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
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(clusterAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.ClusterKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpClusterKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
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
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(portalAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.PortalKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpPortalKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
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
					ObjectID: to.StringPtr("[reference(resourceId('Microsoft.ContainerService/managedClusters', 'aro-aks-cluster-001'), '2020-12-01', 'Full').properties.identityProfile.kubeletidentity.objectId]"),
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
				"scope": to.StringPtr("inner"),
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
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(serviceAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.ServiceKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpServiceKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
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
					LocationName: to.StringPtr("[resourceGroup().location]"),
				},
			},
			DatabaseAccountOfferType: to.StringPtr("Standard"),
			BackupPolicy: &sdkcosmos.PeriodicModeBackupPolicy{
				Type: &backupPolicy,
				PeriodicModeProperties: &sdkcosmos.PeriodicModeProperties{
					BackupIntervalInMinutes:        to.Int32Ptr(240), //4 hours
					BackupRetentionIntervalInHours: to.Int32Ptr(720), //30 days
				},
			},
			MinimalTLSVersion: &minTLSVersion,
			DisableLocalAuth:  to.BoolPtr(true), // Disable local authentication
		},
		Name:     to.StringPtr("[parameters('databaseAccountName')]"),
		Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			"defaultExperience": to.StringPtr("Core (SQL)"),
		},
	}

	r := &arm.Resource{
		Resource:   cosmosdb,
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		Type:       "Microsoft.DocumentDB/databaseAccounts",
	}
	if g.production {
		cosmosdb.Properties.IPRules = []*sdkcosmos.IPAddressOrRange{}
		cosmosdb.Properties.IsVirtualNetworkFilterEnabled = to.BoolPtr(true)
		cosmosdb.Properties.VirtualNetworkRules = []*sdkcosmos.VirtualNetworkRule{}
		cosmosdb.Properties.DisableKeyBasedMetadataWriteAccess = to.BoolPtr(true)
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
		Resource: mgmtauthorization.RoleAssignment{
			Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('" + component + "ServicePrincipalId'), 'DocumentDB Data Contributor'))]"),
			Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlRoleAssignments"),
			RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
				Scope:            &scope,
				RoleDefinitionID: to.StringPtr("[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlRoleDefinitions', parameters('databaseAccountName'), '" + rbac.RoleDocumentDBDataContributor + "')]"),
				PrincipalID:      to.StringPtr("[parameters('" + component + "ServicePrincipalId')]"),
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
					ID: to.StringPtr("[" + databaseName + "]"),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: to.Int32Ptr(cosmosDbStandardProvisionedThroughputHack),
				},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		Type:       "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	}
	hashPartitionKey := sdkcosmos.PartitionKindHash
	portal := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: to.StringPtr("Portal"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							to.StringPtr("/id"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: to.Int32Ptr(-1),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: to.Int32Ptr(cosmosDbPortalProvisionedThroughputHack),
				},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Portal')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
		Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	}

	gateway := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: to.StringPtr("Gateway"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							to.StringPtr("/id"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: to.Int32Ptr(-1),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: to.Int32Ptr(cosmosDbGatewayProvisionedThroughputHack),
				},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Gateway')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
		Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	}

	mimo := &arm.Resource{
		Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
			Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
				Resource: &sdkcosmos.SQLContainerResource{
					ID: to.StringPtr("MaintenanceManifests"),
					PartitionKey: &sdkcosmos.ContainerPartitionKey{
						Paths: []*string{
							to.StringPtr("/clusterResourceID"),
						},
						Kind: &hashPartitionKey,
					},
					DefaultTTL: to.Int32Ptr(-1),
				},
				Options: &sdkcosmos.CreateUpdateOptions{
					Throughput: to.Int32Ptr(cosmosDbGatewayProvisionedThroughputHack),
				},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/MaintenanceManifests')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
		Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	}

	if !g.production {
		database.Resource.(*sdkcosmos.SQLDatabaseCreateUpdateParameters).Properties.Options = &sdkcosmos.CreateUpdateOptions{
			AutoscaleSettings: &sdkcosmos.AutoscaleSettings{
				MaxThroughput: to.Int32Ptr(1000),
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
						ID: to.StringPtr("AsyncOperations"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: to.Int32Ptr(7 * 86400), // 7 days
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/AsyncOperations')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("OpenShiftVersions"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: to.Int32Ptr(-1),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftVersions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("PlatformWorkloadIdentityRoleSets"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: to.Int32Ptr(-1),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/PlatformWorkloadIdentityRoleSets')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("ClusterManagerConfigurations"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/partitionKey"),
							},
							Kind: &hashPartitionKey,
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/ClusterManagerConfigurations')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("Billing"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Billing')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		gateway,
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("Monitors"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
						DefaultTTL: to.Int32Ptr(-1),
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Monitors')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("OpenShiftClusters"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/partitionKey"),
							},
							Kind: &hashPartitionKey,
						},
						UniqueKeyPolicy: &sdkcosmos.UniqueKeyPolicy{
							UniqueKeys: []*sdkcosmos.UniqueKey{
								{
									Paths: []*string{
										to.StringPtr("/key"),
									},
								},
								{
									Paths: []*string{
										to.StringPtr("/clusterResourceGroupIdKey"),
									},
								},
								{
									Paths: []*string{
										to.StringPtr("/clientIdKey"),
									},
								},
							},
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftClusters')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
		portal,
		{
			Resource: &sdkcosmos.SQLContainerCreateUpdateParameters{
				Properties: &sdkcosmos.SQLContainerCreateUpdateProperties{
					Resource: &sdkcosmos.SQLContainerResource{
						ID: to.StringPtr("Subscriptions"),
						PartitionKey: &sdkcosmos.ContainerPartitionKey{
							Paths: []*string{
								to.StringPtr("/id"),
							},
							Kind: &hashPartitionKey,
						},
					},
					Options: &sdkcosmos.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
			Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		},
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
	)

	// Don't deploy the MIMO databases in production yet
	if !g.production {
		rs = append(rs,
			mimo,
			// MIMO DB triggers
			g.rpCosmosDBTriggers(databaseName, "MaintenanceManifests", "renewLease", renewLeaseTriggerFunction, sdkcosmos.TriggerTypePre, sdkcosmos.TriggerOperationAll),
		)
	}

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
					ID:               to.StringPtr(triggerID),
					Body:             to.StringPtr(triggerFunction),
					TriggerOperation: &triggerOperation,
					TriggerType:      &triggerType,
				},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/" + containerName + "/" + triggerID + "')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/triggers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers', parameters('databaseAccountName'), " + databaseName + ", '" + containerName + "')]",
		},
		Type: "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/triggers",
	}
}

func (g *generator) rpCosmosDBAlert(throttledRequestThreshold float64, ruConsumptionThreshold float64, severity int32, name string, evalFreq string, windowSize string) *arm.Resource {
	throttledRequestMetricCriteria := mgmtinsights.MetricCriteria{
		CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
		MetricName:      to.StringPtr("TotalRequests"),
		MetricNamespace: to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
		Name:            to.StringPtr("ThrottledRequestCheck"),
		Operator:        mgmtinsights.OperatorGreaterThan,
		Threshold:       to.Float64Ptr(throttledRequestThreshold),
		TimeAggregation: mgmtinsights.Count,
		Dimensions: &[]mgmtinsights.MetricDimension{
			{
				Name:     to.StringPtr("StatusCode"),
				Operator: to.StringPtr("Include"),
				Values:   &[]string{"429"},
			},
		},
	}

	ruConsumptionMetricCriteria := mgmtinsights.MetricCriteria{
		CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
		MetricName:      to.StringPtr("NormalizedRUConsumption"),
		MetricNamespace: to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
		Name:            to.StringPtr("RUConsumptionCheck"),
		Operator:        mgmtinsights.OperatorGreaterThan,
		Threshold:       to.Float64Ptr(ruConsumptionThreshold),
		TimeAggregation: mgmtinsights.Average,
	}

	return &arm.Resource{
		Resource: mgmtinsights.MetricAlertResource{
			MetricAlertProperties: &mgmtinsights.MetricAlertProperties{
				Actions: &[]mgmtinsights.MetricAlertAction{
					{
						ActionGroupID: to.StringPtr("[resourceId(parameters('subscriptionResourceGroupName'), 'Microsoft.Insights/actionGroups', 'rp-health-ag')]"),
					},
				},
				Enabled:             to.BoolPtr(true),
				EvaluationFrequency: to.StringPtr(evalFreq),
				Severity:            to.Int32Ptr(severity),
				Scopes: &[]string{
					"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
				},
				WindowSize:         to.StringPtr(windowSize),
				TargetResourceType: to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
				AutoMitigate:       to.BoolPtr(true),
				Criteria: mgmtinsights.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf:     &[]mgmtinsights.MetricCriteria{throttledRequestMetricCriteria, ruConsumptionMetricCriteria},
					OdataType: mgmtinsights.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     to.StringPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     to.StringPtr("Microsoft.Insights/metricAlerts"),
			Location: to.StringPtr("global"),
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
			Name: to.StringPtr("48983534-3d06-4dcb-a566-08a694eb1279"),
			Type: to.StringPtr("Microsoft.Authorization/roleDefinitions"),
			RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
				RoleName:         to.StringPtr("ARO v4 ContainerRegistry Token Contributor"),
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
				DataEndpointEnabled: to.BoolPtr(true),
			},
			Name: to.StringPtr("[substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))]"),
			Type: to.StringPtr("Microsoft.ContainerRegistry/registries"),
			// TODO: INT ACR has wrong location - should be redeployed at globalResourceGroupLocation then remove acrLocationOverride configurable.
			Location: to.StringPtr("[if(equals(parameters('acrLocationOverride'), ''), resourceGroup().location, parameters('acrLocationOverride'))]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ContainerRegistry"),
	}
}

func (g *generator) rpACRReplica() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Replication{
			Name:     to.StringPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', parameters('location'))]"),
			Type:     to.StringPtr("Microsoft.ContainerRegistry/registries/replications"),
			Location: to.StringPtr("[parameters('location')]"),
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
			"48983534-3d06-4dcb-a566-08a694eb1279", // ARO v4 ContainerRegistry Token Contributor
			"parameters('fpServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), 'FP / ARO v4 ContainerRegistry Token Contributor')))",
		),
	}
}

func (g *generator) rpVersionStorageAccount() []*arm.Resource {
	return []*arm.Resource{
		g.storageAccount("[parameters('rpVersionStorageAccountName')]", &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess: to.BoolPtr(true),
		}, map[string]*string{
			tagKeyExemptPublicBlob: to.StringPtr(tagValueExemptPublicBlob),
		}),
		{
			Resource: &mgmtstorage.BlobContainer{
				Name: to.StringPtr("[concat(parameters('rpVersionStorageAccountName'), '/default/rpversion')]"),
				Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				ContainerProperties: &mgmtstorage.ContainerProperties{
					PublicAccess: mgmtstorage.PublicAccessContainer,
				},
			},
			APIVersion: azureclient.APIVersion("Microsoft.Storage"),
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts', parameters('rpVersionStorageAccountName'))]",
			},
		},
		{
			Resource: &mgmtstorage.BlobContainer{
				Name: to.StringPtr("[concat(parameters('rpVersionStorageAccountName'), '/default/ocpversions')]"),
				Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				ContainerProperties: &mgmtstorage.ContainerProperties{
					PublicAccess: mgmtstorage.PublicAccessContainer,
				},
			},
			APIVersion: azureclient.APIVersion("Microsoft.Storage"),
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts', parameters('rpVersionStorageAccountName'))]",
			},
		},
	}
}
