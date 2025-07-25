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

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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

	err = m.checkAndUpdateNICsInResourceGroup(ctx, resourceGroup, infraID, lb)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) checkAndUpdateNICsInResourceGroup(ctx context.Context, resourceGroup string, infraID string, lb *armnetwork.LoadBalancer) (err error) {
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

		// Booleans to track if backend pools are updated
		ilbBackendPoolsUpdated, elbBackendPoolsUpdated := false, false

		// Check and update NIC IPConfigurations. Do we ever expect multiple IP configs on an interface?
		for _, ipc := range nic.Properties.IPConfigurations {
			// Skip any NICs that are not in the master subnet
			if ipc.Properties.Subnet != nil && ipc.Properties.Subnet.ID != nil && *ipc.Properties.Subnet.ID != masterSubnetID {
				m.log.Infof("Skipping NIC %s, NIC not in master subnet.", *nic.Name)
				continue NICs
			}

			ilbBackendPoolsUpdated = m.updateILBBackendPools(*ipc, infraID, *nic.Name, *lb.ID)

			// Check if this is a fully private cluster and the internal load balancer backend pools have been updated
			// If both the API and ingress visibility are private, there is no external LB so we continue
			if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPrivate &&
				m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPrivate &&
				ilbBackendPoolsUpdated {
				m.log.Infof("Updating Private Cluster Network Interface %s", *nic.Name)
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

			elb, err := m.armLoadBalancers.Get(ctx, resourceGroup, elbName, nil)
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
		m.log.Infof("Adding NIC %s to Internal Load Balancer SSH Address Pool %s", nicName, sshBackendPoolID)
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

func (m *manager) checkAndUpdateLB(ctx context.Context, resourceGroup string, lbName string) (lb *armnetwork.LoadBalancer, err error) {
	_lb, err := m.armLoadBalancers.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return nil, err
	}

	lb = &_lb.LoadBalancer

	if m.updateLB(lb, lbName) {
		m.log.Infof("Updating Load Balancer %s", lbName)
		err = m.armLoadBalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, *lb, nil)
		if err != nil {
			return nil, err
		}
	}
	return lb, nil
}

func (m *manager) updateLB(lb *armnetwork.LoadBalancer, lbName string) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range lb.Properties.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		m.log.Infof("Adding SSH Backend Address Pool %s to Internal Load Balancer %s", name, lbName)
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
		m.log.Infof("Adding SSH Load Balancing Rule for %s to Internal Load Balancer %s", name, lbName)
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
	m.log.Infof("Adding ssh Health Probe to Internal Load Balancer %s", lbName)
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

func (m *manager) deleteOrphanedNIC(ctx context.Context, nic *armnetwork.Interface, resourceGroup string, masterSubnetID string) error {
	// Delete any IPConfigurations and update the NIC
	nic.Properties.IPConfigurations = []*armnetwork.InterfaceIPConfiguration{{
		Name: pointerutils.ToPtr(*nic.Name),
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &armnetwork.Subnet{ID: pointerutils.ToPtr(masterSubnetID)}}}}
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
