package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitMachineConfigPoolMetrics(t *testing.T) {
	ctx := context.Background()

	mcocli := fake.NewSimpleClientset(&v1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-config-pool",
		},
		Status: v1.MachineConfigPoolStatus{
			Conditions: []v1.MachineConfigPoolCondition{
				{
					Type:   v1.MachineConfigPoolDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.MachineConfigPoolNodeDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.MachineConfigPoolRenderDegraded,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.MachineConfigPoolUpdated,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   v1.MachineConfigPoolUpdating,
					Status: corev1.ConditionTrue,
				},
			},
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		mcocli: mcocli,
		m:      m,
	}

	m.EXPECT().EmitGauge("machineconfigpools.conditions.count", int64(1), map[string]string{
		"machineconfigpool": "machine-config-pool",
		"condition":         "Degraded",
	})

	m.EXPECT().EmitGauge("machineconfigpools.conditions.count", int64(1), map[string]string{
		"machineconfigpool": "machine-config-pool",
		"condition":         "NodeDegraded",
	})

	m.EXPECT().EmitGauge("machineconfigpools.conditions.count", int64(1), map[string]string{
		"machineconfigpool": "machine-config-pool",
		"condition":         "RenderDegraded",
	})

	m.EXPECT().EmitGauge("machineconfigpools.conditions.count", int64(1), map[string]string{
		"machineconfigpool": "machine-config-pool",
		"condition":         "NotUpdated",
	})

	m.EXPECT().EmitGauge("machineconfigpools.conditions.count", int64(1), map[string]string{
		"machineconfigpool": "machine-config-pool",
		"condition":         "Updating",
	})

	err := mon.emitMachineConfigPoolMetrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
