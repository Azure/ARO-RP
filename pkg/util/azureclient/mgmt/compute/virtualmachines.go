package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
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
func NewVirtualMachinesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualMachinesClient {
	client := mgmtcompute.NewVirtualMachinesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &virtualMachinesClient{
		VirtualMachinesClient: client,
	}
}
