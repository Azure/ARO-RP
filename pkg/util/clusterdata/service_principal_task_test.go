package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

func TestServicePrincipalEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		client  kubernetes.Interface
		wantOc  *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "config map object exists - valid json",
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-provider-config",
					Namespace: "openshift-config",
				},
				Data: map[string]string{
					"config": `{
	"cloud": "AzurePublicCloud",
	"tenantId": "fake-tenant-id",
	"aadClientId": "fake-client-id",
	"aadClientSecret": "fake-client-secret"
}`,
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID: "fake-client-id",
						TenantID: "fake-tenant-id",
					},
				},
			},
		},
		{
			name: "config map object exists - invalid json",
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-provider-config",
					Namespace: "openshift-config",
				},
				Data: map[string]string{
					"config": "invalid",
				},
			}),
			wantOc:  &api.OpenShiftCluster{},
			wantErr: "invalid character 'i' looking for beginning of value",
		},
		{
			name: `config map object exists - not "config" key`,
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-provider-config",
					Namespace: "openshift-config",
				},
			}),
			wantOc:  &api.OpenShiftCluster{},
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "config map object does not exist",
			client:  fake.NewSimpleClientset(),
			wantOc:  &api.OpenShiftCluster{},
			wantErr: `configmaps "cloud-provider-config" not found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{}
			e := &servicePrincipalEnricherTask{
				log:    log,
				client: tt.client,
				oc:     oc,
			}
			e.SetDefaults()

			callbacks := make(chan func())
			errors := make(chan error)
			go e.FetchData(callbacks, errors)

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
			}
		})
	}
}
