package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
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
	nodes, err := mon.listNodes(ctx)
	if err != nil {
		return err
	}
	machines := mon.getMachines(ctx)

	mon.emitGauge("node.count", int64(len(nodes.Items)), nil)

	for _, n := range nodes.Items {
		machineNamespacedName := n.Annotations[machineAnnotationKey]
		machine, hasMachine := machines[machineNamespacedName]
		isSpotInstance := hasMachine && isSpotInstance(*machine)

		role := ""
		if hasMachine {
			role = machine.Labels[machineRoleLabelKey]
		}

		machineset := ""
		if hasMachine {
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
	}

	return nil
}

func (mon *Monitor) getMachines(ctx context.Context) map[string]*machinev1beta1.Machine {
	machinesMap := make(map[string]*machinev1beta1.Machine)
	machines, err := mon.maocli.MachineV1beta1().Machines("openshift-machine-api").List(ctx, metav1.ListOptions{})

	if err != nil {
		// when this call fails we may report spot vms as non spot until the next successful call
		mon.log.Error(err)
		return machinesMap
	}

	for _, machine := range machines.Items {
		key := types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}.String()

		var spec machinev1beta1.AzureMachineProviderSpec
		err = json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.Error(err)
			continue
		}
		machine.Spec.ProviderSpec.Value.Object = &spec

		machinesMap[key] = &machine
	}

	return machinesMap
}

func isSpotInstance(m machinev1beta1.Machine) bool {
	amps, ok := m.Spec.ProviderSpec.Value.Object.(*machinev1beta1.AzureMachineProviderSpec)
	return ok && amps.SpotVMOptions != nil
}
