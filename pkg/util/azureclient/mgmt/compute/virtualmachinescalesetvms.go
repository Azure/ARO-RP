package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// VirtualMachineScaleSetVMsClient is a minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	VirtualMachineScaleSetVMsClientAddons
	GetInstanceView(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result mgmtcompute.VirtualMachineScaleSetVMInstanceView, err error)
}

type virtualMachineScaleSetVMsClient struct {
	mgmtcompute.VirtualMachineScaleSetVMsClient
}

var _ VirtualMachineScaleSetVMsClient = &virtualMachineScaleSetVMsClient{}

// NewVirtualMachineScaleSetVMsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualMachineScaleSetVMsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetVMsClient {
	client := mgmtcompute.NewVirtualMachineScaleSetVMsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.PollingDuration = time.Hour
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}
