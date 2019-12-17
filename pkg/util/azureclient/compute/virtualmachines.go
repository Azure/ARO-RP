package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualMachinesClient is a minimal interface for azure VirtualMachinesClient
type VirtualMachinesClient interface {
	VirtualMachinesClientAddons
}

type virtualMachinesClient struct {
	compute.VirtualMachinesClient
}

var _ VirtualMachinesClient = &virtualMachinesClient{}

// NewVirtualMachinesClient creates a new VirtualMachinesClient
func NewVirtualMachinesClient(subscriptionID string, authorizer autorest.Authorizer) VirtualMachinesClient {
	client := compute.NewVirtualMachinesClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachinesClient{
		VirtualMachinesClient: client,
	}
}
