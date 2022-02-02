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

	spotInstances := mon.getSpotInstances(ctx)

	mon.emitGauge("node.count", int64(len(ns.Items)), nil)

	for _, n := range ns.Items {

		for _, c := range n.Status.Conditions {
			if c.Status == nodeConditionsExpected[c.Type] {
				continue
			}

			_, isSpotInstance := spotInstances[n.Name]

			mon.emitGauge("node.conditions", 1, map[string]string{
				"nodeName":     n.Name,
				"status":       string(c.Status),
				"type":         string(c.Type),
				"spotInstance": strconv.FormatBool(isSpotInstance),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "node.conditions",
					"name":         n.Name,
					"status":       c.Status,
					"type":         c.Type,
					"message":      c.Message,
					"spotInstance": isSpotInstance,
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

// getSpotInstances returns a map where the keys are the machine name and only exist if the machine is a spot instance
func (mon *Monitor) getSpotInstances(ctx context.Context) map[string]struct{} {
	spotInstances := make(map[string]struct{})
	machines, err := mon.maocli.MachineV1beta1().Machines("openshift-machine-api").List(ctx, metav1.ListOptions{})

	if err != nil {
		// when this call fails we may report spot vms as non spot until the next successful call
		mon.log.Error(err)
		return spotInstances
	}

	for _, machine := range machines.Items {
		var spec azureproviderv1beta1.AzureMachineProviderSpec
		err = json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.Error(err)
			continue
		}

		if spec.SpotVMOptions != nil {
			spotInstances[machine.Name] = struct{}{}
		}
	}

	return spotInstances
}
