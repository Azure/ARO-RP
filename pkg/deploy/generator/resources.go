package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2019-08-01/documentdb"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtmonitor "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	tenantIDHack = "13805ec3-a223-47ad-ad65-8b2baf92c0fb"
)

var (
	tenantUUIDHack = uuid.Must(uuid.FromString(tenantIDHack))
)

func (g *generator) managedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     to.StringPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     to.StringPtr("[concat('aro-rp-', resourceGroup().location)]"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.ManagedIdentity"],
	}
}

func (g *generator) securityGroupRP() *arm.Resource {
	nsg := &mgmtnetwork.SecurityGroup{
		SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: &[]mgmtnetwork.SecurityRule{
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
			},
		},
		Name:     to.StringPtr("rp-nsg"),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		// override production ARM flag for more open configuration in development
		(*nsg.SecurityRules)[0].SecurityRulePropertiesFormat.SourceAddressPrefix = to.StringPtr("*")

		*nsg.SecurityRules = append(*nsg.SecurityRules, mgmtnetwork.SecurityRule{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(100),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("ssh_in"),
		})
	} else {
		*nsg.SecurityRules = append(*nsg.SecurityRules, mgmtnetwork.SecurityRule{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("443"),
				SourceAddressPrefixes:    to.StringSlicePtr([]string{}),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				DestinationAddressPrefix: to.StringPtr("*"),
				Priority:                 to.Int32Ptr(130),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("rp_in_geneva"),
		})
	}

	return &arm.Resource{
		Resource:   nsg,
		Condition:  g.conditionStanza("deployNSGs"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) securityGroupPE() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.SecurityGroup{
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
			Name:                          to.StringPtr("rp-pe-nsg"),
			Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
			Location:                      to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("deployNSGs"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) proxyVmss() *arm.Resource {
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

	trailer := base64.StdEncoding.EncodeToString([]byte(`yum -y update -x WALinuxAgent
yum -y install docker

firewall-cmd --add-port=443/tcp --permanent

mkdir /root/.docker
cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${PROXYIMAGE%%/*}": {
			"auth": "$PROXYIMAGEAUTH"
		}
	}
}
EOF
systemctl start docker.service
docker pull "$PROXYIMAGE"

mkdir /etc/proxy
base64 -d <<<"$PROXYCERT" >/etc/proxy/proxy.crt
base64 -d <<<"$PROXYKEY" >/etc/proxy/proxy.key
base64 -d <<<"$PROXYCLIENTCERT" >/etc/proxy/proxy-client.crt
chown -R 1000:1000 /etc/proxy
chmod 0600 /etc/proxy/proxy.key

cat >/etc/sysconfig/proxy <<EOF
PROXY_IMAGE='$PROXYIMAGE'
EOF

cat >/etc/systemd/system/proxy.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $PROXY_IMAGE
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

systemctl enable proxy.service

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr(string(mgmtcompute.VirtualMachineSizeTypesStandardD2sV3)),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.Manual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("dev-proxy-"),
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
							Sku:       to.StringPtr("7-RAW"),
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
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("dev-proxy-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("dev-proxy-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("dev-proxy-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: to.StringPtr("[parameters('proxyDomainNameLabel')]"),
														},
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
								Name: to.StringPtr("dev-proxy-vmss-cse"),
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
				},
				Overprovision: to.BoolPtr(false),
			},
			Name:     to.StringPtr("dev-proxy-vmss"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersions["Microsoft.Compute"],
	}
}

func (g *generator) devVpnPip() *arm.Resource {
	return &arm.Resource{
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
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) devVnet() *arm.Resource {
	return &arm.Resource{
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
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) devVPN() *arm.Resource {
	return &arm.Resource{
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
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
		},
	}
}

// halfPeering configures vnetA to peer with vnetB, two symmetrical configurations have to be applied for a peering to work
func (g *generator) halfPeering(vnetA string, vnetB string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetworkPeering{
			VirtualNetworkPeeringPropertiesFormat: &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
				AllowVirtualNetworkAccess: to.BoolPtr(true),
				AllowForwardedTraffic:     to.BoolPtr(true),
				AllowGatewayTransit:       to.BoolPtr(false),
				UseRemoteGateways:         to.BoolPtr(false),
				RemoteVirtualNetwork: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetB)),
				},
			},
			Name: to.StringPtr(fmt.Sprintf("%s/peering-%s", vnetA, vnetB)),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetA),
			fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetB),
		},
		Type:     "Microsoft.Network/virtualNetworks/virtualNetworkPeerings",
		Location: "[resourceGroup().location]",
	}
}

func (g *generator) rpvnet() *arm.Resource {
	subnet := mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/24"),
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
			},
		},
		Name: to.StringPtr("rp-subnet"),
	}

	if g.production {
		subnet.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
			{
				Service:   to.StringPtr("Microsoft.KeyVault"),
				Locations: &[]string{"*"},
			},
			{
				Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
				Locations: &[]string{"*"},
			},
		}
	}

	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.0.0/24",
					},
				},
				Subnets: &[]mgmtnetwork.Subnet{
					subnet,
				},
			},
			Name:     to.StringPtr("rp-vnet"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) pevnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.4.0/22",
					},
				},
				Subnets: &[]mgmtnetwork.Subnet{
					{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.4.0/22"),
							NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
								ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"),
							},
							PrivateEndpointNetworkPolicies: to.StringPtr("Disabled"),
						},
						Name: to.StringPtr("rp-pe-subnet"),
					},
				},
			},
			Name:     to.StringPtr("rp-pe-vnet-001"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) pip() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: mgmtnetwork.Static,
			},
			Name:     to.StringPtr("rp-pip"),
			Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
	}
}

func (g *generator) lb() *arm.Resource {
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
				},
			},
			Name:     to.StringPtr("rp-lb"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]",
		},
	}
}

func (g *generator) actionGroup(name string, shortName string) *arm.Resource {
	return &arm.Resource{
		Resource: mgmtmonitor.ActionGroupResource{
			ActionGroup: &mgmtmonitor.ActionGroup{
				Enabled:        to.BoolPtr(true),
				GroupShortName: to.StringPtr(shortName),
			},
			Name:     to.StringPtr(name),
			Type:     to.StringPtr("Microsoft.Insights/actionGroups"),
			Location: to.StringPtr("Global"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Insights"],
	}
}

// lbAlert generates an alert resource for the rp-lb healthprobe metric
func (g *generator) lbAlert(threshold float64, severity int32, name string, evalFreq string, windowSize string, metric string) *arm.Resource {
	return &arm.Resource{
		Resource: mgmtmonitor.MetricAlertResource{
			MetricAlertProperties: &mgmtmonitor.MetricAlertProperties{
				Actions: &[]mgmtmonitor.MetricAlertAction{
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
				Criteria: mgmtmonitor.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf: &[]mgmtmonitor.MetricCriteria{
						{
							CriterionType:   mgmtmonitor.CriterionTypeStaticThresholdCriterion,
							MetricName:      to.StringPtr(metric),
							MetricNamespace: to.StringPtr("microsoft.network/loadBalancers"),
							Name:            to.StringPtr("HealthProbeCheck"),
							Operator:        mgmtmonitor.OperatorLessThan,
							Threshold:       to.Float64Ptr(threshold),
							TimeAggregation: mgmtmonitor.Average,
						},
					},
					OdataType: mgmtmonitor.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     to.StringPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     to.StringPtr("Microsoft.Insights/metricAlerts"),
			Location: to.StringPtr("global"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Insights"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) vmss() *arm.Resource {
	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{
		"mdmFrontendUrl",
		"mdsdConfigVersion",
		"mdsdEnvironment",
		"acrResourceId",
		"rpImage",
		"rpMode",
		"adminApiClientCertCommonName",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	for _, variable := range []string{
		"adminApiCaBundle",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
		)
	}

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

	trailer := base64.StdEncoding.EncodeToString([]byte(`
yum -y update -x WALinuxAgent

# avoid "error: db5 error(-30969) from dbenv->open: BDB0091 DB_VERSION_MISMATCH: Database environment version mismatch"
rm -f /var/lib/rpm/__db*

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-7
rpm --import https://packages.microsoft.com/keys/microsoft.asc
rpm --import https://packages.fluentbit.io/fluentbit.key

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm

cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes

[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
EOF

cat >/etc/yum.repos.d/td-agent-bit.repo <<'EOF'
[td-agent-bit]
name=td-agent-bit
baseurl=https://packages.fluentbit.io/centos/7
enabled=yes
gpgcheck=yes
EOF

yum -y install azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security docker td-agent-bit

firewall-cmd --add-port=443/tcp --permanent

cat >/etc/td-agent-bit/td-agent-bit.conf <<'EOF'
[INPUT]
	Name systemd
	Tag journald
	Systemd_Filter _COMM=aro

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP

[OUTPUT]
	Name forward
	Port 29230
EOF

az login -i
az account set -s "$SUBSCRIPTIONID"

systemctl start docker.service
az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

MDMIMAGE="${RPIMAGE%%/*}/genevamdm:master_39"
docker pull "$MDMIMAGE"
docker pull "$RPIMAGE"

SVCVAULTURI="$(az keyvault list -g "$RESOURCEGROUPNAME" --query "[?tags.vault=='service'].properties.vaultUri" -o tsv)"
az keyvault secret download --file /etc/mdm.pem --id "${SVCVAULTURI}secrets/rp-mdm"
chmod 0600 /etc/mdm.pem
sed -i -ne '1,/END CERTIFICATE/ p' /etc/mdm.pem

az keyvault secret download --file /etc/mdsd.pem --id "${SVCVAULTURI}secrets/rp-mdsd"
chown syslog:syslog /etc/mdsd.pem
chmod 0600 /etc/mdsd.pem

az logout

mkdir /etc/aro-rp
base64 -d <<<"$ADMINAPICABUNDLE" >/etc/aro-rp/admin-ca-bundle.pem
chown -R 1000:1000 /etc/aro-rp

mkdir /etc/systemd/system/mdsd.service.d
cat >/etc/systemd/system/mdsd.service.d/override.conf <<'EOF'
[Unit]
After=network-online.target
EOF

cat >/etc/default/mdsd <<EOF
MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS="-A -d -r \$MDSD_ROLE_PREFIX"

export SSL_CERT_FILE=/etc/pki/tls/certs/ca-bundle.crt

export MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT=ARORPLogs
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_CERT_CERTFILE=/etc/mdsd.pem
export MONITORING_GCS_CERT_KEYFILE=/etc/mdsd.pem
export MONITORING_GCS_NAMESPACE=ARORPLogs
export MONITORING_CONFIG_VERSION='$MDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=rp
export MONITORING_ROLE_INSTANCE='$(hostname)'
EOF

cat >/etc/sysconfig/mdm <<EOF
MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=rp
MDMSOURCEROLEINSTANCE='$(hostname)'
EOF

mkdir /var/etw
cat >/etc/systemd/system/mdm.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $MDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-rp <<EOF
MDM_ACCOUNT=AzureRedHatOpenShiftRP
MDM_NAMESPACE=RP
ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
RPIMAGE='$RPIMAGE'
RP_MODE='$RPMODE'
EOF

cat >/etc/systemd/system/aro-rp.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e ADMIN_API_CLIENT_CERT_COMMON_NAME \
  -e RP_MODE \
  -e ACR_RESOURCE_ID \
  -p 443:8443 \
  -v /etc/aro-rp:/etc/aro-rp \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  rp
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-monitor <<EOF
MDM_ACCOUNT=AzureRedHatOpenShiftRP
MDM_NAMESPACE=BBM
CLUSTER_MDM_ACCOUNT=AzureRedHatOpenShiftCluster
CLUSTER_MDM_NAMESPACE=BBM
RPIMAGE='$RPIMAGE'
RP_MODE='$RPMODE'
EOF

cat >/etc/systemd/system/aro-monitor.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e RP_MODE \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  monitor
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

for service in aro-monitor aro-rp auoms azsecd azsecmond mdsd mdm chronyd td-agent-bit; do
  systemctl enable $service.service
done

for scan in baseline clamav software; do
  /usr/local/bin/azsecd config -s $scan -d P1D
done

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr("[parameters('vmSize')]"),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(3),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.Manual,
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
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: to.StringPtr("RedHat"),
							Offer:     to.StringPtr("RHEL"),
							Sku:       to.StringPtr("7-RAW"),
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
										"script": script,
									},
								},
							},
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
		APIVersion: azureclient.APIVersions["Microsoft.Compute"],
		DependsOn: []string{
			"[resourceId('Microsoft.Authorization/roleAssignments', guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader'))]",
			"[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]",
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) zone() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtdns.Zone{
			ZoneProperties: &mgmtdns.ZoneProperties{},
			Name:           to.StringPtr("[parameters('domainName')]"),
			Type:           to.StringPtr("Microsoft.Network/dnsZones"),
			Location:       to.StringPtr("global"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Network/dnsZones"],
	}
}

func (g *generator) clusterKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
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

func (g *generator) serviceKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
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

func (g *generator) clustersKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + kvClusterSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			KeyVaultTagName: to.StringPtr(ClustersKeyVaultTagValue),
		},
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.clusterKeyvaultAccessPolicies(),
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
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.KeyVault"],
	}
}

