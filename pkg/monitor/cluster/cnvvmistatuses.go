package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	k6tv1 "kubevirt.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (mon *Monitor) emitCNVVirtualMachineInstanceStatuses(ctx context.Context) error {
	var cont string
	vmiList := &k6tv1.VirtualMachineInstanceList{}

	for {
		err := mon.ocpclientset.List(ctx, vmiList, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			// If the CRD doesn't exist (CNV not installed), log and return without error
			if meta.IsNoMatchError(err) || isCRDNotFoundError(err) {
				mon.log.Debug("VirtualMachineInstance CRD not found, skipping")
				return nil
			}
			return err
		}

		for _, vmi := range vmiList.Items {
			mon.emitGauge("cnv.virtualmachineinstance.info", 1, map[string]string{
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
			})
		}

		cont = vmiList.Continue
		if cont == "" {
			break
		}
	}

	return nil
}

func isCRDNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "no kind is registered") ||
		strings.Contains(errMsg, "no matches for kind")
}
