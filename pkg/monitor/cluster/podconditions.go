package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var podConditionsExpected = map[v1.PodConditionType]v1.ConditionStatus{
	v1.ContainersReady: v1.ConditionTrue,
	v1.PodInitialized:  v1.ConditionTrue,
	v1.PodScheduled:    v1.ConditionTrue,
	v1.PodReady:        v1.ConditionTrue,
}

func (mon *Monitor) emitPodConditions(ctx context.Context) error {
	// to list pods once
	ps, err := mon.cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	mon._emitPodConditions(ps)
	mon._emitPodContainerStatuses(ps)

	return nil
}

func (mon *Monitor) _emitPodConditions(ps *v1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == v1.PodSucceeded {
			continue
		}

		for _, c := range p.Status.Conditions {
			if c.Status == podConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("pod.conditions", 1, map[string]string{
				"name":      p.Name,
				"namespace": p.Namespace,
				"status":    string(c.Status),
				"type":      string(c.Type),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":    "pod.conditions",
					"name":      p.Name,
					"namespace": p.Namespace,
					"status":    c.Status,
					"type":      c.Type,
					"message":   c.Message,
				}).Print()
			}
		}
	}
}

func (mon *Monitor) _emitPodContainerStatuses(ps *v1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == v1.PodSucceeded {
			continue
		}

		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting == nil {
				continue
			}

			mon.emitGauge("pod.containerstatuses", 1, map[string]string{
				"name":          p.Name,
				"namespace":     p.Namespace,
				"containername": cs.Name,
				"reason":        cs.State.Waiting.Reason,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":        "pod.containerstatuses",
					"name":          p.Name,
					"namespace":     p.Namespace,
					"containername": cs.Name,
					"reason":        cs.State.Waiting.Reason,
					"message":       cs.State.Waiting.Message,
				}).Print()
			}
		}
	}
}
