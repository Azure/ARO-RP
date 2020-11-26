package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// VirtualMachinesClient is a minimal interface for azure VirtualMachinesClient
type VirtualMachinesClient interface {
	VirtualMachinesClientAddons
	Get(ctx context.Context, resourceGroupName string, VMName string, expand mgmtcompute.InstanceViewTypes) (result mgmtcompute.VirtualMachine, err error)
}

type virtualMachinesClient struct {
	mgmtcompute.VirtualMachinesClient
}

var _ VirtualMachinesClient = &virtualMachinesClient{}

// NewVirtualMachinesClient creates a new VirtualMachinesClient
func NewVirtualMachinesClient(environment *azure.Environment, subscriptionID string, authorizer autorest.Authorizer) VirtualMachinesClient {
	client := mgmtcompute.NewVirtualMachinesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachinesClient{
		VirtualMachinesClient: client,
	}
}
