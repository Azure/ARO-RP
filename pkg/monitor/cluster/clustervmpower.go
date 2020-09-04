package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/virtualmachines"
)

// emitStoppedVMPowerStatus checks to see if there are any stopped VMs in the cluster.
// If so, it emits a metric for each one and returns (true, nil) to indicate that it found
// at least one stopped VM.
func (mon *Monitor) emitStoppedVMPowerStatus(ctx context.Context) (bool, error) {
	resourceGroupName := stringutils.LastTokenByte(mon.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	stoppedVMs, err := virtualmachines.ListStopped(ctx, mon.vmClient, resourceGroupName)
	if err != nil {
		return false, err
	}
	if len(stoppedVMs) == 0 {
		return false, nil
	}
	for _, vm := range stoppedVMs {
		for _, status := range *vm.VirtualMachineProperties.InstanceView.Statuses {
			if status.Code == nil {
				continue
			}
			if virtualmachines.IsPowerStatus(*status.Code) {
				if virtualmachines.IsStopped(*status.Code) { // Check again in case it has changed
					mon.emitGauge("vmpower.conditions", 1, map[string]string{
						"id":     *vm.ID,
						"status": *status.Code,
					})
					if mon.hourlyRun {
						mon.log.WithFields(logrus.Fields{
							"metric": "vmpower.conditions",
							"id":     *vm.ID,
							"status": *status.Code,
						}).Print()
					}
				}
				break
			}
		}
	}
	return true, nil
}
