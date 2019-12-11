package install

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/machines"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/arm"
	"github.com/jim-minter/rp/pkg/util/restconfig"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

func (i *Installer) installResources(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	g, err := i.getGraph(ctx, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	machinesMaster := g[reflect.TypeOf(&machines.Master{})].(*machines.Master)
	machineMaster := g[reflect.TypeOf(&machine.Master{})].(*machine.Master)

	vnetID, _, err := subnet.Split(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	masterSubnet, err := i.subnets.Get(ctx, doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	_, masterSubnetCIDR, err := net.ParseCIDR(*masterSubnet.AddressPrefix)
	if err != nil {
		return err
	}

	// TODO: make this dynamic and use a DNS alias record
	var lbIP net.IP
	{
		_, last := cidr.AddressRange(masterSubnetCIDR)
		lbIP = cidr.Dec(cidr.Dec(last))
	}

	srvRecords := make([]privatedns.SrvRecord, len(machinesMaster.MachineFiles))
	for i := 0; i < len(machinesMaster.MachineFiles); i++ {
		srvRecords[i] = privatedns.SrvRecord{
			Priority: to.Int32Ptr(10),
			Weight:   to.Int32Ptr(10),
			Port:     to.Int32Ptr(2380),
			Target:   to.StringPtr(fmt.Sprintf("etcd-%d.%s", i, installConfig.Config.ObjectMeta.Name+"."+installConfig.Config.BaseDomain)),
		}
	}

	var objectID string
	{
		spp := &doc.OpenShiftCluster.Properties.ServicePrincipalProfile

		conf := auth.NewClientCredentialsConfig(spp.ClientID, spp.ClientSecret, spp.TenantID)
		conf.Resource = azure.PublicCloud.GraphEndpoint

		spAuthorizer, err := conf.Authorizer()
		if err != nil {
			return err
		}

		applications := graphrbac.NewApplicationsClient(spp.TenantID)
		applications.Authorizer = spAuthorizer

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
			Parameters: map[string]*arm.Parameter{
				"sas": {
					Type: "object",
				},
			},
			Resources: []*arm.Resource{
				{
					Resource: &authorization.RoleAssignment{
						Name: to.StringPtr("[guid(resourceGroup().id, 'SP / Contributor')]"),
						Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
						Properties: &authorization.RoleAssignmentPropertiesWithScope{
							Scope:            to.StringPtr("[resourceGroup().id]"),
							RoleDefinitionID: to.StringPtr("[resourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"),
							PrincipalID:      to.StringPtr(objectID),
						},
					},
					APIVersion: apiVersions["authorization"],
				},
				{
					Resource: &privatedns.PrivateZone{
						Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain),
						Type:     to.StringPtr("Microsoft.Network/privateDnsZones"),
						Location: to.StringPtr("global"),
					},
					APIVersion: apiVersions["privatedns"],
				},
				{
					Resource: &privatedns.RecordSet{
						Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api-int"),
						Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
						RecordSetProperties: &privatedns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: &[]privatedns.ARecord{
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
					Resource: &privatedns.RecordSet{
						Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api"),
						Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
						RecordSetProperties: &privatedns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: &[]privatedns.ARecord{
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
					Resource: &privatedns.RecordSet{
						Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/_etcd-server-ssl._tcp"),
						Type: to.StringPtr("Microsoft.Network/privateDnsZones/SRV"),
						RecordSetProperties: &privatedns.RecordSetProperties{
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
					Resource: &privatedns.RecordSet{
						Name: to.StringPtr("[concat('" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/etcd-', copyIndex())]"),
						Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
						RecordSetProperties: &privatedns.RecordSetProperties{
							TTL: to.Int64Ptr(60),
							ARecords: &[]privatedns.ARecord{
								{
									Ipv4Address: to.StringPtr("[reference(resourceId('Microsoft.Network/networkInterfaces', concat('aro-master', copyIndex(), '-nic')), '2019-07-01').ipConfigurations[0].properties.privateIPAddress]"),
								},
							},
						},
					},
					APIVersion: apiVersions["privatedns"],
					Copy: &arm.Copy{
						Name:  "privatednscopy",
						Count: len(machinesMaster.MachineFiles),
					},
					DependsOn: []string{
						"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
						"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
					},
				},
				{
					Resource: &privatedns.VirtualNetworkLink{
						VirtualNetworkLinkProperties: &privatedns.VirtualNetworkLinkProperties{
							VirtualNetwork: &privatedns.SubResource{
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
					// TODO: upstream doesn't appear to wire this in to any vnet - investigate.
					Resource: &network.RouteTable{
						Name:     to.StringPtr("aro-node-routetable"),
						Type:     to.StringPtr("Microsoft.Network/routeTables"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
				{
					// TODO: we will want to remove this
					Resource: &network.PublicIPAddress{
						Sku: &network.PublicIPAddressSku{
							Name: network.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: network.Static,
						},
						Name:     to.StringPtr("aro-bootstrap-pip"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
				{
					Resource: &network.PublicIPAddress{
						Sku: &network.PublicIPAddressSku{
							Name: network.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: network.Static,
							DNSSettings: &network.PublicIPAddressDNSSettings{
								DomainNameLabel: &doc.OpenShiftCluster.Properties.DomainName,
							},
						},
						Name:     to.StringPtr("aro-pip"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
				{
					Resource: &network.LoadBalancer{
						Sku: &network.LoadBalancerSku{
							Name: network.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &network.PublicIPAddress{
											ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'aro-pip')]"),
										},
									},
									Name: to.StringPtr("public-lb-ip"),
								},
							},
							BackendAddressPools: &[]network.BackendAddressPool{
								{
									Name: to.StringPtr("aro-public-lb-control-plane"),
								},
							},
							LoadBalancingRules: &[]network.LoadBalancingRule{
								{
									LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-public-lb', 'public-lb-ip')]"),
										},
										BackendAddressPool: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
										},
										Probe: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-public-lb', 'api-internal-probe')]"),
										},
										Protocol:             network.TransportProtocolTCP,
										LoadDistribution:     network.LoadDistributionDefault,
										FrontendPort:         to.Int32Ptr(6443),
										BackendPort:          to.Int32Ptr(6443),
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("api-internal"),
								},
							},
							Probes: &[]network.Probe{
								{
									ProbePropertiesFormat: &network.ProbePropertiesFormat{
										Protocol:          network.ProbeProtocolTCP,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("api-internal-probe"),
									Type: to.StringPtr("Microsoft.Network/loadBalancers/probes"),
								},
							},
						},
						Name:     to.StringPtr("aro-public-lb"),
						Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
					DependsOn: []string{
						"Microsoft.Network/publicIPAddresses/aro-pip",
					},
				},
				{
					Resource: &network.LoadBalancer{
						Sku: &network.LoadBalancerSku{
							Name: network.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
										PrivateIPAddress:          to.StringPtr(lbIP.String()),
										PrivateIPAllocationMethod: network.Static,
										Subnet: &network.Subnet{
											ID: to.StringPtr(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
										},
									},
									Name: to.StringPtr("internal-lb-ip"),
								},
							},
							BackendAddressPools: &[]network.BackendAddressPool{
								{
									Name: to.StringPtr("aro-internal-controlplane"),
								},
							},
							LoadBalancingRules: &[]network.LoadBalancingRule{
								{
									LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-internal-lb', 'internal-lb-ip')]"),
										},
										BackendAddressPool: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
										},
										Probe: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-internal-lb', 'api-internal-probe')]"),
										},
										Protocol:             network.TransportProtocolTCP,
										LoadDistribution:     network.LoadDistributionDefault,
										FrontendPort:         to.Int32Ptr(6443),
										BackendPort:          to.Int32Ptr(6443),
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("api-internal"),
								},
								{
									LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'aro-internal-lb', 'internal-lb-ip')]"),
										},
										BackendAddressPool: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
										},
										Probe: &network.SubResource{
											ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'aro-internal-lb', 'sint-probe')]"),
										},
										Protocol:             network.TransportProtocolTCP,
										LoadDistribution:     network.LoadDistributionDefault,
										FrontendPort:         to.Int32Ptr(22623),
										BackendPort:          to.Int32Ptr(22623),
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("sint"),
								},
							},
							Probes: &[]network.Probe{
								{
									ProbePropertiesFormat: &network.ProbePropertiesFormat{
										Protocol:          network.ProbeProtocolTCP,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(10),
										NumberOfProbes:    to.Int32Ptr(3),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
								{
									ProbePropertiesFormat: &network.ProbePropertiesFormat{
										Protocol:          network.ProbeProtocolTCP,
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
					Resource: &network.Interface{
						InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
							IPConfigurations: &[]network.InterfaceIPConfiguration{
								{
									InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
										LoadBalancerBackendAddressPools: &[]network.BackendAddressPool{
											{
												ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
											},
											{
												ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
											},
										},
										Subnet: &network.Subnet{
											ID: to.StringPtr(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
										},
										PublicIPAddress: &network.PublicIPAddress{
											ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'aro-bootstrap-pip')]"),
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
						"Microsoft.Network/publicIPAddresses/aro-bootstrap-pip",
					},
				},
				{
					Resource: &network.Interface{
						InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
							IPConfigurations: &[]network.InterfaceIPConfiguration{
								{
									InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
										LoadBalancerBackendAddressPools: &[]network.BackendAddressPool{
											{
												ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-public-lb', 'aro-public-lb-control-plane')]"),
											},
											{
												ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'aro-internal-lb', 'aro-internal-controlplane')]"),
											},
										},
										Subnet: &network.Subnet{
											ID: to.StringPtr(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
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
						Count: len(machinesMaster.MachineFiles),
					},
					DependsOn: []string{
						"Microsoft.Network/loadBalancers/aro-internal-lb",
						"Microsoft.Network/loadBalancers/aro-public-lb",
					},
				},
				{
					Resource: &compute.Image{
						ImageProperties: &compute.ImageProperties{
							StorageProfile: &compute.ImageStorageProfile{
								OsDisk: &compute.ImageOSDisk{
									OsType:  compute.Linux,
									BlobURI: to.StringPtr("https://cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + ".blob.core.windows.net/vhd/rhcos" + doc.OpenShiftCluster.Properties.StorageSuffix + ".vhd"),
								},
							},
							HyperVGeneration: compute.HyperVGenerationTypesV1,
						},
						Name:     to.StringPtr("aro"),
						Type:     to.StringPtr("Microsoft.Compute/images"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["compute"],
				},
				{
					Resource: &compute.VirtualMachine{
						VirtualMachineProperties: &compute.VirtualMachineProperties{
							HardwareProfile: &compute.HardwareProfile{
								VMSize: compute.VirtualMachineSizeTypesStandardD4sV3,
							},
							StorageProfile: &compute.StorageProfile{
								ImageReference: &compute.ImageReference{
									ID: to.StringPtr("[resourceId('Microsoft.Compute/images', 'aro')]"),
								},
								OsDisk: &compute.OSDisk{
									Name:         to.StringPtr("aro-bootstrap_OSDisk"),
									Caching:      compute.CachingTypesReadWrite,
									CreateOption: compute.DiskCreateOptionTypesFromImage,
									DiskSizeGB:   to.Int32Ptr(100),
									ManagedDisk: &compute.ManagedDiskParameters{
										StorageAccountType: compute.StorageAccountTypesPremiumLRS,
									},
								},
							},
							OsProfile: &compute.OSProfile{
								ComputerName:  to.StringPtr("aro-bootstrap-vm"),
								AdminUsername: to.StringPtr("core"),
								AdminPassword: to.StringPtr("NotActuallyApplied!"),
								CustomData:    to.StringPtr(`[base64(concat('{"ignition":{"version":"2.2.0","config":{"replace":{"source":"https://cluster` + doc.OpenShiftCluster.Properties.StorageSuffix + `.blob.core.windows.net/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + doc.OpenShiftCluster.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`),
								LinuxConfiguration: &compute.LinuxConfiguration{
									DisablePasswordAuthentication: to.BoolPtr(false),
								},
							},
							NetworkProfile: &compute.NetworkProfile{
								NetworkInterfaces: &[]compute.NetworkInterfaceReference{
									{
										ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', 'aro-bootstrap-nic')]"),
									},
								},
							},
							DiagnosticsProfile: &compute.DiagnosticsProfile{
								BootDiagnostics: &compute.BootDiagnostics{
									Enabled:    to.BoolPtr(true),
									StorageURI: to.StringPtr("https://cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + ".blob.core.windows.net/"),
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
						"Microsoft.Compute/images/aro",
						"Microsoft.Network/networkInterfaces/aro-bootstrap-nic",
						"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
					},
				},
				{
					Resource: &compute.VirtualMachine{
						VirtualMachineProperties: &compute.VirtualMachineProperties{
							HardwareProfile: &compute.HardwareProfile{
								VMSize: compute.VirtualMachineSizeTypes(installConfig.Config.ControlPlane.Platform.Azure.InstanceType),
							},
							StorageProfile: &compute.StorageProfile{
								ImageReference: &compute.ImageReference{
									ID: to.StringPtr("[resourceId('Microsoft.Compute/images', 'aro')]"),
								},
								OsDisk: &compute.OSDisk{
									Name:         to.StringPtr("[concat('aro-master-', copyIndex(), '_OSDisk')]"),
									Caching:      compute.CachingTypesReadOnly,
									CreateOption: compute.DiskCreateOptionTypesFromImage,
									DiskSizeGB:   &installConfig.Config.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
									ManagedDisk: &compute.ManagedDiskParameters{
										StorageAccountType: compute.StorageAccountTypesPremiumLRS,
									},
								},
							},
							OsProfile: &compute.OSProfile{
								ComputerName:  to.StringPtr("[concat('aro-master-', copyIndex())]"),
								AdminUsername: to.StringPtr("core"),
								AdminPassword: to.StringPtr("NotActuallyApplied!"),
								CustomData:    to.StringPtr(base64.StdEncoding.EncodeToString(machineMaster.File.Data)),
								LinuxConfiguration: &compute.LinuxConfiguration{
									DisablePasswordAuthentication: to.BoolPtr(false),
								},
							},
							NetworkProfile: &compute.NetworkProfile{
								NetworkInterfaces: &[]compute.NetworkInterfaceReference{
									{
										ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', concat('aro-master', copyIndex(), '-nic'))]"),
									},
								},
							},
							DiagnosticsProfile: &compute.DiagnosticsProfile{
								BootDiagnostics: &compute.BootDiagnostics{
									Enabled:    to.BoolPtr(true),
									StorageURI: to.StringPtr("https://cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + ".blob.core.windows.net/"),
								},
							},
						},
						Zones: &[]string{
							"[copyIndex(1)]",
						},
						Name:     to.StringPtr("[concat('aro-master-', copyIndex())]"),
						Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["compute"],
					Copy: &arm.Copy{
						Name:  "computecopy",
						Count: len(machinesMaster.MachineFiles),
					},
					DependsOn: []string{
						"[concat('Microsoft.Authorization/roleAssignments/', guid(resourceGroup().id, 'SP / Contributor'))]",
						"Microsoft.Compute/images/aro",
						"[concat('Microsoft.Network/networkInterfaces/aro-master', copyIndex(), '-nic')]",
						"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
					},
				},
			},
		}

		i.log.Print("deploying resources template")
		err = i.deployments.CreateOrUpdateAndWait(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: t,
				Parameters: map[string]interface{}{
					"sas": map[string]interface{}{
						"value": map[string]interface{}{
							"signedStart":         doc.OpenShiftCluster.Properties.Install.Now.Format(time.RFC3339),
							"signedExpiry":        doc.OpenShiftCluster.Properties.Install.Now.Add(24 * time.Hour).Format(time.RFC3339),
							"signedPermission":    "rl",
							"signedResourceTypes": "o",
							"signedServices":      "b",
							"signedProtocol":      "https",
						},
					},
				},
				Mode: resources.Incremental,
			},
		})
		if err != nil {
			return err
		}
	}

	{
		restConfig, err := restconfig.RestConfig(doc.OpenShiftCluster.Properties.AdminKubeconfig)
		if err != nil {
			return err
		}

		cli, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		i.log.Print("waiting for bootstrap configmap")
		now := time.Now()
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			cm, err := cli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
			if err == nil && cm.Data["status"] == "complete" {
				break
			}

			if time.Now().Sub(now) > 30*time.Minute {
				return fmt.Errorf("timed out waiting for bootstrap configmap. Last error: %v", err)
			}

			<-t.C
		}
	}

	return nil
}
