package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogVMSerialConsole fetches the serial console from VMs and logs them with
// the associated VM name.
func (m *manager) LogVMSerialConsole(ctx context.Context) (interface{}, error) {
	return m.logVMSerialConsole(ctx, 50)
}

func (m *manager) logVMSerialConsole(ctx context.Context, log_limit_kb int) (interface{}, error) {
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

		// limit what we write to the last log_limit_kb amount
		blobOffset := 0
		blobLength := blob.Len()

		if blobLength > log_limit_kb*1024 {
			blobOffset = blobLength - (log_limit_kb * 1024)
		}

		logForVM := m.log.WithField("failedRoleInstance", vmName)

		reader := bufio.NewReader(blob)
		_, err = reader.Discard(blobOffset)
		if err != nil {
			items = append(items, fmt.Sprintf("blob storage reader discard on %s: %s", vmName, err))
			continue
		}

		// if we're limiting the logs by kb, then consume once to remove any cut-off messages
		if blobOffset > 0 {
			_, err := reader.ReadString('\n')
			if err != nil {
				items = append(items, fmt.Sprintf("blob storage reading after discard on %s: %s", vmName, err))
				continue
			}
		}

		lastLine := ""

		for {
			line, err := reader.ReadString('\n')

			// trim whitespace
			line = strings.TrimSpace(line)

			// don't print empty lines or duplicates
			if line != "" && line != lastLine {
				lastLine = line
				logForVM.Info(line)
			}

			if err == io.EOF {
				break
			} else if err != nil {
				items = append(items, fmt.Sprintf("blob storage reading on %s: %s", vmName, err))
				break
			}
		}
	}

	return items, nil
}
