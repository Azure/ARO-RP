package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var jobConditionsExpected = map[batchv1.JobConditionType]v1.ConditionStatus{
	batchv1.JobComplete: v1.ConditionTrue,
	batchv1.JobFailed:   v1.ConditionFalse,
}

func (mon *Monitor) emitJobConditions(ctx context.Context) error {

	jobs, err := mon.cli.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, job := range jobs.Items {
		if !namespace.IsOpenShift(job.Namespace) {
			continue
		}

		if job.Status.Active > 0 {
			// some pods are still active = job is still running, ignore
			continue
		}

		for _, cond := range job.Status.Conditions {
			if cond.Status == jobConditionsExpected[cond.Type] {
				continue
			}

			mon.emitGauge("job.conditions", 1, map[string]string{
				"name":      job.Name,
				"namespace": job.Namespace,
				"type":      string(cond.Type),
				"status":    string(cond.Status),
			})
		}

	}

	return nil
}
