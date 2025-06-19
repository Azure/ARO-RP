package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// VirtualMachinesClientAddons contains addons for VirtualMachinesClient
type VirtualMachinesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, VMName string, parameters mgmtcompute.VirtualMachine) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, VMName string, forceDeletion *bool) error
	RedeployAndWait(ctx context.Context, resourceGroupName string, VMName string) error
	StartAndWait(ctx context.Context, resourceGroupName string, VMName string) error
	StopAndWait(ctx context.Context, resourceGroupName string, VMName string, deallocateVM bool) error
	List(ctx context.Context, resourceGroupName string) (result []mgmtcompute.VirtualMachine, err error)
	GetSerialConsoleForVM(ctx context.Context, resourceGroupName string, VMName string, target io.Writer) error
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

func (c *virtualMachinesClient) StopAndWait(ctx context.Context, resourceGroupName string, VMName string, deallocateVM bool) error {
	future, err := c.PowerOff(ctx, resourceGroupName, VMName, pointerutils.ToPtr(false))
	if err != nil {
		return err
	}

	err = future.WaitForCompletionRef(ctx, c.Client)
	if err != nil {
		return err
	}

	if deallocateVM {
		future, deallocErr := c.Deallocate(ctx, resourceGroupName, VMName)
		if deallocErr != nil {
			return deallocErr
		}
		err = future.WaitForCompletionRef(ctx, c.Client)
	}

	return err
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

// retrieveBootDiagnosticsData returns the boot diagnostics data for the given
// VM by RG and VMName.
func (c *virtualMachinesClient) retrieveBootDiagnosticsData(ctx context.Context, resourceGroupName string, VMName string) (serialConsoleURI string, err error) {
	resp, err := c.RetrieveBootDiagnosticsData(ctx, resourceGroupName, VMName, pointerutils.ToPtr(int32(60)))
	if err != nil {
		return "", err
	}

	if resp.SerialConsoleLogBlobURI == nil {
		return "", fmt.Errorf("no available serial console URI")
	}

	return *resp.SerialConsoleLogBlobURI, nil
}

// GetSerialConsoleForVM will return the serial console log blob as an
// io.ReadCloser, or an error if it cannot be retrieved.
func (c *virtualMachinesClient) GetSerialConsoleForVM(ctx context.Context, resourceGroupName string, vmName string, target io.Writer) error {
	serialConsoleLogBlobURI, err := c.retrieveBootDiagnosticsData(ctx, resourceGroupName, vmName)
	if err != nil {
		return fmt.Errorf("failure getting boot diagnostics URI Azure: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serialConsoleLogBlobURI, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failure downloading blob URI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got %d instead of 200 downloading blob URI", resp.StatusCode)
	}

	_, err = io.Copy(target, resp.Body)
	if err != nil {
		return fmt.Errorf("failure copying blob URI body: %w", err)
	}

	return nil
}
