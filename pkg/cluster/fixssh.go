package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"

	armnetwork_sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var (
	masterNICRegex               = regexp.MustCompile(`.*(master).*([0-2])-nic`)
	sshBackendPoolRegex          = regexp.MustCompile(`ssh-([0-2])`)
	interfacesCreateOrUpdateOpts = &armnetwork_sdk.InterfacesClientBeginCreateOrUpdateOptions{ResumeToken: ""}
)

func (m *manager) fixSSH(ctx context.Context) error {
	return FixSSH(ctx, m.log, m.armLoadBalancers, m.armInterfaces, m.doc.OpenShiftCluster)
}

func FixSSH(ctx context.Context, log *logrus.Entry, lbClient armnetwork.LoadBalancersClient, interfacesClient armnetwork.InterfacesClient, oc *api.OpenShiftCluster) error {
	infraID := oc.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	var lbName string
	switch oc.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", oc.Properties.ArchitectureVersion)
	}

	resourceGroup := stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')

	lb, err := checkAndUpdateLB(ctx, log, lbClient, resourceGroup, lbName)
	if err != nil {
		log.Errorf("Failed checking and Updating Load Balancer with error: %s", err)
		return err
	}

	err = checkAndUpdateNICsInResourceGroup(ctx, log, lbClient, interfacesClient, oc, infraID, lb)
	if err != nil {
		return err
	}

	return nil
}

func checkAndUpdateNICsInResourceGroup(
	ctx context.Context, log *logrus.Entry, lbClient armnetwork.LoadBalancersClient, interfacesClient armnetwork.InterfacesClient, oc *api.OpenShiftCluster, infraID string, lb *armnetwork_sdk.LoadBalancer,
) (err error) {
	masterSubnetID := oc.Properties.MasterProfile.SubnetID
	resourceGroup := stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')

	interfaces, err := interfacesClient.List(ctx, resourceGroup, &armnetwork_sdk.InterfacesClientListOptions{})
	if err != nil {
		log.Errorf("Error getting network interfaces for resource group %s: %v", resourceGroup, err)
		return err
	} else if len(interfaces) == 0 {
		return fmt.Errorf("interfaces list call for resource group %s returned an empty result", resourceGroup)
	}

NICs:
	for _, nic := range interfaces {
		log.Infof("Checking NIC %s", *nic.Name)
		// Filter out any NICs not associated with a control plane machine, ie workers / NIC for the private link service
		// Not great filtering based on name, but quickest way to skip processing NICs unnecessarily, tags would be better
		if !masterNICRegex.MatchString(*nic.Name) {
			log.Infof("Skipping NIC %s, not associated with a control plane machine.", *nic.Name)
			continue NICs
		}
		//Check for orphaned NICs
		if nic.Properties.VirtualMachine == nil {
			err := deleteOrphanedNIC(ctx, log, interfacesClient, nic, resourceGroup, masterSubnetID)
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
				log.Infof("Skipping NIC %s, NIC not in master subnet.", *nic.Name)
				continue NICs
			}

			ilbBackendPoolsUpdated = updateILBBackendPools(log, oc, *ipc, infraID, *nic.Name, *lb.ID)

			// Check if UserDefinedRouting is enabled for this cluster
			// UDR clusters don't have an external load balancer so stop executing here and continue to next NIC
			if oc.Properties.NetworkProfile.OutboundType == api.OutboundTypeUserDefinedRouting {
				if ilbBackendPoolsUpdated {
					log.Infof("Updating UDR Cluster Network Interface %s", *nic.Name)
					err := interfacesClient.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
					if err != nil {
						return err
					}
				}
				continue NICs
			}

			elbName := infraID
			if oc.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
				elbName = infraID + "-public-lb"
			}

			elb, err := lbClient.Get(ctx, resourceGroup, elbName, nil)
			if err != nil {
				return err
			}

			elbBackendPoolsUpdated = updateELBBackendPools(log, oc, *ipc, infraID, *nic.Name, *elb.ID)
		}

		if ilbBackendPoolsUpdated || elbBackendPoolsUpdated {
			log.Infof("Updating Network Interface %s", *nic.Name)
			err = interfacesClient.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func updateILBBackendPools(log *logrus.Entry, oc *api.OpenShiftCluster, ipc armnetwork_sdk.InterfaceIPConfiguration, infraID string, nicName string, lbID string) bool {
	updated := false
	ilbBackendPoolID := infraID
	if oc.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		ilbBackendPoolID = infraID + "-internal-controlplane-v4"
	}

	ilbBackendPoolID = fmt.Sprintf("%s/backendAddressPools/%s", lbID, ilbBackendPoolID)
	ilbBackendPool := &armnetwork_sdk.BackendAddressPool{ID: &ilbBackendPoolID}
	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork_sdk.BackendAddressPool) bool {
		return *backendPool.ID == *ilbBackendPool.ID
	}) {
		log.Infof("Adding NIC %s to Internal Load Balancer API Address Pool %s", nicName, ilbBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, ilbBackendPool)
		updated = true
	}
	sshBackendPoolID := fmt.Sprintf("%s/backendAddressPools/ssh-%s", lbID, masterNICRegex.FindStringSubmatch(nicName)[2])
	sshBackendPool := &armnetwork_sdk.BackendAddressPool{ID: &sshBackendPoolID}
	// Check for NICs that are in the wrong SSH backend pool and remove them.
	// This covers the case for the bad NIC backend pool placements for CPMS updates to a private cluster
	ipc.Properties.LoadBalancerBackendAddressPools = slices.DeleteFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork_sdk.BackendAddressPool) bool {
		remove := *backendPool.ID != *sshBackendPool.ID && sshBackendPoolRegex.MatchString(*backendPool.ID)
		if remove {
			log.Infof("Removing NIC %s from Internal Load Balancer API Address Pool %s", nicName, *backendPool.ID)
			updated = true
		}
		return remove
	})

	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork_sdk.BackendAddressPool) bool {
		return *backendPool.ID == *sshBackendPool.ID
	}) {
		log.Infof("Adding NIC %s to Internal Load Balancer SSH Address Pool %s", nicName, sshBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, sshBackendPool)
		updated = true
	}

	return updated
}

