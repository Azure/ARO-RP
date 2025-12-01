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

func TestEmitClusterVersionConditions(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		&configv1.ClusterVersion{
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
		ocpclientset: ocpclientset,
		m:            m,
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
