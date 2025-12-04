package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	k6tv1 "kubevirt.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (mon *Monitor) emitCNVVirtualMachineInstanceStatuses(ctx context.Context) error {
	continueToken := ""
	vmiList := &k6tv1.VirtualMachineInstanceList{}

	for {
		err := mon.ocpclientset.List(ctx, vmiList, client.Continue(continueToken), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error listing virtual machine instances: %w", err)
		}

		for _, vmi := range vmiList.Items {
			mon.emitVMIMetrics(vmi)
		}

		continueToken = vmiList.Continue
		if continueToken == "" {
			break
		}
	}

	return nil
}

func (mon *Monitor) emitVMIMetrics(vmi k6tv1.VirtualMachineInstance) {
	labels := mon.buildVMILabels(vmi)
	mon.emitGauge("cnv.virtualmachineinstance.info", 1, labels)
}

func (mon *Monitor) buildVMILabels(vmi k6tv1.VirtualMachineInstance) map[string]string {
	return map[string]string{
		"namespace":               vmi.Namespace,
		"name":                    vmi.Name,
		"phase":                   string(vmi.Status.Phase),
		"os":                      vmi.Labels["os"],
		"workload":                vmi.Labels["workload"],
		"flavor":                  vmi.Labels["flavor"],
		"guest_os_kernel_release": vmi.Status.GuestOSInfo.KernelRelease,
		"guest_os_arch":           vmi.Status.GuestOSInfo.Machine,
		"guest_os_name":           vmi.Status.GuestOSInfo.Name,
		"guest_os_version_id":     vmi.Status.GuestOSInfo.VersionID,
	}
}
