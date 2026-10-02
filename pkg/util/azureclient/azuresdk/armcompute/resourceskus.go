package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// moduleName/moduleVersion identify this client to ARM telemetry.
const (
	moduleName    = "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	moduleVersion = "v1.0.0"
)

type ResourceSKUsClient interface {
	ResourceSKUsClientAddons
}

type resourceSKUsClient struct {
	armClient      *arm.Client
	subscriptionID string
}

var _ ResourceSKUsClient = &resourceSKUsClient{}

// NewDefaultResourceSKUsClient creates a new ResourceSKUsClient with default options
func NewDefaultResourceSKUsClient(environment *azureclient.AROEnvironment, subscriptionId string, credential azcore.TokenCredential) (ResourceSKUsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}

	return NewResourceSKUsClient(subscriptionId, credential, options)
}

// NewResourceSKUsClient creates a new ResourceSKUsClient
func NewResourceSKUsClient(subscriptionId string, credential azcore.TokenCredential, options *arm.ClientOptions) (ResourceSKUsClient, error) {
	armClient, err := arm.NewClient(moduleName, moduleVersion, credential, options)
	if err != nil {
		return nil, err
	}

	return &resourceSKUsClient{
		armClient:      armClient,
		subscriptionID: subscriptionId,
	}, nil
}
