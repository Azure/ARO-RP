package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
)

type VirtualMachineScaleSetVMsClientAddons interface {
	List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) ([]compute.VirtualMachineScaleSetVM, error)
	RunCommandAndWait(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters compute.RunCommandInput) error
}

func (c *virtualMachineScaleSetVMsClient) RunCommandAndWait(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters compute.RunCommandInput) error {
	future, err := c.VirtualMachineScaleSetVMsClient.RunCommand(ctx, resourceGroupName, VMScaleSetName, instanceID, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) ([]compute.VirtualMachineScaleSetVM, error) {
	var scaleSetsVMs []compute.VirtualMachineScaleSetVM
	result, err := c.VirtualMachineScaleSetVMsClient.List(ctx, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand)
	if err != nil {
		return nil, err
	}

	for result.NotDone() {
		scaleSetsVMs = append(scaleSetsVMs, result.Values()...)
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return scaleSetsVMs, nil
}
