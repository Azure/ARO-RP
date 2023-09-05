package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
)

// ensureServiceEndpoints should enable service endpoints on
// subnets for storage account access, but only if egress lockdown is
// not enabled.
func (m *manager) ensureServiceEndpoints(ctx context.Context) error {
	subnetIds, err := m.getSubnetIds()
	if err != nil {
		return err
	}

	return m.subnet.CreateOrUpdateFromIds(ctx, subnetIds, m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled)
}

func (m *manager) getSubnetIds() ([]string, error) {
	subnets := []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
	}
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)

	for _, wp := range workerProfiles {
		if len(wp.SubnetID) == 0 {
			return nil, fmt.Errorf("WorkerProfile '%s' has no SubnetID; check that the corresponding MachineSet is valid", wp.Name)
		}
		subnets = append(subnets, wp.SubnetID)
	}
	return subnets, nil
}
