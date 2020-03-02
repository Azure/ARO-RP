package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

type VirtualMachineScaleSetsClient interface {
	VirtualMachineScaleSetsClientAddons
}

type virtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}
