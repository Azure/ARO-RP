package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (m *manager) dnsPrivateZone(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.PrivateZone{
			Name:     to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain),
			Type:     to.StringPtr("Microsoft.Network/privateDnsZones"),
			Location: to.StringPtr("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/privateDnsZones"),
	}
}

func (m *manager) dnsPrivateRecordAPIINT(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.RecordSet{
			Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api-int"),
			Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
			RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
				TTL: to.Int64Ptr(300),
				ARecords: &[]mgmtprivatedns.ARecord{
					{
						Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/%s-internal', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", m.doc.OpenShiftCluster.Properties.InfraID, azureclient.APIVersion("Microsoft.Network"))),
					},
				},
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/privateDnsZones"),
		DependsOn: []string{
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func (m *manager) dnsPrivateRecordAPI(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtprivatedns.RecordSet{
			Name: to.StringPtr(installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/api"),
			Type: to.StringPtr("Microsoft.Network/privateDnsZones/A"),
			RecordSetProperties: &mgmtprivatedns.RecordSetProperties{
				TTL: to.Int64Ptr(300),
				ARecords: &[]mgmtprivatedns.ARecord{
					{
						Ipv4Address: to.StringPtr(fmt.Sprintf("[reference('Microsoft.Network/loadBalancers/%s-internal', '%s').frontendIpConfigurations[0].properties.privateIPAddress]", m.doc.OpenShiftCluster.Properties.InfraID, azureclient.APIVersion("Microsoft.Network"))),
					},
				},
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network/privateDnsZones"),
		DependsOn: []string{
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func (m *manager) dnsVirtualNetworkLink(installConfig *installconfig.InstallConfig, vnetID string) *arm.Resource {
	return &arm.Resource{
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
		APIVersion: azureclient.APIVersion("Microsoft.Network/privateDnsZones"),
		DependsOn: []string{
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain,
		},
	}
}

func (m *manager) networkBootstrapNIC(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
								},
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
								},
							},
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("bootstrap-nic-ip-v4"),
					},
				},
			},
			Name:     to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap-nic"),
			Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (m *manager) networkMasterNICs(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
								},
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
								},
								{
									ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', concat('ssh-', copyIndex()))]", m.doc.OpenShiftCluster.Properties.InfraID)),
								},
							},
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("pipConfig"),
					},
				},
			},
			Name:     to.StringPtr(fmt.Sprintf("[concat('%s-master', copyIndex(), '-nic')]", m.doc.OpenShiftCluster.Properties.InfraID)),
			Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		Copy: &arm.Copy{
			Name:  "networkcopy",
			Count: int(*installConfig.Config.ControlPlane.Replicas),
		},
	}
}

func (m *manager) computeBootstrapVM(installConfig *installconfig.InstallConfig) *arm.Resource {
	return &arm.Resource{
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
						Name:         to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap_OSDisk"),
						Caching:      mgmtcompute.CachingTypesReadWrite,
						CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   to.Int32Ptr(100),
						ManagedDisk: &mgmtcompute.ManagedDiskParameters{
							StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
						},
					},
				},
				OsProfile: &mgmtcompute.OSProfile{
					ComputerName:  to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap-vm"),
					AdminUsername: to.StringPtr("core"),
					AdminPassword: to.StringPtr("NotActuallyApplied!"),
					CustomData:    to.StringPtr(`[base64(concat('{"ignition":{"version":"3.1.0","config":{"replace":{"source":"https://cluster` + m.doc.OpenShiftCluster.Properties.StorageSuffix + `.blob.` + m.env.Environment().StorageEndpointSuffix + `/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + m.doc.OpenShiftCluster.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`),
					LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(false),
					},
				},
				NetworkProfile: &mgmtcompute.NetworkProfile{
					NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
						{
							ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', '" + m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap-nic')]"),
						},
					},
				},
				DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
					BootDiagnostics: &mgmtcompute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr("https://cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix + ".blob." + m.env.Environment().StorageEndpointSuffix + "/"),
					},
				},
			},
			Name:     to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"Microsoft.Network/networkInterfaces/" + m.doc.OpenShiftCluster.Properties.InfraID + "-bootstrap-nic",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
		},
	}
}

func (m *manager) computeMasterVMs(installConfig *installconfig.InstallConfig, zones *[]string, machineMaster *machine.Master) *arm.Resource {
	return &arm.Resource{
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
						Name:         to.StringPtr("[concat('" + m.doc.OpenShiftCluster.Properties.InfraID + "-master-', copyIndex(), '_OSDisk')]"),
						Caching:      mgmtcompute.CachingTypesReadOnly,
						CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
						DiskSizeGB:   &installConfig.Config.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
						ManagedDisk: &mgmtcompute.ManagedDiskParameters{
							StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
						},
					},
				},
				OsProfile: &mgmtcompute.OSProfile{
					ComputerName:  to.StringPtr("[concat('" + m.doc.OpenShiftCluster.Properties.InfraID + "-master-', copyIndex())]"),
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
							ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', concat('" + m.doc.OpenShiftCluster.Properties.InfraID + "-master', copyIndex(), '-nic'))]"),
						},
					},
				},
				DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
					BootDiagnostics: &mgmtcompute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr("https://cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix + ".blob." + m.env.Environment().StorageEndpointSuffix + "/"),
					},
				},
			},
			Zones:    zones,
			Name:     to.StringPtr("[concat('" + m.doc.OpenShiftCluster.Properties.InfraID + "-master-', copyIndex())]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		Copy: &arm.Copy{
			Name:  "computecopy",
			Count: int(*installConfig.Config.ControlPlane.Replicas),
		},
		DependsOn: []string{
			"[concat('Microsoft.Network/networkInterfaces/" + m.doc.OpenShiftCluster.Properties.InfraID + "-master', copyIndex(), '-nic')]",
			"Microsoft.Network/privateDnsZones/" + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/virtualNetworkLinks/" + installConfig.Config.ObjectMeta.Name + "-network-link",
		},
	}
}
