package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitNodesMetrics() error {
	nodes, err := mon.cli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	mon.emitGauge("nodes.count", int64(len(nodes.Items)), nil)

	counters := map[string]int64{}
	for _, node := range nodes.Items {
		for _, c := range node.Status.Conditions {
			// count 'Unknown' status as unhealthy state for each condition. In this way
			// we can flag issues without creating additional timeseries for each condition.
			// for NodeReady count a node when the status is False (not ready) or Unknown
			// for other conditions count when the status is True or Unknown
			if c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue {
				counters["NotReady"]++
			} else if c.Type != corev1.NodeReady && c.Status != corev1.ConditionFalse {
				counters[string(c.Type)]++
			}
		}
	}

	for condition, count := range counters {
		mon.emitGauge("nodes.conditions.count", count, map[string]string{
			"condition": condition,
		})
	}

	return nil
}
