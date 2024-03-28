package armdns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkdns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ZonesClient is a minimal interface for azure ZonesClient
type ZonesClient interface {
	Get(ctx context.Context, resourceGroupName string, zoneName string, options *sdkdns.ZonesClientGetOptions) (sdkdns.ZonesClientGetResponse, error)
}

type zonesClient struct {
	sdkdns.ZonesClient
}

var _ ZonesClient = &zonesClient{}

// NewZonesClient creates a new ZonesClient
func NewZonesClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) ZonesClient {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	clientFactory, err := sdkdns.NewClientFactory(subscriptionID, credential, &options)
	if err != nil {
		return nil
	}
	client := clientFactory.NewZonesClient()
	return &zonesClient{
		ZonesClient: *client,
	}
}
