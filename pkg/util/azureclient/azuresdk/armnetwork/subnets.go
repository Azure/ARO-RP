package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// SubnetsClient is a minimal interface for azure-sdk-for-go subnets client
type SubnetsClient interface {
	Get(ctx context.Context, resourceGroupName, virtualNetworkName, subnetName string, options *armnetwork.SubnetsClientGetOptions) (armnetwork.SubnetsClientGetResponse, error)
	SubnetsClientAddons
}

type subnetsClient struct {
	*armnetwork.SubnetsClient
}

func NewSubnetsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (SubnetsClient, error) {
	client, err := armnetwork.NewSubnetsClient(subscriptionID, credential, options)

	return &subnetsClient{client}, err
}
