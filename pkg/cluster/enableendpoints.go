package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
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

	subnetsToBeUpdated := subnet.AddEndpointsToSubnets(api.SubnetsEndpoints, subnets)

	return m.subnet.CreateOrUpdateSubnets(ctx, subnetsToBeUpdated)
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
