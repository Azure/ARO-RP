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

func (m *manager) checkAndUpdateLB(ctx context.Context, resourceGroup string, lbName string) (lb mgmtnetwork.LoadBalancer, err error) {
	lb, err = m.loadBalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return lb, err
	}

	if m.updateLB(ctx, &lb, lbName) {
		m.log.Printf("updating Load Balancer %s", lbName)
		err = m.loadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
		if err != nil {
			return lb, err
		}
	}
	return lb, nil
}

func (m *manager) checkandUpdateNIC(ctx context.Context, resourceGroup string, infraID string, lb mgmtnetwork.LoadBalancer) (err error) {
	for i := 0; i < 3; i++ {
		// NIC names might be different if customer re-created master nodes
		// see https://bugzilla.redhat.com/show_bug.cgi?id=1882490 for more details
		// installer naming  - <foo>-master{0,1,2}-nic
		// machineAPI naming - <foo>-master-{0,1,2}-nic
		nicNameInstaller := fmt.Sprintf("%s-master%d-nic", infraID, i)
		nicNameMachineAPI := fmt.Sprintf("%s-master-%d-nic", infraID, i)

		var nic mgmtnetwork.Interface
		nicName := nicNameInstaller
		fallbackNIC := false

		nic, err = m.interfaces.Get(ctx, resourceGroup, nicName, "")
		if err != nil {
			m.log.Warnf("Fetching details for NIC %s has failed with err %s", nicName, err)
			fallbackNIC = true
		} else if nic.InterfacePropertiesFormat != nil && nic.InterfacePropertiesFormat.VirtualMachine == nil {
			err = m.removeBackendPoolsFromNIC(ctx, resourceGroup, nicName, &nic)
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
			nic, err = m.interfaces.Get(ctx, resourceGroup, nicName, "")
			if err != nil {
				m.log.Warnf("Fallback failed with err %s", err)
				return err
			}
		}

		err = m.updateILBAddressPool(ctx, &nic, nicName, &lb, i, resourceGroup, infraID)
		if err != nil {
			return err
		}

		elbName := infraID
		if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
			err = m.updateV1ELBAddressPool(ctx, &nic, nicName, resourceGroup, infraID)
			if err != nil {
				return err
			}
			elbName = infraID + "-public-lb"
		}

		elb, err := m.loadBalancers.Get(ctx, resourceGroup, elbName, "")
		if err != nil {
			return err
		}

		err = m.updateELBAddressPool(ctx, &nic, nicName, &elb, resourceGroup, infraID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) removeBackendPoolsFromNIC(ctx context.Context, resourceGroup, nicName string, nic *mgmtnetwork.Interface) error {
	if nic.InterfacePropertiesFormat.IPConfigurations == nil || len(*nic.InterfacePropertiesFormat.IPConfigurations) == 0 {
		return fmt.Errorf("unable to remove Backend Address Pools from NIC as there are no IP configurations for %s in resource group %s", nicName, resourceGroup)
	}
	ipc := (*nic.InterfacePropertiesFormat.IPConfigurations)[0]
	if ipc.LoadBalancerBackendAddressPools != nil {
		m.log.Printf("Removing Load balancer Backend Address Pools from NIC %s with no VMs attached", nicName)
		*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = []mgmtnetwork.BackendAddressPool{}
		return m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic)
	}
	return nil
}

func (m *manager) updateILBAddressPool(ctx context.Context, nic *mgmtnetwork.Interface, nicName string, lb *mgmtnetwork.LoadBalancer, i int, resourceGroup string, infraID string) error {
	if nic.InterfacePropertiesFormat.IPConfigurations == nil || len(*nic.InterfacePropertiesFormat.IPConfigurations) == 0 {
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

	ipc := (*nic.InterfacePropertiesFormat.IPConfigurations)[0]
	if ipc.LoadBalancerBackendAddressPools == nil {
		emptyBackendAddressPool := make([]mgmtnetwork.BackendAddressPool, 0)
		(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = &emptyBackendAddressPool
	} else {
		for _, p := range *(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools {
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
		*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = append(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools, mgmtnetwork.BackendAddressPool{
			ID: &sshBackendPoolID,
		})
	}

	if updateILBPool {
		m.log.Printf("Adding NIC %s to Internal Load Balancer API Address Pool %s", nicName, ilbBackendPoolID)
		*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = append(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools, mgmtnetwork.BackendAddressPool{
			ID: &ilbBackendPoolID,
		})
	}

	if updateSSHPool || updateILBPool {
		m.log.Printf("updating Network Interface %s", nicName)
		return m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic)
	}
	return nil
}

func (m *manager) updateV1ELBAddressPool(ctx context.Context, nic *mgmtnetwork.Interface, nicName string, resourceGroup string, infraID string) error {
	if nic.InterfacePropertiesFormat.IPConfigurations == nil || len(*nic.InterfacePropertiesFormat.IPConfigurations) == 0 {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", nicName)
	}

	lb, err := m.loadBalancers.Get(ctx, resourceGroup, infraID, "")
	if err != nil {
		return err
	}
	elbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, infraID)
	currentPool := *(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools
	newPool := make([]mgmtnetwork.BackendAddressPool, 0, len(currentPool))
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

	(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = &newPool
	m.log.Printf("Updating Network Interface %s", nicName)
	return m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic)
}

func (m *manager) updateELBAddressPool(ctx context.Context, nic *mgmtnetwork.Interface, nicName string, lb *mgmtnetwork.LoadBalancer, resourceGroup string, infraID string) error {
	if nic.InterfacePropertiesFormat.IPConfigurations == nil || len(*nic.InterfacePropertiesFormat.IPConfigurations) == 0 {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", nicName)
	}

	elbBackendPool := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		elbBackendPool = infraID + "-public-lb-control-plane-v4"
	}

	elbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, elbBackendPool)

	updateELBPool := true
	for _, p := range *(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools {
		if strings.EqualFold(*p.ID, elbBackendPoolID) {
			updateELBPool = false
		}
	}

	if updateELBPool {
		m.log.Printf("Adding NIC %s to Public Load Balancer API Address Pool %s", nicName, elbBackendPoolID)
		*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = append(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools, mgmtnetwork.BackendAddressPool{
			ID: &elbBackendPoolID,
		})
		m.log.Printf("updating Network Interface %s", nicName)
		return m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, nicName, *nic)
	}

	return nil
}

func (m *manager) updateLB(ctx context.Context, lb *mgmtnetwork.LoadBalancer, lbName string) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range *lb.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		m.log.Printf("Adding SSH Backend Address Pool %s to Internal Load Balancer %s", name, lbName)
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
		m.log.Printf("Adding SSH Load Balancing Rule for %s to Internal Load Balancer %s", name, lbName)
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
	m.log.Printf("Adding ssh Health Probe to Internal Load Balancer %s", lbName)
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
