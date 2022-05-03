package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

// VirtualMachinesClientAddons contains addons for VirtualMachinesClient
type VirtualMachinesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, VMName string, parameters mgmtcompute.VirtualMachine) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, VMName string, forceDeletion *bool) error
	RedeployAndWait(ctx context.Context, resourceGroupName string, VMName string) error
	StartAndWait(ctx context.Context, resourceGroupName string, VMName string) error
	StopAndWait(ctx context.Context, resourceGroupName string, VMName string) error
	List(ctx context.Context, resourceGroupName string) (result []mgmtcompute.VirtualMachine, err error)
}

func (c *virtualMachinesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, VMName string, parameters mgmtcompute.VirtualMachine) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, VMName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, VMName string, forceDeletion *bool) error {
	future, err := c.Delete(ctx, resourceGroupName, VMName, forceDeletion)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) RedeployAndWait(ctx context.Context, resourceGroupName string, VMName string) error {
	future, err := c.Redeploy(ctx, resourceGroupName, VMName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) StartAndWait(ctx context.Context, resourceGroupName string, VMName string) error {
	future, err := c.Start(ctx, resourceGroupName, VMName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) StopAndWait(ctx context.Context, resourceGroupName string, VMName string) error {
	future, err := c.PowerOff(ctx, resourceGroupName, VMName, to.BoolPtr(false))
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) List(ctx context.Context, resourceGroupName string) (result []mgmtcompute.VirtualMachine, err error) {
	page, err := c.VirtualMachinesClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
