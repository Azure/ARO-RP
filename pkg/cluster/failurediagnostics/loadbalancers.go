package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogVMSerialConsole fetches the serial console from VMs and logs them with
// the associated VM name.
func (m *manager) LogLoadBalancers(ctx context.Context) ([]string, error) {
	return m.logLoadBalancers(ctx, 50)
}

func (m *manager) logLoadBalancers(ctx context.Context, log_limit_kb int) ([]string, error) {
	items := make([]string, 0)

	if m.loadBalancers == nil {
		items = append(items, "networkclient missing")
		return items, nil
	}

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
		return items, fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	items = append(items, fmt.Sprintf("Load Balancer: %s", lbName))
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	lb, err := m.loadBalancers.Get(ctx, resourceGroupName, lbName, "")
	if lb.Probes != nil {
		for _, probe := range *lb.Probes {
			probeName := "unnamed"
			if probe.Name != nil {
				probeName = *probe.Name
			}
			items = append(items, fmt.Sprintf("Probe %s: %s - status: %s (%d)", probeName, probe.ProvisioningState, probe.Status, probe.StatusCode))
		}
	}
	return items, err
}
