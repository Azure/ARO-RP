package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitAroOperatorHeartbeat(t *testing.T) {
	ctx := context.Background()

	cli := fake.NewSimpleClientset(
		&appsv1.Deployment{ // not available expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name1",
				Namespace: "openshift-azure-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 0,
			},
		}, &appsv1.Deployment{ // available expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name2",
				Namespace: "openshift-azure-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
				Name:      "name2",
				Namespace: "customer",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          2,
				AvailableReplicas: 1,
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("arooperator.heartbeat", int64(1), map[string]string{
		"name":      "name1",
		"available": "false",
	})

	m.EXPECT().EmitGauge("arooperator.heartbeat", int64(1), map[string]string{
		"name":      "name2",
		"available": "true",
	})

	err := mon.emitAroOperatorHeartbeat(ctx)
	if err != nil {
		t.Fatal(err)
	}

}
