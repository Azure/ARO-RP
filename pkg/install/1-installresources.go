package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *Installer) installResources(ctx context.Context) error {
	g, err := i.loadGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	machineMaster := g[reflect.TypeOf(&machine.Master{})].(*machine.Master)

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	vnetID, _, err := subnet.Split(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	srvRecords := make([]mgmtprivatedns.SrvRecord, *installConfig.Config.ControlPlane.Replicas)
	for i := 0; i < int(*installConfig.Config.ControlPlane.Replicas); i++ {
		srvRecords[i] = mgmtprivatedns.SrvRecord{
			Priority: to.Int32Ptr(10),
			Weight:   to.Int32Ptr(10),
			Port:     to.Int32Ptr(2380),
			Target:   to.StringPtr(fmt.Sprintf("etcd-%d.%s", i, installConfig.Config.ObjectMeta.Name+"."+installConfig.Config.BaseDomain)),
		}
	}

	zones, err := zones(installConfig)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters: map[string]*arm.TemplateParameter{
			"sas": {
				Type: "object",
			},
		},
		Resources: []*arm.Resource{
			{
				Resource: &mgmtprivatedns.PrivateZone{
					Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain),
					Type:     to.StringPtr("Microsoft.Network/privateDnsZones"),
					Location: to.StringPtr("global"),
				},
				APIVersion: apiVersions["privatedns"],
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api-int"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL: to.Int64Ptr(300),
						ARecords: &[]mgmtprivatedns.ARecord{
							{
								Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/aro-internal-lb', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", apiVersions["network"])),
							},
						},
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/loadBalancers/aro-internal-lb",
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL: to.Int64Ptr(300),
						ARecords: &[]mgmtprivatedns.ARecord{
							{
								Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/aro-internal-lb', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", apiVersions["network"])),
							},
						},
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/loadBalancers/aro-internal-lb",
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/_etcd-server-ssl._tcp"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/SRV"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL:        to.Int64Ptr(60),
						SrvRecords: &srvRecords,
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr("[concat('" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/etcd-', copyIndex())]"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL: to.Int64Ptr(60),
						ARecords: &[]mgmtprivatedns.ARecord{
							{
								Ipv4Address: to.StringPtr("[reference(resourceId('Microsoft.Network/networkInterfaces', concat('aro-master', copyIndex(), '-nic')), '2019-07-01').ipConfigurations[0].properties.privateIPAddress]"),
							},
						},
					},
				},
				APIVersion: apiVersions["privatedns"],
				Copy: &arm.Copy{
					Name:  "privatednscopy",
					Count: int(*installConfig.Config.ControlPlane.Replicas),
				},
				DependsOn: []string{
					"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
				},
			},
			{
				Resource: &mgmtprivatedns.VirtualNetworkLink{
					VirtualNetworkLinkProperties: &mgmtprivatedns.VirtualNetworkLinkProperties{
						VirtualNetwork: &mgmtprivatedns.SubResource{
							ID: to.StringPtr(vnetID),
						},
						RegistrationEnabled: to.BoolPtr(false),
					},
					Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/" + installConfig.Config.ObjectMeta.Name + "-network-link"),
					Type:     to.StringPtr("Microsoft.Network/privateDnsZones/virtualNetworkLinks"),
					Location: to.StringPtr("global"),
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
					"privatednscopy",
				},
			},
			{
				Resource: &mgmtnetwork.PrivateLinkService{
					PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
						LoadBalancerFrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
							{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-internal-lb', 'internal-lb-ip')]"),
							},
						},
						IPConfigurations: &[]mgmtnetwork.PrivateLinkServiceIPConfiguration{
							{
								PrivateLinkServiceIPConfigurationProperties: &mgmtnetwork.PrivateLinkServiceIPConfigurationProperties{
									Subnet: &mgmtnetwork.Subnet{
										ID: to.StringPtr(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
									},
								},
								Name: to.StringPtr("aro-pls-nic"),
							},
						},
						Visibility: &mgmtnetwork.PrivateLinkServicePropertiesVisibility{
							Subscriptions: &[]string{
								i.env.SubscriptionID(),
							},
						},
						AutoApproval: &mgmtnetwork.PrivateLinkServicePropertiesAutoApproval{
							Subscriptions: &[]string{
								i.env.SubscriptionID(),
							},
						},
					},
					Name:     to.StringPtr("aro-pls"),
					Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
				DependsOn: []string{
					"Microsoft.Network/loadBalancers/aro-internal-lb",
				},
			},
			{
				Resource: &mgmtnetwork.PublicIPAddress{
					Sku: &mgmtnetwork.PublicIPAddressSku{
						Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
					},
					PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
						PublicIPAllocationMethod: mgmtnetwork.Static,
					},
					Name:     to.StringPtr("aro-pip"),
					Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
			},
			i.apiServerPublicLoadBalancer(installConfig.Config.Azure.Region, i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility),
			{
				Resource: &mgmtnetwork.LoadBalancer{
					Sku: &mgmtnetwork.LoadBalancerSku{
						Name: mgmtnetwork.LoadBalancerSkuNameStandard,
					},
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAllocationMethod: mgmtnetwork.Dynamic,
									Subnet: &mgmtnetwork.Subnet{
										ID: to.StringPtr(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
									},
								},
								Name: to.StringPtr("internal-lb-ip"),
							},
						},
						BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
							{
								Name: to.StringPtr("aro-internal-controlplane"),
							},
						},
						LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
							{
								LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
									FrontendIPConfiguration: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-internal-lb', 'internal-lb-ip')]"),
									},
									BackendAddressPool: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
									},
									Probe: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-internal-lb', 'api-internal-probe')]"),
									},
									Protocol:             mgmtnetwork.TransportProtocolTCP,
									LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
									FrontendPort:         to.Int32Ptr(6443),
									BackendPort:          to.Int32Ptr(6443),
									IdleTimeoutInMinutes: to.Int32Ptr(30),
									DisableOutboundSnat:  to.BoolPtr(true),
								},
								Name: to.StringPtr("api-internal"),
							},
							{
								LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
									FrontendIPConfiguration: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-internal-lb', 'internal-lb-ip')]"),
									},
									BackendAddressPool: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
									},
									Probe: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-internal-lb', 'sint-probe')]"),
									},
									Protocol:             mgmtnetwork.TransportProtocolTCP,
									LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
									FrontendPort:         to.Int32Ptr(22623),
									BackendPort:          to.Int32Ptr(22623),
									IdleTimeoutInMinutes: to.Int32Ptr(30),
								},
								Name: to.StringPtr("sint"),
							},
						},
						Probes: &[]mgmtnetwork.Probe{
							{
								ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
									Protocol:          mgmtnetwork.ProbeProtocolTCP,
									Port:              to.Int32Ptr(6443),
									IntervalInSeconds: to.Int32Ptr(10),
									NumberOfProbes:    to.Int32Ptr(3),
								},
								Name: to.StringPtr("api-internal-probe"),
							},
							{
								ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
									Protocol:          mgmtnetwork.ProbeProtocolTCP,
									Port:              to.Int32Ptr(22623),
									IntervalInSeconds: to.Int32Ptr(10),
									NumberOfProbes:    to.Int32Ptr(3),
								},
								Name: to.StringPtr("sint-probe"),
							},
						},
					},
					Name:     to.StringPtr("aro-internal-lb"),
					Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &mgmtnetwork.Interface{
					InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
						IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
							{
								InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
									LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
										{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
										},
										{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
										},
									},
									Subnet: &mgmtnetwork.Subnet{
										ID: to.StringPtr(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
									},
								},
								Name: to.StringPtr("bootstrap-nic-ip"),
							},
						},
					},
					Name:     to.StringPtr("aro-bootstrap-nic"),
					Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
				DependsOn: []string{
					"Microsoft.Network/loadBalancers/aro-internal-lb",
					"Microsoft.Network/loadBalancers/aro-public-lb",
				},
			},
			{
				Resource: &mgmtnetwork.Interface{
					InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
						IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
							{
								InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
									LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
										{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
										},
										{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
										},
									},
									Subnet: &mgmtnetwork.Subnet{
										ID: to.StringPtr(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
									},
								},
								Name: to.StringPtr("pipConfig"),
							},
						},
					},
					Name:     to.StringPtr("[concat('aro-master', copyIndex(), '-nic')]"),
					Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
				Copy: &arm.Copy{
					Name:  "networkcopy",
					Count: int(*installConfig.Config.ControlPlane.Replicas),
				},
				DependsOn: []string{
					"Microsoft.Network/loadBalancers/aro-internal-lb",
					"Microsoft.Network/loadBalancers/aro-public-lb",
				},
			},
			{
				Resource: &mgmtcompute.VirtualMachine{
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						HardwareProfile: &mgmtcompute.HardwareProfile{
							VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD4sV3,
						},
						StorageProfile: &mgmtcompute.StorageProfile{
							ImageReference: &mgmtcompute.ImageReference{
								Publisher: &installConfig.Config.Azure.Image.Publisher,
								Offer:     &installConfig.Config.Azure.Image.Offer,
								Sku:       &installConfig.Config.Azure.Image.SKU,
								Version:   &installConfig.Config.Azure.Image.Version,
							},
							OsDisk: &mgmtcompute.OSDisk{
								Name:         to.StringPtr("aro-bootstrap_OSDisk"),
								Caching:      mgmtcompute.CachingTypesReadWrite,
								CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
								DiskSizeGB:   to.Int32Ptr(100),
								ManagedDisk: &mgmtcompute.ManagedDiskParameters{
									StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
								},
							},
						},
						OsProfile: &mgmtcompute.OSProfile{
							ComputerName:  to.StringPtr("aro-bootstrap-vm"),
							AdminUsername: to.StringPtr("core"),
							AdminPassword: to.StringPtr("NotActuallyApplied!"),
							CustomData:    to.StringPtr(`[base64(concat('{"ignition":{"version":"2.2.0","config":{"replace":{"source":"https://cluster` + i.doc.OpenShiftCluster.Properties.StorageSuffix + `.blob.core.windows.net/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + i.doc.OpenShiftCluster.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`),
							LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
								DisablePasswordAuthentication: to.BoolPtr(false),
							},
						},
						NetworkProfile: &mgmtcompute.NetworkProfile{
							NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
								{
									ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', 'aro-bootstrap-nic')]"),
								},
							},
						},
						DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
							BootDiagnostics: &mgmtcompute.BootDiagnostics{
								Enabled:    to.BoolPtr(true),
								StorageURI: to.StringPtr("https://cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + ".blob.core.windows.net/"),
							},
						},
					},
					Name:     to.StringPtr("aro-bootstrap"),
					Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["compute"],
				DependsOn: []string{
					"Microsoft.Network/networkInterfaces/aro-bootstrap-nic",
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
				},
			},
			{
				Resource: &mgmtcompute.VirtualMachine{
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						HardwareProfile: &mgmtcompute.HardwareProfile{
							VMSize: mgmtcompute.VirtualMachineSizeTypes(installConfig.Config.ControlPlane.Platform.Azure.InstanceType),
						},
						StorageProfile: &mgmtcompute.StorageProfile{
							ImageReference: &mgmtcompute.ImageReference{
								Publisher: &installConfig.Config.Azure.Image.Publisher,
								Offer:     &installConfig.Config.Azure.Image.Offer,
								Sku:       &installConfig.Config.Azure.Image.SKU,
								Version:   &installConfig.Config.Azure.Image.Version,
							},
							OsDisk: &mgmtcompute.OSDisk{
								Name:         to.StringPtr("[concat('aro-master-', copyIndex(), '_OSDisk')]"),
								Caching:      mgmtcompute.CachingTypesReadOnly,
								CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
								DiskSizeGB:   &installConfig.Config.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
								ManagedDisk: &mgmtcompute.ManagedDiskParameters{
									StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
								},
							},
						},
						OsProfile: &mgmtcompute.OSProfile{
							ComputerName:  to.StringPtr("[concat('aro-master-', copyIndex())]"),
							AdminUsername: to.StringPtr("core"),
							AdminPassword: to.StringPtr("NotActuallyApplied!"),
							CustomData:    to.StringPtr(base64.StdEncoding.EncodeToString(machineMaster.File.Data)),
							LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
								DisablePasswordAuthentication: to.BoolPtr(false),
							},
						},
						NetworkProfile: &mgmtcompute.NetworkProfile{
							NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
								{
									ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', concat('aro-master', copyIndex(), '-nic'))]"),
								},
							},
						},
						DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
							BootDiagnostics: &mgmtcompute.BootDiagnostics{
								Enabled:    to.BoolPtr(true),
								StorageURI: to.StringPtr("https://cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + ".blob.core.windows.net/"),
							},
						},
					},
					Zones:    zones,
					Name:     to.StringPtr("[concat('aro-master-', copyIndex())]"),
					Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["compute"],
				Copy: &arm.Copy{
					Name:  "computecopy",
					Count: int(*installConfig.Config.ControlPlane.Replicas),
				},
				DependsOn: []string{
					"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
					"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
				},
			},
			{
				Resource: &mgmtnetwork.PublicIPAddress{
					Sku: &mgmtnetwork.PublicIPAddressSku{
						Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
					},
					PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
						PublicIPAllocationMethod: mgmtnetwork.Static,
					},
					Name:     to.StringPtr("aro-outbound-pip"),
					Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &mgmtnetwork.LoadBalancer{
					Sku: &mgmtnetwork.LoadBalancerSku{
						Name: mgmtnetwork.LoadBalancerSkuNameStandard,
					},
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &mgmtnetwork.PublicIPAddress{
										ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'aro-outbound-pip')]"),
									},
								},
								Name: to.StringPtr("outbound"),
							},
						},
						BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
							{
								Name: to.StringPtr("aro"),
							},
						},
						LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{}, //required to override default LB rules for port 80 and 443
						Probes:             &[]mgmtnetwork.Probe{},             //required to override default LB rules for port 80 and 443
						OutboundRules: &[]mgmtnetwork.OutboundRule{
							{
								OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
									FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
										{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro', 'outbound')]"),
										},
									},
									BackendAddressPool: &mgmtnetwork.SubResource{
										ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro', 'aro')]"),
									},
									Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
									IdleTimeoutInMinutes: to.Int32Ptr(30),
								},
								Name: to.StringPtr("outboundrule"),
							},
						},
					},
					Name:     to.StringPtr("aro"),
					Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: apiVersions["network"],
				DependsOn: []string{
					"Microsoft.Network/publicIPAddresses/aro-outbound-pip",
				},
			},
		},
	}
	return i.deployARMTemplate(ctx, resourceGroup, "resources", t, map[string]interface{}{
		"sas": map[string]interface{}{
			"value": map[string]interface{}{
				"signedStart":         i.doc.OpenShiftCluster.Properties.Install.Now.Format(time.RFC3339),
				"signedExpiry":        i.doc.OpenShiftCluster.Properties.Install.Now.Add(24 * time.Hour).Format(time.RFC3339),
				"signedPermission":    "rl",
				"signedResourceTypes": "o",
				"signedServices":      "b",
				"signedProtocol":      "https",
			},
		},
	})
}

// zones configures how master nodes are distributed across availability zones. In regions where the number of zones matches
// the number of nodes, it's one node per zone. In regions where there are no zones, all the nodes are in the same place.
// Anything else (e.g. 2-zone regions) is currently unsupported.
func zones(installConfig *installconfig.InstallConfig) (zones *[]string, err error) {
	zoneCount := len(installConfig.Config.ControlPlane.Platform.Azure.Zones)
	replicas := int(*installConfig.Config.ControlPlane.Replicas)
	if reflect.DeepEqual(installConfig.Config.ControlPlane.Platform.Azure.Zones, []string{""}) {
		// []string{""} indicates that there are no Azure Zones, so "zones" return value will be nil
	} else if zoneCount == replicas {
		zones = &[]string{"[copyIndex(1)]"}
	} else {
		err = fmt.Errorf("cluster creation with %d zone(s) and %d replica(s) is unimplemented", zoneCount, replicas)
	}
	return
}
