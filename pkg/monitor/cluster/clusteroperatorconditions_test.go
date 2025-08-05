package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitClusterOperatorConditions(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		&configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "apiserver",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: configv1.ConditionTrue,
					},
				},
			},
		},
		&configv1.ClusterOperator{
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
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
					{
						Type:   configv1.OperatorDegraded,
						Status: configv1.ConditionFalse,
					},
					{
						Type:   configv1.OperatorDegraded,
						Status: configv1.ConditionTrue,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: configv1.ConditionFalse,
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
						Type:   configv1.OperatorUpgradeable,
						Status: configv1.ConditionTrue,
					},
					{
						Type:   "dummy",
						Status: configv1.ConditionTrue,
					},
				},
			},
		},
	}

	controller := gomock.NewController(t)
	m := mock_metrics.NewMockEmitter(controller)

	_, log := testlog.New()
	ocpclientset := clienthelper.NewWithClient(log, fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build())

	mon := &Monitor{
		log:          log,
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   1,
	}

	m.EXPECT().EmitGauge("clusteroperator.count", int64(2), map[string]string{})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "console",
		"type":   "Available",
		"status": "False",
	})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "console",
		"type":   "Degraded",
		"status": "True",
	})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "console",
		"type":   "Progressing",
		"status": "True",
	})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "console",
		"type":   "Upgradeable",
		"status": "False",
	})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "console",
		"type":   "dummy",
		"status": "True",
	})

	m.EXPECT().EmitGauge("clusteroperator.conditions", int64(1), map[string]string{
		"name":   "apiserver",
		"type":   "Progressing",
		"status": "True",
	})

	err := mon.emitClusterOperatorConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
