package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestFixUUID(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName"

	for _, tt := range []struct {
		name string
		doc  *api.OpenShiftClusterDocument
	}{
		{
			name: "no UUID",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						UUID:              "",
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
		},
		{
			name: "valid UUID",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						UUID:              "123e4567-e89b-12d3-a456-426614174000",
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
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

			err = m.fixUUID(ctx)
			if err != nil {
				t.Fatal(err)
			}

			checkDoc, err := fakeOpenShiftClustersDatabase.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			if !uuid.IsValid(checkDoc.OpenShiftCluster.Properties.UUID) {
				t.Fatalf("Invalid UUID %s", checkDoc.OpenShiftCluster.Properties.UUID)
			}
		})
	}
}
