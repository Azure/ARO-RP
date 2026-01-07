package armquota

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/quota/armquota/v2"
)

type UsagesClient interface {
	// Get - Get the current usage of a resource.
	// If the operation fails it returns an *azcore.ResponseError type.
	Get(ctx context.Context, resourceName string, scope string, options *armquota.UsagesClientGetOptions) (armquota.UsagesClientGetResponse, error)
}

type usagesClient struct {
	*armquota.UsagesClient
}

var _ UsagesClient = &usagesClient{}

// NewUsageClient creates a new client for azure quota usage
func NewUsagesClient(credential azcore.TokenCredential, options *arm.ClientOptions) (UsagesClient, error) {
	newClient, err := armquota.NewUsagesClient(credential, options)
	if err != nil {
		return nil, err
	}

	return &usagesClient{UsagesClient: newClient}, nil
}

