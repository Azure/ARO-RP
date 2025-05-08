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
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitJobConditions(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{ // will generate no metric
				Name:      "job-running",
				Namespace: "openshift",
			},
			Status: batchv1.JobStatus{
				Active: 1, //1 pod active -> job is running
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
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("job.count", int64(2), map[string]string{})

	m.EXPECT().EmitGauge("job.conditions", int64(1), map[string]string{
		"name":      "job-failing",
		"namespace": "openshift",
		"status":    "True",
		"type":      "Failed",
	})

	ctx := context.Background()

	err := mon.emitJobConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
