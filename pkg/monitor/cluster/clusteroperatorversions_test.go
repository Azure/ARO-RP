package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterOperatorVersion(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset(
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
				Desired: configv1.Release{
					Version: "4.3.1",
				},
			},
		})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	m.EXPECT().EmitGauge("clusteroperator.versions", int64(1), map[string]string{
		"name":    "console",
		"version": "4.3.0",
	})

	err := mon.emitClusterOperatorVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
