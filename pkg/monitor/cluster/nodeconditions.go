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
	workerInfraCount := 0
	machines := mon.getMachines(ctx)

	err := mon.iterateOverNodes(ctx, func(n *corev1.Node) {
		machineNamespacedName := n.Annotations[machineAnnotationKey]
		machine, hasMachine := machines[machineNamespacedName]
		isSpotInstance := hasMachine && isSpotInstance(machine)

		role := ""
		if hasMachine {
			role = machine.Labels[machineRoleLabelKey]
		}

		machineset := ""
		if hasMachine {
			machineset = machine.Labels[machinesetLabelKey]
var role, machineset string                                                
  if hasMachine {                                                            
      role = machine.Labels[machineRoleLabelKey]                             
      machineset = machine.Labels[machinesetLabelKey]                        
  }   

		for _, c := range n.Status.Conditions {
			if c.Status == nodeConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("node.conditions", 1, map[string]string{
				"nodeName":     n.Name,
				"status":       string(c.Status),
				"type":         string(c.Type),
				"spotInstance": strconv.FormatBool(isSpotInstance),
				"role":         role,
				"machineset":   machineset,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "node.conditions",
					"name":         n.Name,
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
			"nodeName":       n.Name,
			"role":           role,
			"kubeletVersion": n.Status.NodeInfo.KubeletVersion,
		})

		if _, ok := n.Labels[masterRoleLabel]; ok {
			masterCount++
		}
		_, isWorker := n.Labels[workerRoleLabel]                                   
        _, isInfra := n.Labels[infraRoleLabel]                                     
        if isWorker || isInfra {                                                   
           workerinfraCount++                                                     
        }  
	})
	if err != nil {
		return err
	}

	mon.emitGauge("node.count", int64(masterCount), map[string]string{"role": "master"})
	mon.emitGauge("node.count", int64(workerinfraCount), map[string]string{"role": "worker"})
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

		for _, n := range l.Items {
			onEach(&n)
		}

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return nil
}
