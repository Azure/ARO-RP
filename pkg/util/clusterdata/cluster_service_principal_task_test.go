package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestClusterServidePrincipalEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	name := "azure-credentials"
	namespace := "kube-system"

	for _, tt := range []struct {
		name    string
		client  kubernetes.Interface
		wantOc  *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "enrich worked",
			client: fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"azure_client_id":     []byte("new-client-id"),
					"azure_client_secret": []byte("new-client-secret"),
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "new-client-id",
						ClientSecret: api.SecureString("new-client-secret"),
					},
				},
			},
		},
		{
			name:   "enrich failed - stale data",
			client: fake.NewSimpleClientset(),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "old-client-id",
						ClientSecret: "old-client-secret",
					},
				},
			},
			wantErr: "secrets \"azure-credentials\" not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID:     "old-client-id",
						ClientSecret: api.SecureString("old-client-secret"),
					},
				},
			}
			e := &clusterServicePrincipalEnricherTask{
				log:    log,
				client: tt.client,
				oc:     oc,
			}
			e.SetDefaults()

			callbacks := make(chan func())
			errors := make(chan error)
			go e.FetchData(context.Background(), callbacks, errors)

			select {
			case f := <-callbacks:
				f()
				if !reflect.DeepEqual(oc, tt.wantOc) {
					t.Error(cmp.Diff(oc, tt.wantOc))
				}
			case err := <-errors:
				if tt.wantErr != err.Error() {
					t.Error(err)
				}
				// we want to make sure we see stale database data in case of failures
				if !reflect.DeepEqual(oc, tt.wantOc) {
					t.Error(cmp.Diff(oc, tt.wantOc))
				}
			}
		})
	}
}
