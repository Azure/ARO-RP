package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var (
	masterNICRegex               = regexp.MustCompile(`.*(master).*([0-2])-nic`)
	sshBackendPoolRegex          = regexp.MustCompile(`ssh-([0-2])`)
	interfacesCreateOrUpdateOpts = &armnetwork.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}
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
		m.log.Errorf("Failed checking and Updating Load Balancer with error: %s", err)
		return err
	}

	err = m.checkAndUpdateNICsInResourceGroup(ctx, resourceGroup, infraID, &lb)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) checkAndUpdateNICsInResourceGroup(ctx context.Context, resourceGroup string, infraID string, lb *mgmtnetwork.LoadBalancer) (err error) {
	masterSubnetID := m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID
	interfaces, err := m.armInterfaces.List(ctx, resourceGroup, &armnetwork.InterfacesClientListOptions{})
	if err != nil {
		m.log.Errorf("Error getting network interfaces for resource group %s: %v", resourceGroup, err)
		return err
	} else if len(interfaces) == 0 {
		return fmt.Errorf("interfaces list call for resource group %s returned an empty result", resourceGroup)
	}

NICs:
	for _, nic := range interfaces {
		m.log.Infof("Checking NIC %s", *nic.Name)
		// Filter out any NICs not associated with a control plane machine, ie workers / NIC for the private link service
		// Not great filtering based on name, but quickest way to skip processing NICs unnecessarily, tags would be better
		if !masterNICRegex.MatchString(*nic.Name) {
			m.log.Infof("Skipping NIC %s, not associated with a control plane machine.", *nic.Name)
			continue NICs
		}
		//Check for orphaned NICs
		if nic.Properties.VirtualMachine == nil {
			err := m.deleteOrphanedNIC(ctx, nic, resourceGroup, masterSubnetID)
			if err != nil {
				return err
			}
			continue NICs
		}
		ilbBackendPoolsUpdated := false
		elbBackendPoolsUpdated := false
		// Check and update NIC IPConfigurations. Do we ever expect multiple IP configs on an interface?
		for _, ipc := range nic.Properties.IPConfigurations {
			// TODO refactor this a bit, one if?
			if ipc.Properties.Subnet != nil {
				// Skip any NICs that are not in the master subnet
				if *ipc.Properties.Subnet.ID != masterSubnetID {
					m.log.Infof("Skipping NIC %s, NIC not in master subnet.", *nic.Name)
					continue NICs
				}
			}

			ilbBackendPoolsUpdated = m.updateILBBackendPools(*ipc, infraID, *nic.Name, *lb.ID)

			if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
				m.log.Infof("Updating UDR Cluster Network Interface %s", *nic.Name)
				err := m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
				if err != nil {
					return err
				}
				continue NICs
			}

			elbName := infraID
			if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
				elbName = infraID + "-public-lb"
			}

			elb, err := m.loadBalancers.Get(ctx, resourceGroup, elbName, "")
			if err != nil {
				return err
			}

			elbBackendPoolsUpdated = m.updateELBBackendPools(*ipc, infraID, *nic.Name, *elb.ID)
		}

		if ilbBackendPoolsUpdated || elbBackendPoolsUpdated {
			m.log.Infof("Updating Network Interface %s", *nic.Name)
			err = m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *manager) updateILBBackendPools(ipc armnetwork.InterfaceIPConfiguration, infraID string, nicName string, lbID string) bool {
	updated := false
	ilbBackendPoolID := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPoolID = infraID + "-internal-controlplane-v4"
	}
	ilbBackendPoolID = fmt.Sprintf("%s/backendAddressPools/%s", lbID, ilbBackendPoolID)
	ilbBackendPool := &armnetwork.BackendAddressPool{ID: &ilbBackendPoolID}
	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork.BackendAddressPool) bool {
		return *backendPool.ID == *ilbBackendPool.ID
	}) {
		m.log.Infof("Adding NIC %s to Internal Load Balancer API Address Pool %s", nicName, ilbBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, ilbBackendPool)
		updated = true
	}
	sshBackendPoolID := fmt.Sprintf("%s/backendAddressPools/ssh-%s", lbID, masterNICRegex.FindStringSubmatch(nicName)[2])
	sshBackendPool := &armnetwork.BackendAddressPool{ID: &sshBackendPoolID}
	// Check for NICs that are in the wrong SSH backend pool and remove them.
	// This covers the case for the bad NIC backend pool placements for CPMS updates to a private cluster
	ipc.Properties.LoadBalancerBackendAddressPools = slices.DeleteFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork.BackendAddressPool) bool {
		remove := *backendPool.ID != *sshBackendPool.ID && sshBackendPoolRegex.MatchString(*backendPool.ID)
		if remove {
			m.log.Infof("Removing NIC %s from Internal Load Balancer API Address Pool %s", nicName, *backendPool.ID)
			updated = true
		}
		return remove
	})

	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork.BackendAddressPool) bool {
		return *backendPool.ID == *sshBackendPool.ID
	}) {
		m.log.Infof("Adding NIC %s to Internal Load Balancer API Address Pool %s", nicName, sshBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, sshBackendPool)
		updated = true
	}

	return updated
}

