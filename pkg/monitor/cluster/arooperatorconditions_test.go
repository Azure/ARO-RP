package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitAROOperatorConditions(t *testing.T) {
	baseCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{},
	}

	for _, tt := range []struct {
		name              string
		conditions        []operatorv1.OperatorCondition
		expectMetricsDims []map[string]string
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
				},
				{
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
				},
				{
					Type:   "DnsmasqClusterControllerAvailable",
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "MachineValid",
					Status: operatorv1.ConditionUnknown,
				},
			},
			expectMetricsDims: []map[string]string{
				{"type": "MachineValid", "status": "Unknown"},
				{"type": "DnsmasqClusterControllerDegraded", "status": "True"},
				{"type": "DnsmasqClusterControllerProgressing", "status": "True"},
				{"type": "DnsmasqClusterControllerAvailable", "status": "False"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx := context.Background()
			baseCluster.Status.Conditions = tt.conditions
			arocli := arofake.NewSimpleClientset(baseCluster)
			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				arocli: arocli,
				m:      m,
			}

			for _, i := range tt.expectMetricsDims {
				m.EXPECT().EmitGauge(operatorConditionsMetricsTopic, int64(1), i)
			}

			err := mon.emitAroOperatorConditions(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
