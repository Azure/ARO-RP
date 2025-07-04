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
func (m *manager) LogVMSerialConsole(ctx context.Context) error {
	return m.logVMSerialConsole(ctx, 50)
}

func (m *manager) logVMSerialConsole(ctx context.Context, log_limit_kb int) error {
	if m.virtualMachines == nil {
		m.log.Infof("skipping step")
		return nil
	}

	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vms, err := m.virtualMachines.List(ctx, resourceGroupName)
	if err != nil {
		m.log.WithError(err).Errorf("failed to list VMs in resource group %s", resourceGroupName)
		return nil
	}

	if len(vms) == 0 {
		m.log.Infof("no VMs found in resource group %s", resourceGroupName)
		return nil
	}

	vmNames := make([]string, 0)
	for _, v := range vms {
		j, err := v.MarshalJSON()
		if err != nil {
			m.log.WithError(err).Errorf("failed to marshal VM: %s", *v.Name)
			return err
		} else {
			vmName := "<unknown>"
			if v.Name != nil {
				vmName = *v.Name
				vmNames = append(vmNames, vmName)
			}
			// Replace double quotes with single quotes for better readability in logs
			s := strings.ReplaceAll(string(j), "\"", "'")
			m.log.WithField("failedRoleInstance", vmName).Infof("VM: %s", s)
		}
	}

	// Fetch boot diagnostics URIs for the VMs
	for _, vmName := range vmNames {
		logForVM := m.log.WithField("failedRoleInstance", vmName)
		blob := &bytes.Buffer{}
		err := m.virtualMachines.GetSerialConsoleForVM(ctx, resourceGroupName, vmName, blob)
		if err != nil {
			logForVM.WithError(err).Errorf("vm boot diagnostics retrieval error for %s", vmName)
			continue
		}

		// limit what we write to the last log_limit_kb amount
		blobOffset := 0
		blobLength := blob.Len()

		if blobLength > log_limit_kb*1024 {
			blobOffset = blobLength - (log_limit_kb * 1024)
		}

		reader := bufio.NewReader(blob)
		_, err = reader.Discard(blobOffset)
		if err != nil {
			logForVM.WithError(err).Errorf("blob storage reader discard on %s", vmName)
			continue
		}

		// if we're limiting the logs by kb, then consume once to remove any cut-off messages
		if blobOffset > 0 {
			_, err := reader.ReadString('\n')
			if err != nil {
				logForVM.WithError(err).Errorf("blob storage reading after discard on %s", vmName)
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
				if m.env != nil && m.env.IsCI() {
					fmt.Printf("%s | %s", vmName, line)
				} else {
					logForVM.Info(line)
				}
			}

			if err == io.EOF {
				break
			} else if err != nil {
				logForVM.WithError(err).Errorf("blob storage reading on %s", vmName)
				break
			}
		}
	}

	return nil
}
