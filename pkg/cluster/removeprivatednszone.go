package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func (m *manager) removePrivateDNSZone(ctx context.Context) error {
	resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
	config := utilnet.PrivateZoneRemovalConfig{
		Log:                m.log,
		PrivateZonesClient: m.privateZones,
		Configcli:          m.configcli,
		Mcocli:             m.mcocli,
		Kubernetescli:      m.kubernetescli,
		VNetLinksClient:    m.virtualNetworkLinks,
		ResourceGroupID:    resourceGroupID,
	}

	return utilnet.RemovePrivateDNSZone(ctx, config)
}
