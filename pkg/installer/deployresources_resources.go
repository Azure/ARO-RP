package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (m *manager) networkBootstrapNIC(installConfig *installconfig.InstallConfig) *arm.Resource {
	// Private clusters without Public IPs will not have valid external load balancers
	lbBackendAddressPool := &[]mgmtnetwork.BackendAddressPool{
		{
			ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.oc.Properties.InfraID)),
		},
	}
	if m.oc.Properties.NetworkProfile.OutboundType == api.OutboundTypeLoadbalancer {
		*lbBackendAddressPool = append(*lbBackendAddressPool, mgmtnetwork.BackendAddressPool{
			ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.oc.Properties.InfraID)),
		})
	}
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: lbBackendAddressPool,
							Subnet: &mgmtnetwork.Subnet{
								ID: &m.oc.Properties.MasterProfile.SubnetID,
							},
						},
						Name: to.StringPtr("bootstrap-nic-ip-v4"),
					},
				},
			},
			Name:     to.StringPtr(m.oc.Properties.InfraID + "-bootstrap-nic"),
			Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
			Location: &installConfig.Config.Azure.Region,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (m *manager) networkMasterNICs(installConfig *installconfig.InstallConfig) *arm.Resource {
	// Private clusters without Public IPs not have valid external load balancers
	lbBackendAddressPool := &[]mgmtnetwork.BackendAddressPool{
		{
			ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.oc.Properties.InfraID)),
		},
		{
			ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', concat('ssh-', copyIndex()))]", m.oc.Properties.InfraID)),
		},
	}
	if m.oc.Properties.NetworkProfile.OutboundType == api.OutboundTypeLoadbalancer {
		*lbBackendAddressPool = append(*lbBackendAddressPool, mgmtnetwork.BackendAddressPool{
			ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.oc.Properties.InfraID)),
		})
	}
	return &arm.Resource{
		Resource: &mgmtnetwork.Interface{
			InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
					{
						InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
							LoadBalancerBackendAddressPools: lbBackendAddressPool,
							Subnet: &mgmtnetwork.Subnet{
								ID: &m.oc.Properties.MasterProfile.SubnetID,
							},
						},
						Name: to.StringPtr("pipConfig"),
					},
				},
			},
			Name:     to.StringPtr(fmt.Sprintf("[concat('%s-master', copyIndex(), '-nic')]", m.oc.Properties.InfraID)),
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
	var customData string
	if m.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
		customData = `[base64(concat('{"ignition":{"version":"3.2.0","proxy":{"httpsProxy":"http://` + m.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP + `"},"config":{"replace":{"source":"https://cluster` + m.oc.Properties.StorageSuffix + `.blob.` + m.env.Environment().StorageEndpointSuffix + `/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + m.oc.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`
	} else {
		customData = `[base64(concat('{"ignition":{"version":"3.2.0","config":{"replace":{"source":"https://cluster` + m.oc.Properties.StorageSuffix + `.blob.` + m.env.Environment().StorageEndpointSuffix + `/ignition/bootstrap.ign?', listAccountSas(resourceId('Microsoft.Storage/storageAccounts', 'cluster` + m.oc.Properties.StorageSuffix + `'), '2019-04-01', parameters('sas')).accountSasToken, '"}}}}'))]`
	}

	vm := &mgmtcompute.VirtualMachine{
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
					Name:         to.StringPtr(m.oc.Properties.InfraID + "-bootstrap_OSDisk"),
					Caching:      mgmtcompute.CachingTypesReadWrite,
					CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
					DiskSizeGB:   to.Int32Ptr(100),
					ManagedDisk: &mgmtcompute.ManagedDiskParameters{
						StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
					},
				},
			},
			OsProfile: &mgmtcompute.OSProfile{
				ComputerName:  to.StringPtr(m.oc.Properties.InfraID + "-bootstrap-vm"),
				AdminUsername: to.StringPtr("core"),
				AdminPassword: to.StringPtr("NotActuallyApplied!"),
				CustomData:    &customData,
				LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(false),
				},
			},
			NetworkProfile: &mgmtcompute.NetworkProfile{
				NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
					{
						ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', '" + m.oc.Properties.InfraID + "-bootstrap-nic')]"),
					},
				},
			},
			DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
				BootDiagnostics: &mgmtcompute.BootDiagnostics{
					Enabled:    to.BoolPtr(true),
					StorageURI: to.StringPtr("https://cluster" + m.oc.Properties.StorageSuffix + ".blob." + m.env.Environment().StorageEndpointSuffix + "/"),
				},
			},
		},
		Name:     to.StringPtr(m.oc.Properties.InfraID + "-bootstrap"),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
		Location: &installConfig.Config.Azure.Region,
	}

	if installConfig.Config.ControlPlane.Platform.Azure.DiskEncryptionSetID != "" {
		vm.StorageProfile.OsDisk.ManagedDisk.DiskEncryptionSet = &mgmtcompute.DiskEncryptionSetParameters{
			ID: &installConfig.Config.ControlPlane.Platform.Azure.DiskEncryptionSetID,
		}
	}

	if installConfig.Config.ControlPlane.Platform.Azure.EncryptionAtHost {
		vm.SecurityProfile = &mgmtcompute.SecurityProfile{
			EncryptionAtHost: &installConfig.Config.ControlPlane.Platform.Azure.EncryptionAtHost,
		}
	}
	return &arm.Resource{
		Resource:   vm,
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"Microsoft.Network/networkInterfaces/" + m.oc.Properties.InfraID + "-bootstrap-nic",
		},
	}
}

