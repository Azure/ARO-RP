package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
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

	m := testmonitor.NewFakeEmitter(t)
	mon := &Monitor{
		mcocli: mcocli,
		m:      m,
	}

	err := mon.emitMachineConfigPoolConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	m.VerifyEmittedMetrics(
		testmonitor.Metric("machineconfigpool.count", int64(1), map[string]string{}),

		testmonitor.Metric("machineconfigpool.conditions", int64(1), map[string]string{
			"name":   "machine-config-pool",
			"type":   "Degraded",
			"status": "True",
		}),

		testmonitor.Metric("machineconfigpool.conditions", int64(1), map[string]string{
			"name":   "machine-config-pool",
			"type":   "NodeDegraded",
			"status": "True",
		}),

		testmonitor.Metric("machineconfigpool.conditions", int64(1), map[string]string{
			"name":   "machine-config-pool",
			"type":   "RenderDegraded",
			"status": "True",
		}),

		testmonitor.Metric("machineconfigpool.conditions", int64(1), map[string]string{
			"name":   "machine-config-pool",
			"type":   "Updated",
			"status": "False",
		}),

		testmonitor.Metric("machineconfigpool.conditions", int64(1), map[string]string{
			"name":   "machine-config-pool",
			"type":   "Updating",
			"status": "True",
		}),
	)

}
