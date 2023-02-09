package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

var (
	resourceGroup = "rg"
	infraID       = "infra"
	ipc           = "internal-lb-ip-v4"
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

func ifBefore(lbID string, i int, ilbBackendPool string) *mgmtnetwork.Interface {
	return &mgmtnetwork.Interface{
		InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
			VirtualMachine: &mgmtnetwork.SubResource{
				ID: to.StringPtr(fmt.Sprintf("master-%d", i)),
			},
			IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
				{
					InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{},
					},
				},
			},
		},
	}
}

func ifNoVmBefore(lbID string, i int, ilbBackendPool string) *mgmtnetwork.Interface {
	return &mgmtnetwork.Interface{
		InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
			VirtualMachine: nil,
			IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
				{
					InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
							{
								ID: to.StringPtr(fmt.Sprintf(lbID+"/backendAddressPools/ssh-%d", i)),
							},
						},
					},
				},
			},
		},
	}
}

func ifNoVmAfter(nic *mgmtnetwork.Interface) *mgmtnetwork.Interface {
	emptyAddressPool := make([]mgmtnetwork.BackendAddressPool, 0)
	(*nic.InterfacePropertiesFormat.IPConfigurations)[0].InterfaceIPConfigurationPropertiesFormat.LoadBalancerBackendAddressPools = &emptyAddressPool
	return nic
}

func ifAfter(lbID string, i int, ilbBackendPool string) *mgmtnetwork.Interface {
	return &mgmtnetwork.Interface{
		InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
			VirtualMachine: &mgmtnetwork.SubResource{
				ID: to.StringPtr(fmt.Sprintf("master-%d", i)),
			},
			IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
				{
					InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
							{
								ID: to.StringPtr(fmt.Sprintf(lbID+"/backendAddressPools/ssh-%d", i)),
							},
							{
								ID: to.StringPtr(fmt.Sprintf(lbID+"/backendAddressPools/%s", ilbBackendPool)),
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
		lb                  string
		lbID                string
		loadbalancer        func(string) *mgmtnetwork.LoadBalancer
		iface               func(string, int, string) *mgmtnetwork.Interface
		iNameF              string
		ifaceNoVmAttached   bool // create the NIC without a master VM attached, to simulate a master node replacement
		lbErrorExpected     bool
		writeExpected       bool // do we expect write to happen as part of this test
		fallbackExpected    bool // do we expect fallback nic.Get as part of this test
		nicErrorExpected    bool
		wantError           string
		ilbBackendPool      string
	}{
		{
			name:          "updates v1 resources correctly",
			lb:            infraID + "-internal-lb",
			lbID:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:  lbBefore,
			iface:         ifBefore,
			iNameF:        "%s-master%d-nic",
			writeExpected: true,
		},
		{
			name:           "v1 noop",
			lb:             infraID + "-internal-lb",
			lbID:           "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal-lb",
			loadbalancer:   lbAfter,
			iface:          ifAfter,
			iNameF:         "%s-master%d-nic",
			ilbBackendPool: infraID + "-internal-controlplane-v4",
		},
		{
			name:                "updates v2 resources correctly",
			architectureVersion: api.ArchitectureVersionV2,
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifBefore,
			iNameF:              "%s-master%d-nic",
			writeExpected:       true,
		},
		{
			name:                "v2 noop",
			architectureVersion: api.ArchitectureVersionV2,
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbAfter,
			iface:               ifAfter,
			iNameF:              "%s-master%d-nic",
			ilbBackendPool:      infraID,
		},
		{
			name:                "updates v2 resources correctly with masters recreated",
			architectureVersion: api.ArchitectureVersionV2,
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifBefore,
			iNameF:              "%s-master-%d-nic",
			writeExpected:       true,
			fallbackExpected:    true,
		},
		{
			name:                "updates v2 resources correctly with masters recreated and no VM attached to the installer NIC",
			architectureVersion: api.ArchitectureVersionV2,
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
			loadbalancer:        lbBefore,
			iface:               ifNoVmBefore,
			iNameF:              "%s-master-%d-nic",
			ifaceNoVmAttached:   true,
			writeExpected:       true,
			fallbackExpected:    true,
		},
		{
			name:                "FixSSH function returns an error while Fetching LB",
			architectureVersion: api.ArchitectureVersionV2,
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
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
			lb:                  infraID + "-internal",
			lbID:                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + infraID + "-internal",
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

			interfaces := mock_network.NewMockInterfacesClient(ctrl)
			loadBalancers := mock_network.NewMockLoadBalancersClient(ctrl)

			if tt.lbErrorExpected {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.lb, "").Return(mgmtnetwork.LoadBalancer{}, fmt.Errorf("Loadbalancer not found"))
			} else {
				loadBalancers.EXPECT().Get(gomock.Any(), resourceGroup, tt.lb, "").Return(*tt.loadbalancer(tt.lbID), nil)
				if tt.writeExpected {
					loadBalancers.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, tt.lb, *lbAfter(tt.lbID))
				}
			}

			for i := 0; i < 3; i++ {
				vmNicBefore := tt.iface(tt.lbID, i, tt.ilbBackendPool)

				if tt.fallbackExpected { // bit of hack to check fallback.
					if tt.ifaceNoVmAttached {
						vmNicBefore = ifNoVmBefore(tt.lbID, i, tt.ilbBackendPool)
						interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), "").Return(*vmNicBefore, nil)
					} else {
						interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), "").Return(mgmtnetwork.Interface{}, fmt.Errorf("nic not found"))
					}
				}

				if tt.nicErrorExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), "").Return(mgmtnetwork.Interface{}, fmt.Errorf("Interface not found"))
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), "").Return(mgmtnetwork.Interface{}, fmt.Errorf("Interface not found"))
					break
				} else if tt.lbErrorExpected {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), "").Times(0)
				} else {
					interfaces.EXPECT().Get(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), "").Return(*vmNicBefore, nil)
				}

				if tt.writeExpected {
					if tt.ifaceNoVmAttached {
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf("%s-master%d-nic", infraID, i), *vmNicBefore)
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), *ifNoVmAfter(vmNicBefore))
					} else {
						interfaces.EXPECT().CreateOrUpdateAndWait(gomock.Any(), resourceGroup, fmt.Sprintf(tt.iNameF, infraID, i), *vmNicBefore)
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
				interfaces:    interfaces,
				loadBalancers: loadBalancers,
			}

			err := m.fixSSH(context.Background())
			if err != nil && err.Error() != tt.wantError ||
				err == nil && tt.wantError != "" {
				t.Error(err)
			}
		})
	}
}
