package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitReplicasetStatuses(t *testing.T) {
	ctx := context.Background()

	cli := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name1",
				Namespace: "openshift",
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:          2,
				AvailableReplicas: 1,
			},
		}, &appsv1.ReplicaSet{ // no metric expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name2",
				Namespace: "openshift",
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:          2,
				AvailableReplicas: 2,
			},
		}, &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
				Name:      "name2",
				Namespace: "customer",
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:          2,
				AvailableReplicas: 1,
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("replicaset.count", int64(3), map[string]string{})

	m.EXPECT().EmitGauge("replicaset.statuses", int64(1), map[string]string{
		"availableReplicas": strconv.Itoa(1),
		"name":              "name1",
		"namespace":         "openshift",
		"replicas":          strconv.Itoa(2),
	})

	err := mon.emitReplicasetStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