func (g *generator) serviceKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + kvServiceSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			KeyVaultTagName: to.StringPtr(ServiceKeyVaultTagValue),
		},
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.serviceKeyvaultAccessPolicies(),
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
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.KeyVault"],
	}
}

func (g *generator) cosmosdb() []*arm.Resource {
	cosmosdb := &mgmtdocumentdb.DatabaseAccountCreateUpdateParameters{
		Kind: mgmtdocumentdb.GlobalDocumentDB,
		DatabaseAccountCreateUpdateProperties: &mgmtdocumentdb.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy: &mgmtdocumentdb.ConsistencyPolicy{
				DefaultConsistencyLevel: mgmtdocumentdb.Strong,
			},
			Locations: &[]mgmtdocumentdb.Location{
				{
					LocationName: to.StringPtr("[resourceGroup().location]"),
				},
			},
			DatabaseAccountOfferType: to.StringPtr(string(mgmtdocumentdb.Standard)),
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
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
	}

	if g.production {
		cosmosdb.IPRangeFilter = to.StringPtr("[concat('104.42.195.92,40.76.54.131,52.176.6.30,52.169.50.45,52.187.184.26', if(equals(parameters('extraCosmosDBIPs'), ''), '', ','), parameters('extraCosmosDBIPs'))]")
		cosmosdb.IsVirtualNetworkFilterEnabled = to.BoolPtr(true)
		cosmosdb.VirtualNetworkRules = &[]mgmtdocumentdb.VirtualNetworkRule{
			{
				ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
			},
		}
		cosmosdb.DisableKeyBasedMetadataWriteAccess = to.BoolPtr(true)

		r.DependsOn = append(r.DependsOn, "[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]")
	}

	rs := []*arm.Resource{
		r,
	}

	if g.production {
		rs = append(rs, g.database("'ARO'", true)...)
	}

	return rs
}

