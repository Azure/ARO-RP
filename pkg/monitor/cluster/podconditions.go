package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	v1 "k8s.io/api/core/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

var podConditionsExpected = map[v1.PodConditionType]v1.ConditionStatus{
	v1.ContainersReady: v1.ConditionTrue,
	v1.PodInitialized:  v1.ConditionTrue,
	v1.PodScheduled:    v1.ConditionTrue,
	v1.PodReady:        v1.ConditionTrue,
}

func (mon *Monitor) emitPodConditions(ctx context.Context) error {
	ps := mon.cache.podList

	for _, p := range ps.Items {
		if !pgknamespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == v1.PodSucceeded {
			continue
		}

		for _, c := range p.Status.Conditions {
			if c.Status == podConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("pods.conditions", 1, map[string]string{
				"name":      p.Name,
				"namespace": p.Namespace,
				"status":    string(c.Status),
				"type":      string(c.Type),
			})
		}
	}

	return nil
}
