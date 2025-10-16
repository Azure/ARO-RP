package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

// machinePhaseExpected defines the expected status for each machine phase
// Phase represents the current phase of machine actuation.
// One of: Failed, Provisioning, Provisioned, Running, Deleting
// Reference: https://pkg.go.dev/github.com/openshift/api@v0.0.0-20240103200955-7ca3a4634e46/machine/v1beta1#MachineStatus
var machineConditionsExpected = map[string]corev1.ConditionStatus{
	"Running":      corev1.ConditionTrue,  // Running - Machine is running (expected state)
	"Provisioned":  corev1.ConditionTrue,  // Provisioned - Machine is provisioned (transitional but acceptable)
	"Provisioning": corev1.ConditionTrue,  // Provisioning - Machine is being provisioned (transitional but acceptable)
	"Deleting":     corev1.ConditionTrue,  // Deleting - Machine is being deleted (transitional but acceptable)
	"Failed":       corev1.ConditionFalse, // Failed - Machine has failed (unexpected state)
}

// Helper function for iterating over machines in a paginated fashion
func (mon *Monitor) iterateOverMachines(ctx context.Context, onEach func(*machinev1beta1.Machine)) error {
	var cont string
	l := &machinev1beta1.MachineList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.InNamespace("openshift-machine-api"), client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			// Log error but don't stop monitoring completely
			// This allows other metrics to continue being collected
			mon.log.WithError(err).Error("failed to list machines")
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
		err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
		if err != nil {
			mon.log.WithError(err).WithField("machineName", machine.Name).Error("failed to unmarshal machine provider spec")
			// Continue without spot instance detection for this machine
		} else {
			machine.Spec.ProviderSpec.Value.Object = &spec
		}

		// Detect if this is a spot VM instance
		// Will return false if unmarshal failed (safe default)
		isSpot := isSpotInstance(*machine)

		// Get the phase from machine status for additional tracking
		phase := ""
		if machine.Status.Phase != nil {
			phase = *machine.Status.Phase
			countByPhase[phase]++
		}

		// Emit conditions (similar to node conditions logic)
		for _, c := range machine.Status.Conditions {
			if c.Status == machineConditionsExpected[string(c.Type)] {
				continue
			}

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
