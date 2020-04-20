package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var nodeConditionsExpected = map[v1.NodeConditionType]v1.ConditionStatus{
	v1.NodeDiskPressure:   v1.ConditionFalse,
	v1.NodeMemoryPressure: v1.ConditionFalse,
	v1.NodePIDPressure:    v1.ConditionFalse,
	v1.NodeReady:          v1.ConditionTrue,
}

func (mon *Monitor) emitNodeConditions(ctx context.Context) error {
	ns, err := mon.cli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	mon.emitGauge("nodes.count", int64(len(ns.Items)), nil)

	for _, n := range ns.Items {
		for _, c := range n.Status.Conditions {
			if c.Status == nodeConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("node.conditions", 1, map[string]string{
				"name":   n.Name,
				"status": string(c.Status),
				"type":   string(c.Type),
			})

			if mon.logMessages {
				mon.log.WithFields(logrus.Fields{
					"metric":  "node.conditions",
					"name":    n.Name,
					"status":  c.Status,
					"type":    c.Type,
					"message": c.Message,
				}).Print()
			}
		}
	}

	return nil
}
