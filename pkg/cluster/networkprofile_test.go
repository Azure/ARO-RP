package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

// TestPopulateMTUSize will test our populate MTUSize function
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
			assert.Equal(t, tt.expectedMTU, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize, "MTUSize was not updated as expected exception")
		})
	}
}

// TestEnsureMTUSize will test our ensureMTUSize function
func TestEnsureMTUSize(t *testing.T) {
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
			name: "MTUSize set to 1500",
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
				mcocli: mcofake.NewSimpleClientset(
					&mcv1.MachineConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
						},
					}),
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU1500,
			expectedErr: nil,
		},
		{
			name: "MTUSize set to 3900",
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							NetworkProfile: api.NetworkProfile{
								MTUSize: api.MTU3900,
							},
						},
					},
				},
				mcocli: mcofake.NewSimpleClientset(
					&mcv1.MachineConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
						},
					}),
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU3900,
			expectedErr: nil,
		},
		{
			name: "No MTUSize & MachineConfig found",
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
				mcocli: mcofake.NewSimpleClientset(
					&mcv1.MachineConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "99-master-mtu",
						},
					}),
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU3900,
			expectedErr: nil,
		},
		{
			name: "No MTUSize & MachineConfig not found",
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
				mcocli: mcofake.NewSimpleClientset(
					&mcv1.MachineConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
						},
					}),
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{},
							},
						},
					},
				},
			},
			expectedMTU: api.MTU1500,
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

			err = tt.m.ensureMTUSize(ctx)

			assert.Equal(t, tt.expectedErr, err, "Unexpected error exception")
			assert.Equal(t, tt.expectedMTU, tt.m.doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize, "MTUSize was not updated as expected exception")
		})
	}
}
