package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

func (mon *Monitor) emitMachineConditions(ctx context.Context) error {
	machines := mon.getMachines(ctx)

	for _, machine := range machines {
		isSpot := isSpotInstance(machine)
		role := machine.Labels[machineRoleLabelKey]
		machineset := machine.Labels[machinesetLabelKey]

		if machine.Status.Phase != nil {

			mon.emitGauge("machine.phase", 1, map[string]string{
				"machineName":  machine.Name,
				"phase":        *machine.Status.Phase,
				"spotInstance": strconv.FormatBool(isSpot),
				"role":         role,
				"machineset":   machineset,
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":       "machine.phase",
					"machineName":  machine.Name,
					"phase":        *machine.Status.Phase,
					"spotInstance": isSpot,
					"role":         role,
					"machineset":   machineset,
				}).Print()
			}
		}
	}

	mon.emitGauge("machine.count", int64(len(machines)), nil)
	return nil
}

func (mon *Monitor) getMachines(ctx context.Context) map[string]*machinev1beta1.Machine {
	machinesMap := make(map[string]*machinev1beta1.Machine)
	var continuationToken string
	machinesList := &machinev1beta1.MachineList{}

	for {
		err := mon.ocpclientset.List(ctx, machinesList, client.InNamespace("openshift-machine-api"), client.Continue(continuationToken), client.Limit(mon.queryLimit))
		if err != nil {
			// when this call fails we may report spot vms as non spot until the next successful call
			mon.log.Error(err)
			return machinesMap
		}

		for i := range machinesList.Items {
			machine := &machinesList.Items[i]
			key := types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}.String()

			var spec machinev1beta1.AzureMachineProviderSpec
			err = json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &spec)
			if err != nil {
				mon.log.Error(err)
				continue
			}
			machine.Spec.ProviderSpec.Value.Object = &spec

			machinesMap[key] = machine
		}

		continuationToken = machinesList.Continue
		if continuationToken == "" {
			break
		}
	}

	return machinesMap
}

func isSpotInstance(m *machinev1beta1.Machine) bool {
	amps, ok := m.Spec.ProviderSpec.Value.Object.(*machinev1beta1.AzureMachineProviderSpec)
	return ok && amps.SpotVMOptions != nil
}
