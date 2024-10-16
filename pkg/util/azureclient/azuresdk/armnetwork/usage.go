package armnetwork

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// UsagesClient is a minimal interface for azure UsageClient
type UsagesClient interface {
	UsageClientAddons
}

type usagesClient struct {
	*armnetwork.UsagesClient
}

// NewUsagesClient creates a new UsageClient
func NewUsagesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (UsagesClient, error) {
	client, err := armnetwork.NewUsagesClient(subscriptionID, credential, options)

	return &usagesClient{client}, err
}
