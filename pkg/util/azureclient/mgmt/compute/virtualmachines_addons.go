package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
)

// VirtualMachinesClientAddons contains addons for VirtualMachinesClient
type VirtualMachinesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, VMName string, parameters compute.VirtualMachine) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, VMName string) error
}

func (c *virtualMachinesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, VMName string, parameters compute.VirtualMachine) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, VMName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualMachinesClient) DeleteAndWait(ctx context.Context, resourceGroupName string, VMName string) error {
	future, err := c.Delete(ctx, resourceGroupName, VMName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
