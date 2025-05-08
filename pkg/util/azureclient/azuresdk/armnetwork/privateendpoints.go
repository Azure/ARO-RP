package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// PrivateEndpointsClient is a minimal interface for azure PrivateEndpointsClient
type PrivateEndpointsClient interface {
	Get(ctx context.Context, resourceGroupName string, privateEndpointName string, options *armnetwork.PrivateEndpointsClientGetOptions) (armnetwork.PrivateEndpointsClientGetResponse, error)
	PrivateEndpointsClientAddons
}

type privateEndpointsClient struct {
	*armnetwork.PrivateEndpointsClient
}

// NewPrivateEndpointsClient creates a new PrivateEndpointsClient
func NewPrivateEndpointsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (PrivateEndpointsClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &privateEndpointsClient{PrivateEndpointsClient: clientFactory.NewPrivateEndpointsClient()}, nil
}
