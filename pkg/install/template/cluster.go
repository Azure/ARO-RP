package template

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type clusterTemplate struct {
	log         *logrus.Entry
	template    *arm.Template
	deployments resources.DeploymentsClient

	ClusterName        string
	SubscriptionID     string
	ResourceGroup      string
	SpID               string
	Location           string
	PrivateZoneName    string
	LbIP               string
	StorageSuffix      string
	SrvRecords         *[]mgmtprivatedns.SrvRecord
	VnetID             string
	Zones              *[]string
	MachineMaster      *machine.Master
	MasterSubnetID     string
	Privatednscopy     int
	Networkcopy        int
	Computecopy        int
	APIServerVisiblity api.Visibility
	InstallTime        time.Time

	ImagePublisher string
	ImageOffer     string
	ImageSku       string
	ImageVersion   string
	DiskSizeGB     int32
	InstanceType   string
}

func NewClusterTemplate(log *logrus.Entry, subscriptionID string, deployments resources.DeploymentsClient, oc *api.OpenShiftCluster, installConfig *types.InstallConfig, lbIP, spID string, master *machine.Master) (Template, error) {
	srvRecords := make([]mgmtprivatedns.SrvRecord, *installConfig.ControlPlane.Replicas)
	for i := 0; i < int(*installConfig.ControlPlane.Replicas); i++ {
		srvRecords[i] = mgmtprivatedns.SrvRecord{
			Priority: to.Int32Ptr(10),
			Weight:   to.Int32Ptr(10),
			Port:     to.Int32Ptr(2380),
			Target:   to.StringPtr(fmt.Sprintf("etcd-%d.%s", i, installConfig.ObjectMeta.Name+"."+installConfig.BaseDomain)),
		}
	}
	var zones *[]string
	switch len(installConfig.ControlPlane.Platform.Azure.Zones) {
	case 1:
	case int(*installConfig.ControlPlane.Replicas):
		zones = &[]string{
			"[copyIndex(1)]",
		}
	default:
		return nil, fmt.Errorf("cluster creation with %d zones is unimplemented", len(installConfig.ControlPlane.Platform.Azure.Zones))
	}

	vnetID, _, err := subnet.Split(oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}
	resourceGroup := oc.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')+1:]

	return &clusterTemplate{
		log:         log,
		deployments: deployments,

		SubscriptionID: subscriptionID,
		SpID:           spID,
		ResourceGroup:  resourceGroup,
		VnetID:         vnetID,
		LbIP:           lbIP,
		Zones:          zones,
		SrvRecords:     &srvRecords,
		MachineMaster:  master,

		StorageSuffix:      oc.Properties.StorageSuffix,
		MasterSubnetID:     oc.Properties.MasterProfile.SubnetID,
		APIServerVisiblity: oc.Properties.APIServerProfile.Visibility,
		InstallTime:        oc.Properties.Install.Now,

		ClusterName:     installConfig.ObjectMeta.Name,
		PrivateZoneName: installConfig.ObjectMeta.Name + "." + installConfig.BaseDomain,
		Location:        installConfig.Azure.Region,
		ImageOffer:      installConfig.Azure.Image.Offer,
		ImagePublisher:  installConfig.Azure.Image.Publisher,
		ImageSku:        installConfig.Azure.Image.SKU,
		ImageVersion:    installConfig.Azure.Image.Version,
		Privatednscopy:  int(*installConfig.ControlPlane.Replicas),
		Networkcopy:     int(*installConfig.ControlPlane.Replicas),
		Computecopy:     int(*installConfig.ControlPlane.Replicas),
		InstanceType:    installConfig.ControlPlane.Platform.Azure.InstanceType,
		DiskSizeGB:      installConfig.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
	}, nil
}

func (c *clusterTemplate) Deploy(ctx context.Context) error {
	parameters := map[string]interface{}{
		"sas": map[string]interface{}{
			"value": map[string]interface{}{
				"signedStart":         c.InstallTime.Format(time.RFC3339),
				"signedExpiry":        c.InstallTime.Add(24 * time.Hour).Format(time.RFC3339),
				"signedPermission":    "rl",
				"signedResourceTypes": "o",
				"signedServices":      "b",
				"signedProtocol":      "https",
			},
		},
	}

	return templateDeploy(ctx, c.log, c.deployments, c.Generate(), parameters, c.ResourceGroup)
}

