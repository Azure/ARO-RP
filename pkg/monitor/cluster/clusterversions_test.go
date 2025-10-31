package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/Azure/ARO-RP/pkg/util/version"
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
		wantErr                                  error
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

func TestPrefetchClusterVersion(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name               string
		cv                 *configv1.ClusterVersion
		wantActualVersion  version.Version
		wantDesiredVersion version.Version
		wantErr            error
		wantLogs           []testlog.ExpectedLogEntry
	}{
		{
			name: "happy path",
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
			wantActualVersion:  version.NewVersion(4, 5, 1),
			wantDesiredVersion: version.NewVersion(4, 5, 3),
		},
		{
			name: "malformed desired Version",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Release{
						Version: "sporngs",
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
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("failure parsing desired ClusterVersion: could not parse version \"sporngs\""),
				},
			},
			wantActualVersion: version.NewVersion(4, 5, 1),
		},
		{
			name: "malformed actual Version",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Release{
						Version: "4.6.1",
					},
					History: []configv1.UpdateHistory{
						{
							State:   configv1.PartialUpdate,
							Version: "4.5.2",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "sporngs",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.5.0",
						},
					},
				},
			},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("failure parsing ClusterVersion: could not parse version \"sporngs\""),
				},
			},
			wantDesiredVersion: version.NewVersion(4, 6, 1),
		},
		{
			name:    "missing clusterversion",
			wantErr: errFetchClusterVersion,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			cb := fake.
				NewClientBuilder()
			if tt.cv != nil {
				cb.WithObjects(tt.cv)
			}

			h, log := testlog.New()
			ocpclientset := clienthelper.NewWithClient(log, cb.
				Build())

			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				ocpclientset: ocpclientset,
				m:            m,
				log:          log,
			}

			err := mon.prefetchClusterVersion(ctx)

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Wanted %v, got %v", err, tt.wantErr)
			} else if tt.wantErr == nil && err != nil {
				t.Fatal(err)
			}

			err = testlog.AssertLoggingOutput(h, tt.wantLogs)
			if err != nil {
				t.Error(err)
			}

			if tt.wantActualVersion != nil && !tt.wantActualVersion.Eq(mon.clusterActualVersion) {
				t.Errorf("actualversion: got %s, wanted %s", mon.clusterActualVersion.String(), tt.wantActualVersion.String())
			}

			if tt.wantDesiredVersion != nil && !tt.wantDesiredVersion.Eq(mon.clusterDesiredVersion) {
				t.Errorf("desiredversion: got %s, wanted %s", mon.clusterDesiredVersion.String(), tt.wantDesiredVersion.String())
			}
		})
	}
}
