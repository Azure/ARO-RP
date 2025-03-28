package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var nicNameRegex = regexp.MustCompile(`.*([0-2])\-nic`)

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

	// err = m.checkandUpdateNIC(ctx, resourceGroup, infraID, lb)
	err = m.checkAndUpdateNICsInResourceGroup(ctx, resourceGroup, infraID, &lb)
	if err != nil {
		m.log.Warnf("Failed checking and Updating Network Interface with error: %s", err)
		return err
	}

	return nil
}

func (m *manager) checkAndUpdateNICsInResourceGroup(ctx context.Context, resourceGroup string, infraID string, lb *mgmtnetwork.LoadBalancer) (err error) {

	opts := &armnetwork.InterfacesClientListAllOptions{}
	pager := m.armInterfaces.NewListAllPager(opts)

	masterSubnetID := m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID
	// r, err := arm.ParseResourceID(masterSubnetID)
	// if err != nil {
	// 	return err
	// }
	// masterSubnet, err := m.armSubnets.Get(ctx, resourceGroup, r.Parent.Name, r.Name, nil)
	masterSubnetFound := false

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			m.log.Warnf("failed to advance page: %v", err)
			return err
		}
		for _, nic := range nextResult.Value {

			// Look for master subnet
			for _, ipc := range nic.Properties.IPConfigurations {
				if *ipc.Properties.Subnet.ID == masterSubnetID {
					masterSubnetFound = true
				}
			}
			// If NIC is not associated with the master subnet continue to next NIC
			if !masterSubnetFound {
				continue
			}

			if nic.Properties != nil && nic.Properties.VirtualMachine == nil {
				err = m.removeBackendPoolsFromNICv2(ctx, resourceGroup, nic, masterSubnetID)
				if err != nil {
					m.log.Warnf("Removing BackendPools from NIC %s has failed with err %s", *nic.Name, err)
					return err
				}
			}

			err = m.updateILBAddressPoolv2(ctx, nic, lb, resourceGroup, infraID, masterSubnetID)
			if err != nil {
				return err
			}

			if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
				continue
			}

			elbName := infraID
			if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
				elbName = infraID + "-public-lb"
			}

			elb, err := m.loadBalancers.Get(ctx, resourceGroup, elbName, "")
			if err != nil {
				return err
			}

			err = m.updateELBAddressPoolv2(ctx, nic, &elb, resourceGroup, infraID, masterSubnetID)
			if err != nil {
				return err
			}
		}

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

func (m *manager) removeBackendPoolsFromNICv2(ctx context.Context, resourceGroup string, nic *armnetwork.Interface, masterSubnetID string) error {
	opts := &armnetwork.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}

	if nic.Properties.IPConfigurations == nil {
		return fmt.Errorf("unable to remove Backend Address Pools from NIC as there are no IP configurations for %s in resource group %s", *nic.Name, resourceGroup)
	}

	for _, ipc := range nic.Properties.IPConfigurations {
		if *ipc.Properties.Subnet.ID == masterSubnetID && ipc.Properties.LoadBalancerBackendAddressPools != nil {
			m.log.Printf("Removing Load balancer Backend Address Pools from NIC %s with no VMs attached", *nic.Name)
			ipc.Properties.LoadBalancerBackendAddressPools = []*armnetwork.BackendAddressPool{}
			return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, opts)
		}
	}

	return nil
}

func (m *manager) updateILBAddressPoolv2(ctx context.Context, nic *armnetwork.Interface, lb *mgmtnetwork.LoadBalancer, resourceGroup string, infraID string, masterSubnetID string) error {
	opts := &armnetwork.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}

	if nic.Properties.IPConfigurations == nil {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", *nic.Name)
	}

	ilbBackendPool := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPool = infraID + "-internal-controlplane-v4"
	}

	for _, ipc := range nic.Properties.IPConfigurations {
		if *ipc.Properties.Subnet.ID == masterSubnetID && ipc.Properties.LoadBalancerBackendAddressPools == nil {
			ipc.Properties.LoadBalancerBackendAddressPools = []*armnetwork.BackendAddressPool{}
		} else if *ipc.Properties.Subnet.ID == masterSubnetID {
			ilbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, ilbBackendPool)

			m.log.Printf("Adding NIC %s to Internal Load Balancer API Address Pool %s", *nic.Name, ilbBackendPoolID)
			ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
				ID: &ilbBackendPoolID,
			})

			// Find the index of the NIC we are working with
			// nicNameRegex `.*([0-2])\-nic`
			nicIndex := nicNameRegex.FindString(*nic.Name)

			sshBackendPoolID := fmt.Sprintf("%s/backendAddressPools/ssh-%s", *lb.ID, nicIndex)
			m.log.Printf("Adding NIC %s to Internal Load Balancer API Address Pool %s", *nic.Name, sshBackendPoolID)
			ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
				ID: &sshBackendPoolID,
			})
		}
	}

	m.log.Printf("updating Network Interface %s", *nic.Name)
	return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, opts)

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

func (m *manager) updateELBAddressPoolv2(ctx context.Context, nic *armnetwork.Interface, lb *mgmtnetwork.LoadBalancer, resourceGroup string, infraID string, masterSubnetID string) error {
	opts := &armnetwork.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}

	if nic.Properties.IPConfigurations == nil {
		return fmt.Errorf("unable to update NIC as there are no IP configurations for %s", *nic.Name)
	}

	elbBackendPool := infraID
	if m.doc.OpenShiftCluster.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		elbBackendPool = infraID + "-public-lb-control-plane-v4"
	}

	elbBackendPoolID := fmt.Sprintf("%s/backendAddressPools/%s", *lb.ID, elbBackendPool)

	for _, ipc := range nic.Properties.IPConfigurations {
		if *ipc.Properties.Subnet.ID == masterSubnetID && ipc.Properties.LoadBalancerBackendAddressPools == nil {
			ipc.Properties.LoadBalancerBackendAddressPools = []*armnetwork.BackendAddressPool{}
		} else if *ipc.Properties.Subnet.ID == masterSubnetID {

			m.log.Printf("Adding NIC %s to Public Load Balancer API Address Pool %s", *nic.Name, elbBackendPoolID)
			ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, &armnetwork.BackendAddressPool{
				ID: &elbBackendPoolID,
			})

		}
	}

	m.log.Printf("updating Network Interface %s", *nic.Name)
	return m.armInterfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, opts)
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
