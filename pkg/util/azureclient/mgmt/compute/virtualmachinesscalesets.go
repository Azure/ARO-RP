package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type VirtualMachineScaleSetsClient interface {
	VirtualMachineScaleSetsClientAddons
}

type virtualMachineScaleSetsClient struct {
	mgmtcompute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetsClient {
	client := mgmtcompute.NewVirtualMachineScaleSetsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}
