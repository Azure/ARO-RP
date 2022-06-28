package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
)

// enableServiceEndpoints should enable service endpoints on
// subnets for storage account access
func (m *manager) enableServiceEndpoints(ctx context.Context) error {
	subnetIds, err := m.getSubnetIds()
	if err != nil {
		return err
	}

	subnets, err := m.getAllSubnets(ctx, subnetIds)
	if err != nil {
		return err
	}

	updatedSubnets := m.subnetsUpdater.AddEndpointsToSubnets(api.SubnetsEndpoints, subnets)

	for _, subnet := range updatedSubnets {
		err := m.subnet.CreateOrUpdate(ctx, *subnet.ID, subnet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) getSubnetIds() ([]string, error) {
	subnets := []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
	}

	for _, wp := range m.doc.OpenShiftCluster.Properties.WorkerProfiles {
		if len(wp.SubnetID) == 0 {
			return nil, fmt.Errorf("WorkerProfile '%s' has no SubnetID; check that the corresponding MachineSet is valid", wp.Name)
		}
		subnets = append(subnets, wp.SubnetID)
	}
	return subnets, nil
}

func (m *manager) getAllSubnets(ctx context.Context, subnetIds []string) ([]*mgmtnetwork.Subnet, error) {
	if len(subnetIds) == 0 {
		return nil, nil
	}

	subnets := make([]*mgmtnetwork.Subnet, len(subnetIds))

	for i, subnetId := range subnetIds {
		subnet, err := m.subnet.Get(ctx, subnetId)
		if err != nil {
			return nil, err
		}

		subnets[i] = subnet
	}
	return subnets, nil
}
