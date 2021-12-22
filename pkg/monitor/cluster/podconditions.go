package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var podConditionsExpected = map[corev1.PodConditionType]corev1.ConditionStatus{
	corev1.ContainersReady: corev1.ConditionTrue,
	corev1.PodInitialized:  corev1.ConditionTrue,
	corev1.PodScheduled:    corev1.ConditionTrue,
	corev1.PodReady:        corev1.ConditionTrue,
}

var restartCounterThreshold int32 = 10

func (mon *Monitor) emitPodConditions(ctx context.Context) error {
	// to list pods once
	var cont string
	var count int64
	for {
		ps, err := mon.cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		count += int64(len(ps.Items))

		mon._emitPodConditions(ps)
		mon._emitPodContainerStatuses(ps)
		mon._emitPodContainerRestartCounter(ps)

		cont = ps.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("pod.count", count, nil)

	return nil
}

func (mon *Monitor) _emitPodConditions(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, c := range p.Status.Conditions {
			if c.Status == podConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("pod.conditions", 1, map[string]string{
				"name":      p.Name,
				"namespace": p.Namespace,
				"nodeName":  p.Spec.NodeName,
				"status":    string(c.Status),
				"type":      string(c.Type),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":    "pod.conditions",
					"name":      p.Name,
					"namespace": p.Namespace,
					"nodeName":  p.Spec.NodeName,
					"status":    c.Status,
					"type":      c.Type,
					"message":   c.Message,
				}).Print()
			}
		}
	}
}

func (mon *Monitor) _emitPodContainerStatuses(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShift(p.Namespace) {
			continue
		}

		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting == nil {
				continue
			}

			mon.emitGauge("pod.containerstatuses", 1, map[string]string{
				"name":          p.Name,
				"namespace":     p.Namespace,
				"nodeName":      p.Spec.NodeName,
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

func (mon *Monitor) _emitPodContainerRestartCounter(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShiftSystemNamespace(p.Namespace) {
			continue
		}

		//Sum up the total number of restarts in the pod to match the number of restarts shown in the 'oc get pods' display
		t := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			t += cs.RestartCount
		}

		if t < restartCounterThreshold {
			continue
		}

		mon.emitGauge("pod.restartcounter", int64(t), map[string]string{
			"name":      p.Name,
			"namespace": p.Namespace,
		})

		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"metric":    "pod.restartcounter",
				"name":      p.Name,
				"namespace": p.Namespace,
			}).Print()
		}
	}
}
