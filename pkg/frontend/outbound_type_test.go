package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestDetermineOutboundType(t *testing.T) {
	ctx := context.Background()

	// Define the DB instance we will use to run the PatchWithLease function
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	// Run tests
	for _, tt := range []struct {
		name                 string
		clusterDoc           *api.OpenShiftClusterDocument
		subscriptionDoc      *api.SubscriptionDocument
		expectedOutboundType api.OutboundType
		expectedErr          error
	}{
		{
			name: "No OutboundType defined and no feature flag registered",
			clusterDoc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						RegisteredFeatures: []api.RegisteredFeatureProfile{},
					},
				},
			},
			expectedOutboundType: api.OutboundTypeLoadbalancer,
			expectedErr:          nil,
		},
		{
			name: "No OutboundType defined and UserDefinedRouting feature flag registered",
			clusterDoc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						APIServerProfile: api.APIServerProfile{
							Visibility: api.VisibilityPrivate,
						},
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPrivate,
							},
						},
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						RegisteredFeatures: []api.RegisteredFeatureProfile{
							{
								Name:  api.FeatureFlagUserDefinedRouting,
								State: "Registered",
							},
						},
					},
				},
			},
			expectedOutboundType: api.OutboundTypeUserDefinedRouting,
			expectedErr:          nil,
		},
		{
			name: "OutboundType specified and UserDefinedRouting feature flag registered",
			clusterDoc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						NetworkProfile: api.NetworkProfile{
							OutboundType: api.OutboundTypeLoadbalancer,
						},
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						RegisteredFeatures: []api.RegisteredFeatureProfile{
							{
								Name:  api.FeatureFlagUserDefinedRouting,
								State: "Registered",
							},
						},
					},
				},
			},
			expectedOutboundType: api.OutboundTypeLoadbalancer,
			expectedErr:          nil,
		},
		{
			name: "OutboundType specified and no feature flag registered",
			clusterDoc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						NetworkProfile: api.NetworkProfile{
							OutboundType: api.OutboundTypeLoadbalancer,
						},
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						RegisteredFeatures: []api.RegisteredFeatureProfile{},
					},
				},
			},
			expectedOutboundType: api.OutboundTypeLoadbalancer,
			expectedErr:          nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run determineOutboundType and assert the correct results
			determineOutboundType(ctx, tt.clusterDoc, tt.subscriptionDoc)
			assert.Equal(t, tt.expectedOutboundType, tt.clusterDoc.OpenShiftCluster.Properties.NetworkProfile.OutboundType, "OutboundType was not updated as expected exception")
		})
	}
}