func (m *manager) updateELBBackendPools(ipc armnetwork.InterfaceIPConfiguration, infraID string, nicName string, lbID string) bool {
	updated := false
	elbBackendPoolID := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		elbBackendPoolID = infraID + "-public-lb-control-plane-v4"
	}
	elbBackendPoolID = fmt.Sprintf("%s/backendAddressPools/%s", lbID, elbBackendPoolID)
	elbBackendPool := &armnetwork.BackendAddressPool{ID: &elbBackendPoolID}
	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork.BackendAddressPool) bool {
		return *backendPool.ID == *elbBackendPool.ID
	}) {
		m.log.Infof("Adding NIC %s to Public Load Balancer API Address Pool %s", nicName, elbBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, elbBackendPool)
		updated = true
	}

	return updated
}

func (m *manager) checkAndUpdateLB(ctx context.Context, resourceGroup string, lbName string) (lb mgmtnetwork.LoadBalancer, err error) {
	lb, err = m.loadBalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return lb, err
	}

	if m.updateLB(&lb, lbName) {
		m.log.Infof("Updating Load Balancer %s", lbName)
		err = m.loadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
		if err != nil {
			return lb, err
		}
	}
	return lb, nil
}

func (m *manager) updateLB(lb *mgmtnetwork.LoadBalancer, lbName string) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range *lb.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		m.log.Infof("Adding SSH Backend Address Pool %s to Internal Load Balancer %s", name, lbName)
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
		m.log.Infof("Adding SSH Load Balancing Rule for %s to Internal Load Balancer %s", name, lbName)
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
	m.log.Infof("Adding ssh Health Probe to Internal Load Balancer %s", lbName)
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

func (m *manager) deleteOrphanedNIC(ctx context.Context, nic *armnetwork.Interface, resourceGroup string, masterSubnetID string) error {
	// Delete any IPConfigurations and update the NIC
	nic.Properties.IPConfigurations = []*armnetwork.InterfaceIPConfiguration{{
		Name: to.StringPtr(*nic.Name),
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &armnetwork.Subnet{ID: to.StringPtr(masterSubnetID)}}}}
	err := m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
	if err != nil {
		m.log.Errorf("Removing IPConfigurations from NIC %s has failed with err %s", *nic.Name, err)
		return err
	}
	// Delete orphaned NIC (no VM associated and at this point we know it's a master NIC that's been removed from all backend pools)
	m.log.Infof("Deleting orphaned control plane machine NIC %s, not associated with any VM.", *nic.Name)
	err = m.armInterfaces.DeleteAndWait(ctx, resourceGroup, *nic.Name, nil)
	if err != nil {
		m.log.Errorf("Failed to delete orphaned NIC %s", *nic.Name)
		return err
	}

	return nil
}
