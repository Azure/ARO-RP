package virtualmachines

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"golang.org/x/sync/errgroup"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
)

// ListStopped returns a list of VMs within a resource group which are stopped or deallocated
func ListStopped(ctx context.Context, vmClient compute.VirtualMachinesClient, resourceGroupName string) ([]mgmtcompute.VirtualMachine, error) {
	vms, err := vmClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	g, groupCtx := errgroup.WithContext(ctx)
	for idx, vm := range vms {
		idx, vm := idx, vm // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() (err error) {
			vms[idx], err = vmClient.Get(groupCtx, resourceGroupName, *vm.Name, mgmtcompute.InstanceView)
			return
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	stoppedVMs := make([]mgmtcompute.VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		if vm.VirtualMachineProperties == nil ||
			vm.VirtualMachineProperties.InstanceView == nil ||
			vm.VirtualMachineProperties.InstanceView.Statuses == nil {
			continue
		}

		for _, status := range *vm.VirtualMachineProperties.InstanceView.Statuses {
			if status.Code == nil {
				continue
			}

			if IsPowerStatus(*status.Code) {
				if IsStopped(*status.Code) {
					stoppedVMs = append(stoppedVMs, vm)
				}
				break
			}
		}
	}
	return stoppedVMs, nil
}

// IsPowerStatus returns true if the VM status code indicates power state
func IsPowerStatus(statusCode string) bool {
	return strings.HasPrefix(statusCode, "PowerState")
}

// IsStopped returns true if a VM is stopped or deallocated
// Ref: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/states-lifecycle
func IsStopped(statusCode string) bool {
	return statusCode == "PowerState/deallocated" || statusCode == "PowerState/stopped"
}