func (g *generator) database(databaseName string, addDependsOn bool) []*arm.Resource {
	rs := []*arm.Resource{
		{
			Resource: &mgmtdocumentdb.SQLDatabaseCreateUpdateParameters{
				SQLDatabaseCreateUpdateProperties: &mgmtdocumentdb.SQLDatabaseCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLDatabaseResource{
						ID: to.StringPtr("[" + databaseName + "]"),
					},
					Options: map[string]*string{
						"throughput": to.StringPtr("500"),
					},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("AsyncOperations"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
						DefaultTTL: to.Int32Ptr(7 * 86400), // 7 days
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/AsyncOperations')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Billing"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Billing')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Monitors"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
						DefaultTTL: to.Int32Ptr(-1),
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Monitors')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("OpenShiftClusters"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/partitionKey",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
						UniqueKeyPolicy: &mgmtdocumentdb.UniqueKeyPolicy{
							UniqueKeys: &[]mgmtdocumentdb.UniqueKey{
								{
									Paths: &[]string{
										"/key",
									},
								},
								{
									Paths: &[]string{
										"/clusterResourceGroupIdKey",
									},
								},
								{
									Paths: &[]string{
										"/clientIdKey",
									},
								},
							},
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftClusters')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Subscriptions"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.DocumentDB"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
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

func (g *generator) roleDefinitionTokenContributor() *arm.Resource {
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
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.Authorization/roleDefinitions"],
	}
}

func (g *generator) rbac() []*arm.Resource {
	return []*arm.Resource{
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'acdd72a7-3385-48ef-bd42-f606fba81ae7')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'FP / Network Contributor')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4d97b98b-1d4f-4787-a291-c67834d212e7')]"),
					PrincipalID:      to.StringPtr("[parameters('fpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('rpServicePrincipalId'), 'RP / DocumentDB Account Contributor'))]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '5bd9cd88-fe45-4216-938b-f97437e15450')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			},
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('domainName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/dnsZones', parameters('domainName')), 'FP / DNS Zone Contributor'))]"),
				Type: to.StringPtr("Microsoft.Network/dnsZones/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'befefa01-2a29-4197-83a8-272ff33ce314')]"),
					PrincipalID:      to.StringPtr("[parameters('fpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]",
			},
		},
	}
}

func (g *generator) acrReplica() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Replication{
			Name:     to.StringPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', parameters('location'))]"),
			Type:     to.StringPtr("Microsoft.ContainerRegistry/registries/replications"),
			Location: to.StringPtr("[parameters('location')]"),
		},
		Condition:  g.conditionStanza("fullDeploy"),
		APIVersion: azureclient.APIVersions["Microsoft.ContainerRegistry"],
	}
}

func (g *generator) acrRbac() []*arm.Resource {
	return []*arm.Resource{
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), parameters('rpServicePrincipalId'), 'RP / AcrPull')))]"),
				Type: to.StringPtr("Microsoft.ContainerRegistry/registries/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[parameters('acrResourceId')]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7ca-43fe172d538d')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), 'FP / ARO v4 ContainerRegistry Token Contributor')))]"),
				Type: to.StringPtr("Microsoft.ContainerRegistry/registries/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[parameters('acrResourceId')]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '48983534-3d06-4dcb-a566-08a694eb1279')]"),
					PrincipalID:      to.StringPtr("[parameters('fpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			Condition:  g.conditionStanza("fullDeploy"),
			APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
		},
	}
}