func (c *clusterTemplate) Generate() *arm.Template {
	return &arm.Template{
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
						PrincipalID:      to.StringPtr(c.SpID),
					},
				},
				APIVersion: apiVersions["authorization"],
			},
			{
				Resource: &mgmtprivatedns.PrivateZone{
					Name:     to.StringPtr(c.PrivateZoneName),
					Type:     to.StringPtr("Microsoft.Network/privateDnsZones"),
					Location: to.StringPtr("global"),
				},
				APIVersion: apiVersions["privatedns"],
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(c.PrivateZoneName + "/api-int"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL: to.Int64Ptr(300),
						ARecords: &[]mgmtprivatedns.ARecord{
							{
								Ipv4Address: to.StringPtr(c.LbIP),
							},
						},
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(c.PrivateZoneName + "/api"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL: to.Int64Ptr(300),
						ARecords: &[]mgmtprivatedns.ARecord{
							{
								Ipv4Address: to.StringPtr(c.LbIP),
							},
						},
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr(c.PrivateZoneName + "/_etcd-server-ssl._tcp"),
					Type: to.StringPtr("Microsoft.Network/privateDnsZones/SRV"),
					RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
						TTL:        to.Int64Ptr(60),
						SrvRecords: c.SrvRecords,
					},
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName,
				},
			},
			{
				Resource: &mgmtprivatedns.RecordSet{
					Name: to.StringPtr("[concat('" + c.PrivateZoneName + "/etcd-', copyIndex())]"),
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
					Count: c.Privatednscopy,
				},
				DependsOn: []string{
					"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName,
				},
			},
			{
				Resource: &mgmtprivatedns.VirtualNetworkLink{
					VirtualNetworkLinkProperties: &mgmtprivatedns.VirtualNetworkLinkProperties{
						VirtualNetwork: &mgmtprivatedns.SubResource{
							ID: to.StringPtr(c.VnetID),
						},
						RegistrationEnabled: to.BoolPtr(false),
					},
					Name:     to.StringPtr(c.PrivateZoneName + "/" + c.ClusterName + "-network-link"),
					Type:     to.StringPtr("Microsoft.Network/privateDnsZones/virtualNetworkLinks"),
					Location: to.StringPtr("global"),
				},
				APIVersion: apiVersions["privatedns"],
				DependsOn: []string{
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName,
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
										ID: to.StringPtr(c.MasterSubnetID),
									},
								},
								Name: to.StringPtr("aro-pls-nic"),
							},
						},
						Visibility: &mgmtnetwork.PrivateLinkServicePropertiesVisibility{
							Subscriptions: &[]string{
								c.SubscriptionID,
							},
						},
						AutoApproval: &mgmtnetwork.PrivateLinkServicePropertiesAutoApproval{
							Subscriptions: &[]string{
								c.SubscriptionID,
							},
						},
					},
					Name:     to.StringPtr("aro-pls"),
					Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
					Location: &c.Location,
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
					Location: &c.Location,
				},
				APIVersion: apiVersions["network"],
			},
			c.apiServerPublicLoadBalancer(),
			{
				Resource: &mgmtnetwork.LoadBalancer{
					Sku: &mgmtnetwork.LoadBalancerSku{
						Name: mgmtnetwork.LoadBalancerSkuNameStandard,
					},
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAddress:          to.StringPtr(c.LbIP),
									PrivateIPAllocationMethod: mgmtnetwork.Static,
									Subnet: &mgmtnetwork.Subnet{
										ID: to.StringPtr(c.MasterSubnetID),
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
					Location: &c.Location,
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
										ID: to.StringPtr(c.MasterSubnetID),
									},
								},
								Name: to.StringPtr("bootstrap-nic-ip"),
							},
						},
					},
					Name:     to.StringPtr("aro-bootstrap-nic"),
					Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
					Location: &c.Location,
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
										ID: to.StringPtr(c.MasterSubnetID),
									},
								},
								Name: to.StringPtr("pipConfig"),
							},
						},
					},
					Name:     to.StringPtr("[concat('aro-master', copyIndex(), '-nic')]"),
					Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
					Location: &c.Location,
				},
				APIVersion: apiVersions["network"],
				Copy: &arm.Copy{
					Name:  "networkcopy",
					Count: c.Networkcopy,
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
								Publisher: &c.ImagePublisher,
								Offer:     &c.ImageOffer,
								Sku:       &c.ImageSku,
								Version:   &c.ImageVersion,
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
							CustomData:    to.StringPtr(`[base64(concat('{"ignition":{"version":"2.2.0","config":{"replace":{"source":"https://cluster` + c.StorageSuffix + `.blob.core.windows.net/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + c.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`),
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
								StorageURI: to.StringPtr("https://cluster" + c.StorageSuffix + ".blob.core.windows.net/"),
							},
						},
					},
					Name:     to.StringPtr("aro-bootstrap"),
					Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
					Location: &c.Location,
				},
				APIVersion: apiVersions["compute"],
				DependsOn: []string{
					"[concat('Microsoft.Authorization/roleAssignments/', guid(resourceGroup().id, 'SP / Contributor'))]",
					"Microsoft.Network/networkInterfaces/aro-bootstrap-nic",
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName + "/virtualNetworkLinks/" + c.ClusterName + "-network-link",
				},
			},
			{
				Resource: &mgmtcompute.VirtualMachine{
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						HardwareProfile: &mgmtcompute.HardwareProfile{
							VMSize: mgmtcompute.VirtualMachineSizeTypes(c.InstanceType),
						},
						StorageProfile: &mgmtcompute.StorageProfile{
							ImageReference: &mgmtcompute.ImageReference{
								Publisher: &c.ImagePublisher,
								Offer:     &c.ImageOffer,
								Sku:       &c.ImageSku,
								Version:   &c.ImageVersion,
							},
							OsDisk: &mgmtcompute.OSDisk{
								Name:         to.StringPtr("[concat('aro-master-', copyIndex(), '_OSDisk')]"),
								Caching:      mgmtcompute.CachingTypesReadOnly,
								CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
								DiskSizeGB:   &c.DiskSizeGB,
								ManagedDisk: &mgmtcompute.ManagedDiskParameters{
									StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
								},
							},
						},
						OsProfile: &mgmtcompute.OSProfile{
							ComputerName:  to.StringPtr("[concat('aro-master-', copyIndex())]"),
							AdminUsername: to.StringPtr("core"),
							AdminPassword: to.StringPtr("NotActuallyApplied!"),
							CustomData:    to.StringPtr(base64.StdEncoding.EncodeToString(c.MachineMaster.File.Data)),
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
								StorageURI: to.StringPtr("https://cluster" + c.StorageSuffix + ".blob.core.windows.net/"),
							},
						},
					},
					Zones:    c.Zones,
					Name:     to.StringPtr("[concat('aro-master-', copyIndex())]"),
					Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
					Location: &c.Location,
				},
				APIVersion: apiVersions["compute"],
				Copy: &arm.Copy{
					Name:  "computecopy",
					Count: c.Computecopy,
				},
				DependsOn: []string{
					"[concat('Microsoft.Authorization/roleAssignments/', guid(resourceGroup().id, 'SP / Contributor'))]",
					"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
					"Microsoft.Network/privateDnsZones/" + c.PrivateZoneName + "/virtualNetworkLinks/" + c.ClusterName + "-network-link",
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
					Location: &c.Location,
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
					Location: &c.Location,
				},
				APIVersion: apiVersions["network"],
				DependsOn: []string{
					"Microsoft.Network/publicIPAddresses/aro-outbound-pip",
				},
			},
		},
	}
}

