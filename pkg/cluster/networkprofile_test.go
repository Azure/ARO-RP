package cluster

import (
	"context"
	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// Test our populate MTUSize function
func TestPopulateMTUSize(t *testing.T) {
	ctx := context.Background()

	// Define the DB instance we will use to run the PatchWithLease function
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	// Run tests
	for _, tt := range []struct {
		name        string
		m           manager
		expectedMTU api.MTUSize
		expectedErr error
	}{
		{
			name: "No MTU size defined, MTU3900 flag",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
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
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagMTU3900,
									State: "Registered",
								},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU3900,
			expectedErr: nil,
		},
		{
			name: "No MTU size defined, No MTU flag",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
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
			},
			expectedMTU: api.MTU1500,
			expectedErr: nil,
		},
		{
			name: "MTU1500 defined, MTU3900 flag",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								MTUSize: api.MTU1500,
							},
						},
					},
				},
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagMTU3900,
									State: "Registered",
								},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU3900,
			expectedErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Create the DB to test the cluster
			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.m.doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}
			tt.m.db = openShiftClustersDatabase

			// Run populateMTUSize and assert the correct results
			err = tt.m.populateMTUSize(ctx)
			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
			assert.Equal(t, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize, tt.expectedMTU, "MTUSize was not updated as expected exception")
		})
	}
}

func TestEnsureMTUSize(t *testing.T) {
	ctx := context.Background()

	// Run tests
	for _, tt := range []struct {
		name        string
		m           manager
		expectedMTU api.MTUSize
		expectedErr error
	}{
		{},
	} {
		t.Run(tt.name, func(t *testing.T) {

			tt.m.ensureMTUSize(ctx)
		})
	}
}
