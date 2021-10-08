package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
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
			if c.Status == nodeConditionsExpected[c.Type] || mon.isSpotInstance(ctx, n) {
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

// isSpotInstance checks if a node represents a spot VM. If so, don't monitor it (creates noise)
func (mon *Monitor) isSpotInstance(ctx context.Context, n corev1.Node) bool {
	machineName, ok := n.Annotations["machine.openshift.io/machine"]
	if ok {
		machine, err := mon.maocli.MachineV1beta1().Machines("openshift-machine-api").Get(ctx, machineName, metav1.GetOptions{})
		if err != nil {
			mon.log.Error(err)
			return false
		}

		var spec azureproviderv1beta1.AzureMachineProviderSpec
		err = json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.Error(err)
			return false
		}

		return spec.SpotVMOptions != nil
	}
	mon.log.Error("node missing annotation 'machine.openshift.io/machine'")
	return false
}
