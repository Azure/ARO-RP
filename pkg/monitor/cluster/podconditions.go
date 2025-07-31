package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/events"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var podConditionsExpected = map[corev1.PodConditionType]corev1.ConditionStatus{
	corev1.ContainersReady: corev1.ConditionTrue,
	corev1.PodInitialized:  corev1.ConditionTrue,
	corev1.PodScheduled:    corev1.ConditionTrue,
	corev1.PodReady:        corev1.ConditionTrue,
}

var restartCounterThreshold int32 = 10

func (mon *Monitor) emitPodConditions(ctx context.Context) error {
	// Only fetch in the namespaces we manage
	for _, ns := range mon.namespacesToMonitor {
		var cont string
		ps := &corev1.PodList{}

		for {
			err := mon.ocpclientset.List(ctx, ps, client.InNamespace(ns), client.Continue(cont), client.Limit(mon.queryLimit))
			if err != nil {
				return fmt.Errorf("error in list operation: %w", err)
			}

			mon._emitPodConditions(ps)
			mon._emitPodContainerStatuses(ps)
			mon._emitPodContainerRestartCounter(ps)

			cont = ps.Continue
			if cont == "" {
				break
			}
		}
	}

	return nil
}

func (mon *Monitor) _emitPodConditions(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		if p.Status.Reason == events.PreemptContainer {
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
		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting == nil {
				continue
			}

			containerStatus := map[string]string{
				"name":                 p.Name,
				"namespace":            p.Namespace,
				"nodeName":             p.Spec.NodeName,
				"containername":        cs.Name,
				"reason":               cs.State.Waiting.Reason,
				"lastTerminationState": "",
			}

			if cs.LastTerminationState.Terminated != nil {
				containerStatus["lastTerminationState"] = cs.LastTerminationState.Terminated.Reason
			}

			mon.emitGauge("pod.containerstatuses", 1, containerStatus)

			if mon.hourlyRun {
				logFields := logrus.Fields{
					"metric":  "pod.containerstatuses",
					"message": cs.State.Waiting.Message,
				}
				for label, labelVal := range containerStatus {
					logFields[label] = labelVal
				}
				mon.log.WithFields(logFields).Print()
			}
		}
	}
}

func (mon *Monitor) _emitPodContainerRestartCounter(ps *corev1.PodList) {
	for _, p := range ps.Items {
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
