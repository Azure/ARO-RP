package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitNodesConditions(t *testing.T) {
	ctx := context.Background()

	cli := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-0",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-master-1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("nodes.count", int64(2), map[string]string{})
	m.EXPECT().EmitGauge("nodes.conditions", int64(1), map[string]string{
		"name":   "aro-master-0",
		"status": "True",
		"type":   "MemoryPressure",
	})
	m.EXPECT().EmitGauge("nodes.conditions", int64(1), map[string]string{
		"name":   "aro-master-1",
		"status": "False",
		"type":   "Ready",
	})

	err := mon.emitNodesConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
