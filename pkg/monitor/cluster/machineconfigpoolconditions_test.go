package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitMachineConfigPoolConditions(t *testing.T) {
	ctx := context.Background()

	mcocli := mcofake.NewSimpleClientset(&mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-config-pool",
		},
		Status: mcv1.MachineConfigPoolStatus{
			Conditions: []mcv1.MachineConfigPoolCondition{
				{
					Type:   mcv1.MachineConfigPoolDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   mcv1.MachineConfigPoolNodeDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   mcv1.MachineConfigPoolRenderDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   mcv1.MachineConfigPoolUpdated,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   mcv1.MachineConfigPoolUpdating,
					Status: corev1.ConditionTrue,
				},
			},
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		mcocli: mcocli,
		m:      m,
	}

	m.EXPECT().EmitGauge("machineconfigpool.count", int64(1), map[string]string{})

	m.EXPECT().EmitGauge("machineconfigpool.conditions", int64(1), map[string]string{
		"name":   "machine-config-pool",
		"type":   "Degraded",
		"status": "True",
	})

	m.EXPECT().EmitGauge("machineconfigpool.conditions", int64(1), map[string]string{
		"name":   "machine-config-pool",
		"type":   "NodeDegraded",
		"status": "True",
	})

	m.EXPECT().EmitGauge("machineconfigpool.conditions", int64(1), map[string]string{
		"name":   "machine-config-pool",
		"type":   "RenderDegraded",
		"status": "True",
	})

	m.EXPECT().EmitGauge("machineconfigpool.conditions", int64(1), map[string]string{
		"name":   "machine-config-pool",
		"type":   "Updated",
		"status": "False",
	})

	m.EXPECT().EmitGauge("machineconfigpool.conditions", int64(1), map[string]string{
		"name":   "machine-config-pool",
		"type":   "Updating",
		"status": "True",
	})

	err := mon.emitMachineConfigPoolConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
