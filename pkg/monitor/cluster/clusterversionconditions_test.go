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

func TestEmitClusterVersionConditions(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset(&configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
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

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	m.EXPECT().EmitGauge("clusterversion.conditions", int64(1), map[string]string{
		"type":   "Available",
		"status": "False",
	})

	m.EXPECT().EmitGauge("clusterversion.conditions", int64(1), map[string]string{
		"type":   "Degraded",
		"status": "True",
	})

	m.EXPECT().EmitGauge("clusterversion.conditions", int64(1), map[string]string{
		"type":   "Progressing",
		"status": "True",
	})

	m.EXPECT().EmitGauge("clusterversion.conditions", int64(1), map[string]string{
		"type":   "Upgradeable",
		"status": "False",
	})

	m.EXPECT().EmitGauge("clusterversion.conditions", int64(1), map[string]string{
		"type":   "dummy",
		"status": "True",
	})

	err := mon.emitClusterVersionConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
