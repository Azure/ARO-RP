package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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
							OperatorFlags: api.OperatorFlags{
								operator.AzureSubnetsNsgManaged: operator.FlagFalse,
							},
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

func TestSyncPreconfiguredNSGManagementPolicy(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/" + subscriptionId + "/resourceGroups/cluster-rg/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster"

	for _, tt := range []struct {
		name             string
		stored           api.PreconfiguredNSG
		nsgResourceGroup string
		nsgManaged       string
		expected         api.PreconfiguredNSG
		expectedLog      string
	}{
		{
			name:             "NSGs outside managed resource group enable PreconfiguredNSG",
			stored:           api.PreconfiguredNSGDisabled,
			nsgResourceGroup: vnetResourceGroup,
			nsgManaged:       operator.FlagFalse,
			expected:         api.PreconfiguredNSGEnabled,
		},
		{
			name:             "NSGs in managed resource group disable PreconfiguredNSG",
			stored:           api.PreconfiguredNSGEnabled,
			nsgResourceGroup: "cluster-rg",
			nsgManaged:       operator.FlagTrue,
			expected:         api.PreconfiguredNSGDisabled,
		},
		{
			name:             "managed policy rejects enabling PreconfiguredNSG",
			stored:           api.PreconfiguredNSGDisabled,
			nsgResourceGroup: vnetResourceGroup,
			nsgManaged:       operator.FlagTrue,
			expected:         api.PreconfiguredNSGDisabled,
			expectedLog:      "inferred PreconfiguredNSG \"Enabled\" conflicts with",
		},
		{
			name:             "unmanaged policy rejects disabling PreconfiguredNSG",
			stored:           api.PreconfiguredNSGEnabled,
			nsgResourceGroup: "cluster-rg",
			nsgManaged:       operator.FlagFalse,
			expected:         api.PreconfiguredNSGEnabled,
			expectedLog:      "inferred PreconfiguredNSG \"Disabled\" conflicts with",
		},
		{
			name:             "missing management flag prevents repair",
			stored:           api.PreconfiguredNSGDisabled,
			nsgResourceGroup: vnetResourceGroup,
			expected:         api.PreconfiguredNSGDisabled,
			expectedLog:      "management flag aro.azuresubnets.nsg.managed is missing or invalid",
		},
		{
			name:             "invalid management flag prevents repair",
			stored:           api.PreconfiguredNSGDisabled,
			nsgResourceGroup: vnetResourceGroup,
			nsgManaged:       "invalid",
			expected:         api.PreconfiguredNSGDisabled,
			expectedLog:      "management flag aro.azuresubnets.nsg.managed is missing or invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			subnetsClient := mock_armnetwork.NewMockSubnetsClient(controller)
			for _, subnetName := range []string{subnetNameMaster, subnetNameWorker} {
				subnetsClient.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetName, nil).
					Return(armnetwork.SubnetsClientGetResponse{
						Subnet: armnetwork.Subnet{
							Properties: &armnetwork.SubnetPropertiesFormat{
								NetworkSecurityGroup: &armnetwork.SecurityGroup{
									ID: new("/subscriptions/" + subscriptionId + "/resourceGroups/" + tt.nsgResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/nsg"),
								},
							},
						},
					}, nil)
			}

			flags := api.OperatorFlags{"unrelated": operator.FlagTrue}
			if tt.nsgManaged != "" {
				flags[operator.AzureSubnetsNsgManaged] = tt.nsgManaged
			}
			db, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(db)
			fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/" + subscriptionId + "/resourceGroups/cluster-rg",
						},
						MasterProfile: api.MasterProfile{SubnetID: subnetIdMaster},
						WorkerProfiles: []api.WorkerProfile{
							{SubnetID: subnetIdWorker},
						},
						NetworkProfile: api.NetworkProfile{PreconfiguredNSG: tt.stored},
						OperatorFlags:  flags,
					},
				},
			})
			require.NoError(t, fixture.Create())
			doc, err := db.Dequeue(ctx)
			require.NoError(t, err)
			require.NotNil(t, doc)

			hook, log := testlog.LogForTesting(t)
			m := &manager{
				log:        log,
				armSubnets: subnetsClient,
				db:         db,
				doc:        doc,
			}
			require.NoError(t, m.syncPreconfiguredNSG(ctx))

			persisted, err := db.Get(ctx, strings.ToLower(resourceID))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, persisted.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG)
			assert.Equal(t, tt.expected, m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG)
			assert.Equal(t, flags, persisted.OpenShiftCluster.Properties.OperatorFlags)
			assert.Equal(t, flags, m.doc.OpenShiftCluster.Properties.OperatorFlags)
			if tt.expectedLog != "" {
				assert.Contains(t, hook.LastEntry().Message, tt.expectedLog)
				assert.Equal(t, "warning", hook.LastEntry().Level.String())
				assert.Equal(t, doc.ETag, persisted.ETag)
			}
		})
	}
}
