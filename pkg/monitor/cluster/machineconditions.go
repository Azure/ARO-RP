package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

// Helper function for iterating over machines in a paginated fashion
func (mon *Monitor) iterateOverMachines(ctx context.Context, onEach func(*machinev1beta1.Machine)) error {
	var cont string
	l := &machinev1beta1.MachineList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.InNamespace("openshift-machine-api"), client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error in Machine list operation: %w", err)
		}

		for _, machineObj := range l.Items {
			onEach(&machineObj)
		}

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return nil
}

func (mon *Monitor) emitMachineConditions(ctx context.Context) error {
	count := 0
	countByPhase := make(map[string]int)
	machines := mon.getMachines(ctx)

	err := mon.iterateOverMachines(ctx, func(machineObj *machinev1beta1.Machine) {
		machineKey := types.NamespacedName{Namespace: machineObj.Namespace, Name: machineObj.Name}.String()
		machine, hasMachine := machines[machineKey]
		isSpot := hasMachine && isSpotInstance(*machine)

		role := ""
		if hasMachine {
			role = machine.Labels[machineRoleLabelKey]
		}

		machineset := ""
		if hasMachine {
			machineset = machine.Labels[machinesetLabelKey]
		}

		// Get the phase from machine status for additional tracking
		phase := ""
		if machineObj.Status.Phase != nil {
			phase = *machineObj.Status.Phase
			countByPhase[phase]++
		}

		// Emit conditions for all machine conditions
		for _, c := range machineObj.Status.Conditions {
			mon.emitGauge("machine.conditions", 1, map[string]string{
				"machineName":  machineObj.Name,
				"status":       string(c.Status),
				"type":         string(c.Type),
				"spotInstance": strconv.FormatBool(isSpot),
				"role":         role,
				"machineset":   machineset,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "machine.conditions",
					"machineName":  machineObj.Name,
					"status":       c.Status,
					"type":         c.Type,
					"message":      c.Message,
					"spotInstance": isSpot,
					"role":         role,
					"machineset":   machineset,
				}).Print()
			}
		}

		count += 1
	})
	if err != nil {
		return err
	}

	// Emit total machine count
	mon.emitGauge("machine.count", int64(count), nil)

	// Emit count by phase for visibility
	for phase, phaseCount := range countByPhase {
		mon.emitGauge("machine.count.phase", int64(phaseCount), map[string]string{
			"phase": phase,
		})
	}

	return nil
}
