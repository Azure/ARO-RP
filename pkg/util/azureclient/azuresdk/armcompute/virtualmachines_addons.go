package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

// VirtualMachinesClientAddons is a convenience interface that wraps the SDK VirtualMachinesClient
// with simplified method signatures (blocking pollers, no options parameters).
type VirtualMachinesClientAddons interface {
	Get(ctx context.Context, resourceGroupName, vmName string) (armcompute.VirtualMachine, error)
	List(ctx context.Context, resourceGroupName string) ([]armcompute.VirtualMachine, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, vmName string, parameters armcompute.VirtualMachine) error
	UpdateAndWait(ctx context.Context, resourceGroupName, vmName string, parameters armcompute.VirtualMachineUpdate) error
	DeallocateAndWait(ctx context.Context, resourceGroupName, vmName string) error
	StartAndWait(ctx context.Context, resourceGroupName, vmName string) error
}

func (c *virtualMachinesClient) Get(ctx context.Context, resourceGroupName, vmName string) (armcompute.VirtualMachine, error) {
	resp, err := c.VirtualMachinesClient.Get(ctx, resourceGroupName, vmName, nil)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	return resp.VirtualMachine, nil
}

func (c *virtualMachinesClient) List(ctx context.Context, resourceGroupName string) ([]armcompute.VirtualMachine, error) {
	var result []armcompute.VirtualMachine
	pager := c.NewListPager(resourceGroupName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vm := range page.Value {
			if vm != nil {
				result = append(result, *vm)
			}
		}
	}
	return result, nil
}

func (c *virtualMachinesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, vmName string, parameters armcompute.VirtualMachine) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, vmName, parameters, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *virtualMachinesClient) UpdateAndWait(ctx context.Context, resourceGroupName, vmName string, parameters armcompute.VirtualMachineUpdate) error {
	poller, err := c.BeginUpdate(ctx, resourceGroupName, vmName, parameters, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *virtualMachinesClient) DeallocateAndWait(ctx context.Context, resourceGroupName, vmName string) error {
	poller, err := c.BeginDeallocate(ctx, resourceGroupName, vmName, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *virtualMachinesClient) StartAndWait(ctx context.Context, resourceGroupName, vmName string) error {
	poller, err := c.BeginStart(ctx, resourceGroupName, vmName, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