func updateELBBackendPools(log *logrus.Entry, oc *api.OpenShiftCluster, ipc armnetwork_sdk.InterfaceIPConfiguration, infraID string, nicName string, lbID string) bool {
	updated := false
	elbBackendPoolID := infraID
	if oc.Properties.ArchitectureVersion == api.ArchitectureVersionV1 {
		elbBackendPoolID = infraID + "-public-lb-control-plane-v4"
	}
	elbBackendPoolID = fmt.Sprintf("%s/backendAddressPools/%s", lbID, elbBackendPoolID)
	elbBackendPool := &armnetwork_sdk.BackendAddressPool{ID: &elbBackendPoolID}
	if !slices.ContainsFunc(ipc.Properties.LoadBalancerBackendAddressPools, func(backendPool *armnetwork_sdk.BackendAddressPool) bool {
		return *backendPool.ID == *elbBackendPool.ID
	}) {
		log.Infof("Adding NIC %s to Public Load Balancer API Address Pool %s", nicName, elbBackendPoolID)
		ipc.Properties.LoadBalancerBackendAddressPools = append(ipc.Properties.LoadBalancerBackendAddressPools, elbBackendPool)
		updated = true
	}

	return updated
}

func checkAndUpdateLB(ctx context.Context, log *logrus.Entry, lbClient armnetwork.LoadBalancersClient, resourceGroup string, lbName string) (lb *armnetwork_sdk.LoadBalancer, err error) {
	_lb, err := lbClient.Get(ctx, resourceGroup, lbName, nil)
	if err != nil {
		return nil, err
	}

	lb = &_lb.LoadBalancer

	if updateLB(log, lb, lbName) {
		log.Infof("Updating Load Balancer %s", lbName)
		err = lbClient.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, *lb, nil)
		if err != nil {
			return nil, err
		}
	}
	return lb, nil
}

func updateLB(log *logrus.Entry, lb *armnetwork_sdk.LoadBalancer, lbName string) (changed bool) {
backendAddressPools:
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("ssh-%d", i)
		for _, p := range lb.Properties.BackendAddressPools {
			if strings.EqualFold(*p.Name, name) {
				continue backendAddressPools
			}
		}

		changed = true
		log.Infof("Adding SSH Backend Address Pool %s to Internal Load Balancer %s", name, lbName)
		lb.Properties.BackendAddressPools = append(lb.Properties.BackendAddressPools, &armnetwork_sdk.BackendAddressPool{
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
		log.Infof("Adding SSH Load Balancing Rule for %s to Internal Load Balancer %s", name, lbName)
		lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules, &armnetwork_sdk.LoadBalancingRule{
			Properties: &armnetwork_sdk.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &armnetwork_sdk.SubResource{
					ID: lb.Properties.FrontendIPConfigurations[0].ID,
				},
				BackendAddressPool: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)),
				},
				Probe: &armnetwork_sdk.SubResource{
					ID: pointerutils.ToPtr(*lb.ID + "/probes/ssh"),
				},
				Protocol:             pointerutils.ToPtr(armnetwork_sdk.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(armnetwork_sdk.LoadDistributionDefault),
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
	log.Infof("Adding ssh Health Probe to Internal Load Balancer %s", lbName)
	lb.Properties.Probes = append(lb.Properties.Probes, &armnetwork_sdk.Probe{
		Properties: &armnetwork_sdk.ProbePropertiesFormat{
			Protocol:          pointerutils.ToPtr(armnetwork_sdk.ProbeProtocolTCP),
			Port:              pointerutils.ToPtr(int32(22)),
			IntervalInSeconds: pointerutils.ToPtr(int32(5)),
			NumberOfProbes:    pointerutils.ToPtr(int32(2)),
		},
		Name: pointerutils.ToPtr("ssh"),
	})

	return changed
}

func deleteOrphanedNIC(ctx context.Context, log *logrus.Entry, interfacesClient armnetwork.InterfacesClient, nic *armnetwork_sdk.Interface, resourceGroup string, masterSubnetID string) error {
	// Delete any IPConfigurations and update the NIC
	nic.Properties.IPConfigurations = []*armnetwork_sdk.InterfaceIPConfiguration{{
		Name: pointerutils.ToPtr(*nic.Name),
		Properties: &armnetwork_sdk.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &armnetwork_sdk.Subnet{ID: pointerutils.ToPtr(masterSubnetID)}}}}
	err := interfacesClient.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, *nic, interfacesCreateOrUpdateOpts)
	if err != nil {
		log.Errorf("Removing IPConfigurations from NIC %s has failed with err %s", *nic.Name, err)
		return err
	}
	// Delete orphaned NIC (no VM associated and at this point we know it's a master NIC that's been removed from all backend pools)
	log.Infof("Deleting orphaned control plane machine NIC %s, not associated with any VM.", *nic.Name)
	err = interfacesClient.DeleteAndWait(ctx, resourceGroup, *nic.Name, nil)
	if err != nil {
		log.Errorf("Failed to delete orphaned NIC %s", *nic.Name)
		return err
	}

	return nil
}
