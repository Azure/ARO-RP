package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitClusterVersion(t *testing.T) {
	ctx := context.Background()

	aroDeployment := &appsv1.Deployment{ // metrics expected
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operator.Namespace,
			Name:      "aro-operator-master",
			Labels: map[string]string{
				"version": "test",
			},
		},
	}

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
			name: "without spec, at nightly",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Release{
						Version: "4.5.0-0.nightly-2025-07-31-063120",
					},
					History: []configv1.UpdateHistory{
						{
							State:   configv1.PartialUpdate,
							Version: "4.5.2",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.5.0-0.ci-2025-08-05-023912",
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
			wantActualVersion:                        "4.5.0-0.ci-2025-08-05-023912",
			wantDesiredVersion:                       "4.5.0-0.nightly-2025-07-31-063120",
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
			name: "with ProvisionedBy unknown",
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
			controller := gomock.NewController(t)

			_, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, fake.
				NewClientBuilder().
				WithObjects(aroDeployment, tt.cv).
				Build())

			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				ocpclientset: ocpclientset,
				m:            m,
				log:          log,
				oc:           tt.oc,
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

			err := mon.prefetchClusterVersion(ctx)
			if err != nil {
				t.Fatal(err)
			}

			err = mon.emitClusterVersions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
