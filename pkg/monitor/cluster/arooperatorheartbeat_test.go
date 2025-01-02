package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitAroOperatorHeartbeat(t *testing.T) {
	ctx := context.Background()

	cli := fake.NewSimpleClientset(
		&appsv1.Deployment{ // not available expected
			ObjectMeta: metav1.ObjectMeta{
				Name:       "aro-operator-master",
				Namespace:  "openshift-azure-operator",
				Generation: 4,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: to.Int32Ptr(1),
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
				Replicas: to.Int32Ptr(1),
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
	)

	m := testmonitor.NewFakeEmitter(t)
	mon := &Monitor{
		cli: cli,
		m:   m,
		log: utillog.GetLogger(),
	}

	err := mon.emitAroOperatorHeartbeat(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.VerifyEmittedMetrics(
		testmonitor.Metric("arooperator.heartbeat", int64(0), map[string]string{
			"name": "aro-operator-master",
		}),
		testmonitor.Metric("arooperator.heartbeat", int64(1), map[string]string{
			"name": "aro-operator-worker",
		}),
	)
}
