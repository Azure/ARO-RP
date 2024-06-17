package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// PrivateLinkServicesClient is a minimal interface for azure PrivateLinkServicesClient
type PrivateLinkServicesClient interface {
	Get(ctx context.Context, resourceGroupName string, serviceName string, options *armnetwork.PrivateLinkServicesClientGetOptions) (armnetwork.PrivateLinkServicesClientGetResponse, error)
	UpdatePrivateEndpointConnection(ctx context.Context, resourceGroupName string, serviceName string, peConnectionName string, parameters armnetwork.PrivateEndpointConnection, options *armnetwork.PrivateLinkServicesClientUpdatePrivateEndpointConnectionOptions) (armnetwork.PrivateLinkServicesClientUpdatePrivateEndpointConnectionResponse, error)
	PrivateLinkServicesClientAddons
}

type privateLinkServicesClient struct {
	*armnetwork.PrivateLinkServicesClient
}

// NewPrivateLinkServicesClient creates a new PrivateLinkServicesClient
func NewPrivateLinkServicesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (PrivateLinkServicesClient, error) {
	clientFactory, err := armnetwork.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &privateLinkServicesClient{PrivateLinkServicesClient: clientFactory.NewPrivateLinkServicesClient()}, nil
}
