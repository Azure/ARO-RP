package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitAroOperatorHeartbeat(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{

		&appsv1.Deployment{ // not available expected
			ObjectMeta: metav1.ObjectMeta{
				Name:       "aro-operator-master",
				Namespace:  "openshift-azure-operator",
				Generation: 4,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointerutils.ToPtr(int32(1)),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				AvailableReplicas:   0,
				UnavailableReplicas: 1,
				UpdatedReplicas:     0,
				ObservedGeneration:  4,
			},
		}, &appsv1.Deployment{ // available expected
			ObjectMeta: metav1.ObjectMeta{
				Name:       "aro-operator-worker",
				Namespace:  "openshift-azure-operator",
				Generation: 4,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointerutils.ToPtr(int32(1)),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
				UpdatedReplicas:     1,
				ObservedGeneration:  4,
			},
		}, &appsv1.Deployment{ // no metric expected - different name
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name3",
				Namespace: "openshift-azure-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				AvailableReplicas:   2,
				UnavailableReplicas: 0,
			},
		}, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
				Name:      "name4",
				Namespace: "customer",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            2,
				AvailableReplicas:   1,
				UnavailableReplicas: 1,
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
		log:          log,
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   1,
	}

	m.EXPECT().EmitGauge("arooperator.heartbeat", int64(0), map[string]string{
		"name": "aro-operator-master",
	})

	m.EXPECT().EmitGauge("arooperator.heartbeat", int64(1), map[string]string{
		"name": "aro-operator-worker",
	})

	err := mon.emitAroOperatorHeartbeat(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
