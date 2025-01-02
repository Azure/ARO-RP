package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitClusterOperatorConditions(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset(&configv1.ClusterOperator{
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
	})

	m := testmonitor.NewFakeEmitter(t)

	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	err := mon.emitClusterOperatorConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.VerifyEmittedMetrics(
		testmonitor.Metric("clusteroperator.count", int64(1), map[string]string{}),
		testmonitor.Metric("clusteroperator.conditions", int64(1), map[string]string{
			"name":   "console",
			"type":   "Available",
			"status": "False",
		}),
		testmonitor.Metric("clusteroperator.conditions", int64(1), map[string]string{
			"name":   "console",
			"type":   "Degraded",
			"status": "True",
		}),
		testmonitor.Metric("clusteroperator.conditions", int64(1), map[string]string{
			"name":   "console",
			"type":   "Progressing",
			"status": "True",
		}),
		testmonitor.Metric("clusteroperator.conditions", int64(1), map[string]string{
			"name":   "console",
			"type":   "Upgradeable",
			"status": "False",
		}),
		testmonitor.Metric("clusteroperator.conditions", int64(1), map[string]string{
			"name":   "console",
			"type":   "dummy",
			"status": "True",
		}),
	)
}
