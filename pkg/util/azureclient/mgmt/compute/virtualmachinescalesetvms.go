package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualMachineScaleSetVMsClient is a minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	VirtualMachineScaleSetVMsClientAddons
}

type virtualMachineScaleSetVMsClient struct {
	compute.VirtualMachineScaleSetVMsClient
}

var _ VirtualMachineScaleSetVMsClient = &virtualMachineScaleSetVMsClient{}

// NewVirtualMachineScaleSetVMsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualMachineScaleSetVMsClient(subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetVMsClient {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}
