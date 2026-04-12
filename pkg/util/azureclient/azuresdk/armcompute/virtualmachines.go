package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// VirtualMachinesClient is a minimal interface for armcompute VirtualMachinesClient,
// used for capacity-reservation-aware VM operations.
type VirtualMachinesClient interface {
	VirtualMachinesClientAddons
}

type virtualMachinesClient struct {
	*armcompute.VirtualMachinesClient
}

var _ VirtualMachinesClient = &virtualMachinesClient{}

// NewDefaultVirtualMachinesClient creates a new VirtualMachinesClient with default options
func NewDefaultVirtualMachinesClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (VirtualMachinesClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	return NewVirtualMachinesClient(subscriptionID, credential, options)
}

// NewVirtualMachinesClient creates a new VirtualMachinesClient
func NewVirtualMachinesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VirtualMachinesClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &virtualMachinesClient{
		VirtualMachinesClient: clientFactory.NewVirtualMachinesClient(),
	}, nil
}
