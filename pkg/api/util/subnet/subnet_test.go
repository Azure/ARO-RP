package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNetworkSecurityGroupID(t *testing.T) {
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
			},
			MasterProfile: api.MasterProfile{
				SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
				},
			},
		},
	}

	for _, tt := range []struct {
		name        string
		infraID     string
		archVersion api.ArchitectureVersion
		subnetID    string
		wantNSGID   string
		wantErr     string
	}{
		{
			name:      "master arch v1",
			subnetID:  "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			wantNSGID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg",
		},
		{
			name:      "worker arch v1",
			infraID:   "test-1234",
			subnetID:  "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-node-nsg",
		},
		{
			name:        "master arch v2",
			archVersion: api.ArchitectureVersionV2,
			subnetID:    "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			wantNSGID:   "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/aro-nsg",
		},
		{
			name:        "worker arch v2",
			infraID:     "test-1234",
			archVersion: api.ArchitectureVersionV2,
			subnetID:    "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID:   "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-nsg",
		},
		{
			name:        "unknown architecture version",
			archVersion: api.ArchitectureVersion(42),
			wantErr:     `unknown architecture version 42`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc.Properties.InfraID = tt.infraID
			oc.Properties.ArchitectureVersion = tt.archVersion

			nsgID, err := NetworkSecurityGroupID(oc, tt.subnetID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if nsgID != tt.wantNSGID {
				t.Error(nsgID)
			}
		})
	}
}
