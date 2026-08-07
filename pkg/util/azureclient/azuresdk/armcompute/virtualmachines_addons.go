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
	// GetWithInstanceView returns the VM with the InstanceView expansion populated,
	// allowing callers to inspect the VM's power state (e.g. PowerState/running).
	GetWithInstanceView(ctx context.Context, resourceGroupName, vmName string) (armcompute.VirtualMachine, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, vmName string, parameters armcompute.VirtualMachine) error
	// UpdateAndWait issues a PATCH (merge-patch) update — use this instead of CreateOrUpdateAndWait
	// when only a subset of VM properties need to change (e.g. clearing capacityReservationGroup).
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

func (c *virtualMachinesClient) GetWithInstanceView(ctx context.Context, resourceGroupName, vmName string) (armcompute.VirtualMachine, error) {
	expand := armcompute.InstanceViewTypesInstanceView
	resp, err := c.VirtualMachinesClient.Get(ctx, resourceGroupName, vmName, &armcompute.VirtualMachinesClientGetOptions{
		Expand: &expand,
	})
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	return resp.VirtualMachine, nil
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
