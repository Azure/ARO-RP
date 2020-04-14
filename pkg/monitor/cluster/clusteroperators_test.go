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

func TestEmitClusterOperatorsMetrics(t *testing.T) {
	ctx := context.Background()

	configcli := fake.NewSimpleClientset(&configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "console",
		},
		Status: configv1.ClusterOperatorStatus{
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorAvailable,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   configv1.OperatorDegraded,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionTrue,
				},
				{
					Type:   configv1.OperatorUpgradeable,
					Status: configv1.ConditionFalse,
				},
				{
					Type:   "dummy",
					Status: configv1.ConditionTrue,
				},
			},
			Versions: []configv1.OperandVersion{
				{
					Name:    "dummy",
					Version: "4.3.2",
				},
				{
					Name:    "operator",
					Version: "4.3.1",
				},
				{
					Name:    "operator",
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

	m.EXPECT().EmitGauge("clusteroperators.conditions.count", int64(1), map[string]string{
		"clusteroperator": "console",
		"condition":       "NotAvailable",
	})

	m.EXPECT().EmitGauge("clusteroperators.conditions.count", int64(1), map[string]string{
		"clusteroperator": "console",
		"condition":       "Degraded",
	})

	m.EXPECT().EmitGauge("clusteroperators.conditions.count", int64(1), map[string]string{
		"clusteroperator": "console",
		"condition":       "Progressing",
	})

	m.EXPECT().EmitGauge("clusteroperators.conditions.count", int64(1), map[string]string{
		"clusteroperator": "console",
		"condition":       "NotUpgradeable",
	})

	m.EXPECT().EmitGauge("clusteroperators.version", int64(1), map[string]string{
		"clusteroperator": "console",
		"version":         "4.3.1",
	})

	err := mon.emitClusterOperatorsMetrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
