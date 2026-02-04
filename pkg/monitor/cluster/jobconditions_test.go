package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitJobConditions(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		namespaceObject("openshift"),
		namespaceObject("customer"),
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{ // will generate no metric
				Name:      "job-running",
				Namespace: "openshift",
			},
			Status: batchv1.JobStatus{
				Active: 1, // 1 pod active -> job is running
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		&batchv1.Job{ // one metric only expected, the job failure
			ObjectMeta: metav1.ObjectMeta{
				Name:      "job-failing",
				Namespace: "openshift",
			},
			Status: batchv1.JobStatus{
				Active: 0,
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		&batchv1.Job{ // no metric expected, customer namespace
			ObjectMeta: metav1.ObjectMeta{
				Name:      "job-failing",
				Namespace: "customer",
			},
			Status: batchv1.JobStatus{
				Active: 0,
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionTrue,
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
		queryLimit:   1,
	}

	err := mon.fetchManagedNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.EXPECT().EmitGauge("job.conditions", int64(1), map[string]string{
		"name":      "job-failing",
		"namespace": "openshift",
		"status":    "True",
		"type":      "Failed",
	})

	err = mon.emitJobConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
