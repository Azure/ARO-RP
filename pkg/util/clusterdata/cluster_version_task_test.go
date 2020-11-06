package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

func TestClusterVersionEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		client  configclient.Interface
		wantOc  *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "version object exists",
			client: fake.NewSimpleClientset(&configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Update{Version: "1.2.3"},
				},
			}),
			wantOc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: "1.2.3",
					},
				},
			},
		},
		{
			name: "version object exists, but desired version is not set",
			client: fake.NewSimpleClientset(&configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
			}),
			wantOc: &api.OpenShiftCluster{},
		},
		{
			name:    "version object does not exist",
			client:  fake.NewSimpleClientset(),
			wantOc:  &api.OpenShiftCluster{},
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{}
			e := &clusterVersionEnricherTask{
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
			}
		})
	}
}
