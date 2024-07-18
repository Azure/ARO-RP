package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogVMSerialConsole fetches the serial console from VMs and logs them with
// the associated VM name.
func (m *manager) LogVMSerialConsole(ctx context.Context) (interface{}, error) {
	items := make([]interface{}, 0)

	if m.virtualMachines == nil {
		items = append(items, "vmclient missing")
		return items, nil
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vms, err := m.virtualMachines.List(ctx, resourceGroupName)
	if err != nil {
		items = append(items, fmt.Sprintf("vm listing error: %s", err))
		return items, nil
	}

	if len(vms) == 0 {
		items = append(items, "no VMs found")
		return items, nil
	}

	vmNames := make([]string, 0)
	for _, v := range vms {
		j, err := v.MarshalJSON()
		if err != nil {
			items = append(items, fmt.Sprintf("vm marshalling error: %s", err))
		} else {
			vmName := "<unknown>"
			if v.Name != nil {
				vmName = *v.Name
				vmNames = append(vmNames, vmName)
			}
			items = append(items, fmt.Sprintf("vm %s: %s", vmName, string(j)))
		}
	}

	// Fetch boot diagnostics URIs for the VMs
	for _, vmName := range vmNames {
		blob := &bytes.Buffer{}
		err := m.virtualMachines.GetSerialConsoleForVM(ctx, resourceGroupName, vmName, blob)
		if err != nil {
			items = append(items, fmt.Sprintf("vm boot diagnostics retrieval error for %s: %s", vmName, err))
			continue
		}

		logForVM := m.log.WithField("failedRoleInstance", vmName)
		scanner := bufio.NewScanner(blob)
		for scanner.Scan() {
			logForVM.Info(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			items = append(items, fmt.Sprintf("blob storage scan on %s: %s", vmName, err))
		}
	}

	return items, nil
}
