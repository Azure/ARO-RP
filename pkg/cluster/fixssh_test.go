package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

var (
	resourceGroup  = "rg"
	infraID        = "infra"
	ipc            = "internal-lb-ip-v4"
	masterSubnetID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"
)

func lbBefore(lbID string) *mgmtnetwork.LoadBalancer {
	return &mgmtnetwork.LoadBalancer{
		ID: to.StringPtr(lbID),
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					ID: to.StringPtr(lbID + "/frontendIPConfigurations/" + ipc),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{},
			LoadBalancingRules:  &[]mgmtnetwork.LoadBalancingRule{},
			Probes:              &[]mgmtnetwork.Probe{},
		},
	}
}

func lbAfter(lbID string) *mgmtnetwork.LoadBalancer {
	return &mgmtnetwork.LoadBalancer{
		ID: to.StringPtr(lbID),
		LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
				{
					ID: to.StringPtr(lbID + "/frontendIPConfigurations/" + ipc),
				},
			},
			BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
				{
					Name: to.StringPtr("ssh-0"),
				},
				{
					Name: to.StringPtr("ssh-1"),
				},
				{
					Name: to.StringPtr("ssh-2"),
				},
			},
			LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
				{
					LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/backendAddressPools/ssh-0"),
						},
						Probe: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/probes/ssh"),
						},
						Protocol:             mgmtnetwork.TransportProtocolTCP,
						LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
						FrontendPort:         to.Int32Ptr(2200),
						BackendPort:          to.Int32Ptr(22),
						IdleTimeoutInMinutes: to.Int32Ptr(30),
						DisableOutboundSnat:  to.BoolPtr(true),
					},
					Name: to.StringPtr("ssh-0"),
				},
				{
					LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/backendAddressPools/ssh-1"),
						},
						Probe: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/probes/ssh"),
						},
						Protocol:             mgmtnetwork.TransportProtocolTCP,
						LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
						FrontendPort:         to.Int32Ptr(2201),
						BackendPort:          to.Int32Ptr(22),
						IdleTimeoutInMinutes: to.Int32Ptr(30),
						DisableOutboundSnat:  to.BoolPtr(true),
					},
					Name: to.StringPtr("ssh-1"),
				},
				{
					LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/backendAddressPools/ssh-2"),
						},
						Probe: &mgmtnetwork.SubResource{
							ID: to.StringPtr(lbID + "/probes/ssh"),
						},
						Protocol:             mgmtnetwork.TransportProtocolTCP,
						LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
						FrontendPort:         to.Int32Ptr(2202),
						BackendPort:          to.Int32Ptr(22),
						IdleTimeoutInMinutes: to.Int32Ptr(30),
						DisableOutboundSnat:  to.BoolPtr(true),
					},
					Name: to.StringPtr("ssh-2"),
				},
			},
			Probes: &[]mgmtnetwork.Probe{
				{
					ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
						Protocol:          mgmtnetwork.ProbeProtocolTCP,
						Port:              to.Int32Ptr(22),
						IntervalInSeconds: to.Int32Ptr(5),
						NumberOfProbes:    to.Int32Ptr(2),
					},
					Name: to.StringPtr("ssh"),
				},
			},
		},
	}
}

func configureInterface(backendPools []string,
	subnet string,
	name string,
	addVM bool,
	addIPConfig bool) *armnetwork.InterfacesClientGetResponse {
	iface := armnetwork.Interface{
		Name:       to.StringPtr(name),
		Properties: &armnetwork.InterfacePropertiesFormat{},
	}

	if addVM {
		iface.Properties.VirtualMachine = &armnetwork.SubResource{ID: to.StringPtr(strings.Replace(name, "-nic", "", -1))}
	} else {
		iface.Properties.VirtualMachine = nil
	}

	if addIPConfig {
		var ipConfigurations []*armnetwork.InterfaceIPConfiguration
		ipConfig := armnetwork.InterfaceIPConfiguration{Name: to.StringPtr(name)}
		ipConfig.Properties = &armnetwork.InterfaceIPConfigurationPropertiesFormat{}
		if backendPools != nil {
			var nicBackendPools []*armnetwork.BackendAddressPool
			for _, backendPool := range backendPools {
				nicBackendPools = append(nicBackendPools, &armnetwork.BackendAddressPool{ID: to.StringPtr(backendPool)})
			}
			ipConfig.Properties.LoadBalancerBackendAddressPools = nicBackendPools
		}
		if subnet != "" {
			ipConfig.Properties.Subnet = &armnetwork.Subnet{ID: to.StringPtr(subnet)}
		}
		ipConfigurations = append(ipConfigurations, &ipConfig)
		iface.Properties.IPConfigurations = ipConfigurations
	} else {
		iface.Properties.IPConfigurations = nil
	}

	return &armnetwork.InterfacesClientGetResponse{Interface: iface}
}

