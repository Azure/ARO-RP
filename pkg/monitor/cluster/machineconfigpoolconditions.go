package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mcv1 "github.com/openshift/api/machineconfiguration/v1"
)

var machineConfigPoolConditionsExpected = map[mcv1.MachineConfigPoolConditionType]corev1.ConditionStatus{
	mcv1.MachineConfigPoolDegraded:       corev1.ConditionFalse,
	mcv1.MachineConfigPoolNodeDegraded:   corev1.ConditionFalse,
	mcv1.MachineConfigPoolRenderDegraded: corev1.ConditionFalse,
	mcv1.MachineConfigPoolUpdated:        corev1.ConditionTrue,
	mcv1.MachineConfigPoolUpdating:       corev1.ConditionFalse,
}

func (mon *Monitor) emitMachineConfigPoolConditions(ctx context.Context) error {
	var cont string
	var count int64
	for {
		mcps, err := mon.mcocli.MachineconfigurationV1().MachineConfigPools().List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		count += int64(len(mcps.Items))

		for _, mcp := range mcps.Items {
			for _, c := range mcp.Status.Conditions {
				if c.Status == machineConfigPoolConditionsExpected[c.Type] {
					continue
				}

				mon.emitGauge("machineconfigpool.conditions", 1, map[string]string{
					"name":   mcp.Name,
					"status": string(c.Status),
					"type":   string(c.Type),
				})

				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"metric":  "machineconfigpool.conditions",
						"name":    mcp.Name,
						"status":  c.Status,
						"type":    c.Type,
						"message": c.Message,
					}).Print()
				}
			}
		}

		cont = mcps.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("machineconfigpool.count", count, nil)

	return nil
}
