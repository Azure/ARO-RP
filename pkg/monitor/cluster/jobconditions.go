package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var jobConditionsExpected = map[batchv1.JobConditionType]corev1.ConditionStatus{
	batchv1.JobComplete: corev1.ConditionTrue,
	batchv1.JobFailed:   corev1.ConditionFalse,
}

func (mon *Monitor) emitJobConditions(ctx context.Context) error {
	var cont string
	var count int64
	for {
		jobs, err := mon.cli.BatchV1().Jobs("").List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		count += int64(len(jobs.Items))

		for _, job := range jobs.Items {
			if !namespace.IsOpenShiftNamespace(job.Namespace) {
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

		cont = jobs.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("job.count", count, nil)

	return nil
}