func (c *clusterTemplate) apiServerPublicLoadBalancer() *arm.Resource {
	lb := &mgmtnetwork.LoadBalancer{
		Sku: &mgmtnetwork.LoadBalancerSku{
			Name: mgmtnetwork.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &mgmtnetwork.PublicIPAddress{
							ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'aro-pip')]"),
						},
					},
					Name: to.StringPtr("public-lb-ip"),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
				{
					Name: to.StringPtr("aro-public-lb-control-plane"),
				},
			},
			OutboundRules: &[]mgmtnetwork.OutboundRule{
				{
					OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
							{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-public-lb', 'public-lb-ip')]"),
							},
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
						},
						Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
						IdleTimeoutInMinutes: to.Int32Ptr(30),
					},
					Name: to.StringPtr("api-internal-outboundrule"),
				},
			},
		},
		Name:     to.StringPtr("aro-public-lb"),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: &c.Location,
	}

	if c.APIServerVisiblity == api.VisibilityPublic {
		lb.LoadBalancingRules = &[]mgmtnetwork.LoadBalancingRule{
			{
				LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-public-lb', 'public-lb-ip')]"),
					},
					BackendAddressPool: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
					},
					Probe: &mgmtnetwork.SubResource{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-public-lb', 'api-internal-probe')]"),
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
		}
		lb.Probes = &[]mgmtnetwork.Probe{
			{
				ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
					Protocol:          mgmtnetwork.ProbeProtocolTCP,
					Port:              to.Int32Ptr(6443),
					IntervalInSeconds: to.Int32Ptr(10),
					NumberOfProbes:    to.Int32Ptr(3),
				},
				Name: to.StringPtr("api-internal-probe"),
				Type: to.StringPtr("Microsoft.Network/loadBalancers/probes"),
			},
		}
	}

	return &arm.Resource{
		Resource:   lb,
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			"Microsoft.Network/publicIPAddresses/aro-pip",
		},
	}
}
