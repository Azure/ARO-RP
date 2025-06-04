package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) fixSSH(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	lb, err := m.checkAndUpdateLB(ctx, resourceGroup, lbName)
	if err != nil {
		m.log.Warnf("Failed checking and Updating Load Balancer with error: %s", err)
		return err
	}

	err = m.checkandUpdateNIC(ctx, resourceGroup, infraID, lb)
	if err != nil {
		m.log.Warnf("Failed checking and Updating Network Interface with error: %s", err)
		return err
	}

	return nil
}

func (m *manager) checkAndUpdateLB(ctx context.Context, resourceGroup string, lbName string) (*armnetwork.LoadBalancer, error) {
	_lb, err := m.armLoadBalancers.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return nil, err
	}

	lb := &_lb.LoadBalancer
	if m.updateLB(ctx, lb, lbName) {
		m.log.Printf("updating Load Balancer %s", lbName)
		err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, *lb, nil)
		if err != nil {
			return nil, err
		}
	}
	return lb, nil
}

func (m *manager) checkandUpdateNIC(ctx context.Context, resourceGroup string, infraID string, lb *armnetwork.LoadBalancer) (err error) {
	for i := 0; i < 3; i++ {
		// NIC names might be different if customer re-created master nodes
		// see https://bugzilla.redhat.com/show_bug.cgi?id=1882490 for more details
		// installer naming  - <foo>-master{0,1,2}-nic
		// machineAPI naming - <foo>-master-{0,1,2}-nic
		nicNameInstaller := fmt.Sprintf("%s-master%d-nic", infraID, i)
		nicNameMachineAPI := fmt.Sprintf("%s-master-%d-nic", infraID, i)

		var nic armnetwork.Interface
		nicName := nicNameInstaller
		fallbackNIC := false

		_nic, err := m.armInterfaces.Get(ctx, resourceGroup, nicName, nil)
		if err != nil {
			m.log.Warnf("Fetching details for NIC %s has failed with err %s", nicName, err)
			fallbackNIC = true
		} else if _nic.Properties != nil && _nic.Properties.VirtualMachine == nil {
			err = m.removeBackendPoolsFromNIC(ctx, resourceGroup, nicName, _nic.Interface)
			if err != nil {
				m.log.Warnf("Removing BackendPools from NIC %s has failed with err %s", nicName, err)
				return err
			}
			m.log.Warnf("Installer provisioned NIC %s has no VM attached", nicName)
			fallbackNIC = true
		}

		if fallbackNIC {
			nicName = nicNameMachineAPI
			m.log.Warnf("Fallback to check MachineAPI Nic name format for %s", nicName)
			_nic, err = m.armInterfaces.Get(ctx, resourceGroup, nicName, nil)
			if err != nil {
				m.log.Warnf("Fallback failed with err %s", err)
				return err
			}
			nic = _nic.Interface
		} else {
			nic = _nic.Interface
		}

		err = m.updateILBAddressPool(ctx, nic, nicName, lb, i, resourceGroup, infraID)
		if err != nil {
			return err
		}

		if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
			return nil
		}

		elbName := infraID
		if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
			err = m.updateV1ELBAddressPool(ctx, &nic, nicName, resourceGroup, infraID)
			if err != nil {
				return err
			}
			elbName = infraID + "-public-lb"
		}

		elb, err := m.armLoadBalancers.Get(ctx, resourceGroup, elbName, nil)
		if err != nil {
			return err
		}

		err = m.updateELBAddressPool(ctx, nic, nicName, &elb.LoadBalancer, resourceGroup, infraID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) removeBackendPoolsFromNIC(ctx context.Context, resourceGroup, nicName string, nic armnetwork.Interface) error {
	if len(nic.Properties.IPConfigurations) == 0 {
		return fmt.Errorf("unable to remove Backend Address Pools from NIC as there are no IP configurations for %s in resource group %s", nicName, resourceGroup)
	}
	ipc := nic.Properties.IPConfigurations[0]
	if ipc.Properties.LoadBalancerBackendAddressPools != nil {
		m.log.Printf("Removing Load balancer Backend Address Pools from NIC %s with no VMs attached", nicName)
		nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = []*armnetwork.BackendAddressPool{}
		return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, nic, nil)
	}
	return nil
}

