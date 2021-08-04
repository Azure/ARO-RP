package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestEnsureDefaults(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName"

	for _, tt := range []struct {
		name string
		doc  *api.OpenShiftClusterDocument
		want *api.OpenShiftClusterDocument
	}{
		{
			name: "doc without defaults",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			want: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						MasterProfile: api.MasterProfile{
							EncryptionAtHost: api.EncryptionAtHostDisabled,
						},
						NetworkProfile: api.NetworkProfile{
							SDNProvider: api.SDNProviderOpenShiftSDN,
						},
					},
				},
			},
		},
		{
			name: "doc with values",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						MasterProfile: api.MasterProfile{
							EncryptionAtHost: api.EncryptionAtHostEnabled,
						},
						NetworkProfile: api.NetworkProfile{
							SDNProvider: api.SDNProviderOVNKubernetes,
						},
					},
				},
			},
			want: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						MasterProfile: api.MasterProfile{
							EncryptionAtHost: api.EncryptionAtHostEnabled,
						},
						NetworkProfile: api.NetworkProfile{
							SDNProvider: api.SDNProviderOVNKubernetes,
						},
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
				doc: clusterdoc,
				db:  fakeOpenShiftClustersDatabase,
			}

			err = m.ensureDefaults(ctx)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := fakeOpenShiftClustersDatabase.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(&doc.OpenShiftCluster, &tt.want.OpenShiftCluster) {
				t.Error(fmt.Errorf("\n%+v\n !=\n%+v", doc.OpenShiftCluster, tt.want.OpenShiftCluster)) // can't use cmp due to cycle imports
			}
		})
	}
}
