package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type ResourceSKUsClient interface {
	ResourceSKUsClientAddons
}

type resourceSKUsClient struct {
	*armcompute.ResourceSKUsClient
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
	clientFactory, err := armcompute.NewClientFactory(subscriptionId, credential, options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewResourceSKUsClient()

	return &resourceSKUsClient{
		ResourceSKUsClient: client,
	}, nil
}
