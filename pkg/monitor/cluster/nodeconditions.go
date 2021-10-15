package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

var nodeConditionsExpected = map[corev1.NodeConditionType]corev1.ConditionStatus{
	corev1.NodeDiskPressure:   corev1.ConditionFalse,
	corev1.NodeMemoryPressure: corev1.ConditionFalse,
	corev1.NodePIDPressure:    corev1.ConditionFalse,
	corev1.NodeReady:          corev1.ConditionTrue,
}

func (mon *Monitor) emitNodeConditions(ctx context.Context) error {
	ns, err := mon.listNodes(ctx)
	if err != nil {
		return err
	}

	mon.emitGauge("node.count", int64(len(ns.Items)), nil)

	for _, n := range ns.Items {
		for _, c := range n.Status.Conditions {
			if c.Status == nodeConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("node.conditions", 1, map[string]string{
				"nodeName": n.Name,
				"status":   string(c.Status),
				"type":     string(c.Type),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":  "node.conditions",
					"name":    n.Name,
					"status":  c.Status,
					"type":    c.Type,
					"message": c.Message,
				}).Print()
			}
		}

		mon.emitGauge("node.kubelet.version", 1, map[string]string{
			"nodeName":       n.Name,
			"kubeletVersion": n.Status.NodeInfo.KubeletVersion,
		})

	}

	return nil
}
