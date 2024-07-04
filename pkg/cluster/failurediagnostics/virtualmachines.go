package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) LogAzureInformation(ctx context.Context) (interface{}, error) {
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
		serialConsoleBlob, err := m.virtualMachines.GetSerialConsoleForVM(ctx, resourceGroupName, vmName)
		if err != nil {
			items = append(items, fmt.Sprintf("vm boot diagnostics retrieval error for %s: %s", vmName, err))
			continue
		}

		logForVM := m.log.WithField("failedRoleInstance", vmName)

		b64Reader := base64.NewDecoder(base64.StdEncoding, serialConsoleBlob)
		scanner := bufio.NewScanner(b64Reader)
		for scanner.Scan() {
			logForVM.Info(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			items = append(items, fmt.Sprintf("blob storage scan on %s: %s", vmName, err))
		}
	}

	return items, nil
}