// Return a list of interfaces that mocks the state of a newly created cluster (no previous CPMS updates)
// 7 NICs total: 3 for masters, 1 for the private link service and 3 workers
func ifListNewCluster(ilbID string, elbID string, withSSHBackendPool bool) []*armnetwork.Interface {
	var ifList []*armnetwork.Interface
	// 3 master NICs with VM attachments
	for i := range 3 {
		var backendPools []string
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", ilbID, infraID, "-internal-controlplane-v4"))
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", elbID, infraID, "-public-lb-control-plane-v4"))
		if withSSHBackendPool {
			backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/ssh-%d", ilbID, i))
		}
		nicName := fmt.Sprintf("%s-master%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(backendPools, masterSubnetID, nicName, true, true).Interface)
	}
	// 1 NIC in the master subnet with a name that does not match the master NIC name regex, ie the private link service NIC
	ifList = append(ifList, &configureInterface(nil, masterSubnetID, "infra-pls-nic", false, true).Interface)
	// 3 NICs in the worker subnet, don't need to add backend pools, these get skipped anyway
	for i := range 3 {
		nicName := fmt.Sprintf("%s-worker-east%d-12345-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, "worker-subnet", nicName, true, false).Interface)
	}

	return ifList
}

// Return a list of interfaces that mocks the state after the first successful CPMS update of a new cluster
// 10 NICs total: 3 for the old masters, 3 for the new masters, 3 workers and 1 for the private link service
func ifListAfterFirstCPMSUpdate(ilbID string, elbID string, withSSHBackendPool bool) []*armnetwork.Interface {
	var ifList []*armnetwork.Interface
	// 3 NICs with VM attachments, not in SSH backend pools, the new NICs for the new VMs
	for i := range 3 {
		var backendPools []string
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", ilbID, infraID, "-internal-controlplane-v4"))
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", elbID, infraID, "-public-lb-control-plane-v4"))
		if withSSHBackendPool {
			backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/ssh-%d", ilbID, i))
		}
		nicName := fmt.Sprintf("%s-master-12345-%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(backendPools, masterSubnetID, nicName, true, true).Interface)
	}
	// 3 NICs with no VM attachment, the orphaned NICs from the deleted VMs
	for i := range 3 {
		nicName := fmt.Sprintf("%s-master%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, masterSubnetID, nicName, false, true).Interface)
	}
	// 1 NIC in the master subnet with a name that does not match the master NIC name regex, ie the private link service NIC
	ifList = append(ifList, &configureInterface(nil, masterSubnetID, "infra-pls-nic", false, true).Interface)
	// 3 NICs in the worker subnet, don't need to add backend pools, these get skipped anyway
	for i := range 3 {
		nicName := fmt.Sprintf("%s-worker-east%d-12345-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, "worker-subnet", nicName, true, false).Interface)
	}

	return ifList
}

// Return a list of interfaces that mocks the state after the first successful CPMS update of a new private cluster
// 10 NICs total: 3 for the old masters, 3 for the new masters all in the ssh-0 backend pool, 3 workers and 1 for the private link service
func ifListAfterFirstCPMSUpdatePrivateCluster(ilbID string, elbID string, withSSHBackendPool bool) []*armnetwork.Interface {
	var ifList []*armnetwork.Interface
	// 3 NICs with VM attachments, all in ssh-0 backend pool or corrected, the new NICs for the new VMs
	for i := range 3 {
		var backendPools []string
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", ilbID, infraID, "-internal-controlplane-v4"))
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", elbID, infraID, "-public-lb-control-plane-v4"))
		if withSSHBackendPool {
			backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/ssh-%d", ilbID, i))
		} else {
			backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/ssh-%d", ilbID, 0))
		}
		nicName := fmt.Sprintf("%s-master-12345-%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(backendPools, masterSubnetID, nicName, true, true).Interface)
	}
	// 3 NICs with no VM attachment, the orphaned NICs from the deleted VMs
	for i := range 3 {
		nicName := fmt.Sprintf("%s-master%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, masterSubnetID, nicName, false, true).Interface)
	}
	// 1 NIC in the master subnet with a name that does not match the master NIC name regex, ie the private link service NIC
	ifList = append(ifList, &configureInterface(nil, masterSubnetID, "infra-pls-nic", false, true).Interface)
	// 3 NICs in the worker subnet, don't need to add backend pools, these get skipped anyway
	for i := range 3 {
		nicName := fmt.Sprintf("%s-worker-east%d-12345-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, "worker-subnet", nicName, true, false).Interface)
	}

	return ifList
}

// Return a list of interfaces that mocks the state after multiple successful CPMS updates of a cluster
// 7 NICs total: 3 for the masters, 1 for the private link service, 3 workers
func ifListAfterMultipleCPMSUpdates(ilbID string, elbID string, withSSHBackendPool bool) []*armnetwork.Interface {
	var ifList []*armnetwork.Interface
	// 3 NICs with VM attachments, not in SSH backend pools
	for i := range 3 {
		var backendPools []string
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", ilbID, infraID, "-internal-controlplane-v4"))
		backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/%s%s", elbID, infraID, "-public-lb-control-plane-v4"))
		if withSSHBackendPool {
			backendPools = append(backendPools, fmt.Sprintf("%s/backendAddressPools/ssh-%d", ilbID, i))
		}
		nicName := fmt.Sprintf("%s-master-12345-%d-nic", infraID, i)
		ifList = append(ifList, &configureInterface(backendPools, masterSubnetID, nicName, true, true).Interface)
	}
	// 1 NIC in the master subnet with a name that does not match the master NIC name regex, ie the private link service NIC
	ifList = append(ifList, &configureInterface(nil, masterSubnetID, "infra-pls-nic", false, true).Interface)
	// 3 NICs in the worker subnet, don't need to add ILB backend pools, these get skipped anyway
	for i := range 3 {
		nicName := fmt.Sprintf("%s-worker-east%d-12345-nic", infraID, i)
		ifList = append(ifList, &configureInterface(nil, "worker-subnet", nicName, true, false).Interface)
	}

	return ifList
}

func ifListOrphanedNIC() []*armnetwork.Interface {
	var ifList []*armnetwork.Interface
	nicName := fmt.Sprintf("%s-master%d-nic", infraID, 0)
	ifList = append(ifList, &configureInterface(nil, masterSubnetID, nicName, false, true).Interface)

	return ifList
}

func TestFixSSH(t *testing.T) {
	for _, tt := range []struct {
		name                               string
		architectureVersion                api.ArchitectureVersion
		ilb                                string
		ilbID                              string
		elb                                string
		elbID                              string
		elbV1ID                            string
		loadbalancer                       func(string) *mgmtnetwork.LoadBalancer
		interfaces                         func(string, string, bool) []*armnetwork.Interface
		iNameNewF                          string
		iNameOldF                          string
		newCluster                         bool
		afterFirstCPMSUpdate               bool
		afterFirstCPMSUpdatePrivateCluster bool
		afterMultipleCPMSUpdates           bool
		interfacesListError                bool
		emptyInterfacesList                bool
		deleteNICError                     bool
		lbErrorExpected                    bool
		writeExpected                      bool // do we expect write to happen as part of this test
		nicErrorExpected                   bool
		wantError                          string
		ilbBackendPool                     string
		elbBackendPool                     string
	}{
		{
			name:          "Updates resources correctly for newly created cluster",
			ilb:           infraID + "-internal-lb",
			ilbID:         "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:  lbBefore,
			interfaces:    ifListNewCluster,
			iNameOldF:     "%s-master%d-nic",
			newCluster:    true,
			writeExpected: true,
			elb:           infraID + "-public-lb",
			elbV1ID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:         "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
		},
		{
			name:                 "Updates public cluster resources correctly after first CPMS update",
			ilb:                  infraID + "-internal-lb",
			ilbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:         lbBefore,
			iNameNewF:            "%s-master-12345-%d-nic",
			iNameOldF:            "%s-master%d-nic",
			afterFirstCPMSUpdate: true,
			writeExpected:        true,
			elb:                  infraID + "-public-lb",
			elbV1ID:              "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
		},
		{
			name:                               "Updates private cluster resources correctly after first CPMS update",
			ilb:                                infraID + "-internal-lb",
			ilbID:                              "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:                       lbBefore,
			interfaces:                         ifListAfterFirstCPMSUpdatePrivateCluster,
			iNameNewF:                          "%s-master-12345-%d-nic",
			iNameOldF:                          "%s-master%d-nic",
			afterFirstCPMSUpdatePrivateCluster: true,
			writeExpected:                      true,
			elb:                                infraID + "-public-lb",
			elbV1ID:                            "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:                              "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
		},
		{
			name:                     "Updates resources correctly after multiple CPMS updates",
			ilb:                      infraID + "-internal-lb",
			ilbID:                    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:             lbBefore,
			interfaces:               ifListAfterMultipleCPMSUpdates,
			iNameNewF:                "%s-master-12345-%d-nic",
			iNameOldF:                "%s-master%d-nic",
			afterMultipleCPMSUpdates: true,
			writeExpected:            true,
			elb:                      infraID + "-public-lb",
			elbV1ID:                  "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:                    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
		},
		{
			name:                "Interfaces list error expected",
			ilb:                 infraID + "-internal-lb",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:        lbBefore,
			iNameNewF:           "%s-master-12345-%d-nic",
			iNameOldF:           "%s-master%d-nic",
			interfacesListError: true,
			writeExpected:       true,
			elb:                 infraID + "-public-lb",
			elbV1ID:             "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
			wantError:           "interfaces list error",
		},
		{
			name:                "Interfaces list no results",
			ilb:                 infraID + "-internal-lb",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:        lbBefore,
			iNameNewF:           "%s-master-12345-%d-nic",
			iNameOldF:           "%s-master%d-nic",
			emptyInterfacesList: true,
			writeExpected:       true,
			elb:                 infraID + "-public-lb",
			elbV1ID:             "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
			wantError:           "interfaces list call for resource group rg returned an empty result",
		},
		{
			name:           "Failed to delete orphaned NIC",
			ilb:            infraID + "-internal-lb",
			ilbID:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:   lbBefore,
			iNameOldF:      "%s-master%d-nic",
			deleteNICError: true,
			writeExpected:  true,
			elb:            infraID + "-public-lb",
			elbV1ID:        "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
			wantError:      "failed to delete orphaned NIC",
		},
		{
			name:                "FixSSH function returns an error while Fetching LB",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			lbErrorExpected:     true,
			nicErrorExpected:    false,
			wantError:           "load balancer not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			loadBalancers := mock_network.NewMockLoadBalancersClient(ctrl)
			armInterfaces := mock_armnetwork.NewMockInterfacesClient(ctrl)
			createOrUpdateOptions := &armnetwork.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}

			if tt.lbErrorExpected {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.ilb, "").Return(mgmtnetwork.LoadBalancer{}, fmt.Errorf("load balancer not found"))
			} else {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.ilb, "").Return(*tt.loadbalancer(tt.ilbID), nil)
				if tt.writeExpected {
					loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, tt.ilb, *lbAfter(tt.ilbID))
				}
			}

			if tt.newCluster {
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(tt.interfaces(tt.ilbID, tt.elbID, true), nil)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
			}

			if tt.afterFirstCPMSUpdate {
				ifList := ifListAfterFirstCPMSUpdate(tt.ilbID, tt.elbID, true)
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(ifListAfterFirstCPMSUpdate(tt.ilbID, tt.elbID, false), nil)
				// New interfaces post CPMS update
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 0), *ifList[0], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 1), *ifList[1], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 2), *ifList[2], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				// Old interfaces from origin cluster install, orphaned and expected to be deleted
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), *ifList[3], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 1), *ifList[4], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 1), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 2), *ifList[5], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 2), nil)
			}

			if tt.afterFirstCPMSUpdatePrivateCluster {
				ifList := ifListAfterFirstCPMSUpdatePrivateCluster(tt.ilbID, tt.elbID, true)
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(ifListAfterFirstCPMSUpdatePrivateCluster(tt.ilbID, tt.elbID, false), nil)
				// New interfaces post CPMS update
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 1), *ifList[1], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 2), *ifList[2], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				// Old interfaces from origin cluster install, orphaned and expected to be deleted
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), *ifList[3], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 1), *ifList[4], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 1), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 2), *ifList[5], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 2), nil)
			}

			if tt.afterMultipleCPMSUpdates {
				ifList := ifListAfterMultipleCPMSUpdates(tt.ilbID, tt.elbID, true)
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(tt.interfaces(tt.ilbID, tt.elbID, false), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 0), *ifList[0], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 1), *ifList[1], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameNewF, infraID, 2), *ifList[2], createOrUpdateOptions)
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, "").Return(*tt.loadbalancer(tt.elbID), nil)
			}

			if tt.interfacesListError {
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(nil, fmt.Errorf("interfaces list error"))
			}

			if tt.emptyInterfacesList {
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return([]*armnetwork.Interface{}, nil)
			}

			if tt.deleteNICError {
				ifList := ifListOrphanedNIC()
				armInterfaces.EXPECT().List(gomock.Any(), resourceGroup, &armnetwork.InterfacesClientListOptions{}).Return(ifListOrphanedNIC(), nil)
				armInterfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), *ifList[0], createOrUpdateOptions)
				armInterfaces.EXPECT().DeleteAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameOldF, infraID, 0), nil).Return(fmt.Errorf("failed to delete orphaned NIC"))
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: tt.architectureVersion,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup,
							},
							MasterProfile: api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.MasterProfile,
							InfraID:       infraID,
						},
					},
				},
				armInterfaces: armInterfaces,
				loadBalancers: loadBalancers,
			}

			err := m.fixSSH(context.Background())
			if err != nil && !strings.Contains(err.Error(), tt.wantError) ||
				err == nil && tt.wantError != "" {
				t.Error(err)
			}
		})
	}
}
