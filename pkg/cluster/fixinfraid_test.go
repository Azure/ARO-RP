package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestFixInfraID(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName"

	for _, tt := range []struct {
		name        string
		doc         *api.OpenShiftClusterDocument
		wantInfraID string
	}{
		{
			name: "no infra id",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						InfraID:           "",
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantInfraID: "aro",
		},
		{
			name: "aro infra id",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						InfraID:           "aro",
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantInfraID: "aro",
		},
		{
			name: "unique random id",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						InfraID:           "cluster-abc",
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantInfraID: "cluster-abc",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeOpenShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(fakeOpenShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			clusterdoc, err := fakeOpenShiftClustersDatabase.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: clusterdoc,
				db:  fakeOpenShiftClustersDatabase,
			}

			err = m.fixInfraID(ctx)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := fakeOpenShiftClustersDatabase.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantInfraID != doc.OpenShiftCluster.Properties.InfraID {
				t.Error(doc.OpenShiftCluster.Properties.InfraID)
			}
		})
	}
}