func (m *manager) updateILBAddressPool(ctx context.Context, nic armnetwork.Interface, nicName string, lb *armnetwork.LoadBalancer, i int, resourceGroup string, infraID string) error {
	if len(nic.Properties.IPConfigurations) == 0 {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", nicName)
	}

	ilbBackendPool := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPool = infraID + "-internal-controlplane-v4"
	}

	sshBackendPoolID := fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)
	ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, ilbBackendPool)

	updateSSHPool := true
	updateILBPool := true

	ipc := nic.Properties.IPConfigurations[0]
	if ipc.Properties.LoadBalancerBackendAddressPools == nil {
		emptyBackendAddressPool := make([]*armnetwork.BackendAddressPool, 0)
		nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = emptyBackendAddressPool
	} else {
		for _, p := range nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools {
			if strings.EqualFold(*p.ID, sshBackendPoolID) {
				updateSSHPool = false
			}
			if strings.EqualFold(*p.ID, ilbBackendPoolID) {
				updateILBPool = false
			}
		}
	}

	if updateSSHPool {
		m.log.Printf("Adding NIC %s to Internal Load Balancer SSH Backend Address Pool %s", nicName, sshBackendPoolID)
		nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = append(nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
			ID: &sshBackendPoolID,
		})
	}

	if updateILBPool {
		m.log.Printf("Adding NIC %s to Internal Load Balancer API Address Pool %s", nicName, ilbBackendPoolID)
		nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = append(nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
			ID: &ilbBackendPoolID,
		})
	}

	if updateSSHPool || updateILBPool {
		m.log.Printf("updating Network Interface %s", nicName)
		return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, nic, nil)
	}
	return nil
}

func (m *manager) updateV1ELBAddressPool(ctx context.Context, nic *armnetwork.Interface, nicName string, resourceGroup string, infraID string) error {
	if len(nic.Properties.IPConfigurations) == 0 {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", nicName)
	}

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroup, infraID, nil)
	if err != nil {
		return err
	}
	elbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, infraID)
	currentPool := nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools
	newPool := make([]*armnetwork.BackendAddressPool, 0, len(currentPool))
	for _, pool := range currentPool {
		if strings.EqualFold(*pool.ID, elbBackendPoolID) {
			m.log.Printf("Removing NIC %s from Public Load Balancer API Address Pool %s", nicName, elbBackendPoolID)
		} else {
			newPool = append(newPool, pool)
		}
	}

	if len(newPool) == len(currentPool) {
		return nil
	}

	nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = newPool
	m.log.Printf("Updating Network Interface %s", nicName)
	return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic, nil)
}

func (m *manager) updateELBAddressPool(ctx context.Context, nic armnetwork.Interface, nicName string, lb *armnetwork.LoadBalancer, resourceGroup string, infraID string) error {
	if len(nic.Properties.IPConfigurations) == 0 {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", nicName)
	}

	elbBackendPool := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		elbBackendPool = infraID + "-public-lb-control-plane-v4"
	}

	elbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, elbBackendPool)

	updateELBPool := true
	for _, p := range nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools {
		if strings.EqualFold(*p.ID, elbBackendPoolID) {
			updateELBPool = false
		}
	}

	if updateELBPool {
		m.log.Printf("Adding NIC %s to Public Load Balancer API Address Pool %s", nicName, elbBackendPoolID)
		nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools = append(nic.Properties.IPConfigurations[0].Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
			ID: &elbBackendPoolID,
		})
		m.log.Printf("updating Network Interface %s", nicName)
		return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, nic, nil)
	}

	return nil
}

func (m *manager) updateLB(ctx context.Context, lb *armnetwork.LoadBalancer, lbName string) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range lb.Properties.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		m.log.Printf("Adding SSH Backend Address Pool %s to Internal Load Balancer %s", name, lbName)
		lb.Properties.BackendAddressPools = append(lb.Properties.BackendAddressPools, &armnetwork.BackendAddressPool{
			Name: &name,
		})
	}

loadBalancingRules:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, r := range lb.Properties.LoadBalancingRules {
			if strings.EqualFold(*r.Name, name) {
				continue loadBalancingRules
			}
		}

		changed = true
		m.log.Printf("Adding SSH Load Balancing Rule for %s to Internal Load Balancer %s", name, lbName)
		lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules, &armnetwork.LoadBalancingRule{
			Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork.SubResource{
					ID: lb.Properties.FrontendIPConfigurations[0].ID,
				},
				BackendAddressPool: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)),
				},
				Probe: &armnetwork.SubResource{
					ID: pointerutils.ToPtr(*lb.ID + "/probes/ssh"),
				},
				Protocol:             pointerutils.ToPtr(armnetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(2200) + int32(i)),
				BackendPort:          pointerutils.ToPtr(int32(22)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
			Name: &name,
		})
	}

	for _, p := range lb.Properties.Probes {
		if strings.EqualFold(*p.Name, "ssh") {
			return changed
		}
	}

	changed = true
	m.log.Printf("Adding ssh Health Probe to Internal Load Balancer %s", lbName)
	lb.Properties.Probes = append(lb.Properties.Probes, &armnetwork.Probe{
		Properties: &armnetwork.ProbePropertiesFormat{
			Protocol:          pointerutils.ToPtr(armnetwork.ProbeProtocolTCP),
			Port:              pointerutils.ToPtr(int32(22)),
			IntervalInSeconds: pointerutils.ToPtr(int32(5)),
			NumberOfProbes:    pointerutils.ToPtr(int32(2)),
		},
		Name: pointerutils.ToPtr("ssh"),
	})

	return changed
}
