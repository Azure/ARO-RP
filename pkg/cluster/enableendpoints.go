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

	subnets, err := m.subnet.GetAll(ctx, subnetIds)
	if err != nil {
		return err
	}

	subnetsToBeUpdated := m.endpointsAdder.AddEndpointsToSubnets(api.SubnetsEndpoints, subnets)

	return m.updateSubnets(subnetsToBeUpdated, ctx)
}

func (m *manager) updateSubnets(subnets []*mgmtnetwork.Subnet, ctx context.Context) error {
	for _, subnet := range subnets {
		if err := m.subnet.CreateOrUpdate(ctx, *subnet.ID, subnet); err != nil {
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
