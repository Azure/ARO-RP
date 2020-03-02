package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
)

type VirtualMachineScaleSetsClientAddons interface {
	List(ctx context.Context, resourceGroupName string) ([]compute.VirtualMachineScaleSet, error)
	DeleteAndWait(ctx context.Context, resourceGroupName, VMScaleSetName string) error
}

func (c *virtualMachineScaleSetsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, VMScaleSetName string) error {
	future, err := c.Delete(ctx, resourceGroupName, VMScaleSetName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

func (c *virtualMachineScaleSetsClient) List(ctx context.Context, resourceGroupName string) ([]compute.VirtualMachineScaleSet, error) {
	var scaleSets []compute.VirtualMachineScaleSet
	result, err := c.VirtualMachineScaleSetsClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}
	for result.NotDone() {
		scaleSets = append(scaleSets, result.Values()...)
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}

	}
	return scaleSets, nil
}