func (m *manager) computeMasterVMs(installConfig *installconfig.InstallConfig, zones *[]string, machineMaster *machine.Master) *arm.Resource {
	vm := &mgmtcompute.VirtualMachine{
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
					Name:         to.StringPtr("[concat('" + m.oc.Properties.InfraID + "-master-', copyIndex(), '_OSDisk')]"),
					Caching:      mgmtcompute.CachingTypesReadOnly,
					CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
					DiskSizeGB:   &installConfig.Config.ControlPlane.Platform.Azure.OSDisk.DiskSizeGB,
					ManagedDisk: &mgmtcompute.ManagedDiskParameters{
						StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
					},
				},
			},
			OsProfile: &mgmtcompute.OSProfile{
				ComputerName:  to.StringPtr("[concat('" + m.oc.Properties.InfraID + "-master-', copyIndex())]"),
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
						ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', concat('" + m.oc.Properties.InfraID + "-master', copyIndex(), '-nic'))]"),
					},
				},
			},
			DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
				BootDiagnostics: &mgmtcompute.BootDiagnostics{
					Enabled:    to.BoolPtr(true),
					StorageURI: to.StringPtr("https://cluster" + m.oc.Properties.StorageSuffix + ".blob." + m.env.Environment().StorageEndpointSuffix + "/"),
				},
			},
		},
		Zones:    zones,
		Name:     to.StringPtr("[concat('" + m.oc.Properties.InfraID + "-master-', copyIndex())]"),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
		Location: &installConfig.Config.Azure.Region,
	}

	if installConfig.Config.ControlPlane.Platform.Azure.DiskEncryptionSetID != "" {
		vm.StorageProfile.OsDisk.ManagedDisk.DiskEncryptionSet = &mgmtcompute.DiskEncryptionSetParameters{
			ID: &installConfig.Config.ControlPlane.Platform.Azure.DiskEncryptionSetID,
		}
	}

	if installConfig.Config.ControlPlane.Platform.Azure.EncryptionAtHost {
		vm.SecurityProfile = &mgmtcompute.SecurityProfile{
			EncryptionAtHost: &installConfig.Config.ControlPlane.Platform.Azure.EncryptionAtHost,
		}
	}

	return &arm.Resource{
		Resource:   vm,
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		Copy: &arm.Copy{
			Name:  "computecopy",
			Count: int(*installConfig.Config.ControlPlane.Replicas),
		},
		DependsOn: []string{
			"[concat('Microsoft.Network/networkInterfaces/" + m.oc.Properties.InfraID + "-master', copyIndex(), '-nic')]",
		},
	}
}
