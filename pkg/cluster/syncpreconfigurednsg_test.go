package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestSyncPreconfiguredNSG(t *testing.T) {
	for _, tt := range []struct {
		name           string
		workerGetError bool
		conflictingNSG bool
		invalidMaster  bool
		emptyWorker    bool
		missingNSG     bool
		expectedLog    string
		expectedErr    string
	}{
		{
			name:        "consistent NSGs leave stored value unchanged",
			expectedLog: "PreconfiguredNSG is consistent",
		},
		{
			name:           "subnet GET failure warns and leaves stored value unchanged",
			workerGetError: true,
			expectedLog:    "unable to inspect subnet " + subnetIdWorker + "; leaving stored value \"Enabled\" unchanged: subnet unavailable",
		},
		{
			name:          "malformed subnet ID warns and leaves stored value unchanged",
			invalidMaster: true,
			expectedLog:   "unable to inspect subnet invalid; leaving stored value \"Enabled\" unchanged:",
		},
		{
			name:        "subnet without an NSG warns and leaves stored value unchanged",
			missingNSG:  true,
			expectedLog: "subnet " + subnetIdWorker + " has no attached NSG; cannot infer PreconfiguredNSG; leaving stored value \"Enabled\" unchanged",
		},
		{
			name:        "empty worker subnet returns validation error",
			emptyWorker: true,
			expectedErr: "WorkerProfile '' has no SubnetID; check that the corresponding MachineSet is valid",
		},
		{
			name:           "conflicting NSGs leave stored value unchanged",
			conflictingNSG: true,
			expectedLog:    "subnets imply conflicting PreconfiguredNSG values",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			subnetsClient := mock_armnetwork.NewMockSubnetsClient(controller)
			hook, log := testlog.LogForTesting(t)
			m := &manager{
				log:        log,
				armSubnets: subnetsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "/subscriptions/" + subscriptionId + "/resourceGroups/cluster-rg",
							},
							MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
							WorkerProfiles: []api.WorkerProfile{
								{SubnetID: subnetIdWorker},
							},
							NetworkProfile: api.NetworkProfile{PreconfiguredNSG: api.PreconfiguredNSGEnabled},
						},
					},
				},
			}
			if tt.invalidMaster {
				m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID = "invalid"
			}
			if tt.emptyWorker {
				m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID = ""
			}

			for _, subnetName := range []string{subnetNameMaster, subnetNameWorker} {
				if tt.emptyWorker || tt.invalidMaster {
					continue
				}
				if subnetName == subnetNameWorker && tt.workerGetError {
					subnetsClient.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetName, nil).
						Return(armnetwork.SubnetsClientGetResponse{}, errors.New("subnet unavailable"))
					continue
				}
				nsgResourceGroup := vnetResourceGroup
				if tt.workerGetError || tt.missingNSG || (subnetName == subnetNameWorker && tt.conflictingNSG) {
					nsgResourceGroup = "cluster-rg"
				}
				subnet := armnetwork.Subnet{
					Properties: &armnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &armnetwork.SecurityGroup{
							ID: new("/subscriptions/" + subscriptionId + "/resourceGroups/" + nsgResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/nsg"),
						},
					},
				}
				if subnetName == subnetNameWorker && tt.missingNSG {
					subnet.Properties.NetworkSecurityGroup = nil
				}
				subnetsClient.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetName, nil).
					Return(armnetwork.SubnetsClientGetResponse{Subnet: subnet}, nil)
			}

			err := m.syncPreconfiguredNSG(context.Background())
			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Contains(t, hook.LastEntry().Message, tt.expectedLog)
				if tt.workerGetError || tt.invalidMaster || tt.missingNSG {
					assert.Equal(t, "warning", hook.LastEntry().Level.String())
				}
			}
			assert.Equal(t, api.PreconfiguredNSGEnabled, m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG)
		})
	}
}
