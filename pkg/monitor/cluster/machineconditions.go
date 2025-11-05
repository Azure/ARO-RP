package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

func (mon *Monitor) emitMachineConditions(ctx context.Context) error {
	count := 0
	countByPhase := make(map[string]int)

	err := mon.iterateOverMachines(ctx, func(machine *machinev1beta1.Machine) {
		var spec machinev1beta1.AzureMachineProviderSpec
		hasMachine := true
		err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.Error(err)
			hasMachine = false
		} else {
			machine.Spec.ProviderSpec.Value.Object = &spec
		}

		// Only check for spot if we successfully unmarshaled
		isSpot := hasMachine && isSpotInstance(*machine)
		role := machine.Labels[machineRoleLabelKey]
		machineset := machine.Labels[machinesetLabelKey]

		// Get the phase from machine status for additional tracking
		phase := ""
		if machine.Status.Phase != nil {
			phase = *machine.Status.Phase
			countByPhase[phase]++
		}

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

	mon.emitGauge("machine.count", int64(count), nil)

	// Emit count by phase for visibility
	for phase, phaseCount := range countByPhase {
		mon.emitGauge("machine.count.phase", int64(phaseCount), map[string]string{
			"phase": phase,
		})
	}

	return nil
}

// Helper functions
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

func (mon *Monitor) getMachines(ctx context.Context) map[string]*machinev1beta1.Machine {
	machinesMap := make(map[string]*machinev1beta1.Machine)

	// Reuse the iterator instead of duplicating pagination logic
	err := mon.iterateOverMachines(ctx, func(machine *machinev1beta1.Machine) {
		key := types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}.String()

		var spec machinev1beta1.AzureMachineProviderSpec
		err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.Error(err)
			return // Skip this machine (don't add to map)
		}
		machine.Spec.ProviderSpec.Value.Object = &spec
		machinesMap[key] = machine
	})

	if err != nil {
		// when this call fails we may report spot vms as non spot until the next successful call
		mon.log.Error(err)
	}

	return machinesMap
}

func isSpotInstance(m machinev1beta1.Machine) bool {
	amps, ok := m.Spec.ProviderSpec.Value.Object.(*machinev1beta1.AzureMachineProviderSpec)
	return ok && amps.SpotVMOptions != nil
}
