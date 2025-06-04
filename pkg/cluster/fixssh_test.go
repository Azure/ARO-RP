package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

var (
	resourceGroup = "rg"
	infraID       = "infra"
	ipc           = "internal-lb-ip-v4"
)

func lbBefore(lbID string) armnetwork.LoadBalancer {
	return armnetwork.LoadBalancer{
		ID: pointerutils.ToPtr(lbID),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					ID: pointerutils.ToPtr(lbID + "/frontendIPConfigurations/" + ipc),
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{},
			LoadBalancingRules:  []*armnetwork.LoadBalancingRule{},
			Probes:              []*armnetwork.Probe{},
		},
	}
}

func lbAfter(lbID string) armnetwork.LoadBalancer {
	return armnetwork.LoadBalancer{
		ID: pointerutils.ToPtr(lbID),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					ID: pointerutils.ToPtr(lbID + "/frontendIPConfigurations/" + ipc),
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: pointerutils.ToPtr("ssh-0"),
				},
				{
					Name: pointerutils.ToPtr("ssh-1"),
				},
				{
					Name: pointerutils.ToPtr("ssh-2"),
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/backendAddressPools/ssh-0"),
						},
						Probe: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2200)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
					Name: pointerutils.ToPtr("ssh-0"),
				},
				{
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/backendAddressPools/ssh-1"),
						},
						Probe: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2201)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
					Name: pointerutils.ToPtr("ssh-1"),
				},
				{
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/frontendIPConfigurations/" + ipc),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/backendAddressPools/ssh-2"),
						},
						Probe: &armnetwork.SubResource{
							ID: pointerutils.ToPtr(lbID + "/probes/ssh"),
						},
						Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
						LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
						FrontendPort:         pointerutils.ToPtr(int32(2202)),
						BackendPort:          pointerutils.ToPtr(int32(22)),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
						DisableOutboundSnat:  pointerutils.ToPtr(true),
					},
					Name: pointerutils.ToPtr("ssh-2"),
				},
			},
			Probes: []*armnetwork.Probe{
				{
					Properties: &armnetwork.ProbePropertiesFormat{
						Protocol:          pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
						Port:              pointerutils.ToPtr(int32(22)),
						IntervalInSeconds: pointerutils.ToPtr(int32(5)),
						NumberOfProbes:    pointerutils.ToPtr(int32(2)),
					},
					Name: pointerutils.ToPtr("ssh"),
				},
			},
		},
	}
}

func ifBefore(ilbID string, elbID string, i int, ilbBackendPool string, elbBackendPool string) armnetwork.Interface {
	return armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			VirtualMachine: &armnetwork.SubResource{
				ID: pointerutils.ToPtr(fmt.Sprintf("master-%d", i)),
			},
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{},
					},
				},
			},
		},
	}
}

func ifNoVmBefore(ilbID string, elbID string, i int, ilbBackendPool string, elbBackendPool string) armnetwork.Interface {
	return armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			VirtualMachine: nil,
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{
							{
								ID: pointerutils.ToPtr(fmt.Sprintf(ilbID+"/backendAddressPools/ssh-%d", i)),
							},
						},
					},
				},
			},
		},
	}
}

func ifNoVmAfter(nic armnetwork.Interface) armnetwork.Interface {
	nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = []*armnetwork.BackendAddressPool{}
	return nic
}

func ifAfter(ilbID string, elbID string, i int, ilbBackendPool string, elbBackendPool string) armnetwork.Interface {
	return armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			VirtualMachine: &armnetwork.SubResource{
				ID: pointerutils.ToPtr(fmt.Sprintf("master-%d", i)),
			},
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: []*armnetwork.BackendAddressPool{
							{
								ID: pointerutils.ToPtr(fmt.Sprintf(ilbID+"/backendAddressPools/ssh-%d", i)),
							},
							{
								ID: pointerutils.ToPtr(fmt.Sprintf(ilbID+"/backendAddressPools/%s", ilbBackendPool)),
							},
							{
								ID: pointerutils.ToPtr(fmt.Sprintf(elbID+"/backendAddressPools/%s", elbBackendPool)),
							},
						},
					},
				},
			},
		},
	}
}

