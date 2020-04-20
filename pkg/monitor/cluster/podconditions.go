package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pgknamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

var podConditionsExpected = map[v1.PodConditionType]v1.ConditionStatus{
	v1.ContainersReady: v1.ConditionTrue,
	v1.PodInitialized:  v1.ConditionTrue,
	v1.PodScheduled:    v1.ConditionTrue,
	v1.PodReady:        v1.ConditionTrue,
}

func (mon *Monitor) emitPodAllConditions(ctx context.Context) error {
	// to list pods once
	ps, err := mon.cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	err = mon._emitPodConditions(ps)
	if err != nil {
		return err
	}

	return mon._emitPodContainersConditions(ps)
}

func (mon *Monitor) _emitPodConditions(ps *v1.PodList) error {
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

func (mon *Monitor) _emitPodContainersConditions(ps *v1.PodList) error {
	sort.SliceStable(ps.Items, func(i, j int) bool { return ps.Items[i].Name < ps.Items[j].Name })
	sort.SliceStable(ps.Items, func(i, j int) bool { return ps.Items[i].Namespace < ps.Items[j].Namespace })
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
