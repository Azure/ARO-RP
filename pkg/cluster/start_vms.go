package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"golang.org/x/sync/errgroup"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// startVMs checks cluster VMs power state and starts deallocated and stopped VMs, if any
func (m *manager) startVMs(ctx context.Context) error {
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	vms, err := m.virtualMachines.List(ctx, resourceGroupName)
	if err != nil {
		return err
	}

	{
		g, groupCtx := errgroup.WithContext(ctx)
		for i, vm := range vms {
			i, vm := i, vm // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() (err error) {
				vms[i], err = m.virtualMachines.Get(groupCtx, resourceGroupName, *vm.Name, mgmtcompute.InstanceView)
				return
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	vmsToStart := make([]mgmtcompute.VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		if vm.VirtualMachineProperties == nil {
			continue
		}

		if vm.VirtualMachineProperties.InstanceView == nil || vm.VirtualMachineProperties.InstanceView.Statuses == nil {
			continue
		}

		for _, status := range *vm.VirtualMachineProperties.InstanceView.Statuses {
			if status.Code == nil {
				continue
			}

			// Ref: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/states-lifecycle
			if strings.HasPrefix(*status.Code, "PowerState") {
				if *status.Code == "PowerState/deallocated" || *status.Code == "PowerState/stopped" {
					vmsToStart = append(vmsToStart, vm)
				}
				break
			}
		}
	}

	{
		g, groupCtx := errgroup.WithContext(ctx)
		for _, vm := range vmsToStart {
			vm := vm // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				return m.virtualMachines.StartAndWait(groupCtx, resourceGroupName, *vm.Name)
			})
		}
		return g.Wait()
	}
}
