package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
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
	l := &mcv1.MachineConfigPoolList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return err
		}

		count += int64(len(l.Items))

		for _, mcp := range l.Items {
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

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	mon.emitGauge("machineconfigpool.count", count, nil)

	return nil
}
