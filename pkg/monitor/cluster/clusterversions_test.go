package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterVersion(t *testing.T) {
	ctx := context.Background()

	cli := fake.NewSimpleClientset(
		&appsv1.Deployment{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Namespace: operator.Namespace,
				Name:      "aro-operator-master",
				Labels: map[string]string{
					"version": "test",
				},
			},
		},
	)

	for _, tt := range []struct {
		name                                     string
		cv                                       *configv1.ClusterVersion
		oc                                       *api.OpenShiftCluster
		wantActualVersion                        string
		wantDesiredVersion                       string
		wantProvisionedByResourceProviderVersion string
		wantAvailableRP                          string
		wantActualMinorVersion                   string
	}{
		{
			name: "without spec",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Release{
						Version: "4.5.3",
					},
					History: []configv1.UpdateHistory{
						{
							State:   configv1.PartialUpdate,
							Version: "4.5.2",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.5.1",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.5.0",
						},
					},
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			wantActualVersion:                        "4.5.1",
			wantDesiredVersion:                       "4.5.3",
			wantProvisionedByResourceProviderVersion: "",
			wantAvailableRP:                          "unknown",
			wantActualMinorVersion:                   "4.5",
		},
		{
			name: "with spec",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{
					DesiredUpdate: &configv1.Update{
						Version: "4.5.4",
					},
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Release{
						Version: "4.5.3",
					},
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			wantDesiredVersion:                       "4.5.4",
			wantProvisionedByResourceProviderVersion: "",
			wantAvailableRP:                          "unknown",
			wantActualMinorVersion:                   "",
		},
		{
			name: "with ProvisionedBy",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisionedBy: "somesha",
				},
			},
			wantProvisionedByResourceProviderVersion: "somesha",
			wantAvailableRP:                          "unknown", // (rpVersion = unknown) != (provisionedByResourceProvider = "")
		},
		{
			name: "with ProvisionedBy",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisionedBy: "unknown",
				},
			},
			wantProvisionedByResourceProviderVersion: "unknown",
			wantAvailableRP:                          "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configcli := configfake.NewSimpleClientset(tt.cv)

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				configcli: configcli,
				m:         m,
				oc:        tt.oc,
				cli:       cli,
			}

			m.EXPECT().EmitGauge("cluster.versions", int64(1), map[string]string{
				"actualVersion":                        tt.wantActualVersion,
				"desiredVersion":                       tt.wantDesiredVersion,
				"provisionedByResourceProviderVersion": tt.wantProvisionedByResourceProviderVersion,
				"operatorVersion":                      "test",
				"resourceProviderVersion":              "unknown",
				"availableRP":                          tt.wantAvailableRP,
				"actualMinorVersion":                   tt.wantActualMinorVersion,
			})

			err := mon.emitClusterVersions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
