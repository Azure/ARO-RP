package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	v1 "k8s.io/api/core/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) emitPodContainersConditions(ctx context.Context) error {
	ps := mon.cache.podList

	for _, p := range ps.Items {
		if !pgknamespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == v1.PodSucceeded {
			continue
		}

		sort.Slice(p.Status.ContainerStatuses, func(i, j int) bool {
			return p.Status.ContainerStatuses[i].Name < p.Status.ContainerStatuses[j].Name
		})
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting == nil {
				continue
			}

			mon.emitGauge("pods.containers.conditions", 1, map[string]string{
				"name":      p.Name,
				"namespace": p.Namespace,
				"reason":    cs.State.Waiting.Reason,
			})
		}
	}

	return nil
}
