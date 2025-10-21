package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

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

		for _, machine := range l.Items {
			onEach(&machine)
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

	err := mon.iterateOverMachines(ctx, func(machine *machinev1beta1.Machine) {
		// Get the role from machine labels
		role := machine.Labels[machineRoleLabelKey]

		// Get the machineset from machine labels
		machineset := machine.Labels[machinesetLabelKey]

		// Unmarshal the provider spec to properly detect spot instances
		var spec machinev1beta1.AzureMachineProviderSpec
		if machine.Spec.ProviderSpec.Value != nil && machine.Spec.ProviderSpec.Value.Raw != nil {
			err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
			if err != nil {
				mon.log.WithError(err).WithField("machineName", machine.Name).Error("failed to unmarshal machine provider spec")
			} else {
				machine.Spec.ProviderSpec.Value.Object = &spec
			}
		}

		// Detect if this is a spot VM instance
		isSpot := isSpotInstance(*machine)

		// Get the phase from machine status for additional tracking
		phase := ""
		if machine.Status.Phase != nil {
			phase = *machine.Status.Phase
			countByPhase[phase]++
		}

		// Emit conditions for all machine conditions
		for _, c := range machine.Status.Conditions {
			mon.emitGauge("machine.conditions", 1, map[string]string{
				"machineName":  machine.Name,
				"status":       string(c.Status),
				"type":         string(c.Type),
				"spotInstance": strconv.FormatBool(isSpot),
				"role":         role,
				"machineset":   machineset,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "machine.conditions",
					"machineName":  machine.Name,
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