func TestFixSSH(t *testing.T) {
	for _, tt := range []struct {
		name                string
		architectureVersion api.ArchitectureVersion
		ilb                 string
		ilbID               string
		elb                 string
		elbID               string
		elbV1ID             string
		loadbalancer        func(string) armnetwork.LoadBalancer
		iface               func(string, string, int, string, string) armnetwork.Interface
		iNameF              string
		ifaceNoVmAttached   bool // create the NIC without a master VM attached, to simulate a master node replacement
		lbErrorExpected     bool
		writeExpected       bool // do we expect write to happen as part of this test
		fallbackExpected    bool // do we expect fallback nic.Get as part of this test
		nicErrorExpected    bool
		noOperationExpected bool
		wantError           string
		ilbBackendPool      string
		elbBackendPool      string
	}{
		{
			name:          "updates v1 resources correctly",
			ilb:           infraID + "-internal-lb",
			ilbID:         "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:  lbBefore,
			iface:         ifBefore,
			iNameF:        "%s-master%d-nic",
			writeExpected: true,
			elb:           infraID + "-public-lb",
			elbV1ID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			elbID:         "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
		},
		{
			name:                "v1 noop",
			ilb:                 infraID + "-internal-lb",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:        lbAfter,
			iface:               ifAfter,
			iNameF:              "%s-master%d-nic",
			elb:                 infraID + "-public-lb",
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-public-lb",
			ilbBackendPool:      infraID + "-internal-controlplane-v4",
			elbBackendPool:      infraID + "-public-lb-control-plane-v4",
			noOperationExpected: true,
		},
		{
			name:                "updates v2 resources correctly",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifBefore,
			iNameF:              "%s-master%d-nic",
			writeExpected:       true,
			elb:                 infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			ilbBackendPool:      infraID,
			elbBackendPool:      infraID,
		},
		{
			name:                "v2 noop",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbAfter,
			iface:               ifAfter,
			iNameF:              "%s-master%d-nic",
			elb:                 infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			ilbBackendPool:      infraID,
			elbBackendPool:      infraID,
			noOperationExpected: true,
		},
		{
			name:                "updates v2 resources correctly with masters recreated",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifBefore,
			iNameF:              "%s-master-%d-nic",
			writeExpected:       true,
			fallbackExpected:    true,
			elb:                 infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			ilbBackendPool:      infraID,
			elbBackendPool:      infraID,
		},
		{
			name:                "updates v2 resources correctly with masters recreated and no VM attached to the installer NIC",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifNoVmBefore,
			iNameF:              "%s-master-%d-nic",
			ifaceNoVmAttached:   true,
			writeExpected:       true,
			fallbackExpected:    true,
			elb:                 infraID,
			elbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID,
			ilbBackendPool:      infraID,
			elbBackendPool:      infraID,
		},
		{
			name:                "FixSSH function returns an error while Fetching LB",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifNoVmBefore,
			iNameF:              "%s-master%d-nic",
			writeExpected:       false,
			fallbackExpected:    false,
			lbErrorExpected:     true,
			nicErrorExpected:    false,
			wantError:           "Loadbalancer not found",
		},
		{
			name:                "FixSSH function returns an error while Fetching NIC",
			architectureVersion: api.ArchitectureVersionV2,
			ilb:                 infraID + "-internal",
			ilbID:               "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifNoVmBefore,
			iNameF:              "%s-master-%d-nic",
			writeExpected:       true,
			fallbackExpected:    false,
			lbErrorExpected:     false,
			nicErrorExpected:    true,
			wantError:           "Interface not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			interfaces := mock_armnetwork.NewMockInterfacesClient(ctrl)
			loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(ctrl)

			if tt.lbErrorExpected {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.ilb, nil).Return(armnetwork.LoadBalancersClientGetResponse{}, fmt.Errorf("Loadbalancer not found"))
			} else {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.ilb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.ilbID)}, nil)
				if tt.writeExpected {
					loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, tt.ilb, lbAfter(tt.ilbID), nil)
				}
			}

			for i := 0; i < 3; i++ {
				vmNicBefore := tt.iface(tt.ilbID, tt.elbID, i, tt.ilbBackendPool, tt.elbBackendPool)

				if tt.fallbackExpected { // bit of hack to check fallback.
					if tt.ifaceNoVmAttached {
						vmNicBefore = ifNoVmBefore(tt.ilbID, tt.elbID, i, tt.ilbBackendPool, tt.elbBackendPool)
						interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{Interface: vmNicBefore}, nil)
					} else {
						interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{}, fmt.Errorf("nic not found"))
					}
				}

				if tt.nicErrorExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{}, fmt.Errorf("Interface not found"))
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{}, fmt.Errorf("Interface not found"))
					break
				} else if tt.lbErrorExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), nil).Times(0)
				} else if tt.architectureVersion == api.ArchitectureVersionV2 && tt.noOperationExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{Interface: vmNicBefore}, nil)
					loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbID)}, nil)
				} else if tt.noOperationExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{Interface: vmNicBefore}, nil)
					loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, infraID, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbV1ID)}, nil)
					loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbID)}, nil)
				} else {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), nil).Return(armnetwork.InterfacesClientGetResponse{Interface: vmNicBefore}, nil)
				}

				if tt.writeExpected {
					if tt.ifaceNoVmAttached {
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), vmNicBefore, nil)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), ifNoVmAfter(vmNicBefore), nil)
						loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbID)}, nil)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), ifNoVmAfter(vmNicBefore), nil)
					} else if tt.architectureVersion == api.ArchitectureVersionV2 {
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), vmNicBefore, nil)
						loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbID)}, nil)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), vmNicBefore, nil)
					} else {
						loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, infraID, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbV1ID)}, nil)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), vmNicBefore, nil)
						loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.elb, nil).Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: tt.loadbalancer(tt.elbID)}, nil)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), vmNicBefore, nil)
					}
				}
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
							InfraID: infraID,
						},
					},
				},
				armInterfaces:    interfaces,
				armLoadBalancers: loadBalancers,
			}

			err := m.fixSSH(context.Background())
			if err != nil && err.Error() != tt.wantError ||
				err == nil && tt.wantError != "" {
				t.Error(err)
			}
		})
	}
}
