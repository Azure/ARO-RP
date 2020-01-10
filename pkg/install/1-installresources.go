package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *Installer) installResources(ctx context.Context) error {
	g, err := i.getGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	machineMaster := g[reflect.TypeOf(&machine.Master{})].(*machine.Master)

	vnetID, _, err := subnet.Split(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	masterSubnet, err := i.subnets.Get(ctx, i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	_, masterSubnetCIDR, err := net.ParseCIDR(*masterSubnet.AddressPrefix)
	if err != nil {
		return err
	}

	var lbIP net.IP
	{
		_, last := cidr.AddressRange(masterSubnetCIDR)
		lbIP = cidr.Dec(cidr.Dec(last))
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

	var zones *[]string
	switch len(installConfig.Config.ControlPlane.Platform.Azure.Zones) {
	case 1:
	case int(*installConfig.Config.ControlPlane.Replicas):
		zones = &[]string{
			"[copyIndex(1)]",
		}
	default:
		return fmt.Errorf("cluster creation with %d zones is unimplemented", len(installConfig.Config.ControlPlane.Platform.Azure.Zones))
	}

	var objectID string
	{
		spp := &i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

		conf := auth.NewClientCredentialsConfig(spp.ClientID, spp.ClientSecret, spp.TenantID)
		conf.Resource = azure.PublicCloud.GraphEndpoint

		spGraphAuthorizer, err := conf.Authorizer()
		if err != nil {
			return err
		}

		applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

		res, err := applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
		if err != nil {
			return err
		}

		objectID = *res.Value
	}

	{
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
					Resource: &mgmtauthorization.RoleAssignment{
						Name: to.StringPtr("[guid(resourceGroup().id, 'SP / Contributor')]"),
						Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
						Properties: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
							Scope:            to.StringPtr("[resourceGroup().id]"),
							RoleDefinitionID: to.StringPtr("[resourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"),
							PrincipalID:      to.StringPtr(objectID),
						},
					},
					APIVersion: apiVersions["authorization"],
				},
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
									Ipv4Address: to.StringPtr(lbIP.String()),
								},
							},
						},
					},
					APIVersion: apiVersions["privatedns"],
					DependsOn: []string{
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
									Ipv4Address: to.StringPtr(lbIP.String()),
								},
							},
						},
					},
					APIVersion: apiVersions["privatedns"],
					DependsOn: []string{
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
					// TODO: upstream doesn't appear to wire this in to any vnet - investigate.
					Resource: &mgmtnetwork.RouteTable{
						Name:     to.StringPtr("aro-node-routetable"),
						Type:     to.StringPtr("Microsoft.Network/routeTables"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
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
										PrivateIPAddress:          to.StringPtr(lbIP.String()),
										PrivateIPAllocationMethod: mgmtnetwork.Static,
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
						"[concat('Microsoft.Authorization/roleAssignments/', guid(resourceGroup().id, 'SP / Contributor'))]",
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
						"[concat('Microsoft.Authorization/roleAssignments/', guid(resourceGroup().id, 'SP / Contributor'))]",
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

		i.log.Print("deploying resources template")
		err = i.deployments.CreateOrUpdateAndWait(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "azuredeploy", mgmtresources.Deployment{
			Properties: &mgmtresources.DeploymentProperties{
				Template: t,
				Parameters: map[string]interface{}{
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
				},
				Mode: mgmtresources.Incremental,
			},
		})
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); ok {
				if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
					requestErr.ServiceError != nil &&
					requestErr.ServiceError.Code == "DeploymentActive" {
					i.log.Print("waiting for resources template")
					err = i.deployments.Wait(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "azuredeploy")
				}
			}
			if err != nil {
				return err
			}
		}
	}

	{
		i.log.Print("creating private endpoint")
		err = i.privateendpoint.Create(ctx, i.doc)
		if err != nil {
			return err
		}
	}

	{
		ipAddress := lbIP.String()

		if i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
			ip, err := i.publicipaddresses.Get(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "aro-pip", "")
			if err != nil {
				return err
			}

			ipAddress = *ip.IPAddress
		}

		err = i.dns.Update(ctx, i.doc.OpenShiftCluster, ipAddress)
		if err != nil {
			return err
		}

		privateEndpointIP, err := i.privateendpoint.GetIP(ctx, i.doc)
		if err != nil {
			return err
		}

		i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.NetworkProfile.PrivateEndpointIP = privateEndpointIP
			doc.OpenShiftCluster.Properties.APIServerProfile.IP = ipAddress
			return nil
		})
		if err != nil {
			return err
		}
	}

	{
		restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
		if err != nil {
			return err
		}

		cli, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		i.log.Print("waiting for bootstrap configmap")
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
			cm, err := cli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
			return err == nil && cm.Data["status"] == "complete", nil

		}, timeoutCtx.Done())
		if err != nil {
			return err
		}
	}

	return nil
}
