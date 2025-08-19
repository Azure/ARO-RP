package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var jobConditionsExpected = map[batchv1.JobConditionType]corev1.ConditionStatus{
	batchv1.JobComplete: corev1.ConditionTrue,
	batchv1.JobFailed:   corev1.ConditionFalse,
}

func (mon *Monitor) emitJobConditions(ctx context.Context) error {
	// Only fetch in the namespaces we manage
	for _, ns := range mon.namespacesToMonitor {
		var cont string
		l := &batchv1.JobList{}

		for {
			err := mon.ocpclientset.List(ctx, l, client.InNamespace(ns), client.Continue(cont), client.Limit(mon.queryLimit))
			if err != nil {
				return fmt.Errorf("error in list operation: %w", err)
			}

			for _, job := range l.Items {
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

			cont = l.Continue
			if cont == "" {
				break
			}
		}
	}

	return nil
}
