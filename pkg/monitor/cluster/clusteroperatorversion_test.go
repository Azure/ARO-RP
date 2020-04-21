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

func TestEmitClusterOperatorsVersion(t *testing.T) {
	ctx := context.Background()

	configcli := fake.NewSimpleClientset(
		&configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "console",
			},
			Status: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: "4.3.0",
					},
					{
						Name:    "operator-good", // no metrics exected
						Version: "4.3.1",
					},
				},
			},
		},
		&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Status: configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: "4.3.1",
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
	mon.cache.clusterVersion, _ = configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	mon.cache.clusterOperatorList, _ = configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})

	m.EXPECT().EmitGauge("clusteroperators.version", int64(1), map[string]string{
		"name":    "console",
		"version": "4.3.0",
	})

	err := mon.emitClusterOperatorsVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
