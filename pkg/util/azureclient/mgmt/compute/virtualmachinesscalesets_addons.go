package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
)

type VirtualMachineScaleSetsClientAddons interface {
	List(ctx context.Context, resourceGroupName string) ([]mgmtcompute.VirtualMachineScaleSet, error)
	DeleteAndWait(ctx context.Context, resourceGroupName, vmScaleSetName string) error
}

func (c *virtualMachineScaleSetsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, vmScaleSetName string) error {
	future, err := c.VirtualMachineScaleSetsClient.Delete(ctx, resourceGroupName, vmScaleSetName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

func (c *virtualMachineScaleSetsClient) List(ctx context.Context, resourceGroupName string) ([]mgmtcompute.VirtualMachineScaleSet, error) {
	var scaleSets []mgmtcompute.VirtualMachineScaleSet
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
