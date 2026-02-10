package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	machineAnnotationKey = "machine.openshift.io/machine"
	machineRoleLabelKey  = "machine.openshift.io/cluster-api-machine-role"
	machinesetLabelKey   = "machine.openshift.io/cluster-api-machineset"
)

var nodeConditionsExpected = map[corev1.NodeConditionType]corev1.ConditionStatus{
	corev1.NodeDiskPressure:   corev1.ConditionFalse,
	corev1.NodeMemoryPressure: corev1.ConditionFalse,
	corev1.NodePIDPressure:    corev1.ConditionFalse,
	corev1.NodeReady:          corev1.ConditionTrue,
}

func (mon *Monitor) emitNodeConditions(ctx context.Context) error {
	masterCount := 0
	workerCount := 0
	infraCount := 0
	totalCount := 0
	unknownCount := 0
	machines := mon.getMachines(ctx)

	err := mon.iterateOverNodes(ctx, func(node *corev1.Node) {
		machineNamespacedName := node.Annotations[machineAnnotationKey]
		machine, hasMachine := machines[machineNamespacedName]
		isSpotInstance := hasMachine && isSpotInstance(machine)

		var role, machineset string
		if hasMachine {
			role = machine.Labels[machineRoleLabelKey]
			machineset = machine.Labels[machinesetLabelKey]
		}

		for _, c := range node.Status.Conditions {
			if c.Status == nodeConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("node.conditions", 1, map[string]string{
				"nodeName":     node.Name,
				"status":       string(c.Status),
				"type":         string(c.Type),
				"spotInstance": strconv.FormatBool(isSpotInstance),
				"role":         role,
				"machineset":   machineset,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "node.conditions",
					"name":         node.Name,
					"status":       c.Status,
					"type":         c.Type,
					"message":      c.Message,
					"spotInstance": isSpotInstance,
					"role":         role,
					"machineset":   machineset,
				}).Print()
			}
		}

		mon.emitGauge("node.kubelet.version", 1, map[string]string{
			"nodeName":       node.Name,
			"role":           role,
			"kubeletVersion": node.Status.NodeInfo.KubeletVersion,
		})

		_, isMaster := node.Labels[masterRoleLabel]
		_, isWorker := node.Labels[workerRoleLabel]
		_, isInfra := node.Labels[infraRoleLabel]

		if isMaster {
			masterCount++
		} else if isWorker {
			workerCount++
		} else if isInfra {
			infraCount++
		}

		if !isMaster && !isWorker && !isInfra {
			unknownCount++
			mon.log.WithFields(logrus.Fields{
				"nodeName": node.Name,
			}).Warning("Node has no role labels")
		}
		totalCount++
	})
	if err != nil {
		return err
	}

	if unknownCount > 0 {
		mon.emitGauge("node.count", int64(unknownCount), map[string]string{"role": "unknown"})
	}

	mon.emitGauge("node.count", int64(masterCount), map[string]string{"role": "master"})
	mon.emitGauge("node.count", int64(workerCount), map[string]string{"role": "worker"})
	mon.emitGauge("node.count", int64(infraCount), map[string]string{"role": "infra"})
	mon.emitGauge("node.count", int64(totalCount), map[string]string{"role": "all"})

	return nil
}

// Helper function for iterating over nodes in a paginated fashion
func (mon *Monitor) iterateOverNodes(ctx context.Context, onEach func(*corev1.Node)) error {
	var cont string
	l := &corev1.NodeList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error in Node list operation: %w", err)
		}

		for _, node := range l.Items {
			onEach(&node)
		}

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return nil
}
