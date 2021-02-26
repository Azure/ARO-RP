package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPopulateCreatedAt(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName"
	mockCreationTimestamp := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name     string
		ns       *corev1.Namespace
		doc      *api.OpenShiftClusterDocument
		wantTime time.Time
		wantErr  string
	}{
		{
			name: "doc without timestamp",
			ns: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.NewTime(mockCreationTimestamp),
				},
			},
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantTime: mockCreationTimestamp,
		},
		{
			name: "doc with timestamp",
			ns: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.NewTime(mockCreationTimestamp),
				},
			},
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						CreatedAt:         time.Date(1970, 1, 1, 0, 0, 0, 1, time.UTC),
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantTime: time.Date(1970, 1, 1, 0, 0, 0, 1, time.UTC),
		},
		{
			name: "default namespace doesn't exist",
			ns:   &corev1.Namespace{},
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
			wantErr: "namespaces \"default\" not found",
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
				log:           logrus.NewEntry(logrus.StandardLogger()),
				kubernetescli: fake.NewSimpleClientset(tt.ns),
				doc:           clusterdoc,
				db:            fakeOpenShiftClustersDatabase,
			}

			err = m.populateCreatedAt(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}

			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}

				doc, err := fakeOpenShiftClustersDatabase.Get(ctx, strings.ToLower(resourceID))
				if err != nil {
					t.Fatal(err)
				}
				if tt.wantTime != doc.OpenShiftCluster.Properties.CreatedAt {
					t.Error(doc.OpenShiftCluster.Properties.CreatedAt)
				}
				if tt.wantTime != m.doc.OpenShiftCluster.Properties.CreatedAt {
					t.Error(m.doc.OpenShiftCluster.Properties.CreatedAt)
				}
			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}
			}
		})
	}
}
