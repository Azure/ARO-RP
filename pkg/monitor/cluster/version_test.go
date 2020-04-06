package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterVersionMetrics(t *testing.T) {
	configcli := fake.NewSimpleClientset(&configv1.ClusterVersion{
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
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	m.EXPECT().EmitGauge("cluster.version", int64(1), map[string]string{
		"actualVersion":  "4.3.1",
		"desiredVersion": "4.3.3",
	})

	err := mon.emitClusterVersionMetrics()
	if err != nil {
		t.Fatal(err)
	}
}
