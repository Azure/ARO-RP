package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
)

type VirtualMachineScaleSetVMsClientAddons interface {
	List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) ([]mgmtcompute.VirtualMachineScaleSetVM, error)
	RunCommandAndWait(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters mgmtcompute.RunCommandInput) error
}

func (c *virtualMachineScaleSetVMsClient) RunCommandAndWait(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters mgmtcompute.RunCommandInput) error {
	future, err := c.RunCommand(ctx, resourceGroupName, VMScaleSetName, instanceID, parameters)
	if err != nil {
		return err
	}

	err = future.WaitForCompletionRef(ctx, c.Client)
	if err != nil {
		return err
	}

	if future.Result == nil {
		return nil
	}

	result, err := future.Result(c.VirtualMachineScaleSetVMsClient)
	if err != nil {
		return err
	}

	return runCommandResultError(VMScaleSetName, instanceID, result)
}

// runCommandResultError reports any error the instance raised while running the
// command.
//
// The long-running operation completes successfully whenever the extension
// managed to run the script at all, whatever the script then did, so waiting on
// it establishes only that the command was delivered. Anything the instance has
// to say about the outcome arrives in the result, which was previously
// discarded, leaving a failed command indistinguishable from a successful one.
//
// The scale set and instance are named in the error because both callers run
// commands across many instances, and a failure which does not say where it
// happened leaves the reader to work that out from surrounding log lines.
func runCommandResultError(vmScaleSetName, instanceID string, result mgmtcompute.RunCommandResult) error {
	if result.Value == nil {
		return nil
	}

	var reported []string
	for _, status := range *result.Value {
		if status.Level != mgmtcompute.Error {
			continue
		}

		parts := make([]string, 0, 2)
		for _, s := range []*string{status.Code, status.Message} {
			if s != nil && *s != "" {
				parts = append(parts, *s)
			}
		}
		if len(parts) == 0 {
			parts = append(parts, "no detail reported")
		}

		reported = append(reported, strings.Join(parts, ": "))
	}

	if len(reported) == 0 {
		return nil
	}

	return fmt.Errorf("run command on scale set %s instance %s reported an error: %s", vmScaleSetName, instanceID, strings.Join(reported, "; "))
}

func (c *virtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) ([]mgmtcompute.VirtualMachineScaleSetVM, error) {
	var scaleSetsVMs []mgmtcompute.VirtualMachineScaleSetVM
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
