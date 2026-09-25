package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	apitesterror "github.com/Azure/ARO-RP/pkg/api/test/error"
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
		name         string
		infraID      string
		archVersion  api.ArchitectureVersion
		subnetID     string
		wpStatus     bool
		wpEmptyFirst bool
		wpAllEmpty   bool
		wantNSGID    string
		wantErr      string
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
		{
			name:        "worker arch v2 to use enriched worker Profile",
			infraID:     "test-1234",
			archVersion: api.ArchitectureVersionV2,
			wpStatus:    true,
			subnetID:    "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/Enrichedworker",
			wantNSGID:   "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-nsg",
		},
		{
			name:         "worker arch v2 skip empty SubnetID when empty comes first",
			infraID:      "test-1234",
			archVersion:  api.ArchitectureVersionV2,
			wpEmptyFirst: true,
			subnetID:     "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID:    "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-nsg",
		},
		{
			name:         "worker arch v1 skip empty SubnetID when empty comes first",
			infraID:      "test-1234",
			wpEmptyFirst: true,
			subnetID:     "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
			wantNSGID:    "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-node-nsg",
		},
		{
			name:       "worker arch v1 all empty SubnetIDs returns master NSG",
			infraID:    "test-1234",
			wpAllEmpty: true,
			subnetID:   "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			wantNSGID:  "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/networkSecurityGroups/test-1234-controlplane-nsg",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc.Properties.InfraID = tt.infraID
			oc.Properties.ArchitectureVersion = tt.archVersion
			oc.Properties.WorkerProfilesStatus = nil

			if tt.wpStatus {
				oc.Properties.WorkerProfilesStatus = []api.WorkerProfile{
					{
						SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/Enrichedworker",
					},
				}
			}
			if tt.wpEmptyFirst {
				oc.Properties.WorkerProfilesStatus = []api.WorkerProfile{
					{Name: "emptyWorker", SubnetID: ""},
					{SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker"},
				}
			}
			if tt.wpAllEmpty {
				oc.Properties.WorkerProfilesStatus = []api.WorkerProfile{
					{Name: "emptyWorker1", SubnetID: ""},
					{Name: "emptyWorker2", SubnetID: ""},
				}
			}

			nsgID, err := NetworkSecurityGroupID(oc, tt.subnetID)
			apitesterror.AssertErrorMessage(t, err, tt.wantErr)

			if nsgID != tt.wantNSGID {
				t.Error(nsgID)
			}
		})
	}
}
