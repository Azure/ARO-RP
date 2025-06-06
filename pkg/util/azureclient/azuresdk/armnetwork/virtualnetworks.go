package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (vnet armnetwork.VirtualNetworksClientGetResponse, err error)
	GetUsage(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientListUsageOptions) (result []*armnetwork.VirtualNetworkUsage, err error)
}

type virtualNetworksClient struct {
	*armnetwork.VirtualNetworksClient
}

func (v *virtualNetworksClient) GetUsage(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientListUsageOptions) (result []*armnetwork.VirtualNetworkUsage, err error) {
	pager := v.VirtualNetworksClient.NewListUsagePager(resourceGroupName, virtualNetworkName, options)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewVirtualNetworksClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VirtualNetworksClient, error) {
	client, err := armnetwork.NewVirtualNetworksClient(subscriptionID, credential, options)

	return &virtualNetworksClient{client}, err
}
