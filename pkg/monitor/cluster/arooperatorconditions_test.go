package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitAROOperatorConditions(t *testing.T) {
	baseCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{},
	}

	for _, tt := range []struct {
		name          string
		conditions    []operatorv1.OperatorCondition
		expectMetrics []testmonitor.ExpectedMetric
	}{
		{
			name: "expected values are ignored",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   "DnsmasqClusterControllerDegraded",
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "DnsmasqClusterControllerProgressing",
					Status: operatorv1.ConditionFalse,
				}, {
					Type:   "DnsmasqClusterControllerAvailable",
					Status: operatorv1.ConditionTrue,
				},
				{
					Type:   "MachineValid",
					Status: operatorv1.ConditionTrue,
				},
			},
		},
		{
			name: "non-expected values are emitted",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   "DnsmasqClusterControllerDegraded",
					Status: operatorv1.ConditionTrue,
				},
				{
					Type:   "DnsmasqClusterControllerProgressing",
					Status: operatorv1.ConditionTrue,
				}, {
					Type:   "DnsmasqClusterControllerAvailable",
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "MachineValid",
					Status: operatorv1.ConditionUnknown,
				},
			},
			expectMetrics: []testmonitor.ExpectedMetric{
				testmonitor.Metric(operatorConditionsMetricsTopic, int64(1), map[string]string{"type": "MachineValid", "status": "Unknown"}),
				testmonitor.Metric(operatorConditionsMetricsTopic, int64(1), map[string]string{"type": "DnsmasqClusterControllerDegraded", "status": "True"}),
				testmonitor.Metric(operatorConditionsMetricsTopic, int64(1), map[string]string{"type": "DnsmasqClusterControllerProgressing", "status": "True"}),
				testmonitor.Metric(operatorConditionsMetricsTopic, int64(1), map[string]string{"type": "DnsmasqClusterControllerAvailable", "status": "False"}),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			baseCluster.Status.Conditions = tt.conditions
			arocli := arofake.NewSimpleClientset(baseCluster)

			m := testmonitor.NewFakeEmitter(t)
			mon := &Monitor{
				arocli: arocli,
				m:      m,
			}

			err := mon.emitAroOperatorConditions(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m.VerifyEmittedMetrics(tt.expectMetrics...)
		})
	}
}
