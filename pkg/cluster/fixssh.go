package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) fixSSH(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	lb, err := m.loadBalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	changed := updateLB(&lb)

	if changed {
		m.log.Printf("updating %s", lbName)
		err = m.loadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
		if err != nil {
			return err
		}
	}

	for i := 0; i < 3; i++ {
		// NIC names might be different if customer re-created master nodes
		// see https://bugzilla.redhat.com/show_bug.cgi?id=1882490 for more details
		// installer naming  - <foo>-master{0,1,2}-nic
		// machineAPI naming - <foo>-master-{0,1,2}-nic
		nicNameInstaller := fmt.Sprintf("%s-master%d-nic", infraID, i)
		nicNameMachineAPI := fmt.Sprintf("%s-master-%d-nic", infraID, i)

		var iErr, err error
		var nic mgmtnetwork.Interface
		nicName := nicNameInstaller

		nic, err = m.interfaces.Get(ctx, resourceGroup, nicName, "")
		if err != nil {
			iErr = err // preserve original error
			nicName = nicNameMachineAPI
			m.log.Warnf("fallback to check MachineAPI Nic name format for %s", nicName)
			nic, err = m.interfaces.Get(ctx, resourceGroup, nicName, "")
			if err != nil {
				m.log.Warnf("fallback failed with err %s", err)
				return iErr
			}
		} else if nic.InterfacePropertiesFormat != nil && nic.InterfacePropertiesFormat.VirtualMachine == nil {
			err = m.removeBackendPoolsFromNIC(ctx, resourceGroup, nicName, &nic)
			if err != nil {
				m.log.Warnf("Removing BackendPools from NIC %s has failed with err %s", nicName, err)
				return err
			}
			nicName = nicNameMachineAPI
			m.log.Warnf("installer provisioned NIC has no VM attached, fallback to check MachineAPI NIC name format for %s", nicName)
			nic, err = m.interfaces.Get(ctx, resourceGroup, nicName, "")
			if err != nil {
				m.log.Warnf("fallback failed with err %s", err)
				return err
			}
		}
		changed = updateNIC(&nic, &lb, i)

		if changed {
			m.log.Printf("updating %s", nicName)
			err = m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, nic)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *manager) removeBackendPoolsFromNIC(ctx context.Context, resourceGroup, nicName string, nic *mgmtnetwork.Interface) error {
	ipc := (*nic.InterfacePropertiesFormat.IPConfigurations)[0]
	if ipc.LoadBalancerBackendAddressPools != nil {
		m.log.Printf("Removing Loadbalancer Backend Address Pools from NIC %s with no VMs attached", nicName)
		*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = []mgmtnetwork.BackendAddressPool{}
		return m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic)
	}
	return nil
}

func updateNIC(nic *mgmtnetwork.Interface, lb *mgmtnetwork.LoadBalancer, i int) bool {
	id := fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)
	ipc := (*nic.InterfacePropertiesFormat.IPConfigurations)[0]
	if ipc.LoadBalancerBackendAddressPools == nil {
		backendAddressPool := make([]mgmtnetwork.BackendAddressPool, 0)
		ipc.LoadBalancerBackendAddressPools = &backendAddressPool
	}
	for _, p := range *(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools {
		if strings.EqualFold(*p.ID, id) {
			return false
		}
	}
	*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = append(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools, mgmtnetwork.BackendAddressPool{
		ID: &id,
	})
	return true
}

func updateLB(lb *mgmtnetwork.LoadBalancer) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range *lb.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		*lb.BackendAddressPools = append(*lb.BackendAddressPools, mgmtnetwork.BackendAddressPool{
			Name: &name,
		})
	}

loadBalancingRules:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, r := range *lb.LoadBalancingRules {
			if strings.EqualFold(*r.Name, name) {
				continue loadBalancingRules
			}
		}

		changed = true
		*lb.LoadBalancingRules = append(*lb.LoadBalancingRules, mgmtnetwork.LoadBalancingRule{
			LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &mgmtnetwork.SubResource{
					ID: (*lb.FrontendIPConfigurations)[0].ID,
				},
				BackendAddressPool: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)),
				},
				Probe: &mgmtnetwork.SubResource{
					ID: to.StringPtr(*lb.ID + "/probes/ssh"),
				},
				Protocol:             mgmtnetwork.TransportProtocolTCP,
				LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
				FrontendPort:         to.Int32Ptr(2200 + int32(i)),
				BackendPort:          to.Int32Ptr(22),
				IdleTimeoutInMinutes: to.Int32Ptr(30),
				DisableOutboundSnat:  to.BoolPtr(true),
			},
			Name: &name,
		})
	}

	for _, p := range *lb.Probes {
		if strings.EqualFold(*p.Name, "ssh") {
			return changed
		}
	}

	changed = true
	*lb.Probes = append(*lb.Probes, mgmtnetwork.Probe{
		ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
			Protocol:          mgmtnetwork.ProbeProtocolTCP,
			Port:              to.Int32Ptr(22),
			IntervalInSeconds: to.Int32Ptr(5),
			NumberOfProbes:    to.Int32Ptr(2),
		},
		Name: to.StringPtr("ssh"),
	})

	return changed
}
