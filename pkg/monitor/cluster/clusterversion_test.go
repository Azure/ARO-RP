package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterVersion(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name               string
		cv                 *configv1.ClusterVersion
		wantActualVersion  string
		wantDesiredVersion string
	}{
		{
			name: "without spec",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Update{
						Version: "4.3.3",
					},
					History: []configv1.UpdateHistory{
						{
							State:   configv1.PartialUpdate,
							Version: "4.3.2",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.3.1",
						},
						{
							State:   configv1.CompletedUpdate,
							Version: "4.3.0",
						},
					},
				},
			},
			wantActualVersion:  "4.3.1",
			wantDesiredVersion: "4.3.3",
		},
		{
			name: "with spec",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{
					DesiredUpdate: &configv1.Update{
						Version: "4.3.4",
					},
				},
				Status: configv1.ClusterVersionStatus{
					Desired: configv1.Update{
						Version: "4.3.3",
					},
				},
			},
			wantDesiredVersion: "4.3.4",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configcli := fake.NewSimpleClientset(tt.cv)

			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockInterface(controller)

			mon := &Monitor{
				configcli: configcli,
				m:         m,
			}

			m.EXPECT().EmitGauge("cluster.version", int64(1), map[string]string{
				"actualVersion":  tt.wantActualVersion,
				"desiredVersion": tt.wantDesiredVersion,
			})

			err := mon.emitClusterVersion(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
